package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// TrainingAssigner opens the required courses a worker is missing. The
// employment service depends on this port so a hire or a driver-type change
// enrols the worker without importing the training service.
type TrainingAssigner interface {
	AssignRequired(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		userID pulid.ID,
	) ([]*worker.WorkerTrainingRecord, error)
}
