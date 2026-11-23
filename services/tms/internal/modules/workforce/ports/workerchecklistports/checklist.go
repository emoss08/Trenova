package workerchecklistports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/shared/pulid"
)

type EventPlan struct {
	Worker   *worker.Worker
	Template *workerchecklist.WorkerChecklistTemplate
	Close    []*workerchecklist.WorkerChecklist
}

type ChecklistSpawner interface {
	PlanForEvent(
		ctx context.Context,
		kind employment.EmploymentEventKind,
		wrk *worker.Worker,
	) (*EventPlan, error)
	SpawnForEvent(
		ctx context.Context,
		event *employment.WorkerEmploymentEvent,
		plan *EventPlan,
		userID pulid.ID,
	) (*workerchecklist.WorkerChecklist, error)
	CloseForEvent(
		ctx context.Context,
		event *employment.WorkerEmploymentEvent,
		plan *EventPlan,
		userID pulid.ID,
	) (int, error)
}
