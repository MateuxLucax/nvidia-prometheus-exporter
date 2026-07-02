package nvidia

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseGPUsConvertsUnitsAndSkipsNA(t *testing.T) {
	fields := []string{
		"index",
		"uuid",
		"name",
		"pci.bus_id",
		"driver_version",
		"pstate",
		"compute_mode",
		"temperature.gpu",
		"utilization.gpu",
		"memory.used",
		"fan.speed",
		"power.draw",
		"clocks.current.graphics",
	}
	specs := []MetricSpec{
		{Key: "temperature.gpu", Field: "temperature.gpu", Unit: unitCelsius},
		{Key: "utilization.gpu", Field: "utilization.gpu", Unit: unitPercentRatio},
		{Key: "memory.used", Field: "memory.used", Unit: unitMiBBytes},
		{Key: "fan.speed", Field: "fan.speed", Unit: unitPercentRatio},
		{Key: "power.draw", Field: "power.draw", Unit: unitWatts},
		{Key: "clocks.current.graphics", Field: "clocks.current.graphics", Unit: unitMHzHertz},
	}
	rows := [][]string{{
		"0",
		"GPU-abc",
		"NVIDIA RTX",
		"00000000:01:00.0",
		"555.42",
		"P2",
		"Default",
		"64",
		"72",
		"4096",
		"N/A",
		"120.5",
		"2100",
	}}

	gpus, err := parseGPUs(fields, specs, rows)
	if err != nil {
		t.Fatal(err)
	}
	gpu := gpus[0]
	assertFloat(t, gpu.Values["utilization.gpu"], 0.72)
	assertFloat(t, gpu.Values["memory.used"], 4096*1024*1024)
	assertFloat(t, gpu.Values["power.draw"], 120.5)
	assertFloat(t, gpu.Values["clocks.current.graphics"], 2100*1000*1000)
	if _, ok := gpu.Values["fan.speed"]; ok {
		t.Fatalf("fan.speed should be skipped for N/A values")
	}
}

func TestParseGPUsNormalizesInfoLabels(t *testing.T) {
	fields := []string{
		"index",
		"uuid",
		"name",
		"pci.bus_id",
		"driver_version",
		"pstate",
		"compute_mode",
	}
	rows := [][]string{{
		"0",
		"GPU-abc",
		"NVIDIA RTX",
		"00000000:01:00.0",
		"555.42",
		"[N/A]",
		"Not Supported",
	}}

	gpus, err := parseGPUs(fields, nil, rows)
	if err != nil {
		t.Fatal(err)
	}
	if gpus[0].PState != "" {
		t.Fatalf("expected normalized pstate, got %q", gpus[0].PState)
	}
	if gpus[0].ComputeMode != "" {
		t.Fatalf("expected normalized compute mode, got %q", gpus[0].ComputeMode)
	}
}

func TestCollectorReadsProcessMetricsWhenEnabled(t *testing.T) {
	path := fakeNvidiaSMI(t)
	collector := Collector{
		Path:                 path,
		Timeout:              time.Second,
		EnableProcessMetrics: true,
	}

	snapshot, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.GPUs) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(snapshot.GPUs))
	}
	if got := snapshot.GPUs[0].CUDAVersion; got != "12.6" {
		t.Fatalf("expected optional cuda version to merge, got %q", got)
	}
	assertFloat(t, snapshot.GPUs[0].Values["power.draw"], 18.42)
	if len(snapshot.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(snapshot.Processes))
	}
	proc := snapshot.Processes[0]
	if proc.GPUIndex != "0" || proc.PID != "1234" || proc.ProcessName != "/usr/bin/python" {
		t.Fatalf("unexpected process: %+v", proc)
	}
	assertFloat(t, proc.UsedMemory, 512*1024*1024)
}

func TestParsePowerSampleAverage(t *testing.T) {
	out := []byte(`
==============NVSMI LOG==============

GPU 00000000:01:00.0
    Power Readings
        Power Draw                      : N/A
    Power Samples
        Duration                        : 18.84 sec
        Number of Samples               : 119
        Max                             : 18.99 W
        Min                             : 17.88 W
        Avg                             : 18.42 W
`)

	got, ok := parsePowerSampleAverage(out)
	if !ok {
		t.Fatal("expected average power sample")
	}
	assertFloat(t, got, 18.42)
}

func TestParsePowerSampleAverageMissing(t *testing.T) {
	if _, ok := parsePowerSampleAverage([]byte("Power Readings\nPower Draw : N/A\n")); ok {
		t.Fatal("expected no average power sample")
	}
}

func TestPowerFallbackDoesNotOverwriteDirectPowerDraw(t *testing.T) {
	gpus, err := parseGPUs(
		[]string{"index", "uuid", "name", "power.draw"},
		[]MetricSpec{{Key: "power.draw", Field: "power.draw", Unit: unitWatts}},
		[][]string{{"0", "GPU-abc", "NVIDIA RTX", "120.5"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	collector := Collector{Path: fakeNvidiaSMI(t), Timeout: time.Second}
	collector.fillPowerDrawFallbacks(context.Background(), collector.Path, gpus)
	assertFloat(t, gpus[0].Values["power.draw"], 120.5)
}

func fakeNvidiaSMI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake is not used on windows")
	}
	path := filepath.Join(t.TempDir(), "nvidia-smi")
	script := `#!/bin/sh
case "$1" in
  --query-gpu=index,uuid,name,pci.bus_id,driver_version,pstate,compute_mode,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.used,memory.free,fan.speed,power.draw,power.limit,clocks.current.graphics,clocks.current.sm,clocks.current.memory,clocks.current.video,pcie.link.gen.current,pcie.link.width.current)
    echo "0, GPU-abc, NVIDIA RTX, 00000000:01:00.0, 555.42, P2, Default, 61, 80, 30, 8192, 4096, 4096, 42, N/A, 250, 2100, 2100, 7000, 1800, 4, 16"
    ;;
  --query-gpu=index,uuid,cuda_version)
    echo "0, GPU-abc, 12.6"
    ;;
  --query-compute-apps=gpu_uuid,pid,process_name,used_memory)
    echo "GPU-abc, 1234, /usr/bin/python, 512"
    ;;
  -i)
    if [ "$2" = "0" ] && [ "$3" = "-q" ] && [ "$4" = "-d" ] && [ "$5" = "POWER" ]; then
      cat <<'OUT'
Power Readings
    Power Draw                      : N/A
Power Samples
    Duration                        : 18.84 sec
    Number of Samples               : 119
    Max                             : 18.99 W
    Min                             : 17.88 W
    Avg                             : 18.42 W
OUT
    else
      exit 1
    fi
    ;;
  *)
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	const epsilon = 0.000001
	if got < want-epsilon || got > want+epsilon {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReadCSVTrimsFields(t *testing.T) {
	rows, err := readCSV([]byte("0, GPU-abc, NVIDIA RTX\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(rows[0], "|") != "0|GPU-abc|NVIDIA RTX" {
		t.Fatalf("unexpected row: %#v", rows[0])
	}
}
