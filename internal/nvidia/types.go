package nvidia

import "time"

// Collector shells out to nvidia-smi and converts its CSV output into a
// normalized snapshot for the Prometheus exporter.
type Collector struct {
	Path                 string
	Timeout              time.Duration
	EnableProcessMetrics bool
}

type Snapshot struct {
	GPUs        []GPU
	Processes   []Process
	CollectedAt time.Time
}

type GPU struct {
	Index         string
	UUID          string
	Name          string
	PCIBusID      string
	DriverVersion string
	CUDAVersion   string
	PState        string
	ComputeMode   string
	MIGMode       string

	Values map[string]float64
}

type Process struct {
	GPUUUID     string
	GPUIndex    string
	PID         string
	ProcessName string
	UsedMemory  float64
}

type MetricSpec struct {
	Key       string
	Field     string
	Name      string
	Help      string
	Unit      unit
	Extra     map[string]string
	ValueType string
}

type unit int

const (
	unitRaw unit = iota
	unitPercentRatio
	unitMiBBytes
	unitMHzHertz
	unitWatts
	unitCelsius
)
