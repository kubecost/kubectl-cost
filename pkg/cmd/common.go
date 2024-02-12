package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/kubecost"
)

// CostOptions holds common options for querying and displaying
// data from the kubecost API
type CostOptions struct {
	window          string
	filterNamespace string
	includeIdle     bool

	isHistorical bool

	query.QueryBackendOptions
}

func addCostOptionsFlags(cmd *cobra.Command, options *CostOptions) {
	cmd.Flags().StringVar(&options.window, "window", "1d", "The window of data to query. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")
	cmd.Flags().BoolVar(&options.isHistorical, "historical", false, "show the total cost during the window instead of the projected monthly rate based on the data in the window")
	cmd.Flags().BoolVar(&options.includeIdle, "idle", true, "include the __idle__ cost row in the response")

	query.AddQueryBackendOptionsFlags(cmd, &options.QueryBackendOptions)
}

func (co *CostOptions) Complete(restConfig *rest.Config) error {
	if err := co.QueryBackendOptions.Complete(restConfig); err != nil {
		return fmt.Errorf("complete backend opts: %s", err)
	}
	return nil
}

func (co *CostOptions) Validate() error {
	// make sure window parses client-side, may not be necessary but allows
	// for a nicer error message for the user
	if _, err := kubecost.ParseWindowWithOffset(co.window, 0); err != nil {
		return fmt.Errorf("failed to parse window: %s", err)
	}

	if err := co.QueryBackendOptions.Validate(); err != nil {
		return fmt.Errorf("validating query options: %s", err)
	}

	return nil
}
