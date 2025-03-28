package util

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type Endpoint struct {
	Host string
	Port int32
}

func NewEndpoint(addr string) (Endpoint, bool) {
	split := strings.Split(addr, ":")
	if len(split) != 2 {
		slog.Error("Invalid endpoint address", "addr", addr)
		return Endpoint{}, false
	}
	if port, err := strconv.Atoi(split[1]); err != nil || port < 0 || port > 65535 {
		slog.Error("Invalid endpoint Port", "Port", split[1])
		return Endpoint{}, false
	} else {
		return Endpoint{
			Host: split[0],
			Port: int32(port),
		}, true
	}
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

type EndpointGroup struct {
	endpoints []Endpoint
}

func NewEndpointGroup(endpoints []Endpoint) *EndpointGroup {
	return &EndpointGroup{
		endpoints: endpoints,
	}
}

func (eg *EndpointGroup) addEndpoint(endpoint Endpoint) {
	for _, e := range eg.endpoints {
		if e.Host == endpoint.Host && e.Port == endpoint.Port {
			return
		}
	}
	eg.endpoints = append(eg.endpoints, endpoint)
}

func (eg *EndpointGroup) removeEndpoint(endpoint Endpoint) {
	for i, e := range eg.endpoints {
		if e.Host == endpoint.Host && e.Port == endpoint.Port {
			eg.endpoints = append(eg.endpoints[:i], eg.endpoints[i+1:]...)
			break
		}
	}
}
