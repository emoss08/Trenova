package samsync

import "context"

type Runner struct {
	jobs []SyncJob
}

func NewRunner(jobs ...SyncJob) *Runner {
	return &Runner{jobs: jobs}
}

func (r *Runner) Run(ctx context.Context, tenantID string) error {
	for _, job := range r.jobs {
		if err := job.Run(ctx, tenantID); err != nil {
			return err
		}
	}
	return nil
}
