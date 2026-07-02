package exporter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MateuxLucax/nvidia-prometheus-exporter/internal/nvidia"
)

type collector interface {
	Collect(context.Context) (nvidia.Snapshot, error)
}

type Exporter struct {
	collector collector

	mu              sync.RWMutex
	snapshot        nvidia.Snapshot
	lastErr         error
	lastAttempt     time.Time
	lastSuccess     time.Time
	lastDurationSec float64
}

func New(collector collector) *Exporter {
	return &Exporter{collector: collector}
}

func (e *Exporter) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	e.collect(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.collect(ctx)
		}
	}
}

func (e *Exporter) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	e.mu.RLock()
	defer e.mu.RUnlock()

	var b strings.Builder
	writeHealth(&b, e)
	writeGPUInfo(&b, e.snapshot.GPUs)
	writeGPUMetrics(&b, e.snapshot.GPUs)
	writeProcessMetrics(&b, e.snapshot.Processes)
	_, _ = w.Write([]byte(b.String()))
}

func (e *Exporter) collect(ctx context.Context) {
	start := time.Now()
	snapshot, err := e.collector.Collect(ctx)
	duration := time.Since(start).Seconds()

	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastAttempt = start
	e.lastDurationSec = duration
	if err != nil {
		e.lastErr = err
		log.Printf("collect failed: %v", err)
		return
	}
	e.snapshot = snapshot
	e.lastErr = nil
	e.lastSuccess = snapshot.CollectedAt
}

func writeHealth(b *strings.Builder, e *Exporter) {
	up := 1.0
	if e.lastErr != nil || e.lastSuccess.IsZero() {
		up = 0
	}
	emitHeader(b, "nvidia_smi_up", "Whether the last nvidia-smi collection succeeded.", "gauge")
	emitSample(b, "nvidia_smi_up", nil, up)
	emitHeader(b, "nvidia_smi_collect_duration_seconds", "Duration of the last collection attempt in seconds.", "gauge")
	emitSample(b, "nvidia_smi_collect_duration_seconds", nil, e.lastDurationSec)
	emitHeader(b, "nvidia_smi_collect_last_attempt_timestamp_seconds", "Unix timestamp of the last collection attempt.", "gauge")
	emitSample(b, "nvidia_smi_collect_last_attempt_timestamp_seconds", nil, timestamp(e.lastAttempt))
	emitHeader(b, "nvidia_smi_collect_last_success_timestamp_seconds", "Unix timestamp of the last successful collection.", "gauge")
	emitSample(b, "nvidia_smi_collect_last_success_timestamp_seconds", nil, timestamp(e.lastSuccess))
}

func writeGPUInfo(b *strings.Builder, gpus []nvidia.GPU) {
	emitHeader(b, "nvidia_smi_gpu_info", "Static GPU information.", "gauge")
	for _, gpu := range gpus {
		labels := gpuLabels(gpu)
		labels["pci_bus_id"] = gpu.PCIBusID
		labels["driver_version"] = gpu.DriverVersion
		labels["cuda_version"] = gpu.CUDAVersion
		labels["pstate"] = gpu.PState
		labels["compute_mode"] = gpu.ComputeMode
		labels["mig_mode"] = gpu.MIGMode
		emitSample(b, "nvidia_smi_gpu_info", labels, 1)
	}
}

func writeGPUMetrics(b *strings.Builder, gpus []nvidia.GPU) {
	specs := nvidia.AllMetricSpecs()
	emittedHeaders := map[string]bool{}
	for _, gpu := range gpus {
		base := gpuLabels(gpu)
		for _, spec := range specs {
			value, ok := gpu.Values[spec.Key]
			if !ok {
				continue
			}
			if !emittedHeaders[spec.Name] {
				emitHeader(b, spec.Name, spec.Help, "gauge")
				emittedHeaders[spec.Name] = true
			}
			labels := cloneLabels(base)
			for k, v := range spec.Extra {
				labels[k] = v
			}
			emitSample(b, spec.Name, labels, value)
		}
	}
}

func writeProcessMetrics(b *strings.Builder, processes []nvidia.Process) {
	if len(processes) == 0 {
		return
	}
	emitHeader(b, "nvidia_smi_process_used_memory_bytes", "GPU memory used by a compute process in bytes.", "gauge")
	for _, proc := range processes {
		emitSample(b, "nvidia_smi_process_used_memory_bytes", map[string]string{
			"gpu_uuid":     proc.GPUUUID,
			"gpu_index":    proc.GPUIndex,
			"pid":          proc.PID,
			"process_name": proc.ProcessName,
		}, proc.UsedMemory)
	}
}

func gpuLabels(gpu nvidia.GPU) map[string]string {
	return map[string]string{
		"index": gpu.Index,
		"uuid":  gpu.UUID,
		"name":  gpu.Name,
	}
}

func cloneLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels))
	for k, v := range labels {
		cloned[k] = v
	}
	return cloned
}

func emitHeader(b *strings.Builder, name, help, typ string) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", name, typ)
}

func emitSample(b *strings.Builder, name string, labels map[string]string, value float64) {
	b.WriteString(name)
	if len(labels) > 0 {
		keys := make([]string, 0, len(labels))
		for k := range labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(k)
			b.WriteString("=\"")
			b.WriteString(escapeLabel(labels[k]))
			b.WriteByte('"')
		}
		b.WriteByte('}')
	}
	b.WriteByte(' ')
	b.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	b.WriteByte('\n')
}

func escapeLabel(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	return v
}

func timestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UnixNano()) / float64(time.Second)
}
