package metrics

import (
	"github.com/nakabonne/tstorage"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"time"
)

type InternalClient struct {
	storage tstorage.Storage
}

func NewInternalClient() *InternalClient {
	storage, _ := tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Milliseconds),
	)
	return &InternalClient{
		storage: storage,
	}
}

func (client *InternalClient) Count(key string, tags ...tstorage.Label) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: 1},
			Labels:    tags,
		},
	})
}

func (client *InternalClient) Gauge(key string, value float64, tags ...tstorage.Label) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: value},
			Labels:    tags,
		},
	})
}

func (client *InternalClient) Time(key string, value time.Duration, tags ...tstorage.Label) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: float64(value.Milliseconds())},
			Labels:    tags,
		},
	})
}

func (client *InternalClient) ReadCount(key string, tags ...tstorage.Label) float64 {
	points := client.readWithFallback(key, tags...)
	if len(points) == 0 {
		return 0
	}
	// sum all points
	var sum float64
	for _, point := range points {
		sum += point.Value
	}
	return sum
}

func (client *InternalClient) ReadGauge(key string, tags ...tstorage.Label) float64 {
	points := client.readWithFallback(key, tags...)
	if len(points) == 0 {
		return 0
	}
	// return the latest point
	var latestValue float64
	var latestTimestamp int64
	for _, point := range points {
		if point.Timestamp > latestTimestamp {
			latestTimestamp = point.Timestamp
			latestValue = point.Value
		}
	}
	return latestValue
}

func (client *InternalClient) ReadTime(key string, tags ...tstorage.Label) time.Duration {
	points := client.readWithFallback(key, tags...)
	if len(points) == 0 {
		return 0
	}
	// return the max point
	var maxTime float64
	for _, point := range points {
		if point.Value > maxTime {
			maxTime = point.Value
		}
	}
	return time.Duration(maxTime) * time.Millisecond
}

func (client *InternalClient) Close() {
	_ = client.storage.Close()
}

func (client *InternalClient) readWithFallback(key string, tags ...tstorage.Label) []*tstorage.DataPoint {
	start, end := getCurrentInterval()
	points, _ := client.storage.Select(key, tags, start, end)
	if len(points) == 0 {
		// fallback to last interval
		start, end = getLastInterval()
		points, _ = client.storage.Select(key, tags, start, end)
	}
	return points
}

func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func getCurrentInterval() (int64, int64) {
	current := time.Now().Unix()
	return (current - int64(config.C.MetricsAggregationInterval)) * 1000, current * 1000
}

func getLastInterval() (int64, int64) {
	current := time.Now().Unix()
	return (current - int64(config.C.MetricsAggregationInterval*2)) * 1000, (current - int64(config.C.MetricsAggregationInterval)) * 1000
}
