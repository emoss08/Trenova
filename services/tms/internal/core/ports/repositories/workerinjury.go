package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerInjuriesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerID scopes the list to one worker; leave it nil for the log, which
	// is read for the whole establishment.
	WorkerID pulid.ID `json:"workerId"`
	// CaseYear scopes to one calendar year, which is how the 300 log is read.
	// Zero means every year.
	CaseYear int16 `json:"caseYear"`
	// RecordableOnly narrows to the cases that are the 300 log; the rest stay
	// on file because the decision not to record is itself worth a record.
	RecordableOnly  bool `json:"recordableOnly"`
	OpenOnly        bool `json:"openOnly"`
	IncludeWorker   bool `json:"includeWorker"`
	IncludeDocument bool `json:"includeDocument"`
	IncludeEvent    bool `json:"includeEvent"`
	Limit           int  `json:"limit"`
}

type GetWorkerInjuryByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
	IncludeEvent    bool                  `json:"includeEvent"`
}

// NextCaseNumberRequest asks for the next line on the log. Case numbers restart
// each calendar year, which is how the 300 log reads.
type NextCaseNumberRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CaseYear   int16                 `json:"caseYear"`
}

type GetOSHASummaryRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Year       int16                 `json:"year"`
}

type ListOSHASummariesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
}

type WorkerInjuryRepository interface {
	ListInjuries(
		ctx context.Context,
		req *ListWorkerInjuriesRequest,
	) ([]*worker.WorkerInjury, error)
	GetInjuryByID(
		ctx context.Context,
		req *GetWorkerInjuryByIDRequest,
	) (*worker.WorkerInjury, error)
	CreateInjury(ctx context.Context, entity *worker.WorkerInjury) (*worker.WorkerInjury, error)
	UpdateInjury(ctx context.Context, entity *worker.WorkerInjury) (*worker.WorkerInjury, error)
	DeleteInjury(ctx context.Context, req *GetWorkerInjuryByIDRequest) error
	NextCaseNumber(ctx context.Context, req *NextCaseNumberRequest) (int32, error)

	// GetSummary returns the 300A for a year, or nil when none has been
	// started. A year with no summary row is the ordinary case until somebody
	// opens one.
	GetSummary(
		ctx context.Context,
		req *GetOSHASummaryRequest,
	) (*worker.OSHAAnnualSummary, error)
	ListSummaries(
		ctx context.Context,
		req *ListOSHASummariesRequest,
	) ([]*worker.OSHAAnnualSummary, error)
	CreateSummary(
		ctx context.Context,
		entity *worker.OSHAAnnualSummary,
	) (*worker.OSHAAnnualSummary, error)
	UpdateSummary(
		ctx context.Context,
		entity *worker.OSHAAnnualSummary,
	) (*worker.OSHAAnnualSummary, error)
}
