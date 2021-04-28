package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"

	"github.com/spf13/cobra"

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

	currencyCode, err := query.QueryCurrencyCode(ko.clientset, *ko.configFlags.Namespace, no.serviceName, context.Background())
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "deployment", "", context.Background())
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		// don't show unallocated deployment data
		delete(aggs, "__unallocated__")

		applyNamespaceFilter(aggs, no.filterNamespace)

		writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"namespace", "deployment"},
			deploymentTitleExtractor,
			no.displayOptions,
			currencyCode,
		)
	} else {
		// Not supported because the allocation API does not return deployment names.
		return fmt.Errorf("kubectl cost deployment does not yet support historical queries")
	}

	return nil
}

// Applies the filter in place by deleting all entries from aggData that are not in
// the namespace, unless it is an empty string in which case nothing is done.
func applyNamespaceFilter(aggData map[string]query.Aggregation, namespaceFilter string) {
	if namespaceFilter == "" {
		return
	}

	for aggName, _ := range aggData {
		sp, err := deploymentTitleExtractor(aggName)
		if err != nil {
			klog.Warningf("failed to extract namespace info from aggregation title %s, skipping", aggName)
			continue
		}
		namespace := sp[0]

		if namespace != namespaceFilter {
			delete(aggData, aggName)
		}
	}

	return
}
