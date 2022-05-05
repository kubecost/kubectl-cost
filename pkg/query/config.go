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

type CurrencyCodeParameters struct {
	RestConfig *rest.Config
	Ctx        context.Context

	QueryBackendOptions
}

func QueryCurrencyCode(p CurrencyCodeParameters) (string, error) {
	var bytes []byte
	var err error

	if p.UseProxy {
		clientset, err := kubernetes.NewForConfig(p.RestConfig)
		if err != nil {
			return "", fmt.Errorf("failed to create clientset: %s", err)
		}

		bytes, err = clientset.CoreV1().Services(p.KubecostNamespace).ProxyGet("", p.ServiceName, "9090", "/model/getConfigs", nil).DoRaw(p.Ctx)

		if err != nil {
			return "", fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
		}
	} else {
		bytes, err = portForwardedQueryService(p.RestConfig, p.KubecostNamespace, p.ServiceName, "model/getConfigs", nil, p.Ctx)
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
