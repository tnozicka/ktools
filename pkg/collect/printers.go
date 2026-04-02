package collect

import (
	"fmt"

	"github.com/tnozicka/k8s-controller-lib/pkg/kubetypes"
	clnaming "github.com/tnozicka/k8s-controller-lib/pkg/naming"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/yaml"
)

type PrinterInterface interface {
	PrintObject(resourceInfo *ResourceInfo, obj kubetypes.Object) ([]byte, bool, error)
	GetExtension() string
	GetPrinterName() string
}

type OmitManagedFieldsPrinter struct {
	Delegate PrinterInterface
}

var _ PrinterInterface = &OmitManagedFieldsPrinter{}

func (p *OmitManagedFieldsPrinter) PrintObject(resourceInfo *ResourceInfo, obj kubetypes.Object) ([]byte, bool, error) {
	if obj == nil {
		return p.Delegate.PrintObject(resourceInfo, obj)
	}

	a, err := meta.Accessor(obj)
	if err != nil {
		return nil, false, err
	}

	a.SetManagedFields(nil)

	return p.Delegate.PrintObject(resourceInfo, obj)
}

func (p *OmitManagedFieldsPrinter) GetExtension() string {
	return p.Delegate.GetExtension()
}

func (p *OmitManagedFieldsPrinter) GetPrinterName() string {
	return p.Delegate.GetPrinterName()
}

type YAMLPrinter struct{}

var _ PrinterInterface = &YAMLPrinter{}

func (p *YAMLPrinter) PrintObject(resourceInfo *ResourceInfo, obj kubetypes.Object) ([]byte, bool, error) {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return nil, false, fmt.Errorf("can't marshal object %q into yaml: %w", clnaming.ObjResourceNN(resourceInfo.GroupVersionResource, obj), err)
	}

	return bytes, true, nil
}

func (p *YAMLPrinter) GetExtension() string {
	return ".yaml"
}

func (p *YAMLPrinter) GetPrinterName() string {
	return "yaml"
}
