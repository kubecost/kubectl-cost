package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencost/opencost/pkg/kubecost"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type assetResponse struct {
	Code int                    `json:"code"`
	Data []map[string]AssetNode `json:"data"`
}

type AssetParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	Window             string
	Aggregate          string
	DisableAdjustments bool
	Accumulate         string
	FilterTypes        string

	QueryBackendOptions
}

// QueryAssets queries /model/assets by proxying a request to Kubecost
// through the Kubernetes API server if useProxy is true or, if it isn't, by
// temporarily port forwarding to a Kubecost pod.
func QueryAssets(p AssetParameters) ([]map[string]AssetNode, error) {

	// aggregate, accumulate, and disableAdjustments are hardcoded;
	// as other asset types are added in to be filtered by, this may change,
	// but for now anything beyond isn't needed.

	requestParams := map[string]string{
		"window":      p.Window,
		"accumulate":  p.Accumulate,
		"filterTypes": p.FilterTypes,
	}

	if p.Aggregate != "" {
		requestParams["aggregate"] = p.Aggregate
	}

	var bytes []byte
	var err error
	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, string(p.ServicePort), "/model/assets", requestParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, "model/assets", p.ServicePort, requestParams, p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	var ar assetResponse
	err = json.Unmarshal(bytes, &ar)
	if err != nil {
		return ar.Data, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return ar.Data, nil
}

type AssetNode struct {
	Type         string                   `json:"type"`
	Properties   kubecost.AssetProperties `json:"properties"`
	Labels       kubecost.AssetLabels     `json:"labels"`
	Start        string                   `json:"start"`
	End          string                   `json:"end"`
	Minutes      float64                  `json:"minutes"`
	NodeType     string                   `json:"nodeType"`
	CpuCores     float64                  `json:"cpuCores"`
	RamBytes     float64                  `json:"ramBytes"`
	CPUCoreHours float64                  `json:"cpuCoreHours"`
	RAMByteHours float64                  `json:"ramByteHours"`
	GPUHours     float64                  `json:"GPUHours"`
	CPUBreakdown kubecost.Breakdown       `json:"cpuBreakdown"`
	GPUBreakdown kubecost.Breakdown       `json:"ramBreakdown"`
	Preemptible  float64                  `json:"preemptible"`
	Discount     float64                  `json:"discount"`
	CPUCost      float64                  `json:"cpuCost"`
	GPUCost      float64                  `json:"gpuCost"`
	GPUCount     float64                  `json:"gpuCount"`
	RAMCost      float64                  `json:"ramCost"`
	Adjustment   float64                  `json:"adjustment"`
	TotalCost    float64                  `json:"totalCost"`
}
