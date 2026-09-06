package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
)

// EmploymentEventRecorder lets the worker service open a timeline when a
// worker is created without importing the employment service.
type EmploymentEventRecorder interface {
	RecordHired(ctx context.Context, wrk *worker.Worker, userID pulid.ID) error
}
