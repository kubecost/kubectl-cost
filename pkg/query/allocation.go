package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/opencost/opencost/pkg/kubecost"
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
	var bytes []byte
	var err error

	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, "9090", "/model/allocation", p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, "model/allocation", p.QueryParams, p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	var ar allocationResponse
	err = json.Unmarshal(bytes, &ar)
	if err != nil {
		return ar.Data, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar.Data, nil
}
