package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListLeaveCasesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerID scopes the list to one worker; leave it nil for the whole
	// organisation, which is what the leave queue reads.
	WorkerID pulid.ID `json:"workerId"`
	OpenOnly bool     `json:"openOnly"`
	// CertificationOutstandingOnly narrows to the cases still waiting on
	// paperwork the employee owes.
	CertificationOutstandingOnly bool `json:"certificationOutstandingOnly"`
	IncludeWorker                bool `json:"includeWorker"`
	IncludeDocument              bool `json:"includeDocument"`
	IncludeEntries               bool `json:"includeEntries"`
	Limit                        int  `json:"limit"`
}

type GetLeaveCaseByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeDocument bool                  `json:"includeDocument"`
	IncludeEntries  bool                  `json:"includeEntries"`
}

type ListLeaveEntriesRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	WorkerID    pulid.ID              `json:"workerId"`
	LeaveCaseID pulid.ID              `json:"leaveCaseId"`
	// Since limits the list to days on or after this instant; the entitlement
	// only ever needs the measurement window, not a worker's whole history.
	Since int64 `json:"since"`
	Limit int   `json:"limit"`
}

type ListLeaveEntriesByCaseIDsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	LeaveCaseIDs []pulid.ID            `json:"leaveCaseIds"`
}

type GetLeaveEntryByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type WorkerLeaveRepository interface {
	// GetControl returns the organisation's leave settings, creating the row
	// with the statutory defaults when none exists. It never returns nil: an
	// absent control would read as an entitlement of zero.
	GetControl(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*worker.LeaveControl, error)
	UpdateControl(
		ctx context.Context,
		entity *worker.LeaveControl,
	) (*worker.LeaveControl, error)

	ListCases(ctx context.Context, req *ListLeaveCasesRequest) ([]*worker.WorkerLeaveCase, error)
	// CountCases answers how many cases match without carrying them back. The
	// home tile only ever needs the number.
	CountCases(ctx context.Context, req *ListLeaveCasesRequest) (int, error)
	GetCaseByID(
		ctx context.Context,
		req *GetLeaveCaseByIDRequest,
	) (*worker.WorkerLeaveCase, error)
	CreateCase(
		ctx context.Context,
		entity *worker.WorkerLeaveCase,
	) (*worker.WorkerLeaveCase, error)
	UpdateCase(
		ctx context.Context,
		entity *worker.WorkerLeaveCase,
	) (*worker.WorkerLeaveCase, error)

	ListEntries(
		ctx context.Context,
		req *ListLeaveEntriesRequest,
	) ([]*worker.WorkerLeaveEntry, error)
	ListEntriesByCaseIDs(
		ctx context.Context,
		req *ListLeaveEntriesByCaseIDsRequest,
	) (map[pulid.ID][]*worker.WorkerLeaveEntry, error)
	GetEntryByID(
		ctx context.Context,
		req *GetLeaveEntryByIDRequest,
	) (*worker.WorkerLeaveEntry, error)
	CreateEntry(
		ctx context.Context,
		entity *worker.WorkerLeaveEntry,
	) (*worker.WorkerLeaveEntry, error)
	UpdateEntry(
		ctx context.Context,
		entity *worker.WorkerLeaveEntry,
	) (*worker.WorkerLeaveEntry, error)
	DeleteEntry(ctx context.Context, req *GetLeaveEntryByIDRequest) error
}
