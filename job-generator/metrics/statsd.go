package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"time"
)

type Tag string

func NewTag(key, value string) Tag {
	return Tag(fmt.Sprintf("%s:%s", key, value))
}

type DogStatsDClient struct {
	client *statsd.Client
}

func NewDogStatsDClient(endpoint util.Endpoint) (*DogStatsDClient, error) {
	client, err := statsd.New(fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port), statsd.WithAggregationInterval(time.Second*time.Duration(config.C.MetricsAggregationInterval)))
	if err != nil {
		return nil, err
	}
	return &DogStatsDClient{client: client}, nil
}

func (client *DogStatsDClient) Count(key string, tags ...string) {
	_ = client.client.Count(key, 1, tags, 1)
}

func (client *DogStatsDClient) Gauge(key string, value float64, tags ...string) {
	_ = client.client.Gauge(key, value, tags, 1)
}

func (client *DogStatsDClient) Time(key string, value time.Duration, tags ...string) {
	_ = client.client.TimeInMilliseconds(key, float64(value.Milliseconds()), tags, 1)
}

func (client *DogStatsDClient) Close() {
	_ = client.client.Close()
}

func DiscoverDogStatsDEndpoint(client *kubernetes.Clientset) (util.Endpoint, error) {
	namespaces, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		slog.Error("Failed to list namespaces using k8s api when auto discover datadog agent", "err", err.Error())
		return util.Endpoint{}, err
	}

	for _, ns := range namespaces.Items {

		// search for dagStatsD service
		services, err := client.CoreV1().Services(ns.Name).List(context.Background(), metav1.ListOptions{
			FieldSelector: "metadata.name=datadog-agent",
		})
		if err != nil {
			slog.Error("Failed to list services using k8s api when auto discover datadog agent", "err", err.Error())
			return util.Endpoint{}, err
		}
		if len(services.Items) > 0 {
			service := services.Items[0]
			host := service.Spec.ClusterIP
			port := 8125
			if len(service.Spec.Ports) > 0 {
				port = int(service.Spec.Ports[0].Port)
			}
			slog.Info("Auto discovered datadog agent, using dogStatD metrics", "host", host, "port", port)
			return util.Endpoint{
				Host: host,
				Port: int32(port),
			}, nil
		}

	}
	slog.Info("datadog agent not found")
	return util.Endpoint{}, errors.New("datadog agent not found")
}
