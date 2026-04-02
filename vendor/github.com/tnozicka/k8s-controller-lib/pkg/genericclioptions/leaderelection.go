package genericclioptions

import (
	"context"
	"time"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"

	clleaderelection "github.com/tnozicka/k8s-controller-lib/pkg/leaderelection"
)

type LeaderElection struct {
	LeaderElectionLeaseDuration time.Duration
	LeaderElectionRenewDeadline time.Duration
	LeaderElectionRetryPeriod   time.Duration
	WatchDog                    *leaderelection.HealthzAdaptor

	programName string
}

func NewLeaderElection(
	programName string,
) *LeaderElection {
	return &LeaderElection{
		programName: programName,
		WatchDog:    leaderelection.NewLeaderHealthzAdaptor(time.Second * 20),

		LeaderElectionLeaseDuration: 60 * time.Second,
		LeaderElectionRenewDeadline: 35 * time.Second,
		LeaderElectionRetryPeriod:   10 * time.Second,
	}
}

func (le *LeaderElection) AddFlags(flagSet *flag.FlagSet) {
	flagSet.DurationVar(&le.LeaderElectionLeaseDuration, "leader-election-lease-duration", le.LeaderElectionLeaseDuration, "LeaseDuration is the duration that non-leader candidates will wait to force acquire leadership.")
	flagSet.DurationVar(&le.LeaderElectionRenewDeadline, "leader-election-renew-deadline", le.LeaderElectionRenewDeadline, "RenewDeadline is the duration that the acting master will retry refreshing leadership before giving up.")
	flagSet.DurationVar(&le.LeaderElectionRetryPeriod, "leader-election-retry-period", le.LeaderElectionRetryPeriod, "RetryPeriod is the duration the LeaderElector clients should wait between tries of actions.")
}

func (le *LeaderElection) Validate() error {
	return nil
}

func (le *LeaderElection) Complete() error {
	return nil
}

func (le *LeaderElection) Run(
	programCtx context.Context,
	programName string,
	lease types.NamespacedName,
	client kubernetes.Interface,
	f func(context.Context) error,
) error {
	return clleaderelection.LeaderElectAndRun(
		programCtx,
		programName,
		lease,
		client,
		le.LeaderElectionLeaseDuration,
		le.LeaderElectionRenewDeadline,
		le.LeaderElectionRetryPeriod,
		le.WatchDog,
		f,
	)
}
