package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newCmdVersion(streams genericclioptions.IOStreams) *cobra.Command {
	buildInfo, ok := debug.ReadBuildInfo()
	// Zero out deps information because it isn't helpful for this command.
	buildInfo.Deps = nil

	cmd := &cobra.Command{
		Use:   "version",
		Short: "view installed version of kubectl cost",
		RunE: func(c *cobra.Command, args []string) error {
			if !ok {
				fmt.Fprintf(streams.ErrOut, "Build info is unavailable in this binary.")
				return fmt.Errorf("Build info is unavailable in this binary.")
			}

			fmt.Fprintf(
				streams.ErrOut,
				"kubectl cost version/build info:\n%s\n",
				buildInfo.String(),
			)

			return nil
		},
	}

	return cmd
}
