package core

import (
	"github.com/nakabonne/tstorage"
	"github.com/paopaoyue/kscale/job-genrator/api"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"github.com/paopaoyue/kscale/job-genrator/util"
	"time"
)

type JobWorker struct {
	Endpoint util.Endpoint
	Hostname string

	Active bool

	JobScheduler *JobScheduler

	stopChan chan struct{}

	metricsTags        []tstorage.Label
	dataDogMetricsTags []string
}

func NewJobWorker(endpoint util.Endpoint, hostname string, jobScheduler *JobScheduler) *JobWorker {
	return &JobWorker{
		Endpoint: endpoint,
		Hostname: hostname,

		JobScheduler: jobScheduler,
		stopChan:     make(chan struct{}),

		metricsTags: []tstorage.Label{
			{Name: "hostname", Value: hostname},
			{Name: "endpoint", Value: endpoint.String()},
		},
		dataDogMetricsTags: []string{
			string(metrics.NewTag("hostname", hostname)),
			string(metrics.NewTag("endpoint", endpoint.String())),
		},
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
					jw.processJob(job)
				case job, ok := <-jw.JobScheduler.RetryChan:
					if !ok {
						return
					}
					jw.processJob(job)
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
	t1 := time.Now()
	job.StartTime = t1.UnixMilli()

	err := api.GenerateImage("http://"+jw.Endpoint.String(), job.Param, job.Id)

	if err != nil {
		if job.Retry < config.C.MaxRetryCount {
			job.Retry++
			jw.JobScheduler.RetryChan <- job
			return
		} else {

			metrics.Client.Count(metrics.JobFailure, jw.metricsTags...)
			metrics.DatadogClient.Count(metrics.JobFailure, jw.dataDogMetricsTags...)

			jw.JobScheduler.OutputChan <- job
			return
		}
	}

	t2 := time.Now()
	job.EndTime = t2.UnixMilli()

	metrics.Client.Count(metrics.JobSuccess, jw.metricsTags...)
	metrics.DatadogClient.Count(metrics.JobSuccess, jw.dataDogMetricsTags...)

	metrics.Client.Time(metrics.JobDuration, t2.Sub(t1), jw.metricsTags...)
	metrics.DatadogClient.Time(metrics.JobDuration, t2.Sub(t1), jw.dataDogMetricsTags...)

	metrics.Client.Time(metrics.JobLatency, t2.Sub(time.UnixMilli(job.RequestTime)), jw.metricsTags...)
	metrics.DatadogClient.Time(metrics.JobLatency, t2.Sub(time.UnixMilli(job.RequestTime)), jw.dataDogMetricsTags...)

	jw.JobScheduler.OutputChan <- job
}
