# Build a static dependency-free Go binary, then run it in an NVIDIA base image
# so nvidia-smi is available when the container is started with GPU access.
FROM golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/exporter .

FROM nvidia/cuda:12.6.3-base-ubuntu24.04

COPY --from=build /out/exporter /exporter

ENV PORT=3000
ENV COLLECT_INTERVAL=5s
ENV COLLECT_TIMEOUT=3s
ENV NVIDIA_SMI_PATH=nvidia-smi
ENV ENABLE_PROCESS_METRICS=false

EXPOSE 3000
USER 65532:65532
ENTRYPOINT ["/exporter"]
