package genericclioptions

import (
	"fmt"

	flag "github.com/spf13/pflag"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/tnozicka/k8s-controller-lib/pkg/version"
)

func BuildVersionedUserAgent(cmdName string) string {
	return fmt.Sprintf(
		"%s/%s (%s) %s",
		cmdName,
		version.Get().GitVersion,
		version.Get().Platform,
		version.Get().GitCommit,
	)
}

type ClientConfig struct {
	QPS                  float32
	Burst                int
	UserAgentCommandName string
	Kubeconfig           string
	Context              string
	RestConfig           *restclient.Config
	ProtoConfig          *restclient.Config
}

func NewClientConfig(userAgentName string) *ClientConfig {
	return &ClientConfig{
		QPS:                  50,
		Burst:                75,
		UserAgentCommandName: userAgentName,
		Kubeconfig:           "",
		RestConfig:           nil,
		ProtoConfig:          nil,
	}
}

func (cc *ClientConfig) AddFlags(flagSet *flag.FlagSet) {
	flagSet.Float32VarP(&cc.QPS, "qps", "", cc.QPS, "Maximum allowed number of queries per second.")
	flagSet.IntVarP(&cc.Burst, "burst", "", cc.Burst, "Allows extra queries to accumulate when a client is exceeding its rate.")
	flagSet.StringVarP(&cc.Kubeconfig, "kubeconfig", "", cc.Kubeconfig, "Path to the kubeconfig file.")
	flagSet.StringVarP(&cc.Context, "context", "", cc.Context, "The name of the kubeconfig context to use.")
}

func (cc *ClientConfig) Validate() error {
	return nil
}

func (cc *ClientConfig) Complete() error {
	var err error

	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	// Use explicit kubeconfig if set.
	loader.ExplicitPath = cc.Kubeconfig
	cc.RestConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader,
		&clientcmd.ConfigOverrides{
			CurrentContext: cc.Context,
		},
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("can't create client config: %w", err)
	}

	cc.RestConfig.QPS = cc.QPS
	cc.RestConfig.Burst = cc.Burst
	cc.RestConfig.UserAgent = BuildVersionedUserAgent(cc.UserAgentCommandName)

	cc.ProtoConfig = restclient.CopyConfig(cc.RestConfig)
	cc.ProtoConfig.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	cc.ProtoConfig.ContentType = "application/vnd.kubernetes.protobuf"

	return nil
}
