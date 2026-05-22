package main

import (
	"os"

	cmd "github.com/tnozicka/ktools/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/component-base/cli"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/tnozicka/k8s-controller-lib/pkg/leaderelection"
	"github.com/tnozicka/k8s-controller-lib/pkg/signals"
)

func main() {
	leaderelection.ElectionLostCallback = func() {
		klog.Flush()
	}

	command := cmd.NewKToolCommand(genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	command.SetContext(signals.GetContext())

	err := cli.RunNoErrOutput(command)
	if err != nil {
		cmdutil.CheckErr(err)
	}
}
