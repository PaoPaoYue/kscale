package core

import (
	"errors"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/util"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type JobScheduler struct {
	active            bool
	jobBatchStartTime time.Time
	jobBatchName      string
	jobBatchSize      int

	jobChan    chan Job
	outputChan chan Job
	stopChan   chan struct{}

	worker *JobWorker
	scaler *Scaler

	mu        *sync.Mutex
	jobTicker *time.Ticker
}

func NewJobScheduler() *JobScheduler {
	return &JobScheduler{
		active:       false,
		jobBatchName: "",
		jobBatchSize: 0,
		jobChan:      make(chan Job, config.C.MaxQueueSize),
		outputChan:   make(chan Job),
		stopChan:     make(chan struct{}),
		mu:           &sync.Mutex{},
	}
}

func (js *JobScheduler) Start() {
	ep, _ := util.NewEndpoint(config.C.APIEndpoint)
	js.worker = NewJobWorker(ep, js.jobChan, js.outputChan)
	js.worker.Start()

}

func (js *JobScheduler) Stop() {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.jobTicker != nil {
		js.jobTicker.Stop()
	}
	time.Sleep(time.Duration(config.C.ShutdownPeriod) * time.Second) // wait for workers to finish

	js.worker.Stop()
	close(js.jobChan)
	close(js.outputChan)
	close(js.stopChan)
}

func (js *JobScheduler) SubmitJobs(jobBatchName string, file multipart.File) error {
	if js.active {
		slog.Warn("Job scheduler is already active")
		return errors.New("job scheduler is already active")
	}
	iter, err := ReadJobCSV(file)
	if err != nil {
		return err
	}

	go func() {
		// catch panic
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Ticker recovered from panic", "error", r)
			}
		}()

		js.active = true
		js.jobBatchName = jobBatchName
		js.jobBatchSize = iter.Size()
		js.jobBatchStartTime = time.Now()
		js.scaler = NewScaler()
		js.scaler.Start(jobBatchName, js.jobBatchStartTime)

		js.processOutput()

		js.jobTicker = time.NewTicker(1 * time.Millisecond)

		var (
			job     Job
			hasNext bool
		)
		if job, hasNext = iter.Next(); hasNext {
			for {
				select {
				case <-js.stopChan:
					return
				case <-js.jobTicker.C:
					current := time.Now()
					timeElapsed := current.Sub(js.jobBatchStartTime)
					jobTime := job.RequestTime.Sub(time.UnixMilli(0))
					for timeElapsed >= jobTime {
						job.RequestTime = current
						js.scaler.PreProcessJob(job)
						js.jobChan <- job
						if job, hasNext = iter.Next(); hasNext {
							jobTime = job.RequestTime.Sub(time.UnixMilli(0))
							if timeElapsed < jobTime {
								js.jobTicker.Reset(jobTime - timeElapsed)
							}
						} else {
							break
						}
					}
				}
				if !hasNext {
					js.jobTicker.Stop()
					break
				}
			}
		}
	}()

	return nil
}

func (js *JobScheduler) processOutput() {
	file := OpenCSVAndWriteHeader(filepath.Join(config.C.OutputFilePath, js.jobBatchName+"-result.csv"),
		[]string{
			"Id",
			"Success",
			"Retry",
			"RequestTime",
			"EndTime",
			"Duration",
			"Latency",
		})
	go func() {
		defer file.Close()
		var count int
		for {
			select {
			case <-js.stopChan:
				return
			case job := <-js.outputChan:
				js.scaler.PostProcessJob(job)
				AppendCSV(file, []string{
					job.Id,
					strconv.FormatBool(job.Success),
					strconv.Itoa(job.Retry),
					formatTimeWithMillis(job.RequestTime),
					formatTimeWithMillis(job.EndTime),
					strconv.FormatInt(job.Duration.Milliseconds(), 10),
					strconv.FormatInt(job.EndTime.Sub(job.RequestTime).Milliseconds(), 10),
				})
				count++
			}
			if count >= js.jobBatchSize {
				break
			}
		}

		slog.Info("Job batch completed", "Name", js.jobBatchName, "Size", js.jobBatchSize, "Duration", time.Since(js.jobBatchStartTime))
		js.active = false
		js.jobBatchName = ""
		js.jobBatchSize = 0
		js.jobBatchStartTime = time.Time{}
		js.scaler.Stop()
	}()
}
