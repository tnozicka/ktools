package snapshotmanifests

import (
	"context"
	"errors"
	"fmt"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/tnozicka/k8s-controller-lib/pkg/genericclioptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"

	"github.com/tnozicka/ktools/pkg/collect"
)

type SnapshotManifestsOptions struct {
	*genericclioptions.ClientConfig

	OutputDir   string
	KeepGoing   bool
	SkipSecrets bool

	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
}

func NewSnapshotManifestsOptions() *SnapshotManifestsOptions {
	cc := genericclioptions.NewClientConfig("snapshot-manifests")
	// Rely on PaF by default.
	cc.QPS = -1
	cc.Burst = 0
	return &SnapshotManifestsOptions{
		ClientConfig: cc,
		OutputDir:    ".",
		KeepGoing:    true,
		SkipSecrets:  true,
	}
}

func NewSnapshotManifestsCmd() *cobra.Command {
	o := NewSnapshotManifestsOptions()
	cmd := &cobra.Command{
		Use:   "snapshot-manifests",
		Short: "Snapshot all resources from a live Kubernetes cluster.",
		Long: dedent.Dedent(`
		Snapshot all resources from a live Kubernetes cluster into a local directory structure.
		
		(At this point this based on existing collection logic and filters resources only after fetching them from the API.)
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate()
			if err != nil {
				return err
			}

			err = o.Complete()
			if err != nil {
				return err
			}

			err = o.Run(cmd.Context(), cmd)
			if err != nil {
				return err
			}

			return nil
		},

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	o.ClientConfig.AddFlags(cmd.Flags())

	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", o.OutputDir, "Directory to place the collected manifests.")
	cmd.Flags().BoolVarP(&o.KeepGoing, "keep-going", "", o.KeepGoing, "Continue collecting on errors instead of stopping at the first error.")
	cmd.Flags().BoolVarP(&o.SkipSecrets, "skip-secrets", "", o.SkipSecrets, "Skip collecting secrets.")

	return cmd
}

func (o *SnapshotManifestsOptions) Validate() error {
	var errs []error

	errs = append(errs, o.ClientConfig.Validate())

	return errors.Join(errs...)
}

func (o *SnapshotManifestsOptions) Complete() error {
	err := o.ClientConfig.Complete()
	if err != nil {
		return err
	}

	o.kubeClient, err = kubernetes.NewForConfig(o.ProtoConfig)
	if err != nil {
		return fmt.Errorf("can't create kubernetes clientset: %w", err)
	}

	o.dynamicClient, err = dynamic.NewForConfig(o.RestConfig)
	if err != nil {
		return fmt.Errorf("can't create dynamic client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(o.ProtoConfig)
	if err != nil {
		return fmt.Errorf("can't create kubernetes clientset: %w", err)
	}

	o.discoveryClient = clientset.Discovery()

	return nil
}

func (o *SnapshotManifestsOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	cliflag.PrintFlags(cmd.Flags())

	cachedDiscoveryClient := memory.NewMemCacheClient(o.discoveryClient)
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)

	filteredGKs := filteredGroupKinds.Clone()
	if o.SkipSecrets {
		filteredGKs.Insert(schema.GroupKind{
			Group: corev1.SchemeGroupVersion.Group,
			Kind:  "Secret",
		})
	}

	collector := collect.NewDefaultCollector(
		o.dynamicClient,
		o.discoveryClient,
		o.kubeClient.CoreV1(),
		restMapper,
		o.OutputDir,
		true,
		false,
		o.KeepGoing,
	)
	collector.GroupResources = false
	collector.Printers = []collect.PrinterInterface{
		&SnapshotPrinter{
			Delegate: &collect.OmitManagedFieldsPrinter{
				Delegate: &collect.YAMLPrinter{},
			},
			FilterGroupKinds:   filteredGKs,
			filteredNamespaces: sets.Set[string]{},
		},
	}

	clusterScopedResourceInfos, err := collector.DiscoverResources(ctx, func(gv string, r *metav1.APIResource) bool {
		return !r.Namespaced && discovery.SupportsAllVerbs{
			Verbs: []string{"list"},
		}.Match(gv, r)
	})
	if err != nil {
		return fmt.Errorf("can't discover cluster-scoped resources: %w", err)
	}

	clusterScopedResourceInfos, err = collect.ReplaceIsometricResources(clusterScopedResourceInfos)
	if err != nil {
		return fmt.Errorf("can't replace isometric resources: %w", err)
	}

	klog.InfoS("Discovered cluster-scoped resources", "Count", len(clusterScopedResourceInfos))

	var errs []error
	// Collecting related resources will make sure we collect all namespaced objects through the v1.Namespace.
	for _, ri := range clusterScopedResourceInfos {
		klog.V(1).InfoS("Collecting resource", "Resource", ri.GroupVersionResource)
		err = collector.CollectResources(ctx, ri, "")
		if err != nil {
			errs = append(errs, fmt.Errorf("can't collect %s: %w", ri.GroupVersionResource, err))

			if !o.KeepGoing {
				break
			}

			klog.ErrorS(err, "Can't collect resource", "Resource", ri.GroupVersionResource)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("error collecting resources: %w", errors.Join(errs...))
	}

	klog.InfoS("Successfully collected all resources")

	return nil
}
