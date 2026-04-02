package leaderelection

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

const (
	// terminationGracePeriod is the time given to the program to gracefully exit
	// before a forceful termination is initiated.
	terminationGracePeriod = 10 * time.Second

	// forcefulShutdownTimeout is the time given to the emergency sequence and callbacks to finish
	// before the program is forcefully terminated.
	forcefulShutdownTimeout = 5 * time.Second
)

var ElectionLostCallback func()

type ElectionLostError struct{}

var _ error = &ElectionLostError{}

func (e *ElectionLostError) Error() string {
	return "leader election lost"
}

// LeaderElectAndRun runs the given function while holding the leader lease.
// Graceful termination and releasing the lock happen simultaneously, so the other instance can take over
// as soon as possible. This means that in the absolute worst case both can run in parallel for a short period of time,
// but the instance with canceled context is unlikely to do any meaningful action. On the other hand, it mitigates
// the risk of a single instance having running leader election but nonoperational controllers.
func LeaderElectAndRun(
	programCtx context.Context,
	programName string,
	lease types.NamespacedName,
	client kubernetes.Interface,
	leaderelectionLeaseDuration time.Duration,
	leaderelectionRenewDeadline time.Duration,
	leaderelectionRetryPeriod time.Duration,
	watchDog *leaderelection.HealthzAdaptor,
	f func(context.Context) error,
) error {
	leCtx, leCtxCancel := context.WithCancelCause(programCtx)
	defer leCtxCancel(nil)

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	// Add a unique identifier so that two processes on the same host don't accidentally both become active.
	id := hostname + "_" + string(uuid.NewUUID())
	klog.V(4).Infof("Leader election ID is %q", id)

	resourceLock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		lease.Namespace,
		lease.Name,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		},
	)
	if err != nil {
		return fmt.Errorf("can't create resource lock: %w", err)
	}

	var fErr error
	var finishedLeading atomic.Bool
	leConfig := leaderelection.LeaderElectionConfig{
		Lock:            resourceLock,
		LeaseDuration:   leaderelectionLeaseDuration,
		RenewDeadline:   leaderelectionRenewDeadline,
		RetryPeriod:     leaderelectionRetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				defer finishedLeading.Store(true)
				// Cancel the leader election (if not already canceled).
				// (This is important for the case where we say exit on error and not because of a leader election loss.)
				defer leCtxCancel(nil)

				fErr = f(ctx)
				if fErr != nil {
					// We are returning fErr at the end, but it's better to log it in case we fail along the way.
					klog.ErrorS(fErr, "Failed to run leader elected application.")
				}
			},
			OnStoppedLeading: func() {
				// Set up a forceful termination after the grace period but still
				// try to flush the logs and call the custom callback.
				time.AfterFunc(terminationGracePeriod, func() {
					// Add failsafe to ensure the forceful shutdown finishes at all cost.
					time.AfterFunc(forcefulShutdownTimeout, func() {
						os.Exit(254)
					})

					if ElectionLostCallback != nil {
						klog.V(4).InfoS("Executing custom leader election lost callback.")
						klog.Flush()

						ElectionLostCallback()
					}

					klog.InfoS("Exiting because of leader election loss.")
					klog.Flush()
					os.Exit(255)
				})

				// Start terminating our controllers early (if they are still running).
				if !finishedLeading.Load() {
					klog.InfoS("Leader election lost, initiating emergency shutdown.", "GracePeriod", terminationGracePeriod)
					leCtxCancel(&ElectionLostError{})
				} else {
					leCtxCancel(nil)
				}
			},
		},
		Name:     programName,
		WatchDog: watchDog,
	}
	le, err := leaderelection.NewLeaderElector(leConfig)
	if err != nil {
		return fmt.Errorf("can't create leaderelector: %w", err)
	}

	klog.InfoS("Starting leader election",
		"Name", programName,
		"Lease", lease,
		"Identity", id,
		"LeaseDuration", leConfig.LeaseDuration,
		"RenewDeadline", leConfig.RenewDeadline,
		"RetryPeriod", leConfig.RetryPeriod,
	)
	le.Run(leCtx)

	return fErr
}
