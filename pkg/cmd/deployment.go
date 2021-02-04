package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

type CostOptionsDeployment struct {
	isRate      bool
	showCPU     bool
	showMemory  bool
	showGPU     bool
	showPV      bool
	showNetwork bool
}

func newCmdCostDeployment(streams genericclioptions.IOStreams) *cobra.Command {
	commonO := NewCommonCostOptions(streams)
	deploymentO := &CostOptionsDeployment{}

	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "view cost information aggregated by deployment",
		RunE: func(c *cobra.Command, args []string) error {
			if err := commonO.Complete(c, args); err != nil {
				return err
			}
			if err := commonO.Validate(); err != nil {
				return err
			}

			return runCostDeployment(commonO, deploymentO)
		},
	}

	cmd.Flags().StringVar(&commonO.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&deploymentO.isRate, "rate", false, "show the projected monthly rate based on data in the window instead of the total cost during the window")
	cmd.Flags().BoolVar(&deploymentO.showCPU, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&deploymentO.showMemory, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&deploymentO.showGPU, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&deploymentO.showPV, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&deploymentO.showNetwork, "show-network", false, "show data for network cost")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostDeployment(co *CostOptionsCommon, no *CostOptionsDeployment) error {

	do := displayOptions{
		showCPUCost:     no.showCPU,
		showMemoryCost:  no.showMemory,
		showGPUCost:     no.showGPU,
		showPVCost:      no.showPV,
		showNetworkCost: no.showNetwork,
	}

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if no.isRate {
		aggCMResp, err := queryAggCostModel(clientset, co.costWindow, "deployment")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(co.Out, aggCMResp.Data, "deployment", do)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		return fmt.Errorf("kubectl cost does not yet support non-rate queries by deployment")
	}

	return nil
}
