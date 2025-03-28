package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/nakabonne/tstorage"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"net/http"
	"strings"
)

type MetricsRequest struct {
	Key  string   `json:"key"`  // metrics key
	Tags []string `json:"tags"` // ["key:value", "key2:value2"]
}

func MetricsHandler(c *gin.Context) {
	var req MetricsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	tags := make([]tstorage.Label, 0)
	for _, tag := range req.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			tags = append(tags, tstorage.Label{Name: parts[0], Value: parts[1]})
		}
	}

	var value float64
	switch req.Key {
	case metrics.JobRequest, metrics.JobSuccess, metrics.JobFailure:
		value = metrics.Client.ReadCount(req.Key, tags...)
	case metrics.JobDuration, metrics.JobLatency, metrics.WorkerStartDuration:
		value = float64(metrics.Client.ReadTime(req.Key, tags...))
	case metrics.QueueSize, metrics.RetryQueueSize, metrics.WorkerNum, metrics.NodeNum:
		value = metrics.Client.ReadGauge(req.Key, tags...)
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   req.Key,
		"value": value,
		"tags":  req.Tags,
	})
}
