package core

import (
	"context"
	"fmt"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/metrics"
	"github.com/paopaoyue/kscale/job-genrator/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type JobScheduler struct {
	client *kubernetes.Clientset

	Active            bool
	JobBatchStartTIme time.Time
	JobBatchName      string
	JobBatchSize      int

	JobChan    chan Job
	OutputChan chan Job
	StopChan   chan struct{}

	worker *JobWorker

	mu            *sync.Mutex
	jobTicker     *time.Ticker
	metricsTicker *time.Ticker

	queueSize  atomic.Int32
	workerPods map[util.Endpoint]string
}

func NewJobScheduler(client *kubernetes.Clientset) *JobScheduler {
	return &JobScheduler{
		client:       client,
		Active:       false,
		JobBatchName: "",
		JobBatchSize: 0,
		JobChan:      make(chan Job, config.C.MaxQueueSize),
		OutputChan:   make(chan Job),
		StopChan:     make(chan struct{}),
		mu:           &sync.Mutex{},
		workerPods:   make(map[util.Endpoint]string),
	}
}

func (js *JobScheduler) Start() {
	ep, _ := util.NewEndpoint(config.C.APIEndpoint)
	js.worker = NewJobWorker(ep, js)

	// Get list of pods for the deployment
	pods, err := js.client.CoreV1().Pods(config.C.Environment).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", config.C.AppName),
	})
	if err != nil {
		slog.Error("Failed to list pods using k8s api", "err", err.Error())
	}

	// Loop through the pods and get IP addresses of running pods
	for _, pod := range pods.Items {
		if isPodReady(&pod) {
			js.addWorkerPod(&pod)
		}
	}

	go js.watchEndpoints()
	go js.watchMetrics()
}

func (js *JobScheduler) Stop() {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.jobTicker != nil {
		js.jobTicker.Stop()
	}
	time.Sleep(time.Duration(config.C.ShutdownPeriod) * time.Second) // wait for workers to finish

	js.worker.Stop()
	close(js.JobChan)
	close(js.OutputChan)
	close(js.StopChan)
	if js.metricsTicker != nil {
		js.metricsTicker.Stop()
	}
}

func (js *JobScheduler) SubmitJobs(jobBatchName string, file multipart.File) error {
	if js.Active {
		slog.Warn("Job scheduler is already active")
	}
	iter, err := ReadCSV(file)
	if err != nil {
		return err
	}
	go func() {
		js.Active = true
		js.JobBatchName = jobBatchName
		js.JobBatchSize = iter.Size()
		js.JobBatchStartTIme = time.Now()

		js.processOutput()

		js.jobTicker = time.NewTicker(time.Second)

		if job, ok := iter.Next(); ok {
			for range js.jobTicker.C {
				current := time.Now()
				for current.Sub(js.JobBatchStartTIme) > job.RequestTime.Sub(time.UnixMilli(0)) {
					job.RequestTime = current
					metrics.Client.Count(metrics.JobRequest)
					metrics.DatadogClient.Count(metrics.JobRequest)
					js.queueSize.Add(1)
					js.JobChan <- job
					if job, ok = iter.Next(); !ok {
						js.jobTicker.Stop()
						break
					}
				}
			}
		}
	}()

	return nil
}

func (js *JobScheduler) watchEndpoints() {

	factory := informers.NewSharedInformerFactory(js.client, 30*time.Second)
	podInformer := factory.Core().V1().Pods().Informer()

	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Discovery recovered from panic", "error", r)
				}
			}()

			oldState := oldObj.(*v1.Pod)
			newState := newObj.(*v1.Pod)
			if newState.Labels["app"] != config.C.AppName {
				return
			}

			if !isPodReady(oldState) && isPodReady(newState) {
				js.addWorkerPod(newState)
			}

			if !isPodTerminating(oldState) && isPodTerminating(newState) {
				js.removeWorkerPod(newState)
			}
		},
		DeleteFunc: func(obj interface{}) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Discovery recovered from panic", "error", r)
				}
			}()

			state := obj.(*v1.Pod)

			if state.Labels["app"] != config.C.AppName {
				return
			}

			js.removeWorkerPod(state)
		},
	})
	if err != nil {
		slog.Error("Failed to add event handler to Pod informer", err, err.Error())
		return
	}

	factory.Start(js.StopChan)
	factory.WaitForCacheSync(js.StopChan)
}

func (js *JobScheduler) watchMetrics() {
	js.metricsTicker = time.NewTicker(time.Second)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Discovery recovered from panic", "error", r)
			}
		}()
		for range js.metricsTicker.C {
			var workerNum int
			var hostnames []string
			for _, hostname := range js.workerPods {
				workerNum++
				for _, name := range hostnames {
					if name != hostname {
						hostnames = append(hostnames, hostname)
					}
				}
			}

			metrics.Client.Gauge(metrics.QueueSize, float64(js.queueSize.Load()))
			metrics.Client.Gauge(metrics.WorkerNum, float64(workerNum))
			metrics.Client.Gauge(metrics.NodeNum, float64(len(hostnames)))

			metrics.DatadogClient.Gauge(metrics.QueueSize, float64(js.queueSize.Load()))
			metrics.DatadogClient.Gauge(metrics.WorkerNum, float64(workerNum))
			metrics.DatadogClient.Gauge(metrics.NodeNum, float64(len(hostnames)))
		}
	}()
}

func (js *JobScheduler) processOutput() {
	file := OpenCSVAndWriteHeader(filepath.Join(config.C.OutputFilePath, js.JobBatchName+"-result.csv"))
	go func() {
		defer file.Close()
		var count int
		for job := range js.OutputChan {
			js.queueSize.Add(-1)
			AppendCSV(file, job)
			count++
			if count >= js.JobBatchSize {
				break
			}
		}

		slog.Info("Job batch completed", "Name", js.JobBatchName, "Size", js.JobBatchSize, "Duration", time.Since(js.JobBatchStartTIme))
		js.Active = false
		js.JobBatchName = ""
		js.JobBatchSize = 0
		js.JobBatchStartTIme = time.Time{}
	}()
}

func (js *JobScheduler) addWorkerPod(pod *v1.Pod) {
	js.mu.Lock()
	defer js.mu.Unlock()
	ep, hostname, ok := extractPodSpec(pod)
	if ok {
		slog.Info("Adding pod Endpoint", "Name", pod.Name, "Host", ep.Host, "Port", ep.Port, "hostname", hostname)
		js.workerPods[ep] = hostname
	}
}

func (js *JobScheduler) removeWorkerPod(pod *v1.Pod) {
	js.mu.Lock()
	defer js.mu.Unlock()
	ep, hostname, ok := extractPodSpec(pod)
	if ok {
		for iep, _ := range js.workerPods {
			if iep == ep {
				slog.Info("Removing pod Endpoint", "Name", pod.Name, "Host", ep.Host, "Port", ep.Port, "hostname", hostname)
				delete(js.workerPods, ep)
				break
			}
		}
	}
}

func isPodReady(pod *v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isPodTerminating(pod *v1.Pod) bool {
	return pod.DeletionTimestamp != nil
}

func extractPodSpec(pod *v1.Pod) (util.Endpoint, string, bool) {
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			return util.Endpoint{
				Host: pod.Status.PodIP,
				Port: port.ContainerPort,
			}, pod.Spec.NodeName, true
		}
	}
	return util.Endpoint{}, "", false
}
