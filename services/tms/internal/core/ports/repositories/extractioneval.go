package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetExtractionEvalCaseRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListExtractionEvalCaseConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

type ListActiveExtractionEvalCasesRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
	Limit      int
}

type CountExtractionEvalCasesRequest struct {
	TenantInfo pagination.TenantInfo
	Task       aicorrection.Task
}

type ExtractionEvalCaseCounts struct {
	Candidate int
	Active    int
	Retired   int
}

type ExtractionEvalCaseRepository interface {
	Create(
		ctx context.Context,
		entity *extractioneval.ExtractionCase,
	) (*extractioneval.ExtractionCase, error)
	GetByID(
		ctx context.Context,
		req GetExtractionEvalCaseRequest,
	) (*extractioneval.ExtractionCase, error)
	GetBySourceCorrection(
		ctx context.Context,
		tenant pagination.TenantInfo,
		correctionID pulid.ID,
	) (*extractioneval.ExtractionCase, error)
	Update(
		ctx context.Context,
		entity *extractioneval.ExtractionCase,
	) (*extractioneval.ExtractionCase, error)
	Delete(ctx context.Context, req GetExtractionEvalCaseRequest) error
	ListConnection(
		ctx context.Context,
		req *ListExtractionEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error)
	ListActive(
		ctx context.Context,
		req ListActiveExtractionEvalCasesRequest,
	) ([]*extractioneval.ExtractionCase, error)
	Counts(
		ctx context.Context,
		req CountExtractionEvalCasesRequest,
	) (*ExtractionEvalCaseCounts, error)
}

type GetExtractionEvalRunRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListExtractionEvalRunConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

type PurgeExtractionEvalRunsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type ExtractionEvalRunRepository interface {
	Create(
		ctx context.Context,
		run *extractioneval.ExtractionRun,
		results []*extractioneval.ExtractionResult,
	) (*extractioneval.ExtractionRun, error)
	GetByID(
		ctx context.Context,
		req GetExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	Update(
		ctx context.Context,
		entity *extractioneval.ExtractionRun,
	) (*extractioneval.ExtractionRun, error)
	ListConnection(
		ctx context.Context,
		req *ListExtractionEvalRunConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error)
	ListRecentFinished(
		ctx context.Context,
		tenant pagination.TenantInfo,
		task aicorrection.Task,
		limit int,
	) ([]*extractioneval.ExtractionRun, error)
	PurgeBefore(ctx context.Context, req PurgeExtractionEvalRunsRequest) (int64, error)
}

type GetExtractionEvalResultRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListPendingExtractionEvalResultsRequest struct {
	TenantInfo   pagination.TenantInfo
	RunID        pulid.ID
	AfterOrdinal int
	Limit        int
}

type ListExtractionEvalResultConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	RunID   pulid.ID
	Columns []string
}

type ExtractionEvalResultRepository interface {
	GetByID(
		ctx context.Context,
		req GetExtractionEvalResultRequest,
	) (*extractioneval.ExtractionResult, error)
	Save(
		ctx context.Context,
		entity *extractioneval.ExtractionResult,
	) (*extractioneval.ExtractionResult, error)
	ListPending(
		ctx context.Context,
		req ListPendingExtractionEvalResultsRequest,
	) ([]*extractioneval.ExtractionResult, error)
	ListByRun(
		ctx context.Context,
		tenant pagination.TenantInfo,
		runID pulid.ID,
	) ([]*extractioneval.ExtractionResult, error)
	SkipPending(ctx context.Context, tenant pagination.TenantInfo, runID pulid.ID) (int64, error)
	ListConnection(
		ctx context.Context,
		req *ListExtractionEvalResultConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error)
}
