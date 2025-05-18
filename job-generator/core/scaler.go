package core

import (
	"github.com/paopaoyue/kscale/job-genrator/api"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

type DataPoint struct {
	ExpectedWorker int
	RunningWorker  int
	TotalWorker    int
	NewJob         int
	OngoingJob     int
	CompletedJob   int
	AvgDuration    float64 // in milliseconds
	AvgDelay       float64 // in milliseconds
	Reward         float64
}

type Scaler struct {
	time              int // in seconds
	jobBatchName      string
	jobBatchStartTime time.Time

	dataPointList []DataPoint

	stepTicker   *time.Ticker
	reportTicker *time.Ticker
	stopChan     chan struct{}

	expectedWorker     int
	queueSize          atomic.Int32
	windowCompletedJob []Job
	reward             float64
}

func NewScaler() *Scaler {
	return &Scaler{
		dataPointList: []DataPoint{},

		stepTicker:   time.NewTicker(time.Duration(config.C.MetricsWindow) * time.Second),
		reportTicker: time.NewTicker(1 * time.Second),
		stopChan:     make(chan struct{}),

		queueSize:          atomic.Int32{},
		windowCompletedJob: []Job{},
	}
}

func (s *Scaler) Start(jobBatchName string, jobBatchStartTime time.Time) {
	s.time = 0
	s.reward = 0
	s.expectedWorker = config.C.InitWorkerCount
	s.jobBatchName = jobBatchName
	s.jobBatchStartTime = jobBatchStartTime
	file := OpenCSVAndWriteHeader(filepath.Join(config.C.OutputFilePath, s.jobBatchName+"-metrics.csv"),
		[]string{
			"Time",
			"Expected Worker",
			"Running Worker",
			"Total Worker",
			"New Job",
			"Ongoing Job",
			"Completed Job",
			"Avg Duration",
			"Avg Delay",
			"Reward",
		})
	go func() {
		// catch panic
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Scaler recovered from panic", "error", r)
			}
		}()
		defer file.Close()
		for {
			select {
			case <-s.stepTicker.C:
				s.step(file)
			case <-s.reportTicker.C:
				s.report()
			case <-s.stopChan:
				return
			}
		}
	}()

}

func (s *Scaler) Stop() {
	s.stepTicker.Stop()
	s.reportTicker.Stop()
	close(s.stopChan)
}

func (s *Scaler) PreProcessJob(job Job) {
	s.queueSize.Add(1)
	metrics.Client.Count(metrics.JobRequest)
	metrics.DatadogClient.Count(metrics.JobRequest)
}

func (s *Scaler) PostProcessJob(job Job) {
	s.queueSize.Add(-1)

	if job.Success {
		metrics.Client.Count(metrics.JobSuccess)
		metrics.DatadogClient.Count(metrics.JobSuccess)

		metrics.Client.Time(metrics.JobDuration, job.Duration)
		metrics.DatadogClient.Time(metrics.JobDuration, job.Duration)

		metrics.Client.Time(metrics.JobLatency, job.EndTime.Sub(job.RequestTime))
		metrics.DatadogClient.Time(metrics.JobLatency, job.EndTime.Sub(job.RequestTime))
	} else {
		metrics.Client.Count(metrics.JobFailure)
		metrics.DatadogClient.Count(metrics.JobFailure)
	}

	s.windowCompletedJob = append(s.windowCompletedJob, job)
}

func (s *Scaler) step(outputFile *os.File) {
	s.time += config.C.MetricsWindow

	// calculate data point
	aggregateTime := s.jobBatchStartTime.Add(time.Duration(s.time) * time.Second)
	dp := DataPoint{
		ExpectedWorker: s.expectedWorker,
		RunningWorker:  int(math.Round(metrics.Client.ReadGauge(aggregateTime, metrics.RunningWorkerNum))),
		TotalWorker:    int(math.Round(metrics.Client.ReadGauge(aggregateTime, metrics.WorkerNum))),
		NewJob:         int(math.Round(metrics.Client.ReadCount(aggregateTime, metrics.JobRequest))),
		OngoingJob:     int(s.queueSize.Load()),
		CompletedJob:   int(math.Round(metrics.Client.ReadCount(aggregateTime, metrics.JobSuccess))),
		AvgDuration:    float64(metrics.Client.ReadTime(aggregateTime, metrics.JobDuration).Milliseconds()),
		AvgDelay:       float64(metrics.Client.ReadTime(aggregateTime, metrics.JobLatency).Milliseconds()),
	}
	for _, job := range s.windowCompletedJob {
		delay := job.EndTime.Sub(job.RequestTime)
		if job.Success && delay.Milliseconds() < int64(config.C.LatencyThreshold) {
			s.reward += config.C.JobReward
		}
	}
	s.reward -= config.C.WorkerCostPerHour * float64(config.C.MetricsWindow) / 3600.0 * float64(dp.TotalWorker)
	dp.Reward = s.reward

	s.dataPointList = append(s.dataPointList, dp)

	// scale worker
	if config.C.EnableAutoScaling {
		go func() {
			param := api.CalcWorkerCountRequestParam{
				Time:   s.time,
				Points: []api.DataPoint{},
			}
			dataPointLen := len(s.dataPointList)
			for i := config.C.ForecastWindow - 1; i >= 0; i-- {
				if dataPointLen-i-1 < 0 {
					continue
				} else {
					dp := s.dataPointList[dataPointLen-i-1]
					param.Points = append(param.Points, api.DataPoint{
						RunningWorker: dp.RunningWorker,
						NewJob:        dp.NewJob,
						OngoingJob:    dp.OngoingJob,
						CompletedJob:  dp.CompletedJob,
						AvgDuration:   dp.AvgDuration,
						AvgDelay:      dp.AvgDelay,
					})
				}
			}
			expectedWorker, err := api.CalcWorkerCount("http://"+config.C.APIEndpoint, param)
			if err != nil {
				slog.Error("Failed to calculate worker count", "err", err)
				return
			}
			if expectedWorker != s.expectedWorker {
				s.expectedWorker = expectedWorker
				err := api.ScaleWorker("http://"+config.C.RayDashboardEndpoint, expectedWorker)
				if err != nil {
					slog.Error("Failed to scale worker", "err", err)
					return
				}
			}
		}()
	}

	// write to file
	AppendCSV(outputFile, []string{
		strconv.Itoa(s.time),
		strconv.Itoa(dp.ExpectedWorker),
		strconv.Itoa(dp.RunningWorker),
		strconv.Itoa(dp.TotalWorker),
		strconv.Itoa(dp.NewJob),
		strconv.Itoa(dp.OngoingJob),
		strconv.Itoa(dp.CompletedJob),
		strconv.FormatFloat(dp.AvgDuration, 'f', 2, 64),
		strconv.FormatFloat(dp.AvgDelay, 'f', 2, 64),
		strconv.FormatFloat(dp.Reward, 'f', 8, 64),
	})

	// reset window completed job
	s.windowCompletedJob = []Job{}
}

func (s *Scaler) report() {
	running, total, err := api.GetWorkerCount("http://" + config.C.RayDashboardEndpoint)
	if err != nil {
		slog.Error("Failed to get worker count", "err", err)
		return
	}

	metrics.Client.Gauge(metrics.QueueSize, float64(s.queueSize.Load()))
	metrics.DatadogClient.Gauge(metrics.QueueSize, float64(s.queueSize.Load()))

	metrics.Client.Gauge(metrics.ExpectedWorkerNum, float64(s.expectedWorker))
	metrics.DatadogClient.Gauge(metrics.ExpectedWorkerNum, float64(s.expectedWorker))

	metrics.Client.Gauge(metrics.RunningWorkerNum, float64(running))
	metrics.DatadogClient.Gauge(metrics.RunningWorkerNum, float64(running))

	metrics.Client.Gauge(metrics.WorkerNum, float64(total))
	metrics.DatadogClient.Gauge(metrics.WorkerNum, float64(total))
}
