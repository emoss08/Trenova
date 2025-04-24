package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type WorkerFilterOptions struct {
	Status         string `query:"status"`
	IncludeProfile bool   `query:"includeProfile"`
	IncludePTO     bool   `query:"includePTO"`
}

type GetWorkerByIDRequest struct {
	WorkerID      pulid.ID
	BuID          pulid.ID
	OrgID         pulid.ID
	UserID        pulid.ID
	FilterOptions WorkerFilterOptions `query:"filterOptions"`
}

type ListWorkerRequest struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions WorkerFilterOptions `query:"filterOptions"`
}

type UpdateWorkerOptions struct {
	OrgID pulid.ID
	BuID  pulid.ID
}

type WorkerRepository interface {
	List(ctx context.Context, req *ListWorkerRequest) (*ports.ListResult[*worker.Worker], error)
	GetByID(ctx context.Context, req *GetWorkerByIDRequest) (*worker.Worker, error)
	Create(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	Update(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	GetWorkerPTO(ctx context.Context, ptoID, workerID, buID, orgID pulid.ID) (*worker.WorkerPTO, error)
}
