package exporter

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MateuxLucax/nvidia-prometheus-exporter/internal/nvidia"
)

type fakeCollector struct {
	snapshot nvidia.Snapshot
	err      error
}

func (c *fakeCollector) Collect(context.Context) (nvidia.Snapshot, error) {
	return c.snapshot, c.err
}

func TestServeHTTPRendersSnapshotAndEscapesLabels(t *testing.T) {
	exp := New(nvidia.Collector{})
	exp.snapshot = nvidia.Snapshot{
		GPUs: []nvidia.GPU{{
			Index:         "0",
			UUID:          "GPU-abc",
			Name:          `RTX "Test"`,
			PCIBusID:      "00000000:01:00.0",
			DriverVersion: "555.42",
			CUDAVersion:   "12.6",
			PState:        "P2",
			ComputeMode:   "Default",
			Values: map[string]float64{
				"utilization.gpu": 0.8,
				"memory.used":     1024,
			},
		}},
		CollectedAt: time.Unix(100, 0),
	}
	exp.lastAttempt = time.Unix(100, 0)
	exp.lastSuccess = time.Unix(100, 0)
	exp.lastDurationSec = 0.02

	rec := httptest.NewRecorder()
	exp.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	assertContains(t, body, "nvidia_smi_up 1")
	assertContains(t, body, `name="RTX \"Test\""`)
	assertContains(t, body, `nvidia_smi_utilization_ratio{index="0",name="RTX \"Test\"",unit="gpu",uuid="GPU-abc"} 0.8`)
	assertContains(t, body, `nvidia_smi_memory_bytes{index="0",kind="used",name="RTX \"Test\"",uuid="GPU-abc"} 1024`)
}

func TestServeHTTPMarksDownBeforeSuccessfulCollect(t *testing.T) {
	exp := New(nvidia.Collector{})
	exp.lastAttempt = time.Unix(100, 0)

	rec := httptest.NewRecorder()
	exp.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	assertContains(t, rec.Body.String(), "nvidia_smi_up 0")
}

func TestServeHTTPDoesNotEmitEmptyMetricFamilyHeaders(t *testing.T) {
	exp := New(nvidia.Collector{})
	exp.snapshot = nvidia.Snapshot{
		GPUs: []nvidia.GPU{{
			Index:  "0",
			UUID:   "GPU-abc",
			Name:   "NVIDIA RTX",
			Values: map[string]float64{},
		}},
		CollectedAt: time.Unix(100, 0),
	}
	exp.lastAttempt = time.Unix(100, 0)
	exp.lastSuccess = time.Unix(100, 0)

	rec := httptest.NewRecorder()
	exp.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	if strings.Contains(body, "# HELP nvidia_smi_temperature_celsius") {
		t.Fatalf("unexpected empty metric family header:\n%s", body)
	}
}

func TestCollectFailureKeepsLastSnapshotAndMarksDown(t *testing.T) {
	collector := &fakeCollector{
		snapshot: nvidia.Snapshot{
			GPUs: []nvidia.GPU{{
				Index:  "0",
				UUID:   "GPU-abc",
				Name:   "NVIDIA RTX",
				Values: map[string]float64{"temperature.gpu": 42},
			}},
			CollectedAt: time.Unix(100, 0),
		},
	}
	exp := New(collector)
	exp.collect(context.Background())

	collector.err = errors.New("driver unavailable")
	exp.collect(context.Background())

	rec := httptest.NewRecorder()
	exp.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	assertContains(t, body, "nvidia_smi_up 0")
	assertContains(t, body, `nvidia_smi_gpu_info`)
	assertContains(t, body, `nvidia_smi_temperature_celsius{index="0",name="NVIDIA RTX",uuid="GPU-abc"} 42`)
}

func assertContains(t *testing.T, body, needle string) {
	t.Helper()
	if !strings.Contains(body, needle) {
		t.Fatalf("expected body to contain %q\n%s", needle, body)
	}
}
