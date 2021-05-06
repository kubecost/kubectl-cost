package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type configsResponse struct {
	Data struct {
		CurrencyCode string `json:"currencyCode"`
	} `json:"data"`
}

func QueryCurrencyCode(restConfig *rest.Config, kubecostNamespace, serviceName string, useProxy bool, ctx context.Context) (string, error) {
	var bytes []byte
	var err error

	if useProxy {
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return "", fmt.Errorf("failed to create clientset: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(kubecostNamespace).ProxyGet("", serviceName, "9090", "/model/getConfigs", nil).DoRaw(ctx)

		if err != nil {
			return "", fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = portForwardedQueryService(restConfig, kubecostNamespace, serviceName, "model/getConfigs", nil, ctx)
		if err != nil {
			return "", fmt.Errorf("failed to forward get kubecost: %s", err)
		}
	}

	var resp configsResponse
	err = json.Unmarshal(bytes, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal allocation response: %s", err)
	}

	// Empty currency code is considered equivalent to USD
	if resp.Data.CurrencyCode == "" {
		return "USD", nil
	}

	return resp.Data.CurrencyCode, nil
}
