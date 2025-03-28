package core

import (
	"github.com/paopaoyue/kscale/job-genrator/api"
)

type Job struct {
	Id          string
	Param       api.GenerateRequestParam
	Retry       int
	RequestTime int64
	StartTime   int64
	EndTime     int64
}

func NewJob(id string, param api.GenerateRequestParam) *Job {
	return &Job{
		Id:    id,
		Param: param,
	}
}
