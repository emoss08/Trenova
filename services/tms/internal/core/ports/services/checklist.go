package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
)

// ChecklistSpawner starts the default checklist an employment event calls
// for. The employment service depends on this port rather than the checklist
// service so the two can evolve without an import cycle.
type ChecklistSpawner interface {
	SpawnForEvent(
		ctx context.Context,
		event *worker.WorkerEmploymentEvent,
		wrk *worker.Worker,
		userID pulid.ID,
	) (*worker.WorkerChecklist, error)
}
