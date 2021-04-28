package query

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

type configsResponse struct {
	Data struct {
		CurrencyCode string `json:"currencyCode"`
	} `json:"data"`
}

func QueryCurrencyCode(clientset *kubernetes.Clientset, kubecostNamespace, serviceName string, ctx context.Context) (string, error) {
	bytes, err := clientset.CoreV1().Services(kubecostNamespace).ProxyGet("", serviceName, "9090", "/model/getConfigs", nil).DoRaw(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to proxy get kubecost. err: %s; data: %s", err, bytes)
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
