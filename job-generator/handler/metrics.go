package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"net/http"
	"path/filepath"
	"time"
)

type MetricsRequest struct {
	Key string `json:"key"` // metrics key
}

func MetricsHandler(c *gin.Context) {
	var req MetricsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var value float64
	switch req.Key {
	case metrics.JobRequest, metrics.JobSuccess, metrics.JobFailure:
		value = metrics.Client.(*metrics.InternalClient).ReadCount(time.Now(), req.Key)
	case metrics.JobDuration, metrics.JobLatency:
		value = float64(metrics.Client.(*metrics.InternalClient).ReadTime(time.Now(), req.Key))
	case metrics.QueueSize, metrics.WorkerNum, metrics.RunningWorkerNum, metrics.ExpectedWorkerNum:
		value = metrics.Client.(*metrics.InternalClient).ReadGauge(time.Now(), req.Key)
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   req.Key,
		"value": value,
	})
}

func DownloadMetricsHandler(c *gin.Context) {
	batchName := c.DefaultQuery("batchname", "")

	if batchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batchname is required"})
		return
	}

	filePath := filepath.Join(config.C.OutputFilePath, fmt.Sprintf("%s-metrics.csv", batchName))

	if _, err := filepath.Abs(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("File %s not found", filePath)})
		return
	}

	c.File(filePath)
}
