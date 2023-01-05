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
	DerivedCostPerCPUCoreHour    float64 `json:"derivedCostPerCPUCoreHour"`
	DerivedCostPerMemoryByteHour float64 `json:"derivedCostPerMemoryByteHour"`
	DerivedCostPerGPUHour        float64 `json:"derivedCostPerGPUHour"`

	MonthlyCPUCoreHours    float64 `json:"monthlyCPUCoreHours"`
	MonthlyMemoryByteHours float64 `json:"monthlyMemoryByteHours"`
	MonthlyGPUHours        float64 `json:"monthlyGPUHours"`

	MonthlyCostMemory float64 `json:"monthlyCostMemory"`
	MonthlyCostCPU    float64 `json:"monthlyCostCPU"`
	MonthlyCostGPU    float64 `json:"monthlyCostGPU"`
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

type PVCostPredictParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryParams map[string]string

	QueryBackendOptions
}

type PVCostPredictionResponse struct {
	DerivedCostPerGiBHour float64 `json:"derivedCostPerGiBHour"`

	MonthlyGiBHours float64 `json:"monthlyGiBHours"`

	MonthlyCostTotal float64 `json:"monthlyCostTotal"`
}

func QueryPredictPVCost(p ResourcePredictParameters) (PVCostPredictionResponse, error) {
	var bytes []byte
	var err error

	// TODO: genericize query logic further?
	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return PVCostPredictionResponse{}, fmt.Errorf("failed to create clientset for proxied query: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, fmt.Sprint(p.ServicePort), p.PredictPVCostPath, p.QueryParams).DoRaw(p.Ctx)
		if err != nil {
			return PVCostPredictionResponse{}, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryGet(p.Ctx, p.PredictResourceCostPath, p.QueryParams)
		if err != nil {
			return PVCostPredictionResponse{}, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	log.Debugf("Prediction response raw: %s", string(bytes))

	var resp PVCostPredictionResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return resp, nil
}
