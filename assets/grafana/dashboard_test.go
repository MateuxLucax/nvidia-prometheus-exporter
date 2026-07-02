package grafana_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDashboardJSONIsValid(t *testing.T) {
	data, err := os.ReadFile("nvidia-smi-grafana-dashboard.json")
	if err != nil {
		t.Fatal(err)
	}
	var dashboard map[string]any
	if err := json.Unmarshal(data, &dashboard); err != nil {
		t.Fatal(err)
	}
	if dashboard["title"] != "NVIDIA SMI Exporter" {
		t.Fatalf("unexpected dashboard title: %v", dashboard["title"])
	}
}
