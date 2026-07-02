package nvidia

var coreFields = []string{
	"index",
	"uuid",
	"name",
	"pci.bus_id",
	"driver_version",
	"pstate",
	"compute_mode",
	"temperature.gpu",
	"utilization.gpu",
	"utilization.memory",
	"memory.total",
	"memory.used",
	"memory.free",
	"fan.speed",
	"power.draw",
	"power.limit",
	"clocks.current.graphics",
	"clocks.current.sm",
	"clocks.current.memory",
	"clocks.current.video",
	"pcie.link.gen.current",
	"pcie.link.width.current",
}

var coreSpecs = []MetricSpec{
	{Key: "temperature.gpu", Field: "temperature.gpu", Name: "nvidia_smi_temperature_celsius", Help: "Current GPU temperature in degrees Celsius.", Unit: unitCelsius},
	{Key: "utilization.gpu", Field: "utilization.gpu", Name: "nvidia_smi_utilization_ratio", Help: "Current GPU utilization ratio.", Unit: unitPercentRatio, Extra: map[string]string{"unit": "gpu"}},
	{Key: "utilization.memory", Field: "utilization.memory", Name: "nvidia_smi_utilization_ratio", Help: "Current GPU utilization ratio.", Unit: unitPercentRatio, Extra: map[string]string{"unit": "memory"}},
	{Key: "memory.total", Field: "memory.total", Name: "nvidia_smi_memory_bytes", Help: "GPU memory in bytes.", Unit: unitMiBBytes, Extra: map[string]string{"kind": "total"}},
	{Key: "memory.used", Field: "memory.used", Name: "nvidia_smi_memory_bytes", Help: "GPU memory in bytes.", Unit: unitMiBBytes, Extra: map[string]string{"kind": "used"}},
	{Key: "memory.free", Field: "memory.free", Name: "nvidia_smi_memory_bytes", Help: "GPU memory in bytes.", Unit: unitMiBBytes, Extra: map[string]string{"kind": "free"}},
	{Key: "fan.speed", Field: "fan.speed", Name: "nvidia_smi_fan_speed_ratio", Help: "Fan speed ratio.", Unit: unitPercentRatio},
	{Key: "power.draw", Field: "power.draw", Name: "nvidia_smi_power_watts", Help: "GPU power in watts.", Unit: unitWatts, Extra: map[string]string{"kind": "draw"}},
	{Key: "power.limit", Field: "power.limit", Name: "nvidia_smi_power_watts", Help: "GPU power in watts.", Unit: unitWatts, Extra: map[string]string{"kind": "limit"}},
	{Key: "clocks.current.graphics", Field: "clocks.current.graphics", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "graphics", "kind": "current"}},
	{Key: "clocks.current.sm", Field: "clocks.current.sm", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "sm", "kind": "current"}},
	{Key: "clocks.current.memory", Field: "clocks.current.memory", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "memory", "kind": "current"}},
	{Key: "clocks.current.video", Field: "clocks.current.video", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "video", "kind": "current"}},
	{Key: "pcie.link.gen.current", Field: "pcie.link.gen.current", Name: "nvidia_smi_pcie_link_generation", Help: "Current PCIe link generation.", Unit: unitRaw},
	{Key: "pcie.link.width.current", Field: "pcie.link.width.current", Name: "nvidia_smi_pcie_link_width_lanes", Help: "Current PCIe link width in lanes.", Unit: unitRaw},
}

var optionalGroups = []struct {
	fields []string
	specs  []MetricSpec
}{
	{
		fields: []string{
			"index",
			"uuid",
			"cuda_version",
		},
		specs: nil,
	},
	{
		fields: []string{
			"index",
			"uuid",
			"memory.reserved",
			"clocks.max.graphics",
			"clocks.max.sm",
			"clocks.max.memory",
			"clocks.max.video",
			"utilization.encoder",
			"utilization.decoder",
			"encoder.stats.sessionCount",
			"encoder.stats.averageFps",
			"encoder.stats.averageLatency",
		},
		specs: []MetricSpec{
			{Key: "memory.reserved", Field: "memory.reserved", Name: "nvidia_smi_memory_bytes", Help: "GPU memory in bytes.", Unit: unitMiBBytes, Extra: map[string]string{"kind": "reserved"}},
			{Key: "clocks.max.graphics", Field: "clocks.max.graphics", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "graphics", "kind": "max"}},
			{Key: "clocks.max.sm", Field: "clocks.max.sm", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "sm", "kind": "max"}},
			{Key: "clocks.max.memory", Field: "clocks.max.memory", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "memory", "kind": "max"}},
			{Key: "clocks.max.video", Field: "clocks.max.video", Name: "nvidia_smi_clock_hertz", Help: "GPU clock frequency in hertz.", Unit: unitMHzHertz, Extra: map[string]string{"clock": "video", "kind": "max"}},
			{Key: "utilization.encoder", Field: "utilization.encoder", Name: "nvidia_smi_utilization_ratio", Help: "Current GPU utilization ratio.", Unit: unitPercentRatio, Extra: map[string]string{"unit": "encoder"}},
			{Key: "utilization.decoder", Field: "utilization.decoder", Name: "nvidia_smi_utilization_ratio", Help: "Current GPU utilization ratio.", Unit: unitPercentRatio, Extra: map[string]string{"unit": "decoder"}},
			{Key: "encoder.stats.sessionCount", Field: "encoder.stats.sessionCount", Name: "nvidia_smi_encoder_sessions", Help: "Current encoder session count.", Unit: unitRaw},
			{Key: "encoder.stats.averageFps", Field: "encoder.stats.averageFps", Name: "nvidia_smi_encoder_average_fps", Help: "Average encoder frames per second.", Unit: unitRaw},
			{Key: "encoder.stats.averageLatency", Field: "encoder.stats.averageLatency", Name: "nvidia_smi_encoder_average_latency", Help: "Average encoder latency as reported by nvidia-smi.", Unit: unitRaw},
		},
	},
	{
		fields: []string{
			"index",
			"uuid",
			"mig.mode.current",
			"clocks_throttle_reasons.gpu_idle",
			"clocks_throttle_reasons.applications_clocks_setting",
			"clocks_throttle_reasons.sw_power_cap",
			"clocks_throttle_reasons.hw_slowdown",
			"clocks_throttle_reasons.hw_thermal_slowdown",
			"clocks_throttle_reasons.hw_power_brake_slowdown",
			"clocks_throttle_reasons.sw_thermal_slowdown",
			"clocks_throttle_reasons.sync_boost",
		},
		specs: []MetricSpec{
			{Key: "clocks_throttle_reasons.gpu_idle", Field: "clocks_throttle_reasons.gpu_idle", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "gpu_idle"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.applications_clocks_setting", Field: "clocks_throttle_reasons.applications_clocks_setting", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "applications_clocks_setting"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.sw_power_cap", Field: "clocks_throttle_reasons.sw_power_cap", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "sw_power_cap"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.hw_slowdown", Field: "clocks_throttle_reasons.hw_slowdown", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "hw_slowdown"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.hw_thermal_slowdown", Field: "clocks_throttle_reasons.hw_thermal_slowdown", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "hw_thermal_slowdown"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.hw_power_brake_slowdown", Field: "clocks_throttle_reasons.hw_power_brake_slowdown", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "hw_power_brake_slowdown"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.sw_thermal_slowdown", Field: "clocks_throttle_reasons.sw_thermal_slowdown", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "sw_thermal_slowdown"}, ValueType: "bool"},
			{Key: "clocks_throttle_reasons.sync_boost", Field: "clocks_throttle_reasons.sync_boost", Name: "nvidia_smi_clock_throttle_reason", Help: "Clock throttle reason status, 1 when active.", Unit: unitRaw, Extra: map[string]string{"reason": "sync_boost"}, ValueType: "bool"},
		},
	},
	{
		fields: []string{
			"index",
			"uuid",
			"ecc.errors.corrected.volatile.total",
			"ecc.errors.uncorrected.volatile.total",
			"ecc.errors.corrected.aggregate.total",
			"ecc.errors.uncorrected.aggregate.total",
			"retired_pages.single_bit_ecc.count",
			"retired_pages.double_bit.count",
			"remapped_rows.correctable",
			"remapped_rows.uncorrectable",
		},
		specs: []MetricSpec{
			{Key: "ecc.errors.corrected.volatile.total", Field: "ecc.errors.corrected.volatile.total", Name: "nvidia_smi_ecc_errors_total", Help: "ECC error count.", Unit: unitRaw, Extra: map[string]string{"correction": "corrected", "scope": "volatile"}},
			{Key: "ecc.errors.uncorrected.volatile.total", Field: "ecc.errors.uncorrected.volatile.total", Name: "nvidia_smi_ecc_errors_total", Help: "ECC error count.", Unit: unitRaw, Extra: map[string]string{"correction": "uncorrected", "scope": "volatile"}},
			{Key: "ecc.errors.corrected.aggregate.total", Field: "ecc.errors.corrected.aggregate.total", Name: "nvidia_smi_ecc_errors_total", Help: "ECC error count.", Unit: unitRaw, Extra: map[string]string{"correction": "corrected", "scope": "aggregate"}},
			{Key: "ecc.errors.uncorrected.aggregate.total", Field: "ecc.errors.uncorrected.aggregate.total", Name: "nvidia_smi_ecc_errors_total", Help: "ECC error count.", Unit: unitRaw, Extra: map[string]string{"correction": "uncorrected", "scope": "aggregate"}},
			{Key: "retired_pages.single_bit_ecc.count", Field: "retired_pages.single_bit_ecc.count", Name: "nvidia_smi_retired_pages_total", Help: "Retired GPU memory page count.", Unit: unitRaw, Extra: map[string]string{"reason": "single_bit_ecc"}},
			{Key: "retired_pages.double_bit.count", Field: "retired_pages.double_bit.count", Name: "nvidia_smi_retired_pages_total", Help: "Retired GPU memory page count.", Unit: unitRaw, Extra: map[string]string{"reason": "double_bit_ecc"}},
			{Key: "remapped_rows.correctable", Field: "remapped_rows.correctable", Name: "nvidia_smi_remapped_rows_total", Help: "Remapped memory row count.", Unit: unitRaw, Extra: map[string]string{"reason": "correctable"}},
			{Key: "remapped_rows.uncorrectable", Field: "remapped_rows.uncorrectable", Name: "nvidia_smi_remapped_rows_total", Help: "Remapped memory row count.", Unit: unitRaw, Extra: map[string]string{"reason": "uncorrectable"}},
		},
	},
}

func AllMetricSpecs() []MetricSpec {
	specs := append([]MetricSpec{}, coreSpecs...)
	for _, group := range optionalGroups {
		specs = append(specs, group.specs...)
	}
	return specs
}
