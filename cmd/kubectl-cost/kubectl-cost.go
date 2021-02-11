package main

import (
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/pflag"

	"github.com/kubecost/kubectl-cost/pkg/cmd"
)

type assetsQuery struct {
	field1 string
}

func main() {
	flags := pflag.NewFlagSet("kubectl-ns", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdCost(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}

}
