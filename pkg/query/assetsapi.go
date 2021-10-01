package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubecost/cost-model/pkg/kubecost"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type AssetParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	KubecostNamespace  string
	ServiceName        string
	Window             string
	Aggregate          string
	DisableAdjustments bool
	Accumulate         string
	UseProxy           bool
	FilterTypes        string
}

// QueryAssets queries /model/assets by proxying a request to Kubecost
// through the Kubernetes API server if useProxy is true or, if it isn't, by
// temporarily port forwarding to a Kubecost pod.
func QueryAssets(p AssetParameters) ([]map[string]kubecost.Asset, error) {

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

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, "9090", "/model/assets", requestParams).DoRaw(p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, "model/assets", requestParams, p.Ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to port forward query: %s", err)
		}
	}

	var asrl []map[string]kubecost.Asset

	var ar kubecost.AssetAPIResponse

	err = json.Unmarshal(bytes, &ar)
	responseList := ar.Data.Assets

	for _, resp := range responseList {

		as := make(map[string]kubecost.Asset)

		for str, asset := range resp.Assets {
			as[str] = asset
		}

		asrl = append(asrl, as)
	}

	if err != nil {
		return asrl, fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	return asrl, nil
}
