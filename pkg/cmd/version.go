package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const versionFormat = `kubectl cost version info
    Git Commit:   %s
    Git Branch:   %s
    Git State:    %s
    Git Summary:  %s
    Build Date:   %s
`

func newCmdVersion(
	streams genericclioptions.IOStreams,
	GitCommit string,
	GitBranch string,
	GitState string,
	GitSummary string,
	BuildDate string,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "view installed version of kubectl cost",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(streams.ErrOut, versionFormat,
				GitCommit,
				GitBranch,
				GitState,
				GitSummary,
				BuildDate,
			)

			return nil
		},
	}

	return cmd
}
