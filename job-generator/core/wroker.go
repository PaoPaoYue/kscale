package core

import (
	"github.com/paopaoyue/kscale/job-genrator/api"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/util"
	"log/slog"
	"time"
)

type JobWorker struct {
	Endpoint util.Endpoint
	Hostname string

	jobChan    chan Job
	outputChan chan Job

	stopChan chan struct{}
}

func NewJobWorker(endpoint util.Endpoint, jobChan, outputChan chan Job) *JobWorker {
	return &JobWorker{
		Endpoint: endpoint,

		jobChan:    jobChan,
		outputChan: outputChan,
		stopChan:   make(chan struct{}),
	}
}

func (jw *JobWorker) Start() {

	go func() {
		for {
			select {
			case <-jw.stopChan:
				return
			default:
				select {
				case job, ok := <-jw.jobChan:
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

		job.Success = false
		job.EndTime = time.Now()
		jw.outputChan <- job
		return
	}

	jw.outputChan <- job
}
