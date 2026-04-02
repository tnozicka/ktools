package naming

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func ObjNN(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

type ObjectWithGroupVersionKind struct {
	schema.GroupVersionKind
	types.NamespacedName
}

func (o ObjectWithGroupVersionKind) String() string {
	groupFmt := "%s."
	if len(o.GroupVersionKind.Group) == 0 {
		groupFmt = "%s"
	}

	return fmt.Sprintf(
		groupFmt+"%s.%s_%s/%s",
		o.GroupVersionKind.Group,
		o.GroupVersionKind.Version,
		o.GroupVersionKind.Kind,
		o.Namespace,
		o.Name,
	)
}

func ObjKindNN(gvk schema.GroupVersionKind, obj metav1.Object) ObjectWithGroupVersionKind {
	return ObjectWithGroupVersionKind{
		GroupVersionKind: gvk,
		NamespacedName:   ObjNN(obj),
	}
}

type ObjectWithGroupVersionResource struct {
	schema.GroupVersionResource
	types.NamespacedName
}

func (o ObjectWithGroupVersionResource) String() string {
	groupFmt := "%s."
	if len(o.GroupVersionResource.Group) == 0 {
		groupFmt = "%s"
	}

	return fmt.Sprintf(
		groupFmt+"%s.%s_%s/%s",
		o.GroupVersionResource.Group,
		o.GroupVersionResource.Version,
		o.GroupVersionResource.Resource,
		o.Namespace,
		o.Name,
	)
}

func ObjResourceNN(gvr schema.GroupVersionResource, obj metav1.Object) ObjectWithGroupVersionResource {
	return ObjectWithGroupVersionResource{
		GroupVersionResource: gvr,
		NamespacedName:       ObjNN(obj),
	}
}

type ConciseGVK schema.GroupVersionKind

func (gvk ConciseGVK) String() string {
	if len(gvk.Group) == 0 {
		return fmt.Sprintf("%s/%s", gvk.Kind, gvk.Version)
	}
	return fmt.Sprintf("%s.%s/%s", gvk.Kind, gvk.Group, gvk.Version)
}

type ConciseGVR schema.GroupVersionResource

func (gvr ConciseGVR) String() string {
	if len(gvr.Group) == 0 {
		return fmt.Sprintf("%s/%s", gvr.Resource, gvr.Version)
	}
	return fmt.Sprintf("%s.%s/%s", gvr.Resource, gvr.Group, gvr.Version)
}
