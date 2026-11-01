package query

import (
	"net/url"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
)

func TestNewPortForwardDialer(t *testing.T) {
	restConfig := &rest.Config{Host: "https://kubernetes.example.test"}
	reqURL, err := url.Parse("https://kubernetes.example.test/api/v1/namespaces/kubecost/pods/cost-analyzer/portforward")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("websockets with SPDY fallback by default", func(t *testing.T) {
		dialer, err := newPortForwardDialer(restConfig, reqURL, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := dialer.(*portforward.FallbackDialer); !ok {
			t.Fatalf("got %T, want *portforward.FallbackDialer", dialer)
		}
	})

	t.Run("SPDY only when websockets are disabled", func(t *testing.T) {
		dialer, err := newPortForwardDialer(restConfig, reqURL, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := dialer.(*portforward.FallbackDialer); ok {
			t.Fatal("got a fallback dialer, want plain SPDY")
		}
	})
}
