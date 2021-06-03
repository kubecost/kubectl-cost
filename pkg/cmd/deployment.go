package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsDeployment contains the standard CostOptions and any
// options specific to deployment queries.
type CostOptionsDeployment struct {
	filterNamespace string

	CostOptions
}

func newCmdCostDeployment(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	deploymentO := &CostOptionsDeployment{}

	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "view cost information aggregated by deployment",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			deploymentO.CostOptions.Complete()

			if err := deploymentO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostDeployment(kubeO, deploymentO)
		},
	}

	cmd.Flags().StringVarP(&deploymentO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")

	addCostOptionsFlags(cmd, &deploymentO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostDeployment(ko *KubeOptions, no *CostOptionsDeployment) error {

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		RestConfig:        ko.restConfig,
		Ctx:               context.Background(),
		KubecostNamespace: *ko.configFlags.Namespace,
		ServiceName:       no.serviceName,
		UseProxy:          no.useProxy,
	})
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	// if !no.isHistorical {
	// aggs, err := query.QueryAggCostModel(query.AggCostModelParameters{
	// 	RestConfig:        ko.restConfig,
	// 	Ctx:               context.Background(),
	// 	KubecostNamespace: *ko.configFlags.Namespace,
	// 	ServiceName:       no.serviceName,
	// 	Window:            no.window,
	// 	Aggregate:         "deployment",
	// 	UseProxy:          no.useProxy,
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to query agg cost model: %s", err)
	// }

	// // don't show unallocated deployment data
	// delete(aggs, "__unallocated__")

	// applyNamespaceFilter(aggs, no.filterNamespace)

	// writeAggregationRateTable(
	// 	ko.Out,
	// 	aggs,
	// 	[]string{"namespace", "deployment"},
	// 	deploymentTitleExtractor,
	// 	no.displayOptions,
	// 	currencyCode,
	// )

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		RestConfig:        ko.restConfig,
		Ctx:               context.Background(),
		KubecostNamespace: *ko.configFlags.Namespace,
		ServiceName:       no.serviceName,
		Window:            no.window,
		Aggregate:         "deployment",
		UseProxy:          no.useProxy,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use allocations[0] because the query accumulates to a single result
	applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

	writeAllocationTable(ko.Out, "Deployment", allocations[0], no.displayOptions, currencyCode, true, no.isHistorical)

	// } else {
	// 	allocations, err := query.QueryAllocation(query.AllocationParameters{
	// 		RestConfig:        ko.restConfig,
	// 		Ctx:               context.Background(),
	// 		KubecostNamespace: *ko.configFlags.Namespace,
	// 		ServiceName:       no.serviceName,
	// 		Window:            no.window,
	// 		Aggregate:         "deployment",
	// 		UseProxy:          no.useProxy,
	// 	})
	// 	if err != nil {
	// 		return fmt.Errorf("failed to query allocation API: %s", err)
	// 	}

	// 	// Use allocations[0] because the query accumulates to a single result
	// 	applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

	// 	writeAllocationTable(ko.Out, "Deployment", allocations[0], no.displayOptions, currencyCode, true)
	// }

	return nil
}

// Applies the filter in place by deleting all entries from aggData that are not in
// the namespace, unless it is an empty string in which case nothing is done.
// func applyNamespaceFilter(aggData map[string]query.Aggregation, namespaceFilter string) {
// 	if namespaceFilter == "" {
// 		return
// 	}

// 	for aggName, _ := range aggData {
// 		sp, err := deploymentTitleExtractor(aggName)
// 		if err != nil {
// 			klog.Warningf("failed to extract namespace info from aggregation title %s, skipping", aggName)
// 			continue
// 		}
// 		namespace := sp[0]

// 		if namespace != namespaceFilter {
// 			delete(aggData, aggName)
// 		}
// 	}

// 	return
// }

func applyNamespaceFilterAllocation(allocData map[string]kubecost.Allocation, namespaceFilter string) {
	if namespaceFilter == "" {
		return
	}

	for allocName, alloc := range allocData {
		ns := alloc.Properties.Namespace
		if ns != namespaceFilter {
			delete(allocData, allocName)
		}
	}
}
