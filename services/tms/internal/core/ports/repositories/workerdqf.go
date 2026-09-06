package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEmploymentVerificationsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerID scopes the list to one driver; leave it nil for the whole
	// organisation, which is what the outstanding-investigations queue reads.
	WorkerID        pulid.ID `json:"workerId"`
	OutstandingOnly bool     `json:"outstandingOnly"`
	IncludeWorker   bool     `json:"includeWorker"`
	IncludeDocument bool     `json:"includeDocument"`
	Limit           int      `json:"limit"`
}

type ListEmploymentVerificationsByWorkerIDsRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	WorkerIDs       []pulid.ID            `json:"workerIds"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
}

type GetEmploymentVerificationByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeDocument bool                  `json:"includeDocument"`
}

// ListDQFRetentionCandidatesRequest finds terminated drivers whose file has
// passed its retention window. The window is applied in SQL so the caller does
// not have to read every terminated worker to find the handful that qualify.
type ListDQFRetentionCandidatesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// RetentionSeconds is the organisation's hold period, already converted
	// from days.
	RetentionSeconds int64 `json:"retentionSeconds"`
	AsOf             int64 `json:"asOf"`
	Limit            int   `json:"limit"`
}

// DQFRetentionCandidate is one file eligible for purge.
type DQFRetentionCandidate struct {
	WorkerID        pulid.ID `bun:"worker_id"`
	OrganizationID  pulid.ID `bun:"organization_id"`
	BusinessUnitID  pulid.ID `bun:"business_unit_id"`
	FirstName       string   `bun:"first_name"`
	LastName        string   `bun:"last_name"`
	TerminationDate int64    `bun:"termination_date"`
}

type WorkerDQFRepository interface {
	ListVerifications(
		ctx context.Context,
		req *ListEmploymentVerificationsRequest,
	) ([]*worker.WorkerEmploymentVerification, error)
	ListVerificationsByWorkerIDs(
		ctx context.Context,
		req *ListEmploymentVerificationsByWorkerIDsRequest,
	) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error)
	GetVerificationByID(
		ctx context.Context,
		req *GetEmploymentVerificationByIDRequest,
	) (*worker.WorkerEmploymentVerification, error)
	CreateVerification(
		ctx context.Context,
		entity *worker.WorkerEmploymentVerification,
	) (*worker.WorkerEmploymentVerification, error)
	UpdateVerification(
		ctx context.Context,
		entity *worker.WorkerEmploymentVerification,
	) (*worker.WorkerEmploymentVerification, error)
	DeleteVerification(ctx context.Context, req *GetEmploymentVerificationByIDRequest) error

	ListRetentionCandidates(
		ctx context.Context,
		req *ListDQFRetentionCandidatesRequest,
	) ([]DQFRetentionCandidate, error)
}
