package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

type clusterinfoResponse struct {
	Data struct {
		ClusterID string `json:"id"`
	} `json:"data"`
}

type ClusterInfoParameters struct {
	Ctx context.Context

	QueryBackendOptions
}

func QueryClusterID(p ClusterInfoParameters) (string, error) {
	var bytes []byte
	var err error

	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.restConfig)
		if err != nil {
			return "", fmt.Errorf("failed to create clientset: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, string(p.ServicePort), "/model/clusterInfo", nil).DoRaw(p.Ctx)

		if err != nil {
			return "", fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = p.QueryBackendOptions.pfQuerier.queryGet(p.Ctx, "model/clusterInfo", nil)
		if err != nil {
			return "", fmt.Errorf("failed to forward get kubecost: %s", err)
		}
	}

	var resp clusterinfoResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return resp.Data.ClusterID, nil
}
