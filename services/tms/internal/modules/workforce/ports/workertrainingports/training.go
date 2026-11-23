package workertrainingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TrainingAssigner interface {
	PlanAssignRequiredFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		wrk *worker.Worker,
	) ([]*workertraining.TrainingCourse, error)
	AssignPlanned(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		courses []*workertraining.TrainingCourse,
		userID pulid.ID,
	) []*workertraining.WorkerTrainingRecord
}
