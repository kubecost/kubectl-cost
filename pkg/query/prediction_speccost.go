package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencost/opencost/pkg/log"

	"k8s.io/client-go/rest"
)

type CostPrediction struct {
	TotalMonthlyRate float64 `json:"totalMonthlyRate"`
	CPUMonthlyRate   float64 `json:"cpuMonthlyRate"`
	RAMMonthlyRate   float64 `json:"ramMonthlyRate"`
	GPUMonthlyRate   float64 `json:"gpuMonthlyRate"`

	MonthlyCPUCoreHours float64 `json:"monthlyCPUCoreHours"`
	MonthlyRAMByteHours float64 `json:"monthlyRAMByteHours"`
	MonthlyGPUHours     float64 `json:"monthlyGPUHours"`
}

type SpecCostDiff struct {
	Namespace      string `json:"namespace"`
	ControllerKind string `json:"controllerKind"`
	ControllerName string `json:"controllerName"`

	CostBefore CostPrediction `json:"costBefore"`
	CostAfter  CostPrediction `json:"costAfter"`
	CostChange CostPrediction `json:"costChange"`
}

type SpecCostParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	SpecBytes   []byte
	QueryParams map[string]string

	QueryBackendOptions
}

type SpecCostResponse = []SpecCostDiff

func QuerySpecCost(p SpecCostParameters) (SpecCostResponse, error) {
	var bytes []byte
	var err error

	if p.UseProxy {
		return SpecCostResponse{}, fmt.Errorf("spec cost does not yet support using proxy to query due to limitations in the K8s libraries")
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryPost(
			p.Ctx,
			p.PredictSpecCostPath,
			p.QueryParams,
			nil,
			p.SpecBytes,
		)
		if err != nil {
			return SpecCostResponse{}, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	log.Debugf("Response raw: %s", string(bytes))

	var resp SpecCostResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return resp, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return resp, nil
}
