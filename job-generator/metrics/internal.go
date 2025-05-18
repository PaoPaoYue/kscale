package metrics

import (
	"github.com/nakabonne/tstorage"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"time"
)

type InternalClient struct {
	DummyClient
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

func (client *InternalClient) Count(key string) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: 1},
		},
	})
}

func (client *InternalClient) Gauge(key string, value float64) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: value},
		},
	})
}

func (client *InternalClient) Time(key string, value time.Duration) {
	_ = client.storage.InsertRows([]tstorage.Row{
		{
			Metric:    key,
			DataPoint: tstorage.DataPoint{Timestamp: getCurrentTimestamp(), Value: float64(value.Milliseconds())},
		},
	})
}

func (client *InternalClient) ReadCount(t time.Time, key string) float64 {
	points := client.readWithFallback(t, key)
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

func (client *InternalClient) ReadGauge(t time.Time, key string) float64 {
	points := client.readWithFallback(t, key)
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

func (client *InternalClient) ReadTime(t time.Time, key string) time.Duration {
	points := client.readWithFallback(t, key)
	if len(points) == 0 {
		return 0
	}
	// return the avg point
	var sum float64
	var count int
	for _, point := range points {
		sum += point.Value
		count++
	}
	if count == 0 {
		return 0
	}
	avg := sum / float64(count)
	return time.Duration(avg) * time.Millisecond
}

func (client *InternalClient) Close() {
	_ = client.storage.Close()
}

func (client *InternalClient) readWithFallback(t time.Time, key string) []*tstorage.DataPoint {
	start, end := getCurrentInterval(t)
	points, _ := client.storage.Select(key, nil, start, end)
	if len(points) == 0 {
		// fallback to last interval
		start, end = getLastInterval(t)
		points, _ = client.storage.Select(key, nil, start, end)
	}
	return points
}

func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func getCurrentInterval(t time.Time) (int64, int64) {
	target := t.Unix()
	return (target - int64(config.C.MetricsWindow)) * 1000, target * 1000
}

func getLastInterval(t time.Time) (int64, int64) {
	target := t.Unix()
	return (target - int64(config.C.MetricsWindow*2)) * 1000, (target - int64(config.C.MetricsWindow)) * 1000
}
