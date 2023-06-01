package operator

import (
	"github.com/spf13/cobra"
	"github.com/tnozicka/ktools/pkg/cmd/splitmanifests"
	"github.com/tnozicka/ktools/pkg/cmd/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewKToolCommand(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ktool",
		Short: "Run the ktool.",
		Long:  `Run the ktool.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(version.NewVersionCmd(streams))
	cmd.AddCommand(splitmanifests.NewSplitManifestsCmd(streams))

	return cmd
}
