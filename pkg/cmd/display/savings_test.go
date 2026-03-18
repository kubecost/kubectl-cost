package display

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

func TestMakeSavingsTable_Empty(t *testing.T) {
	tw := MakeSavingsTable(nil, "USD")
	out := tw.Render()

	if !strings.Contains(strings.ToUpper(out), "NAMESPACE") {
		t.Error("expected header to contain Namespace")
	}
	if !strings.Contains(out, "TOTAL") {
		t.Error("expected footer to contain TOTAL")
	}
	if !strings.Contains(out, "0.00 USD") {
		t.Errorf("expected total of 0.00 USD, got:\n%s", out)
	}
}

func TestMakeSavingsTable_SingleRec(t *testing.T) {
	recs := []query.RequestSizingRecommendation{
		{
			ClusterID:      "cluster-one",
			Namespace:      "default",
			ControllerKind: "Deployment",
			ControllerName: "nginx",
			ContainerName:  "nginx",
			RecommendedRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "100m", Memory: "128Mi"},
			MonthlySavings: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
			}{CPU: 5.50, Memory: 2.30},
			LatestKnownRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "500m", Memory: "512Mi"},
			CurrentEfficiency: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
				Total  float64 `json:"total"`
			}{CPU: 0.20, Memory: 0.25, Total: 0.225},
		},
	}

	tw := MakeSavingsTable(recs, "EUR")
	out := tw.Render()

	checks := []string{
		"default",
		"Deployment/nginx",
		"nginx",
		"500m",
		"100m",
		"512Mi",
		"128Mi",
		"20%",
		"25%",
		"7.80 EUR",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("expected table to contain %q, got:\n%s", want, out)
		}
	}
}

func TestMakeSavingsTable_MultipleRecs_SortedBySavings(t *testing.T) {
	recs := []query.RequestSizingRecommendation{
		{
			Namespace:      "ns-a",
			ControllerKind: "Deployment",
			ControllerName: "small-saver",
			ContainerName:  "app",
			RecommendedRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "50m", Memory: "64Mi"},
			MonthlySavings: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
			}{CPU: 1.00, Memory: 0.50},
			LatestKnownRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "100m", Memory: "128Mi"},
			CurrentEfficiency: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
				Total  float64 `json:"total"`
			}{CPU: 0.50, Memory: 0.50, Total: 0.50},
		},
		{
			Namespace:      "ns-b",
			ControllerKind: "StatefulSet",
			ControllerName: "big-saver",
			ContainerName:  "db",
			RecommendedRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "1", Memory: "1Gi"},
			MonthlySavings: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
			}{CPU: 20.00, Memory: 10.00},
			LatestKnownRequest: struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			}{CPU: "4", Memory: "8Gi"},
			CurrentEfficiency: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
				Total  float64 `json:"total"`
			}{CPU: 0.25, Memory: 0.125, Total: 0.1875},
		},
	}

	tw := MakeSavingsTable(recs, "USD")
	out := tw.Render()

	// big-saver (30.00) should appear before small-saver (1.50) due to DscNumeric sort
	bigIdx := strings.Index(out, "big-saver")
	smallIdx := strings.Index(out, "small-saver")
	if bigIdx == -1 || smallIdx == -1 {
		t.Fatalf("expected both rows in output, got:\n%s", out)
	}
	if bigIdx > smallIdx {
		t.Errorf("expected big-saver before small-saver (descending savings sort), got:\n%s", out)
	}
	if !strings.Contains(out, "30.00 USD") {
		t.Errorf("expected 30.00 USD in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1.50 USD") {
		t.Errorf("expected 1.50 USD in output, got:\n%s", out)
	}

	// Total should be 31.50
	if !strings.Contains(out, "31.50 USD") {
		t.Errorf("expected total of 31.50 USD, got:\n%s", out)
	}
}

func TestWriteSavingsTable_WritesToOutput(t *testing.T) {
	var buf bytes.Buffer
	WriteSavingsTable(&buf, nil, "USD")

	out := buf.String()
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(strings.ToUpper(out), "SAVINGS/MO") {
		t.Errorf("expected header in output, got:\n%s", out)
	}
}
