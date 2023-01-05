package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencost/opencost/pkg/log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ResourcePredictParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type ResourceCostPredictionResponse struct {
	DerivedCostPerCoreHour float64 `json:"derivedCostPerCoreHour"`
	DerivedCostPerByteHour float64 `json:"derivedCostPerByteHour"`

	MonthlyCoreHours float64 `json:"monthlyCoreHours"`
	MonthlyByteHours float64 `json:"monthlyByteHours"`

	MonthlyCostMemory float64 `json:"monthlyCostMemory"`
	MonthlyCostCPU    float64 `json:"monthlyCostCPU"`
	MonthlyCostTotal  float64 `json:"monthlyCostTotal"`
}

func QueryPredictResourceCost(p ResourcePredictParameters) (ResourceCostPredictionResponse, error) {
	var bytes []byte
	var err error

	// TODO: genericize query logic further?
	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return ResourceCostPredictionResponse{}, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, fmt.Sprint(p.ServicePort), p.PredictResourceCostPath, p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return ResourceCostPredictionResponse{}, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryGet(p.Ctx, p.PredictResourceCostPath, p.QueryParams)
		if err != nil {
			return ResourceCostPredictionResponse{}, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	log.Debugf("Prediction response raw: %s", string(bytes))

	var resp ResourceCostPredictionResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return resp, nil
}
