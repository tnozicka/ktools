package leaderelect

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/tnozicka/k8s-controller-lib/pkg/genericclioptions"
	"k8s.io/apimachinery/pkg/types"
	cliruntime "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

const programName = "run-leader-elected"

type RunLeaderElectedOptions struct {
	*genericclioptions.ClientConfig
	*genericclioptions.InClusterReflection
	*genericclioptions.LeaderElection

	streams cliruntime.IOStreams

	name    string
	cmd     string
	cmdArgs []string

	kubeClient kubernetes.Interface
}

func NewRunLeaderElectedOptions(streams cliruntime.IOStreams) *RunLeaderElectedOptions {
	return &RunLeaderElectedOptions{
		ClientConfig:        genericclioptions.NewClientConfig(programName),
		InClusterReflection: genericclioptions.NewInClusterReflection(),
		LeaderElection:      genericclioptions.NewLeaderElection(programName),
		streams:             streams,
	}
}

func NewRunLeaderElectedCmd(streams cliruntime.IOStreams) *cobra.Command {
	o := NewRunLeaderElectedOptions(streams)
	cmd := &cobra.Command{
		Use:   "run-leader-elected [--namespace <namespace>] <name> -- <command> [args...]",
		Short: "Run a command while holding a Kubernetes Lease.",
		Long: dedent.Dedent(`
		Acquires the Kubernetes Lease and, while holding it, runs the
		given command as a child process.

		If a leadership is lost it will kill the child process.
		`),
		Args: o.ValidateArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate()
			if err != nil {
				return err
			}

			err = o.Complete(args)
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
	o.InClusterReflection.AddFlags(cmd.Flags())
	o.LeaderElection.AddFlags(cmd.Flags())

	return cmd
}

func (o *RunLeaderElectedOptions) Validate() error {
	var errs []error

	errs = append(errs, o.ClientConfig.Validate())
	errs = append(errs, o.InClusterReflection.Validate())
	errs = append(errs, o.LeaderElection.Validate())

	return errors.Join(errs...)
}

func (o *RunLeaderElectedOptions) ValidateArgs(cmd *cobra.Command, args []string) error {
	if cmd.ArgsLenAtDash() != 1 {
		return fmt.Errorf("expected exactly one positional argument <name> before \"--\"")
	}

	if len(args) < 2 {
		return fmt.Errorf("expected a command after \"--\"")
	}

	return nil
}

func (o *RunLeaderElectedOptions) Complete(args []string) error {
	err := o.ClientConfig.Complete()
	if err != nil {
		return err
	}

	err = o.InClusterReflection.Complete()
	if err != nil {
		return err
	}

	err = o.LeaderElection.Complete()
	if err != nil {
		return err
	}

	o.name = args[0]
	o.cmd = args[1]
	o.cmdArgs = args[2:]

	o.kubeClient, err = kubernetes.NewForConfig(o.ProtoConfig)
	if err != nil {
		return fmt.Errorf("can't create kubernetes clientset: %w", err)
	}

	return nil
}

func (o *RunLeaderElectedOptions) Run(ctx context.Context, cmd *cobra.Command) error {
	cliflag.PrintFlags(cmd.Flags())

	lease := types.NamespacedName{Namespace: o.InClusterReflection.Namespace, Name: o.name}

	return o.LeaderElection.Run(
		ctx,
		programName,
		lease,
		o.kubeClient,
		func(leaderCtx context.Context) error {
			klog.InfoS("Acquired leadership; starting child process",
				"Lease", lease.String(),
				"Cmd", o.cmd,
				"Args", o.cmdArgs,
			)

			child := exec.CommandContext(leaderCtx, o.cmd, o.cmdArgs...)
			child.Stdin = o.streams.In
			child.Stdout = o.streams.Out
			child.Stderr = o.streams.ErrOut

			runErr := child.Run()
			klog.InfoS("Child process exited",
				"Err", runErr,
				"Cause", context.Cause(leaderCtx),
			)
			return runErr
		},
	)
}
