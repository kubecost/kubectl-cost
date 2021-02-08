package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"

	// "github.com/kubecost/cost-model/pkg/costmodel"
	"github.com/kubecost/cost-model/pkg/kubecost"
	"github.com/kubecost/cost-model/pkg/util"
)

type aggCostModelResponse struct {
	Code int `json:"code"`
	// Data map[string]costmodel.Aggregation `json:"data"`
	Data map[string]aggregation `json:"data"`
}

func queryAggCostModel(clientset *kubernetes.Clientset, window, aggregate string) (aggCostModelResponse, error) {
	params := map[string]string{
		"window":      window,
		"aggregation": aggregate,
		"rate":        "monthly",
	}

	ctx := context.Background()
	bytes, err := clientset.CoreV1().Services("kubecost").ProxyGet("", "kubecost-cost-analyzer", "9090", "/model/aggregatedCostModel", params).DoRaw(ctx)

	if err != nil {
		return aggCostModelResponse{}, fmt.Errorf("failed to proxy get kubecost: %s", err)
	}

	var ar aggCostModelResponse
	err = json.Unmarshal(bytes, &ar)
	if err != nil {
		return ar, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar, nil
}

// Hardcoded instead of imported because of dependency problems introduced when
// github.com/kubecost/cost-model/pkg/costmodel is imported. The breakage involves
// Azure's go-autorest, the azure-sdk-for-go, and k8s client-go.
type aggregation struct {
	Aggregator                 string               `json:"aggregation"`
	Subfields                  []string             `json:"subfields,omitempty"`
	Environment                string               `json:"environment"`
	Cluster                    string               `json:"cluster,omitempty"`
	Properties                 *kubecost.Properties `json:"-"`
	CPUAllocationHourlyAverage float64              `json:"cpuAllocationAverage"`
	CPUAllocationVectors       []*util.Vector       `json:"-"`
	CPUAllocationTotal         float64              `json:"-"`
	CPUCost                    float64              `json:"cpuCost"`
	CPUCostVector              []*util.Vector       `json:"cpuCostVector,omitempty"`
	CPUEfficiency              float64              `json:"cpuEfficiency"`
	CPURequestedVectors        []*util.Vector       `json:"-"`
	CPUUsedVectors             []*util.Vector       `json:"-"`
	Efficiency                 float64              `json:"efficiency"`
	GPUAllocationHourlyAverage float64              `json:"gpuAllocationAverage"`
	GPUAllocationVectors       []*util.Vector       `json:"-"`
	GPUCost                    float64              `json:"gpuCost"`
	GPUCostVector              []*util.Vector       `json:"gpuCostVector,omitempty"`
	GPUAllocationTotal         float64              `json:"-"`
	RAMAllocationHourlyAverage float64              `json:"ramAllocationAverage"`
	RAMAllocationVectors       []*util.Vector       `json:"-"`
	RAMAllocationTotal         float64              `json:"-"`
	RAMCost                    float64              `json:"ramCost"`
	RAMCostVector              []*util.Vector       `json:"ramCostVector,omitempty"`
	RAMEfficiency              float64              `json:"ramEfficiency"`
	RAMRequestedVectors        []*util.Vector       `json:"-"`
	RAMUsedVectors             []*util.Vector       `json:"-"`
	PVAllocationHourlyAverage  float64              `json:"pvAllocationAverage"`
	PVAllocationVectors        []*util.Vector       `json:"-"`
	PVAllocationTotal          float64              `json:"-"`
	PVCost                     float64              `json:"pvCost"`
	PVCostVector               []*util.Vector       `json:"pvCostVector,omitempty"`
	NetworkCost                float64              `json:"networkCost"`
	NetworkCostVector          []*util.Vector       `json:"networkCostVector,omitempty"`
	SharedCost                 float64              `json:"sharedCost"`
	TotalCost                  float64              `json:"totalCost"`
	TotalCostVector            []*util.Vector       `json:"totalCostVector,omitempty"`
}
