package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/tnozicka/ktools/pkg/cmd/snapshotmanifests"
	"github.com/tnozicka/ktools/pkg/cmd/splitmanifests"
	"github.com/tnozicka/ktools/pkg/cmd/version"
)

func NewKToolCommand(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ktool",
		Short: "Run the ktool.",
		Long:  `Run the ktool.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(version.NewVersionCmd(streams))
	cmd.AddCommand(splitmanifests.NewSplitManifestsCmd(streams))
	cmd.AddCommand(snapshotmanifests.NewSnapshotManifestsCmd())

	return cmd
}
