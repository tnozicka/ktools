package version

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	version "github.com/tnozicka/ktools/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cliflag "k8s.io/component-base/cli/flag"
)

type VersionOptions struct {
	Streams genericclioptions.IOStreams
}

func NewVersionOptions(streams genericclioptions.IOStreams) *VersionOptions {
	return &VersionOptions{
		Streams: streams,
	}
}

func NewVersionCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewVersionOptions(streams)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version info.",
		Long:  "Show version info.",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.Validate()
			if err != nil {
				return err
			}

			err = o.Complete()
			if err != nil {
				return err
			}

			err = o.Run(streams, cmd)
			if err != nil {
				return err
			}

			return nil
		},

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	return cmd
}

func (o *VersionOptions) Validate() error {
	var errs []error

	return errors.Join(errs...)
}

func (o *VersionOptions) Complete() error {
	return nil
}

func (o *VersionOptions) Run(streams genericclioptions.IOStreams, cmd *cobra.Command) error {
	cliflag.PrintFlags(cmd.Flags())

	v := version.Get()

	for _, item := range []struct {
		name, value string
	}{
		{
			name:  "Revision",
			value: version.OptionalToString(v.Revision),
		},
		{
			name:  "RevisionTime",
			value: version.OptionalToString(v.RevisionTime),
		},
		{
			name:  "Modified",
			value: version.OptionalToString(v.Modified),
		},
		{
			name:  "GoVersion",
			value: version.OptionalToString(v.GoVersion),
		},
	} {
		_, err := fmt.Fprintf(streams.Out, "%s: %v\n", item.name, item.value)
		if err != nil {
			return fmt.Errorf("can't print field %q: %w", item.name, err)
		}
	}

	return nil
}
