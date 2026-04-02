package genericclioptions

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
)

type InClusterReflection struct {
	Namespace string
}

func NewInClusterReflection() *InClusterReflection {
	return &InClusterReflection{}
}

func (o *InClusterReflection) AddFlags(flagSet *flag.FlagSet) {
	flagSet.StringVarP(&o.Namespace, "namespace", "", o.Namespace, "Namespace where the program is running. Auto-detected if run inside a cluster.")
}

func (o *InClusterReflection) Validate() error {
	return nil
}

func (o *InClusterReflection) Complete() error {
	if len(o.Namespace) == 0 {
		// Autodetect if running inside a cluster.
		bytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return fmt.Errorf("can't autodetect namespace: %w", err)
		}

		o.Namespace = string(bytes)
	}

	return nil
}
