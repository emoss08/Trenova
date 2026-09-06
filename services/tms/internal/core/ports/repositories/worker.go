package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type WorkerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	IncludeProfile     bool                           `json:"includeProfile"`
	// OwnerOperatorsOnly narrows results to owner-operators: workers of type
	// Contractor, or workers whose currently effective pay profile carries the
	// OwnerOperator classification (covers contractors not yet assigned a
	// profile and mislabeled workers on owner-op pay).
	OwnerOperatorsOnly bool `json:"ownerOperatorsOnly"`
}

type GetWorkerByIDRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	IncludeProfile bool                  `json:"includeProfile"`
	IncludeState   bool                  `json:"includeState"`
}

type WorkerSyncReadinessCounts struct {
	TotalWorkers        int `json:"totalWorkers"        bun:"total_workers"`
	ActiveWorkers       int `json:"activeWorkers"       bun:"active_workers"`
	SyncedActiveWorkers int `json:"syncedActiveWorkers" bun:"synced_active_workers"`
}

const (
	WorkerSyncDriftTypeMissingMapping      = "missing_mapping"
	WorkerSyncDriftTypeMissingRemoteDriver = "missing_remote_driver"
	WorkerSyncDriftTypeMappingMismatch     = "mapping_mismatch"
	WorkerSyncDriftTypeRemoteDeactivated   = "remote_deactivated"
)

type WorkerSyncDriftRecord struct {
	WorkerID        string `json:"workerId"`
	WorkerName      string `json:"workerName"`
	DriftType       string `json:"driftType"`
	Message         string `json:"message"`
	LocalExternalID string `json:"localExternalId,omitempty"`
	RemoteDriverID  string `json:"remoteDriverId,omitempty"`
	DetectedAt      int64  `json:"detectedAt"`
}

type WorkerRepository interface {
	List(
		ctx context.Context,
		req *ListWorkersRequest,
	) (*pagination.CursorListResult[*worker.Worker], error)
	SelectOptions(
		ctx context.Context,
		req *WorkerSelectOptionsRequest,
	) (*pagination.ListResult[*worker.Worker], error)
	GetByID(
		ctx context.Context,
		req GetWorkerByIDRequest,
	) (*worker.Worker, error)
	Create(
		ctx context.Context,
		entity *worker.Worker,
	) (*worker.Worker, error)
	Update(
		ctx context.Context,
		entity *worker.Worker,
	) (*worker.Worker, error)
	GetWorkerSyncReadinessCounts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*WorkerSyncReadinessCounts, error)
	ReplaceWorkerSyncDrifts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		drifts []WorkerSyncDriftRecord,
	) error
	ListWorkerSyncDrifts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]WorkerSyncDriftRecord, error)
	PatchProfileCredentialField(
		ctx context.Context,
		req *PatchProfileCredentialFieldRequest,
	) error
	UpdateProfileComplianceStatus(
		ctx context.Context,
		req *UpdateProfileComplianceStatusRequest,
	) error
	UpdateProfileQualification(
		ctx context.Context,
		req *UpdateProfileQualificationRequest,
	) error
	UpdateProfileTrainingRollup(
		ctx context.Context,
		req *UpdateProfileTrainingRollupRequest,
	) error
	UpdateProfileSafetyRollup(
		ctx context.Context,
		req *UpdateProfileSafetyRollupRequest,
	) error
	CountRosterAttention(
		ctx context.Context,
		req *CountRosterAttentionRequest,
	) (*RosterAttention, error)
}

// CountRosterAttentionRequest asks how many active workers need attention.
// ExpiryHorizon is the cutoff for "expiring soon"; the caller owns the clock
// so the answer is stable within a request.
type CountRosterAttentionRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	ExpiryHorizon int64                 `json:"expiryHorizon"`
}

// RosterAttention counts the roster in the terms HR works in. Every number is
// read from the roll-up columns on worker_profiles in one pass, which is the
// reason those columns exist.
type RosterAttention struct {
	ActiveWorkers   int `bun:"active_workers"`
	NonCompliant    int `bun:"non_compliant"`
	TrainingOverdue int `bun:"training_overdue"`
	AtRisk          int `bun:"at_risk"`
	ExpiringSoon    int `bun:"expiring_soon"`
}
