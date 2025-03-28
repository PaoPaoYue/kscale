package metrics

var Client *InternalClient
var DatadogClient *DogStatsDClient

const (
	JobRequest  = "job_generator.job_request"
	JobSuccess  = "job_generator.job_success"
	JobFailure  = "job_generator.job_failure"
	JobLatency  = "job_generator.job_latency"
	JobDuration = "job_generator.job_duration"

	QueueSize      = "job_generator.queue_size"
	RetryQueueSize = "job_generator.retry_queue_size"

	WorkerNum           = "job_generator.worker_num"
	WorkerStartDuration = "job_generator.worker_start_duration"
	NodeNum             = "job_generator.node_num"
)
