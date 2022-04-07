package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/spf13/cobra"
)

// Note that the auth/gcp import is necessary https://github.com/kubernetes/client-go/issues/242
// Similar may be required for AWS

var (
	costExample = `
    # Show the projected monthly rate for each namespace
    # based on the last 5 days of activity.
    %[1]s cost namespace --window 5d

    # Show how much each namespace cost over the past 5 days
    # with additional CPU and memory cost and without efficiency.
    %[1]s cost namespace \
      --historical \
      --window 5d \
      --show-cpu \
      --show-memory \
      --show-efficiency=false

    # Show the projected monthly rate for each controller
    # based on the last 5 days of activity with PV (persistent
    # volume) cost breakdown.
    %[1]s cost controller --window 5d --show-pv

    # Show costs over the past 5 days broken down by the value
    # of the "app" label:
    %[1]s cost label --historical -l app

    # Show the projected monthly rate for each deployment
    # based on the last month of activity with CPU, memory,
    # GPU, PV, and network cost breakdown.
    %[1]s cost deployment --window month -A

    # Show the projected monthly rate for each deployment
    # in the "kubecost" namespace based on the last 3 days
    # of activity with CPU cost breakdown.
    %[1]s cost deployment \
      --window 3d \
      --show-cpu \
      -n kubecost

    # The same, but with a non-standard Kubecost deployment
    # in the namespace "kubecost-staging" with the cost
    # analyzer service called "kubecost-staging-cost-analyzer".
    %[1]s cost deployment \
      --window 3d \
      --show-cpu \
      -n kubecost \
      -N kubecost-staging \
      --service-name kubecost-staging-cost-analyzer

    # Show how much each pod in the "kube-system" namespace
    # cost yesterday, including CPU-specific cost.
    %[1]s cost pod \
      --historical \
      --window yesterday \
      --show-cpu \
      -n kube-system
`

	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
)

// KubeOptions provides information required to communicate
// with the Kubernetes API
type KubeOptions struct {
	configFlags *genericclioptions.ConfigFlags

	restConfig *rest.Config
	args       []string

	genericclioptions.IOStreams
}

// NewCommonCostOptions creates the default set of cost options
func NewKubeOptions(streams genericclioptions.IOStreams) *KubeOptions {
	return &KubeOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdCost provides a cobra command that acts as a parent command
// for all subcommands. It provides only basic usage information. See
// common.go and the subcommands for the actual functionality.
func NewCmdCost(
	streams genericclioptions.IOStreams,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cost",
		Short:        "View cluster cost information.",
		Example:      fmt.Sprintf(costExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return fmt.Errorf("please use a subcommand")
		},
	}

	// Show usage on error because this command is just a base
	// for the subcommands
	cmd.SilenceUsage = false

	cmd.AddCommand(newCmdCostNamespace(streams))
	cmd.AddCommand(newCmdCostDeployment(streams))
	cmd.AddCommand(newCmdCostController(streams))
	cmd.AddCommand(newCmdCostLabel(streams))
	cmd.AddCommand(newCmdCostPod(streams))
	cmd.AddCommand(newCmdCostNode(streams))
	cmd.AddCommand(newCmdTUI(streams))
	cmd.AddCommand(newCmdVersion(streams))

	return cmd
}

// Complete sets all information required for getting cost information
func (o *KubeOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error

	o.restConfig, err = o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *KubeOptions) Validate() error {

	return nil
}
