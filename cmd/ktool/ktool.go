package main

import (
	"os"

	cmd "github.com/tnozicka/ktools/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/component-base/cli"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	command := cmd.NewKToolCommand(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	err := cli.RunNoErrOutput(command)
	if err != nil {
		cmdutil.CheckErr(err)
	}
}
