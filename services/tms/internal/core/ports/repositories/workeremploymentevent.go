package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerEmploymentEventsRequest struct {
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	WorkerID   pulid.ID                     `json:"workerId"`
	Kinds      []worker.EmploymentEventKind `json:"kinds"`
	// Ascending orders oldest first, which is what the state machine needs;
	// the timeline reads newest first.
	Ascending       bool `json:"ascending"`
	IncludeDocument bool `json:"includeDocument"`
	IncludeActors   bool `json:"includeActors"`
	Limit           int  `json:"limit"`
}

type GetWorkerEmploymentEventByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeDocument bool                  `json:"includeDocument"`
	IncludeActors   bool                  `json:"includeActors"`
}

type WorkerEmploymentEventRepository interface {
	List(
		ctx context.Context,
		req *ListWorkerEmploymentEventsRequest,
	) ([]*worker.WorkerEmploymentEvent, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerEmploymentEventByIDRequest,
	) (*worker.WorkerEmploymentEvent, error)
	Create(
		ctx context.Context,
		entity *worker.WorkerEmploymentEvent,
	) (*worker.WorkerEmploymentEvent, error)
	Update(
		ctx context.Context,
		entity *worker.WorkerEmploymentEvent,
	) (*worker.WorkerEmploymentEvent, error)
}
