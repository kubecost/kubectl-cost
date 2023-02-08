package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencost/opencost/pkg/log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ResourceDiffPredictParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type ResourceCostDiffPredictionResponse struct {
	MonthlyCostMemory float64 `json:"monthlyCostMemory"`
	MonthlyCostCPU    float64 `json:"monthlyCostCPU"`
	MonthlyCostGPU    float64 `json:"monthlyCostGPU"`
	MonthlyCostTotal  float64 `json:"monthlyCostTotal"`

	MonthlyCostMemoryDiff float64 `json:"monthlyCostMemoryDiff"`
	MonthlyCostCPUDiff    float64 `json:"monthlyCostCPUDiff"`
}

func QueryPredictResourceCostDiff(p ResourceDiffPredictParameters) (ResourceCostDiffPredictionResponse, error) {
	var bytes []byte
	var err error

	// TODO: genericize query logic further?
	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return ResourceCostDiffPredictionResponse{}, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, fmt.Sprint(p.ServicePort), p.PredictResourceCostDiffPath, p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return ResourceCostDiffPredictionResponse{}, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryGet(p.Ctx, p.PredictResourceCostDiffPath, p.QueryParams)
		if err != nil {
			return ResourceCostDiffPredictionResponse{}, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	log.Debugf("Prediction response raw: %s", string(bytes))

	var resp ResourceCostDiffPredictionResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return resp, nil
}
