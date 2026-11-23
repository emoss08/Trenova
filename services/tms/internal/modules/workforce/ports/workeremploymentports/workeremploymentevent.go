package workeremploymentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerEmploymentEventsRequest struct {
	TenantInfo      pagination.TenantInfo            `json:"tenantInfo"`
	WorkerID        pulid.ID                         `json:"workerId"`
	Kinds           []employment.EmploymentEventKind `json:"kinds"`
	IncludeDocument bool                             `json:"includeDocument"`
	IncludeActors   bool                             `json:"includeActors"`
	Limit           int                              `json:"limit"`
	Cursor          pagination.CursorInfo            `json:"cursor"`
}

type SummarizeWorkerEmploymentEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type EmploymentEventSummary struct {
	EventCount  int    `json:"eventCount"`
	LastHiredAt *int64 `json:"lastHiredAt"`
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
	ListPage(
		ctx context.Context,
		req *ListWorkerEmploymentEventsRequest,
	) (*pagination.CursorListResult[*employment.WorkerEmploymentEvent], error)
	Summarize(
		ctx context.Context,
		req *SummarizeWorkerEmploymentEventsRequest,
	) (*EmploymentEventSummary, error)
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
