package cmd

import (
	"fmt"

	"github.com/kubecost/kubectl-cost/pkg/cmd/deprecated/oldpredict"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/opencost/opencost/pkg/log"
)

// Note that the auth/gcp import is necessary https://github.com/kubernetes/client-go/issues/242
// Similar may be required for AWS

var (
	costExample = `
    # Show the projected monthly rate for each namespace
    # based on the last 5 days of activity.
    %[1]s cost namespace --window 5d

    # Predict the cost of the Deployment defined in k8s-deployment.yaml.
    %[1]s cost predict -f 'k8s-deployment.yaml' \
      --show-cost-per-resource-hr

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
		Use:   "cost",
		Short: "View cluster cost information.",
		Long: `
kubectl-cost is a CLI frontend for Kubecost, a highly accurate provider of
Kubernetes cluster cost information and optimization opportunities.

kubectl-cost requires Kubecost to be installed in your Kubernetes cluster. Make
sure to check out the examples and full set of flags (--help) if you have a
non-default Kubecost install, like running in a custom namespace!

If you don't have Kubecost installed yet, all it takes is Helm and two minutes:

kubectl create namespace kubecost
helm repo add kubecost https://kubecost.github.io/cost-analyzer/
helm install \
    kubecost \
    kubecost/cost-analyzer \
    --namespace kubecost \
    --set kubecostToken="WljaGFdC5jctl20df98"
`,
		Example:      fmt.Sprintf(costExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			return fmt.Errorf("please use a subcommand")
		},
	}

	cmd.PersistentFlags().String("log-level", "info", "Set the log level from one of: 'trace', 'debug', 'info', 'warn', 'error'.")

	viper.BindPFlag("log-level", cmd.Flag("log-level"))

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		log.InitLogging(false)

		return nil
	}

	// Show usage on error because this command is just a base
	// for the subcommands
	cmd.SilenceUsage = false

	// TODO: disable cluster in single-cluster case
	cmd.AddCommand(buildStandardAggregatedAllocationCommand(streams,
		"namespace",
		[]string{"ns"},
		[]string{"cluster", "namespace"},
		false,
	))
	cmd.AddCommand(buildStandardAggregatedAllocationCommand(streams,
		"deployment",
		[]string{"deploy"},
		[]string{"cluster", "namespace", "deployment"},
		true,
	))
	cmd.AddCommand(buildStandardAggregatedAllocationCommand(streams,
		"controller",
		nil,
		[]string{"cluster", "namespace", "controller"},
		true,
	))
	cmd.AddCommand(buildStandardAggregatedAllocationCommand(streams,
		"pod",
		[]string{"po"},
		[]string{"cluster", "namespace", "pod"},
		true,
	))
	cmd.AddCommand(newCmdCostLabel(streams))
	cmd.AddCommand(newCmdCostNode(streams))
	cmd.AddCommand(newCmdTUI(streams))
	cmd.AddCommand(newCmdVersion(streams, GitCommit, GitBranch, GitState, GitSummary, BuildDate))
	cmd.AddCommand(oldpredict.NewCmdOldPredict(streams))
	cmd.AddCommand(NewCmdPredict(streams))

	return cmd
}
