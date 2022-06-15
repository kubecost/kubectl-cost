package main

import (
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/rs/zerolog"
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
	logLevel := flags.String("log-level", "info", "Set the log level. Options: 'trace', 'debug', 'info', 'warn', 'error'.")
	if logLevel == nil {
	} else if *logLevel == "trace" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if *logLevel == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if *logLevel == "info" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if *logLevel == "warn" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else if *logLevel == "error" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
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
