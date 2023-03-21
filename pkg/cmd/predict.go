package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/kubecost/kubectl-cost/pkg/cmd/display"
	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"

	"github.com/opencost/opencost/pkg/log"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// PredictOptions contains options specific to prediction queries.
type PredictOptions struct {
	avgUsageWindow     string
	resourceCostWindow string

	clusterID string

	// The file containing the workload definition to be predicted.
	filepath string

	noUsage bool

	query.QueryBackendOptions
	display.PredictDisplayOptions
}

func NewCmdPredict(
	streams genericclioptions.IOStreams,
) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	predictO := &PredictOptions{}

	cmd := &cobra.Command{
		Use:   "predict",
		Short: "Estimate the monthly cost rate of a workload based on tracked cluster resource costs and historical usage.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return fmt.Errorf("complete k8s options: %s", err)
			}
			if err := kubeO.Validate(); err != nil {
				return fmt.Errorf("validate k8s options: %s", err)
			}

			if err := predictO.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("complete: %s", err)
			}
			if err := predictO.Validate(); err != nil {
				return fmt.Errorf("validate: %s", err)
			}

			return runCostPredict(kubeO, predictO)
		},
	}
	cmd.Flags().StringVarP(&predictO.filepath, "filepath", "f", "", "The file containing the workload definition whose cost should be predicted. E.g. a file might be 'test-deployment.yaml' containing an apps/v1 Deployment definition. '-' can also be passed, in which case workload definitions will be read from stdin.")
	cmd.Flags().StringVarP(&predictO.clusterID, "cluster-id", "c", "", "The cluster ID (in Kubecost) of the presumed cluster which the workload will be deployed to. This is used to determine resource costs. Defaults to local cluster.")
	cmd.Flags().StringVar(&predictO.avgUsageWindow, "window-usage", "2d", "The window of Kubecost data to calculate historical average usage from, if historical data exists. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")
	cmd.Flags().StringVar(&predictO.resourceCostWindow, "window-cost", "7d offset 48h", "The window of Kubecost data to base resource costs on. Defaults with an offset of 48h to incorporate reconciled data if reconciliation is set up. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")
	cmd.Flags().BoolVar(&predictO.noUsage, "no-usage", false, "Set true ignore historical usage data (if any exists) when performing cost prediction.")
	cmd.Flags().BoolVar(&predictO.ShowTotal, "show-total", false, "Show the total cost of the new spec(s). See --hide-diff for a similar option..")
	cmd.Flags().BoolVar(&predictO.HideDiff, "hide-diff", false, "Hide the cost difference of applying the new spec(s). See --show-total for a similar option..")

	query.AddQueryBackendOptionsFlags(cmd, &predictO.QueryBackendOptions)
	display.AddPredictDisplayOptionsFlags(cmd, &predictO.PredictDisplayOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	cmd.SilenceUsage = true

	return cmd
}

func (predictO *PredictOptions) Validate() error {
	if predictO.filepath != "-" {
		if _, err := os.Stat(predictO.filepath); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("file '%s' does not exist, not a valid option", predictO.filepath)
		}
	}

	if err := predictO.QueryBackendOptions.Validate(); err != nil {
		return fmt.Errorf("validating query options: %s", err)
	}

	if err := predictO.PredictDisplayOptions.Validate(); err != nil {
		return fmt.Errorf("validating display options: %s", err)
	}

	return nil
}

func (predictO *PredictOptions) Complete(restConfig *rest.Config) error {
	if err := predictO.QueryBackendOptions.Complete(restConfig); err != nil {
		return fmt.Errorf("complete backend opts: %s", err)
	}
	return nil
}

func runCostPredict(ko *utilities.KubeOptions, no *PredictOptions) error {
	var b []byte
	var err error

	// Filepath of - means read from stdin.
	if no.filepath == "-" {
		reader := bufio.NewReader(ko.In)

		scratch := make([]byte, 1024)
		for {
			n, err := reader.Read(scratch)
			b = append(b, scratch[:n]...)
			if err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("reading from stdin: %s", err)
			}
		}
	} else {
		b, err = ioutil.ReadFile(no.filepath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %s", no.filepath, err)
		}
	}

	// If the user doesn't provide a cluster ID, default to the "local" (the
	// cluster ID of the API).
	// TODO: Should we at some point distinguish between cluster ID of API and
	// cluster ID of the actual configured cluster? Env var retrieval or
	// something?
	if len(no.clusterID) == 0 {
		clusterID, err := query.QueryClusterID(query.ClusterInfoParameters{
			Ctx:                 context.Background(),
			QueryBackendOptions: no.QueryBackendOptions,
		})
		if err != nil {
			return fmt.Errorf("acquiring cluster ID from service: %s", err)
		}
		no.clusterID = clusterID
		log.Debugf("Cluster ID for query set to: %s", no.clusterID)
	}

	rows, err := query.QuerySpecCost(query.SpecCostParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
		SpecBytes:           b,
		QueryParams: map[string]string{
			"noUsage":            fmt.Sprint(no.noUsage),
			"windowAvgUsage":     no.avgUsageWindow,
			"windowResourceCost": no.resourceCostWindow,
			"clusterID":          no.clusterID,
			"defaultNamespace":   ko.DefaultNamespace,
		},
	})
	if err != nil {
		return fmt.Errorf("Failed querying the speccost API. This API requires a version of Kubecost >= 1.101, which may be why this query failed. If running Kubecost v1.100, you can downgrade kubectl cost to v0.4 for old-style prediction. Error: %s", err)
	}
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	display.WritePredictionTable(ko.Out, rows, currencyCode, no.PredictDisplayOptions)
	return nil
}
