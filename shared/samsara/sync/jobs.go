package samsync

import "context"

type JobFunc func(ctx context.Context, tenantID string) error

type FunctionalJob struct {
	name string
	run  JobFunc
}

func NewFunctionalJob(name string, run JobFunc) *FunctionalJob {
	return &FunctionalJob{name: name, run: run}
}

func (j *FunctionalJob) Name() string {
	return j.name
}

func (j *FunctionalJob) Run(ctx context.Context, tenantID string) error {
	if j.run == nil {
		return nil
	}
	return j.run(ctx, tenantID)
}
