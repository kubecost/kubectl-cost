package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

// Note that the auth/gcp import is necessary https://github.com/kubernetes/client-go/issues/242
// Similar may be required for AWS

var (
	costExample = `
	# view the general cost breakdown
	%[1]s cost

	# view the general cost breakdown for the last 4 days
	%[1]s cost --window 4d

	# view the general cost breakdown for the last 4 days for the kubecost namespace
	%[1]s cost --window 4d --cost-namespace kubecost
`

	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
)

const (
	idleString = "__idle__"
)

// CostOptions provides information required to get
// cost informatin from the kubecost API
type CostOptions struct {
	configFlags *genericclioptions.ConfigFlags

	costWindow    string
	costNamespace string

	restConfig *rest.Config
	args       []string

	genericclioptions.IOStreams
}

// NewCostOptions creates the default set of cost options
func NewCostOptions(streams genericclioptions.IOStreams) *CostOptions {
	return &CostOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdCost provides a cobra command wrapping CostOptions
func NewCmdCost(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewCostOptions(streams)

	cmd := &cobra.Command{
		Use:          "cost [category] [flags]",
		Short:        "View cluster cost information",
		Example:      fmt.Sprintf(costExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().StringVar(&o.costNamespace, "cost-namespace", "", "filter results to only include a specific namespace, leave blank for all namespaces")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for getting cost information
func (o *CostOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error

	o.restConfig, err = o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *CostOptions) Validate() error {
	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	// just make sure window parses client-side, perhaps not necessary
	if _, err := kubecost.ParseWindowWithOffset(o.costWindow, 0); err != nil {
		return fmt.Errorf("failed to parse window: %s", err)
	}

	return nil
}

func (o *CostOptions) Run() error {

	clientset, err := kubernetes.NewForConfig(o.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	allocResp, err := queryAllocation(clientset, o.costWindow)
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// using allocResp.Data[0] is fine because we set the accumulate
	// flag in the allocation API
	err = filterAllocations(allocResp.Data[0], o.costNamespace)
	if err != nil {
		return fmt.Errorf("failed to filter allocations: %s", err)
	}
	writeAllocationTable(o.Out, allocResp.Data[0])

	return nil
}

// edits allocation map without copying
func filterAllocations(allocations map[string]kubecost.Allocation, namespace string) error {
	// empty filter parameter means no filtering occurs
	if namespace == "" {
		return nil
	}

	for name, _ := range allocations {
		// idle allocation has no namespace
		if name == idleString {
			delete(allocations, name)
		} else {
			_, _, allocNamespace, _, _, err := parseAllocationName(name)
			if err != nil {
				return fmt.Errorf("failed to parse allocation name: %s", err)
			}
			if allocNamespace != namespace {
				delete(allocations, name)
			}
		}
	}

	return nil
}

func writeAllocationTable(out io.Writer, allocations map[string]kubecost.Allocation) error {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:      "Cluster",
			AutoMerge: true,
		},
		{
			Name:      "Namespace",
			AutoMerge: true,
		},
		{
			Name:     "Pod",
			WidthMax: 26,
		},
		{
			Name:     "Container",
			WidthMax: 26,
		},
		{
			Name:        "Total Cost",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
	})

	t.AppendHeader(table.Row{"Cluster", "Namespace", "Pod", "Container", "Total Cost"})
	t.SortBy([]table.SortBy{
		{
			Name: "Cluster",
			Mode: table.Dsc,
		},
		{
			Name: "Namespace",
			Mode: table.Dsc,
		},
		{
			Name: "Total Cost",
			Mode: table.Dsc,
		},
	})

	var summedCost float64

	for allocName, alloc := range allocations {

		// idle allocation is a special case where information
		// cannot be parsed from the allocation name
		if alloc.Name == idleString {
			namespace := idleString
			cluster, _ := alloc.Properties.GetCluster()
			totalCost := fmt.Sprintf("%.6f", alloc.TotalCost)
			t.AppendRow(table.Row{
				cluster, namespace, "", "", totalCost,
			})
			continue
		}

		cluster, _, namespace, pod, container, err := parseAllocationName(allocName)
		if err != nil {
			return fmt.Errorf("failed to parse allocation name: %s", err)
		}

		totalCost := fmt.Sprintf("%.6f", alloc.TotalCost)
		t.AppendRow(table.Row{
			cluster, namespace, pod, container, totalCost,
		})
		summedCost += alloc.TotalCost
	}
	t.AppendFooter(table.Row{"SUMMED", "", "", "", fmt.Sprintf("%.6f", summedCost)})
	t.Render()

	return nil
}

func parseAllocationName(allocationName string) (cluster, node, namespace, pod, container string, err error) {

	if allocationName == idleString {
		return "", "", "", "", "", fmt.Errorf("can't parse allocation information for special idle case")
	}

	// We use the allocation name instead of properties
	// because a recent performance-motivated change
	// that means properties is not guaranteed to have
	// information beyond cluster and node. In the future,
	// we should be able to rely on properties to have
	// accurate information.
	allocNameSplit := strings.Split(allocationName, "/")

	if len(allocNameSplit) != 5 {
		return "", "", "", "", "", fmt.Errorf("allocation name %s could not be split into the correct number of fields", allocationName)
	}

	cluster = allocNameSplit[0]
	node = allocNameSplit[1]
	namespace = allocNameSplit[2]
	pod = allocNameSplit[3]
	container = allocNameSplit[4]

	return cluster, node, namespace, pod, container, nil
}

type allocationResponse struct {
	Code int                              `json:"code"`
	Data []map[string]kubecost.Allocation `json:"data"`
}

func queryAllocation(clientset *kubernetes.Clientset, window string) (allocationResponse, error) {

	params := map[string]string{
		// if we set this to false, output would be
		// per-day (we could use it in a more
		// complicated way to build in-terminal charts)
		"accumulate": "true",
		"window":     window,
	}

	ctx := context.Background()
	bytes, err := clientset.CoreV1().Services("kubecost").ProxyGet("", "kubecost-cost-analyzer", "9090", "/model/allocation", params).DoRaw(ctx)

	if err != nil {
		return allocationResponse{}, fmt.Errorf("failed to proxy get kubecost: %s", err)
	}

	var ar allocationResponse
	err = json.Unmarshal(bytes, &ar)
	if err != nil {
		return ar, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar, nil
}
