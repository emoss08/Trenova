// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

var WorkerFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"status":    true,
		"firstName": true,
		"lastName":  true,
		"type":      true,
	},
	SortableFields: map[string]bool{
		"status":    true,
		"firstName": true,
		"lastName":  true,
	},
	FieldMap: map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
		"status":    "status",
	},
	EnumMap: map[string]bool{
		"status": true,
		"type":   true,
	},
}

type WorkerFilterOptions struct {
	Status         string `query:"status"`
	IncludeProfile bool   `query:"includeProfile"`
	IncludePTO     bool   `query:"includePTO"`
}

func BuildWorkerListOptions(
	filter *ports.QueryOptions,
	additionalOpts *ListWorkerRequest,
) *ListWorkerRequest {
	return &ListWorkerRequest{
		Filter:              filter,
		WorkerFilterOptions: additionalOpts.WorkerFilterOptions,
	}
}

type GetWorkerByIDRequest struct {
	WorkerID      pulid.ID
	BuID          pulid.ID
	OrgID         pulid.ID
	UserID        pulid.ID
	FilterOptions WorkerFilterOptions `query:"filterOptions"`
}

type ListWorkerRequest struct {
	Filter              *ports.QueryOptions `json:"filter"              query:"filter"`
	WorkerFilterOptions `json:"workerFilterOptions" query:"workerFilterOptions"`
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
	GetWorkerPTO(
		ctx context.Context,
		ptoID, workerID, buID, orgID pulid.ID,
	) (*worker.WorkerPTO, error)
}
