package main

import (
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/pflag"

	"github.com/kubecost/kubectl-cost/pkg/cmd"
)

// The following are set by https://github.com/ahmetb/govvv during
// build with linker flags. Should be replaced with different logic
// once https://github.com/golang/go/issues/37475 is complete and
// available.
var GitCommit string
var GitBranch string
var GitState string
var GitSummary string
var BuildDate string

type assetsQuery struct {
	field1 string
}

func main() {
	flags := pflag.NewFlagSet("kubectl-ns", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdCost(
		genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		GitCommit,
		GitBranch,
		GitState,
		GitSummary,
		BuildDate,
	)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}

}
