package main

import (
	"context"
	"fmt"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/core"
	"github.com/paopaoyue/kscale/job-genrator/handler"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig()

	r := gin.Default()

	r.POST("/submit-job", handler.SubmitJobHandler)
	r.GET("/download-result", handler.DownloadResultHandler)
	r.GET("/download-metrics", handler.DownloadMetricsHandler)
	r.GET("/metrics", handler.MetricsHandler)

	port := fmt.Sprintf(":%d", config.C.Port)
	slog.Info("Server starting...", "port", config.C.Port)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	initialize()
	defer shutdown()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err.Error())
		}
	}()

	<-ctx.Done()
	slog.Info("Shutdown signal received, exiting gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err.Error())
	} else {
		slog.Info("Server exited gracefully")
	}
}

func initialize() {
	metrics.Client = metrics.NewInternalClient()

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("Failed to create in-cluster config", "error", err.Error())
	} else {
		k8sClient, err := kubernetes.NewForConfig(k8sConfig)
		if err != nil {
			slog.Error("Failed to create Kubernetes client", "error", err.Error())
		} else {
			if endpoint, _ := metrics.DiscoverDogStatsDEndpoint(k8sClient); err == nil {
				metrics.DatadogClient, _ = metrics.NewDogStatsDClient(endpoint)
			}
		}
	}
	if metrics.DatadogClient == nil {
		slog.Warn("Datadog client cannot be auto-discovered, using internal metrics only")
		metrics.DatadogClient = metrics.NewDummyClient()
	}

	core.Scheduler = core.NewJobScheduler()
	core.Scheduler.Start()
}

func shutdown() {
	core.Scheduler.Stop()

	metrics.Client.Close()
	metrics.DatadogClient.Close()
}
