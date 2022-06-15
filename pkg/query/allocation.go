package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/rs/zerolog/log"

	"github.com/kubecost/opencost/pkg/kubecost"
)

type AllocationParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type allocationResponse struct {
	Code int                              `json:"code"`
	Data []map[string]kubecost.Allocation `json:"data"`
}

// QueryAllocation queries the Allocation API by proxying a request to Kubecost
// through the Kubernetes API server if useProxy is true or, if it isn't, by
// temporarily port forwarding to a Kubecost pod.
func QueryAllocation(p AllocationParameters) ([]map[string]kubecost.Allocation, error) {
	var responseBytes []byte
	var queryErr error

	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		responseBytes, queryErr = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, string(p.ServicePort), p.AllocationPath, p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get: %s", queryErr)
		}
	} else {
		responseBytes, queryErr = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, p.AllocationPath, p.ServicePort, p.QueryParams, p.Ctx)
		if queryErr != nil {
			return nil, fmt.Errorf("failed to port-forwarded query: %s", queryErr)
		}

	}

	var ar allocationResponse
	err := json.Unmarshal(responseBytes, &ar)
	if err != nil {
		return ar.Data, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar.Data, nil
}
