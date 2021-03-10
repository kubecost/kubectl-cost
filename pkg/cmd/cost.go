package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/tools/clientcmd/api"

	"github.com/spf13/cobra"
	// "github.com/kubecost/cost-model/pkg/kubecost"
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
`

	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
)

// KubeOptions provides information required to communicate
// with the Kubernetes API
type KubeOptions struct {
	configFlags *genericclioptions.ConfigFlags

	restConfig *rest.Config
	clientset  *kubernetes.Clientset
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
	GitCommit string,
	GitBranch string,
	GitState string,
	GitSummary string,
	BuildDate string,
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
	cmd.AddCommand(newCmdTUI(streams))
	cmd.AddCommand(newCmdVersion(streams, GitCommit, GitBranch, GitState, GitSummary, BuildDate))

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

	if *o.configFlags.Namespace == "" {
		// Don't log here, as this is expected behavior. This is hard to communicate
		// in the --help output because the --namespace flag is set up by
		// genericclioptions.
		*o.configFlags.Namespace = "kubecost"
	}

	o.clientset, err = kubernetes.NewForConfig(o.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *KubeOptions) Validate() error {

	return nil
}
