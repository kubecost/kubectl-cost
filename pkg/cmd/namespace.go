package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsNamespace contains the standard CostOptions and any
// options specific to namespace queries.
type CostOptionsNamespace struct {
	CostOptions
}

func newCmdCostNamespace(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	namespaceO := &CostOptionsNamespace{}

	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "view cost information aggregated by namespace",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			namespaceO.Complete()

			return runCostNamespace(kubeO, namespaceO)
		},
	}

	addCostOptionsFlags(cmd, &namespaceO.CostOptions)
	kubeO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (no *CostOptionsNamespace) Complete() {
	if no.showAll {
		no.showCPUCost = true
		no.showMemoryCost = true
		no.showGPUCost = true
		no.showPVCost = true
		no.showNetworkCost = true
	}
}

func runCostNamespace(co *KubeOptions, no *CostOptionsNamespace) error {

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(clientset, *co.configFlags.Namespace, no.serviceName, no.window, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(
			co.Out,
			aggs,
			[]string{"namespace"},
			noopTitleExtractor,
			no.displayOptions,
		)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		allocations, err := query.QueryAllocation(clientset, *co.configFlags.Namespace, no.serviceName, no.window, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query allocation API: %s", err)
		}

		// Use Data[0] because the query accumulates
		err = writeNamespaceTable(co.Out, allocations[0], no.displayOptions)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}
