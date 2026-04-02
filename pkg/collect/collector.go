package collect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tnozicka/k8s-controller-lib/pkg/kubetypes"
	clnaming "github.com/tnozicka/k8s-controller-lib/pkg/naming"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

const (
	clusterScopeDirName = "cluster-scoped"
	namespacesDirName   = "namespaces"
)

type ResourceInfo struct {
	meta.RESTScope
	schema.GroupVersionResource
	meta.RESTMapper
}

type Collector struct {
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	coreClient      corev1client.CoreV1Interface
	restMapper      meta.RESTMapper

	Printers []PrinterInterface

	collectedResources sets.Set[clnaming.ObjectWithGroupVersionResource]
	destDir            string
	relatedResources   bool
	collectLogs        bool
	GroupResources     bool
	keepGoing          bool
	logsLimitBytes     int64
}

func NewDefaultCollector(
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	coreClient corev1client.CoreV1Interface,
	restMapper meta.RESTMapper,
	destDir string,
	relatedResources bool,
	collectLogs bool,
	keepGoing bool,
) *Collector {
	return &Collector{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		coreClient:      coreClient,
		restMapper:      restMapper,
		Printers: []PrinterInterface{
			&OmitManagedFieldsPrinter{
				Delegate: &YAMLPrinter{},
			},
		},
		collectedResources: sets.Set[clnaming.ObjectWithGroupVersionResource]{},
		destDir:            destDir,
		relatedResources:   relatedResources,
		collectLogs:        collectLogs,
		GroupResources:     true,
		keepGoing:          keepGoing,
		logsLimitBytes:     0,
	}
}

func (c *Collector) getResourceLocation(obj metav1.Object, resourceInfo *ResourceInfo) (string, error) {
	gvk, err := c.restMapper.KindFor(resourceInfo.GroupVersionResource)
	if err != nil {
		return "", fmt.Errorf("can't get kind for %q: %w", resourceInfo.GroupVersionResource, err)
	}

	gkString := strings.ToLower(gvk.GroupKind().String())

	var resourceSubDir, resourceFileName string
	if c.GroupResources {
		resourceSubDir = gkString
		resourceFileName = obj.GetName()
	} else {
		resourceSubDir = ""
		resourceFileName = fmt.Sprintf("%s.%s", obj.GetName(), gkString)
	}

	scope := resourceInfo.RESTScope.Name()
	switch scope {
	case meta.RESTScopeNameRoot:
		return filepath.Join(
			c.destDir,
			clusterScopeDirName,
			resourceSubDir,
			resourceFileName,
		), nil

	case meta.RESTScopeNameNamespace:
		return filepath.Join(
			c.destDir,
			namespacesDirName,
			obj.GetNamespace(),
			resourceSubDir,
			resourceFileName,
		), nil

	default:
		return "", fmt.Errorf("unknown scope %q", scope)
	}
}

func (c *Collector) writeObject(ctx context.Context, resourceInfo *ResourceInfo, obj kubetypes.Object) error {
	resourceLocation, err := c.getResourceLocation(obj, resourceInfo)
	if err != nil {
		return fmt.Errorf("can't get resourceDir: %q", err)
	}

	objWithGVR := clnaming.ObjResourceNN(resourceInfo.GroupVersionResource, obj)
	var dirCreated bool
	var shouldPrint bool
	for _, printer := range c.Printers {
		filePath := resourceLocation + printer.GetExtension()

		var bytes []byte
		bytes, shouldPrint, err = printer.PrintObject(resourceInfo, obj)
		if err != nil {
			return fmt.Errorf("can't print object %q with printer %q: %w", objWithGVR, printer.GetPrinterName(), err)
		}

		if !shouldPrint {
			continue
		}

		if !dirCreated {
			resourceDir := filepath.Dir(resourceLocation)
			err = os.MkdirAll(resourceDir, 0770)
			if err != nil {
				return fmt.Errorf("can't create resource dir %q: %w", resourceDir, err)
			}
			dirCreated = true
		}

		err = os.WriteFile(filePath, bytes, 0770)
		if err != nil {
			return fmt.Errorf("can't write file %q: %w", filePath, err)
		}

		klog.V(4).InfoS(
			"Wrote object",
			"Ref", objWithGVR,
			"Path", filePath,
			"Printer", printer.GetPrinterName(),
		)
	}

	return nil
}

func (c *Collector) collectObject(
	ctx context.Context,
	obj kubetypes.Object,
	resourceInfo *ResourceInfo,
) error {
	err := c.writeObject(ctx, resourceInfo, obj)
	if err != nil {
		return fmt.Errorf("can't write object: %w", err)
	}

	return nil
}

func (c *Collector) CollectObject(ctx context.Context, u *unstructured.Unstructured, resourceInfo *ResourceInfo) error {
	objWithGVR := clnaming.ObjResourceNN(resourceInfo.GroupVersionResource, u)
	if c.collectedResources.Has(objWithGVR) {
		return nil
	}
	c.collectedResources.Insert(objWithGVR)

	switch resourceInfo.GroupVersionResource.GroupResource() {
	case corev1.SchemeGroupVersion.WithResource("secrets").GroupResource():
		return c.collectSecret(ctx, resourceInfo, u)

	case corev1.SchemeGroupVersion.WithResource("namespaces").GroupResource():
		return c.collectNamespace(ctx, resourceInfo, u)

	case corev1.SchemeGroupVersion.WithResource("pods").GroupResource():
		return c.collectPod(ctx, resourceInfo, u)

	default:
		return c.collectObject(ctx, u, resourceInfo)
	}
}

func (c *Collector) CollectResource(ctx context.Context, resourceInfo *ResourceInfo, namespace, name string) error {
	obj, err := c.dynamicClient.Resource(resourceInfo.GroupVersionResource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't get resource %q: %w", resourceInfo.Resource, err)
	}

	return c.CollectObject(ctx, obj, resourceInfo)
}

func (c *Collector) CollectResources(ctx context.Context, resourceInfo *ResourceInfo, namespace string) error {
	l, err := c.dynamicClient.Resource(resourceInfo.GroupVersionResource).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("can't list resource %q: %w", resourceInfo.GroupVersionResource, err)
	}

	var errs []error
	for _, obj := range l.Items {
		err = c.CollectObject(ctx, &obj, resourceInfo)
		if err != nil {
			errs = append(errs, err)

			if !c.keepGoing {
				break
			}

			// If we keep going, we should surface the error for early feedback.
			// The error will also be processed the regular way at the end.
			klog.ErrorS(err, "can't collect object", "Object", klog.KObj(&obj))
		}
	}

	return errors.Join(errs...)
}
