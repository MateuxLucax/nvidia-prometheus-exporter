package nvidia

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var optionalQueryLogOnce sync.Map

func (c Collector) Collect(ctx context.Context) (Snapshot, error) {
	path := c.Path
	if path == "" {
		path = "nvidia-smi"
	}

	rows, err := c.queryGPU(ctx, path, coreFields)
	if err != nil {
		return Snapshot{}, err
	}

	gpus, err := parseGPUs(coreFields, coreSpecs, rows)
	if err != nil {
		return Snapshot{}, err
	}
	byKey := map[string]*GPU{}
	for i := range gpus {
		byKey[gpus[i].key()] = &gpus[i]
	}

	for _, group := range optionalGroups {
		rows, err := c.queryGPU(ctx, path, group.fields)
		if err != nil {
			logOptionalQueryFailure("gpu:"+strings.Join(group.fields, ","), err)
			continue
		}
		_ = mergeGPUFields(group.fields, group.specs, rows, byKey)
	}
	c.fillPowerDrawFallbacks(ctx, path, gpus)

	var processes []Process
	if c.EnableProcessMetrics {
		var err error
		processes, err = c.queryProcesses(ctx, path, gpus)
		if err != nil {
			logOptionalQueryFailure("compute-apps", err)
		}
	}

	return Snapshot{
		GPUs:        gpus,
		Processes:   processes,
		CollectedAt: time.Now(),
	}, nil
}

func (c Collector) queryGPU(ctx context.Context, path string, fields []string) ([][]string, error) {
	ctx, cancel := c.commandContext(ctx)
	defer cancel()

	args := []string{
		"--query-gpu=" + strings.Join(fields, ","),
		"--format=csv,noheader,nounits",
	}
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return nil, commandError(err, out)
	}
	return readCSV(out)
}

func (c Collector) queryProcesses(ctx context.Context, path string, gpus []GPU) ([]Process, error) {
	ctx, cancel := c.commandContext(ctx)
	defer cancel()

	args := []string{
		"--query-compute-apps=gpu_uuid,pid,process_name,used_memory",
		"--format=csv,noheader,nounits",
	}
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return nil, commandError(err, out)
	}
	rows, err := readCSV(out)
	if err != nil {
		return nil, err
	}

	indexByUUID := map[string]string{}
	for _, gpu := range gpus {
		indexByUUID[gpu.UUID] = gpu.Index
	}

	processes := make([]Process, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		mem, ok := parseNumber(row[3], unitMiBBytes)
		if !ok {
			continue
		}
		uuid := clean(row[0])
		processes = append(processes, Process{
			GPUUUID:     uuid,
			GPUIndex:    indexByUUID[uuid],
			PID:         clean(row[1]),
			ProcessName: clean(row[2]),
			UsedMemory:  mem,
		})
	}
	return processes, nil
}

func (c Collector) fillPowerDrawFallbacks(ctx context.Context, path string, gpus []GPU) {
	for i := range gpus {
		if _, ok := gpus[i].Values["power.draw"]; ok {
			continue
		}
		value, err := c.queryPowerSampleAverage(ctx, path, gpus[i].Index)
		if err != nil {
			logOptionalQueryFailure("power-samples:"+gpus[i].Index, err)
			continue
		}
		gpus[i].Values["power.draw"] = value
	}
}

func (c Collector) queryPowerSampleAverage(ctx context.Context, path, index string) (float64, error) {
	if index == "" {
		return 0, fmt.Errorf("missing gpu index")
	}

	ctx, cancel := c.commandContext(ctx)
	defer cancel()

	out, err := exec.CommandContext(ctx, path, "-i", index, "-q", "-d", "POWER").CombinedOutput()
	if err != nil {
		return 0, commandError(err, out)
	}
	value, ok := parsePowerSampleAverage(out)
	if !ok {
		return 0, fmt.Errorf("power sample average not found")
	}
	return value, nil
}

func parseGPUs(fields []string, specs []MetricSpec, rows [][]string) ([]GPU, error) {
	gpus := make([]GPU, 0, len(rows))
	for _, row := range rows {
		values := fieldMap(fields, row)
		gpu := GPU{
			Index:         normalizeInfo(values["index"]),
			UUID:          normalizeInfo(values["uuid"]),
			Name:          normalizeInfo(values["name"]),
			PCIBusID:      normalizeInfo(values["pci.bus_id"]),
			DriverVersion: normalizeInfo(values["driver_version"]),
			PState:        normalizeInfo(values["pstate"]),
			ComputeMode:   normalizeInfo(values["compute_mode"]),
			Values:        map[string]float64{},
		}
		if gpu.Index == "" && gpu.UUID == "" {
			return nil, fmt.Errorf("nvidia-smi returned a GPU row without index or uuid")
		}
		fillMetrics(&gpu, specs, values)
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

func mergeGPUFields(fields []string, specs []MetricSpec, rows [][]string, byKey map[string]*GPU) error {
	for _, row := range rows {
		values := fieldMap(fields, row)
		key := values["uuid"]
		if key == "" {
			key = values["index"]
		}
		gpu, ok := byKey[key]
		if !ok {
			continue
		}
		if v := normalizeInfo(values["cuda_version"]); v != "" {
			gpu.CUDAVersion = v
		}
		if v := normalizeInfo(values["mig.mode.current"]); v != "" {
			gpu.MIGMode = v
		}
		fillMetrics(gpu, specs, values)
	}
	return nil
}

func fillMetrics(gpu *GPU, specs []MetricSpec, values map[string]string) {
	for _, spec := range specs {
		raw := values[spec.Field]
		var (
			v  float64
			ok bool
		)
		if spec.ValueType == "bool" {
			v, ok = parseBoolMetric(raw)
		} else {
			v, ok = parseNumber(raw, spec.Unit)
		}
		if ok {
			gpu.Values[spec.Key] = v
		}
	}
}

func readCSV(out []byte) ([][]string, error) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}
	reader := csv.NewReader(bytes.NewReader(out))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] = clean(rows[i][j])
		}
	}
	return rows, nil
}

func fieldMap(fields []string, row []string) map[string]string {
	values := map[string]string{}
	for i, field := range fields {
		if i < len(row) {
			values[field] = clean(row[i])
		}
	}
	return values
}

func parseNumber(raw string, unit unit) (float64, bool) {
	raw = clean(raw)
	if raw == "" || raw == "N/A" || raw == "[N/A]" || raw == "Not Supported" {
		return 0, false
	}
	raw = strings.TrimSuffix(raw, "%")
	raw = strings.TrimSuffix(raw, "MiB")
	raw = strings.TrimSuffix(raw, "MHz")
	raw = strings.TrimSuffix(raw, "W")
	raw = strings.TrimSpace(raw)

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case unitPercentRatio:
		return v / 100, true
	case unitMiBBytes:
		return v * 1024 * 1024, true
	case unitMHzHertz:
		return v * 1000 * 1000, true
	default:
		return v, true
	}
}

func parseBoolMetric(raw string) (float64, bool) {
	switch strings.ToLower(clean(raw)) {
	case "active", "enabled", "yes", "true", "1":
		return 1, true
	case "not active", "disabled", "no", "false", "0":
		return 0, true
	default:
		return parseNumber(raw, unitRaw)
	}
}

func parsePowerSampleAverage(out []byte) (float64, bool) {
	inPowerSamples := false
	for _, line := range strings.Split(string(out), "\n") {
		line = clean(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "Power Samples") {
			inPowerSamples = true
			continue
		}
		if !inPowerSamples || !strings.HasPrefix(line, "Avg") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return 0, false
		}
		return parseNumber(parts[1], unitWatts)
	}
	return 0, false
}

func clean(v string) string {
	return strings.TrimSpace(v)
}

func normalizeInfo(v string) string {
	v = clean(v)
	switch v {
	case "N/A", "[N/A]", "Not Supported":
		return ""
	default:
		return v
	}
}

func commandError(err error, out []byte) error {
	msg := strings.TrimSpace(string(out))
	if msg != "" {
		return fmt.Errorf("nvidia-smi failed: %s", msg)
	}
	return fmt.Errorf("nvidia-smi failed: %w", err)
}

func logOptionalQueryFailure(name string, err error) {
	if _, loaded := optionalQueryLogOnce.LoadOrStore(name, struct{}{}); loaded {
		return
	}
	log.Printf("optional nvidia-smi query %q failed: %v", name, err)
}

func (c Collector) commandContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return context.WithTimeout(ctx, timeout)
}

func (g GPU) key() string {
	if g.UUID != "" {
		return g.UUID
	}
	return g.Index
}
