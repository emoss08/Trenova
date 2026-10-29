package workerchecklistports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/shared/pulid"
)

// ChecklistSpawner starts the default checklist an employment event calls
// for. The employment service depends on this port rather than the checklist
// service so the two can evolve without an import cycle.
type ChecklistSpawner interface {
	SpawnForEvent(
		ctx context.Context,
		event *employment.WorkerEmploymentEvent,
		wrk *worker.Worker,
		userID pulid.ID,
	) (*workerchecklist.WorkerChecklist, error)
	// CloseForEvent cancels the open checklists the event makes moot and
	// reports how many it closed.
	CloseForEvent(
		ctx context.Context,
		event *employment.WorkerEmploymentEvent,
		wrk *worker.Worker,
		userID pulid.ID,
	) (int, error)
}
