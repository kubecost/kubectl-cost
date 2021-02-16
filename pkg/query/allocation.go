package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

const (
	idleString = "__idle__"
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

func QueryAllocation(clientset *kubernetes.Clientset, kubecostNamespace, serviceName, window, aggregate string) ([]map[string]kubecost.Allocation, error) {

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
	bytes, err := clientset.CoreV1().Services(kubecostNamespace).ProxyGet("", serviceName, "9090", "/model/allocation", params).DoRaw(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to proxy get kubecost: %s", err)
	}

	var ar allocationResponse
	err = json.Unmarshal(bytes, &ar)
	if err != nil {
		return ar.Data, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar.Data, nil
}
