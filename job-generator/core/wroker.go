package core

import (
	"github.com/paopaoyue/kscale/job-genrator/api"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"github.com/paopaoyue/kscale/job-genrator/util"
	"log/slog"
	"time"
)

type JobWorker struct {
	Endpoint util.Endpoint
	Hostname string

	Active bool

	JobScheduler *JobScheduler

	stopChan chan struct{}
}

func NewJobWorker(endpoint util.Endpoint, jobScheduler *JobScheduler) *JobWorker {
	return &JobWorker{
		Endpoint: endpoint,

		JobScheduler: jobScheduler,
		stopChan:     make(chan struct{}),
	}
}

func (jw *JobWorker) Start() {

	jw.Active = true
	go func() {
		for {
			select {
			case <-jw.stopChan:
				return
			default:
				select {
				case job, ok := <-jw.JobScheduler.JobChan:
					if !ok {
						return
					}
					go func() {
						// catch panic
						defer func() {
							if r := recover(); r != nil {
								slog.Error("Worker recovered from panic", "jobId", job.Id, "err", r)
							}
						}()
						jw.processJob(job)
					}()
				case <-jw.stopChan:
					return
				}
			}
		}
	}()
}

func (jw *JobWorker) Stop() {
	close(jw.stopChan)
}

func (jw *JobWorker) processJob(job Job) {
	for ; job.Retry < config.C.MaxRetryCount; job.Retry++ {
		slog.Info("Processing job", "jobId", job.Id, "retry", job.Retry)
		duration, err := api.GenerateImage("http://"+jw.Endpoint.String(), job.Param, job.Id)

		if err != nil {
			slog.Error("Error generating image, retrying...", "err", err, "jobId", job.Id, "retry", job.Retry+1)
		} else {
			job.Success = true
			job.EndTime = time.Now()
			job.Duration = duration
			break
		}
	}
	if job.Retry >= config.C.MaxRetryCount {
		slog.Error("Error generating image, max retries reached", "jobId", job.Id)
		metrics.Client.Count(metrics.JobFailure)
		metrics.DatadogClient.Count(metrics.JobFailure)

		job.Success = false
		job.EndTime = time.Now()
		jw.JobScheduler.OutputChan <- job
		return
	}

	metrics.Client.Count(metrics.JobSuccess)
	metrics.DatadogClient.Count(metrics.JobSuccess)

	metrics.Client.Time(metrics.JobDuration, job.Duration)
	metrics.DatadogClient.Time(metrics.JobDuration, job.Duration)

	metrics.Client.Time(metrics.JobLatency, job.EndTime.Sub(job.RequestTime))
	metrics.DatadogClient.Time(metrics.JobLatency, job.EndTime.Sub(job.RequestTime))

	jw.JobScheduler.OutputChan <- job
}
