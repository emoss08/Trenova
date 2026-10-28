package workerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
	// WithoutCredential keeps the workers with no valid credential of the
	// type this code names: none on file, archived, or expired.
	WithoutCredential string `json:"withoutCredential"`
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
	ID                  pulid.ID              `json:"id"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IncludeProfile      bool                  `json:"includeProfile"`
	IncludeState        bool                  `json:"includeState"`
	IncludeFleetCode    bool                  `json:"includeFleetCode"`
	IncludeManager      bool                  `json:"includeManager"`
	IncludeLicenseState bool                  `json:"includeLicenseState"`
}

type WorkerSyncReadinessCounts struct {
	TotalWorkers        int `json:"totalWorkers"`
	ActiveWorkers       int `json:"activeWorkers"`
	SyncedActiveWorkers int `json:"syncedActiveWorkers"`
}

const WorkerSyncDriftTypeMissingMapping = "missing_mapping"

const WorkerSyncDriftTypeMissingRemoteDriver = "missing_remote_driver"

const WorkerSyncDriftTypeMappingMismatch = "mapping_mismatch"

const WorkerSyncDriftTypeRemoteDeactivated = "remote_deactivated"

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
	ActiveWorkers   int
	NonCompliant    int
	TrainingOverdue int
	AtRisk          int
	ExpiringSoon    int
}

type PatchProfileCredentialFieldRequest struct {
	TenantInfo pagination.TenantInfo                   `json:"tenantInfo"`
	WorkerID   pulid.ID                                `json:"workerId"`
	Field      workercredential.CredentialProfileField `json:"field"`
	ExpiresAt  *int64                                  `json:"expiresAt"`
	Number     string                                  `json:"number"`
}

type UpdateProfileQualificationRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Qualified  bool                  `json:"qualified"`
}

type UpdateProfileComplianceStatusRequest struct {
	TenantInfo pagination.TenantInfo   `json:"tenantInfo"`
	WorkerID   pulid.ID                `json:"workerId"`
	Status     worker.ComplianceStatus `json:"status"`
	// NextExpiry is the earliest expiry among the worker's required
	// credentials, denormalised so the roster can sort by what lapses next.
	// Nil clears it, meaning nothing on file expires.
	NextExpiry *int64 `json:"nextExpiry"`
}

// UpdateProfileTrainingRollupRequest carries the training half of the roster
// cache. The training service owns these columns because it is the only place
// that can compute them.
type UpdateProfileTrainingRollupRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Health     worker.TrainingHealth `json:"health"`
	NextDue    *int64                `json:"nextDue"`
}

// UpdateProfileSafetyRollupRequest carries the safety half, owned by the
// safety service for the same reason.
type UpdateProfileSafetyRollupRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Rating     worker.SafetyRating   `json:"rating"`
	Score      int16                 `json:"score"`
}

// WorkerTenantRef names a worker and the tenant they belong to, for sweeps
// that work across tenants.
type WorkerTenantRef struct {
	WorkerID       pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}
