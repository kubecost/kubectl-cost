package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubecost/opencost/pkg/kubecost"
)

const (
	idleString = "__idle__"
)

// Summary allocations do not marshal the Properties field, so we parse the
// relevant data from the SummaryAllocation name.
func parseAllocationName(allocationName string) (cluster, node, namespace, pod, container string, err error) {
	if allocationName == idleString {
		return "", "", "", "", "", fmt.Errorf("can't parse allocation information for special idle case")
	}

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

type AllocationParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type summaryAllocationResponse struct {
	Data kubecost.SummaryAllocationSetRange `json:"data"`
}

// QuerySummaryAllocation queries /model/allocation/summary by proxying a
// request to Kubecost through the Kubernetes API server if useProxy is true or,
// if it isn't, by temporarily port forwarding to a Kubecost pod.
func QuerySummaryAllocation(p AllocationParameters) (*kubecost.SummaryAllocationSetRange, error) {
	var responseBytes []byte
	var err error
	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset: %s", err)
		}

		responseBytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, "9090", "/model/allocation/summary", p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, responseBytes)
		}
	} else {
		responseBytes, err = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, "model/allocation/summary", p.QueryParams, p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	// The response is wrapped in a JSON like this: {code: XXXX, data: SASR}
	var sas summaryAllocationResponse
	err = json.Unmarshal(responseBytes, &sas)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return &sas.Data, nil
}
