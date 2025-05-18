package core

import (
	"github.com/paopaoyue/kscale/job-genrator/api"
	"time"
)

type Job struct {
	Id          string
	Success     bool
	Param       api.GenerateRequestParam
	Retry       int
	RequestTime time.Time
	EndTime     time.Time
	Duration    time.Duration
}

func NewJob(id string, param api.GenerateRequestParam) *Job {
	return &Job{
		Id:    id,
		Param: param,
	}
}
