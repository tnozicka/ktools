package collect

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	clnaming "github.com/tnozicka/k8s-controller-lib/pkg/naming"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

// ReplaceIsometricResources removes old isometric resources if their newer variant is present.
// The replacement is "one-pass" and order-dependent.
func ReplaceIsometricResources(resourceInfos []*ResourceInfo) ([]*ResourceInfo, error) {
	replacements := []struct {
		old schema.GroupVersionResource
		new schema.GroupVersionResource
	}{
		{
			old: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "events",
			},
			new: schema.GroupVersionResource{
				Group:    "events.k8s.io",
				Version:  "v1",
				Resource: "events",
			},
		},
	}

	resourceInfosMap := make(map[schema.GroupVersionResource]*ResourceInfo, len(resourceInfos))
	for _, m := range resourceInfos {
		resourceInfosMap[m.GroupVersionResource] = m
	}

	for _, replacement := range replacements {
		_, found := resourceInfosMap[replacement.new]
		if found {
			delete(resourceInfosMap, replacement.old)
		}
	}

	return slices.Collect(maps.Values(resourceInfosMap)), nil
}

func (c *Collector) DiscoverResources(ctx context.Context, filter discovery.ResourcePredicateFunc) ([]*ResourceInfo, error) {
	all, err := c.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("can't discover preferred resources: %w", err)
	}

	resourceLists := discovery.FilteredBy(filter, all)

	resourceInfos := make([]*ResourceInfo, 0, len(resourceLists))
	for _, rl := range resourceLists {
		var gv schema.GroupVersion
		gv, err = schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("can't parse gv %q: %w", rl.GroupVersion, err)
		}

		for _, r := range rl.APIResources {
			var scope meta.RESTScope
			if r.Namespaced {
				scope = meta.RESTScopeNamespace
			} else {
				scope = meta.RESTScopeRoot
			}
			resourceInfos = append(resourceInfos, &ResourceInfo{
				GroupVersionResource: gv.WithResource(r.Name),
				RESTScope:            scope,
				RESTMapper:           c.restMapper,
			})
		}
	}

	return resourceInfos, nil
}

func (c *Collector) collectNamespace(ctx context.Context, resourceInfo *ResourceInfo, u *unstructured.Unstructured) error {
	err := c.writeObject(ctx, resourceInfo, u)
	if err != nil {
		return fmt.Errorf("can't write namespace %q: %w", clnaming.ObjResourceNN(resourceInfo.GroupVersionResource, u), err)
	}

	if !c.relatedResources {
		return nil
	}

	// TODO: The discovery should be cached / global for all namespaces.
	namespacedResourceInfos, err := c.DiscoverResources(ctx, func(gv string, r *metav1.APIResource) bool {
		return r.Namespaced && discovery.SupportsAllVerbs{
			Verbs: []string{"list"},
		}.Match(gv, r)
	})
	if err != nil {
		return fmt.Errorf("can't discover resource: %w", err)
	}

	// Filter out kube resources that share storage across API groups.
	namespacedResourceInfos, err = ReplaceIsometricResources(namespacedResourceInfos)
	if err != nil {
		return fmt.Errorf("can't repalce isometric resources: %w", err)
	}

	namespace := u.GetName()
	var errs []error
	for _, m := range namespacedResourceInfos {
		err = c.CollectResources(ctx, m, namespace)
		if err != nil {
			errs = append(errs, fmt.Errorf("can't collect %s in namespace %s: %w", m.Resource, namespace, err))

			if !c.keepGoing {
				break
			}

			klog.Error(err, "Can't collect resource", "Resource", m.Resource, "Namespace", namespace)
		}
	}

	return errors.Join(errs...)
}
