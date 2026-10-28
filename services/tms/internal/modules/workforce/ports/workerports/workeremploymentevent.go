package workerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerEmploymentEventsRequest struct {
	TenantInfo pagination.TenantInfo            `json:"tenantInfo"`
	WorkerID   pulid.ID                         `json:"workerId"`
	Kinds      []employment.EmploymentEventKind `json:"kinds"`
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
	) ([]*employment.WorkerEmploymentEvent, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerEmploymentEventByIDRequest,
	) (*employment.WorkerEmploymentEvent, error)
	Create(
		ctx context.Context,
		entity *employment.WorkerEmploymentEvent,
	) (*employment.WorkerEmploymentEvent, error)
	Update(
		ctx context.Context,
		entity *employment.WorkerEmploymentEvent,
	) (*employment.WorkerEmploymentEvent, error)
}
