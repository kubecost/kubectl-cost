package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

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

func queryAllocation(clientset *kubernetes.Clientset, window, aggregate string) (allocationResponse, error) {

	params := map[string]string{
		// if we set this to false, output would be
		// per-day (we could use it in a more
		// complicated way to build in-terminal charts)
		"accumulate": "true",
		"window":     window,
	}

	if aggregate != "" {
		params["aggregate"] = aggregate
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
