package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

const SavingsRequestSizingPath = "/model/savings/requestSizingV2"

type SavingsParameters struct {
	Ctx context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type RequestSizingRecommendation struct {
	ClusterID      string `json:"clusterID"`
	Namespace      string `json:"namespace"`
	ControllerKind string `json:"controllerKind"`
	ControllerName string `json:"controllerName"`
	ContainerName  string `json:"containerName"`

	RecommendedRequest struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"recommendedRequest"`

	MonthlySavings struct {
		CPU    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
	} `json:"monthlySavings"`

	LatestKnownRequest struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"latestKnownRequest"`

	CurrentEfficiency struct {
		CPU    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
		Total  float64 `json:"total"`
	} `json:"currentEfficiency"`
}

// QuerySavings queries the Kubecost savings/requestSizingV2 API.
func QuerySavings(p SavingsParameters) ([]RequestSizingRecommendation, error) {
	var bytes []byte
	var err error

	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, fmt.Sprint(p.ServicePort), SavingsRequestSizingPath, p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get savings: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryGet(p.Ctx, SavingsRequestSizingPath, p.QueryParams)
		if err != nil {
			return nil, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	var recs []RequestSizingRecommendation
	err = json.Unmarshal(bytes, &recs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal savings response: %s", err)
	}

	return recs, nil
}
