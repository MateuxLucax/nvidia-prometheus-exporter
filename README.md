# NVIDIA Prometheus Exporter

Dependency-free Go exporter for NVIDIA GPU stats from `nvidia-smi`.

It polls `nvidia-smi` in the background, caches the latest successful snapshot, and exposes Prometheus-compatible metrics at `/metrics`. This keeps scrapes fast while still giving Grafana data for GPU usage, memory, clocks, temperature, power, fan speed, PCIe state, ECC, encoder/decoder utilization, and optional process memory.

## Run on the host

```bash
go run .
curl http://localhost:3000/metrics
```

The host must have a working NVIDIA driver and `nvidia-smi` in `PATH`.

## Run with Docker Compose

Docker usage requires the NVIDIA Container Toolkit and a Compose version that supports `gpus: all`.

```bash
docker compose up -d
curl http://localhost:3000/metrics
```

The compose file binds to `127.0.0.1:3000` because `/metrics` is unauthenticated. If Prometheus runs elsewhere, expose the port only on a trusted network.

To build the image locally instead of pulling the GHCR image:

```bash
docker compose -f docker-compose.local.yml up -d --build
```

## Docker image

Images are published to GitHub Container Registry when CI runs on `main` or a semver tag:

```bash
docker pull ghcr.io/mateuxlucax/nvidia-prometheus-exporter:latest
docker run --rm --gpus all -p 127.0.0.1:3000:3000 ghcr.io/mateuxlucax/nvidia-prometheus-exporter:latest
```

Available tags include `latest`, branch tags, `sha-<shortsha>`, and semver tags like `v0.1.0`, `0.1.0`, `0.1`, and `0`.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `3000` | HTTP port for `/metrics` |
| `COLLECT_INTERVAL` | `5s` | Background polling interval as a Go duration |
| `COLLECT_TIMEOUT` | `3s` | Timeout for each `nvidia-smi` command |
| `NVIDIA_SMI_PATH` | `nvidia-smi` | Path to the `nvidia-smi` binary |
| `ENABLE_PROCESS_METRICS` | `false` | Enable per-process GPU memory metrics |

## Metrics

Health metrics:

- `nvidia_smi_up`
- `nvidia_smi_collect_duration_seconds`
- `nvidia_smi_collect_last_attempt_timestamp_seconds`
- `nvidia_smi_collect_last_success_timestamp_seconds`

GPU metrics:

- `nvidia_smi_gpu_info`
- `nvidia_smi_utilization_ratio`
- `nvidia_smi_memory_bytes`
- `nvidia_smi_temperature_celsius`
- `nvidia_smi_power_watts`
- `nvidia_smi_fan_speed_ratio`
- `nvidia_smi_clock_hertz`
- `nvidia_smi_pcie_link_generation`
- `nvidia_smi_pcie_link_width_lanes`
- `nvidia_smi_clock_throttle_reason`
- `nvidia_smi_encoder_sessions`
- `nvidia_smi_encoder_average_fps`
- `nvidia_smi_encoder_average_latency`
- `nvidia_smi_ecc_errors_total`
- `nvidia_smi_retired_pages_total`
- `nvidia_smi_remapped_rows_total`

Optional process metric:

- `nvidia_smi_process_used_memory_bytes`

Unsupported or `N/A` values are skipped instead of emitted as zero.

## Prometheus

```yaml
scrape_configs:
  - job_name: nvidia
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:3000']
```

## Grafana

Import `assets/grafana/nvidia-smi-grafana-dashboard.json` and choose your Prometheus data source. The dashboard includes variables for `job`, `instance`, and `gpu_uuid`.

## Development

```bash
CGO_ENABLED=0 go test ./...
go run .
```

The tests use sample and fake `nvidia-smi` output; a real GPU is only required for manual acceptance.

## CI

The GitHub Actions workflow runs formatting checks, unit tests, and a static Go build on every pull request and push. It also builds the Docker image for pull requests, then publishes it to GHCR on `main`, semver tags, or manual workflow runs.
