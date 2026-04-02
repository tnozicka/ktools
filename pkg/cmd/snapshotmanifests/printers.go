package snapshotmanifests

import (
	"fmt"
	"regexp"

	"github.com/tnozicka/k8s-controller-lib/pkg/kubetypes"
	"github.com/tnozicka/k8s-controller-lib/pkg/naming"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/tnozicka/ktools/pkg/collect"
)

type groupKindObj struct {
	schema.GroupKind
	Name string
}

var (
	filteredGroupKinds = sets.New(
		schema.GroupKind{
			Group: "",
			Kind:  "ComponentStatus",
		},
		schema.GroupKind{
			Group: "",
			Kind:  "Node",
		},
		schema.GroupKind{
			Group: "",
			Kind:  "Endpoints",
		},
		schema.GroupKind{
			Group: "discovery.k8s.io",
			Kind:  "EndpointSlice",
		},
		schema.GroupKind{
			Group: "events.k8s.io",
			Kind:  "Event",
		},
		schema.GroupKind{
			Group: "metrics.k8s.io",
			Kind:  "NodeMetrics",
		},
		schema.GroupKind{
			Group: "metrics.k8s.io",
			Kind:  "PodMetrics",
		},
		// Leases are created on demand in kube machinery.
		schema.GroupKind{
			Group: "coordination.k8s.io",
			Kind:  "Lease",
		},
		schema.GroupKind{
			Group: "cloud.google.com",
			Kind:  "ComputeClass",
		},
		schema.GroupKind{
			Group: "networking.k8s.io",
			Kind:  "ServiceCIDR",
		},
	)
	crdFilterRegex  = regexp.MustCompile(`^.*(\.gke\.io)$`)
	filteredObjects = sets.New(
		groupKindObj{
			GroupKind: corev1.SchemeGroupVersion.WithKind("ConfigMap").GroupKind(),
			Name:      "kube-root-ca.crt",
		},
		groupKindObj{
			GroupKind: corev1.SchemeGroupVersion.WithKind("ServiceAccount").GroupKind(),
			Name:      "default",
		},
	)
	managedLabels = sets.New(
		"addonmanager.kubernetes.io/mode",
		"kube-aggregator.kubernetes.io/automanaged",
		"ipaddress.kubernetes.io/managed-by",
		"kubernetes.io/bootstrapping",
		"networking.gke.io/common-webhooks",
		"istio.io/config",
	)
	managedAnnotations = sets.New(
		"components.gke.io/component-name",
		"operator.prometheus.io/version",
		"apf.kubernetes.io/autoupdate-spec",
	)
)

type SnapshotPrinter struct {
	Delegate           collect.PrinterInterface
	FilterGroupKinds   sets.Set[schema.GroupKind]
	filteredNamespaces sets.Set[string]
}

var _ collect.PrinterInterface = &SnapshotPrinter{}

func (p *SnapshotPrinter) isManaged(metaObj metav1.Object) (bool, error) {
	for l := range metaObj.GetLabels() {
		if managedLabels.Has(l) {
			return true, nil
		}
	}

	for a := range metaObj.GetAnnotations() {
		if managedAnnotations.Has(a) {
			return true, nil
		}
	}

	return false, nil
}

func (p *SnapshotPrinter) PrintObject(resourceInfo *collect.ResourceInfo, obj kubetypes.Object) ([]byte, bool, error) {
	if obj == nil {
		return p.Delegate.PrintObject(resourceInfo, obj)
	}

	if p.filteredNamespaces.Has(obj.GetNamespace()) {
		klog.V(4).InfoS("Skipping object in filtered namespace", "Ref", naming.ObjNN(obj))
		return nil, false, nil
	}

	data, shouldPrint, err := p.printObject(resourceInfo, obj)
	if err != nil {
		return nil, false, err
	}

	if !shouldPrint && obj.GetObjectKind().GroupVersionKind().GroupKind() == corev1.SchemeGroupVersion.WithKind("Namespace").GroupKind() {
		p.filteredNamespaces.Insert(obj.GetName())
	}

	return data, shouldPrint, nil
}

func (p *SnapshotPrinter) printObject(resourceInfo *collect.ResourceInfo, obj kubetypes.Object) ([]byte, bool, error) {
	if obj.GetDeletionTimestamp() != nil {
		klog.V(4).InfoS("Skipping deleted object", "Ref", naming.ObjNN(obj))
		return nil, false, nil
	}

	if metav1.GetControllerOf(obj) != nil {
		klog.V(4).InfoS("Skipping managed object", "Ref", naming.ObjNN(obj))
		return nil, false, nil
	}

	a, err := meta.Accessor(obj)
	if err != nil {
		return nil, false, err
	}

	// Filter out managed objects from known components that don't have controllerRef.
	isManaged, err := p.isManaged(a)
	if err != nil {
		return nil, false, fmt.Errorf("can't check if object managed: %w", err)
	}
	if isManaged {
		klog.V(4).InfoS("Skipping managed object", "Ref", naming.ObjNN(obj))
		return nil, false, nil
	}

	a.SetManagedFields(nil)
	a.SetGeneration(0)
	a.SetResourceVersion("")
	a.SetCreationTimestamp(metav1.Time{})
	a.SetUID("")

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, false, fmt.Errorf("can't convert object %T to Unstructured: %w", obj, err)
	}

	unstructured.RemoveNestedField(u, "status")

	obj = &unstructured.Unstructured{
		Object: u,
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	gko := groupKindObj{
		GroupKind: gvk.GroupKind(),
		Name:      obj.GetName(),
	}

	if p.FilterGroupKinds.Has(gko.GroupKind) {
		klog.V(4).InfoS("Skipping filtered GroupKind", "Ref", naming.ObjKindNN(gvk, obj))
		return nil, false, nil
	}

	if filteredObjects.Has(gko) {
		klog.V(4).InfoS("Skipping filtered object", "Ref", naming.ObjKindNN(gvk, obj))
		return nil, false, nil
	}

	if gvk.GroupKind() == apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition").GroupKind() &&
		crdFilterRegex.MatchString(obj.GetName()) {
		klog.V(4).InfoS("Skipping filtered CRD", "Ref", naming.ObjKindNN(gvk, obj))
		return nil, false, nil
	}

	return p.Delegate.PrintObject(resourceInfo, obj)
}

func (p *SnapshotPrinter) GetExtension() string {
	return p.Delegate.GetExtension()
}

func (p *SnapshotPrinter) GetPrinterName() string {
	return p.Delegate.GetPrinterName()
}
