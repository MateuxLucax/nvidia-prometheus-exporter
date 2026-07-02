// Command nvidia-prometheus-exporter polls nvidia-smi and exposes cached GPU
// metrics in Prometheus text format at /metrics.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MateuxLucax/nvidia-prometheus-exporter/internal/exporter"
	"github.com/MateuxLucax/nvidia-prometheus-exporter/internal/nvidia"
)

func main() {
	var (
		port           = env("PORT", "3000")
		interval       = envDuration("COLLECT_INTERVAL", 5*time.Second)
		timeout        = envDuration("COLLECT_TIMEOUT", 3*time.Second)
		nvidiaSMIPath  = env("NVIDIA_SMI_PATH", "nvidia-smi")
		processMetrics = envBool("ENABLE_PROCESS_METRICS", false)
	)

	exp := exporter.New(nvidia.Collector{
		Path:                 nvidiaSMIPath,
		Timeout:              timeout,
		EnableProcessMetrics: processMetrics,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go exp.Start(ctx, interval)

	mux := http.NewServeMux()
	mux.Handle("/metrics", exp)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("nvidia-prometheus-exporter listening on :%s, polling %s every %s", port, nvidiaSMIPath, interval)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		log.Printf("invalid %s=%q, using default %s", key, v, fallback)
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		switch v {
		case "1", "t", "T", "true", "TRUE", "True", "yes", "YES", "on", "ON":
			return true
		case "0", "f", "F", "false", "FALSE", "False", "no", "NO", "off", "OFF":
			return false
		default:
			log.Printf("invalid %s=%q, using default %t", key, v, fallback)
		}
	}
	return fallback
}
