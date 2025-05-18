package metrics

import "time"

var Client MetricsClient
var DatadogClient MetricsClient

const (
	JobRequest  = "job_generator.job_request"
	JobSuccess  = "job_generator.job_success"
	JobFailure  = "job_generator.job_failure"
	JobLatency  = "job_generator.job_latency"
	JobDuration = "job_generator.job_duration"

	QueueSize = "job_generator.queue_size"

	WorkerNum         = "job_generator.worker_num"
	RunningWorkerNum  = "job_generator.running_worker_num"
	ExpectedWorkerNum = "job_generator.expected_worker_num"
)

type MetricsClient interface {
	Count(key string)
	Gauge(key string, value float64)
	Time(key string, value time.Duration)

	ReadCount(t time.Time, key string) float64
	ReadGauge(t time.Time, key string) float64
	ReadTime(t time.Time, key string) time.Duration

	Close()
}

type DummyClient struct{}

func NewDummyClient() *DummyClient {
	return &DummyClient{}
}

func (c *DummyClient) Count(key string)                     {}
func (c *DummyClient) Gauge(key string, value float64)      {}
func (c *DummyClient) Time(key string, value time.Duration) {}
func (c *DummyClient) ReadCount(t time.Time, key string) float64 {
	return 0
}
func (c *DummyClient) ReadGauge(t time.Time, key string) float64 {
	return 0
}
func (c *DummyClient) ReadTime(t time.Time, key string) time.Duration {
	return 0
}
func (c *DummyClient) Close() {}
