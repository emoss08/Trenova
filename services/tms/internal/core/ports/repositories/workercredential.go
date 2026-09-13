package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCredentialTypesRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	Cursor   pagination.CursorInfo    `json:"cursor"`
	Status   string                   `json:"status"`
	Category string                   `json:"category"`
}

type WorkerCredentialTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type GetCredentialTypeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CredentialTypeCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type CountCredentialsByTypeRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TypeID     pulid.ID              `json:"typeId"`
	ActiveOnly bool                  `json:"activeOnly"`
}

type CountCredentialsByTypeIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TypeIDs    []pulid.ID            `json:"typeIds"`
	ActiveOnly bool                  `json:"activeOnly"`
}

type ListWorkerCredentialsRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	WorkerID        pulid.ID              `json:"workerId"`
	IncludeArchived bool                  `json:"includeArchived"`
	IncludeType     bool                  `json:"includeType"`
	IncludeDocument bool                  `json:"includeDocument"`
}

type GetWorkerCredentialByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeType     bool                  `json:"includeType"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
}

type CreateWorkerCredentialRequest struct {
	Entity *worker.WorkerCredential `json:"entity"`
	// SupersedeReason, when set, archives any other active credential of the
	// same type for the worker inside the same transaction so the new row can
	// take the single active slot.
	SupersedeReason string   `json:"supersedeReason"`
	SupersededByID  pulid.ID `json:"supersededById"`
}

// ListExpiringWorkerCredentialsRequest walks active credentials whose expiry
// falls inside [now - GraceDays, now + HorizonDays]. With a zero TenantInfo the
// walk crosses every tenant, which is what the nightly sweep needs; AfterID
// pages by credential id so a large fleet never loads in one go.
type ListExpiringWorkerCredentialsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	HorizonDays int                   `json:"horizonDays"`
	GraceDays   int                   `json:"graceDays"`
	AfterID     pulid.ID              `json:"afterId"`
	Limit       int                   `json:"limit"`
	// RequiredOnly restricts the walk to credential types flagged required.
	RequiredOnly bool `json:"requiredOnly"`
}

type PatchProfileCredentialFieldRequest struct {
	TenantInfo pagination.TenantInfo         `json:"tenantInfo"`
	WorkerID   pulid.ID                      `json:"workerId"`
	Field      worker.CredentialProfileField `json:"field"`
	ExpiresAt  *int64                        `json:"expiresAt"`
	Number     string                        `json:"number"`
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

type WorkerCredentialRepository interface {
	ListTypes(
		ctx context.Context,
		req *ListCredentialTypesRequest,
	) (*pagination.CursorListResult[*worker.WorkerCredentialType], error)
	ListActiveTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.WorkerCredentialType, error)
	TypeSelectOptions(
		ctx context.Context,
		req *WorkerCredentialTypeSelectOptionsRequest,
	) (*pagination.ListResult[*worker.WorkerCredentialType], error)
	GetTypeByID(
		ctx context.Context,
		req *GetCredentialTypeByIDRequest,
	) (*worker.WorkerCredentialType, error)
	TypeCodeExists(ctx context.Context, req *CredentialTypeCodeExistsRequest) (bool, error)
	CreateType(
		ctx context.Context,
		entity *worker.WorkerCredentialType,
	) (*worker.WorkerCredentialType, error)
	UpdateType(
		ctx context.Context,
		entity *worker.WorkerCredentialType,
	) (*worker.WorkerCredentialType, error)
	EnsureSystemTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		types []*worker.WorkerCredentialType,
	) (int, error)
	CountCredentialsByType(ctx context.Context, req *CountCredentialsByTypeRequest) (int, error)
	CountCredentialsByTypeIDs(
		ctx context.Context,
		req *CountCredentialsByTypeIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerCredentialsRequest,
	) ([]*worker.WorkerCredential, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerCredentialByIDRequest,
	) (*worker.WorkerCredential, error)
	Create(
		ctx context.Context,
		req *CreateWorkerCredentialRequest,
	) (*worker.WorkerCredential, error)
	Update(ctx context.Context, entity *worker.WorkerCredential) (*worker.WorkerCredential, error)
	ListExpiring(
		ctx context.Context,
		req *ListExpiringWorkerCredentialsRequest,
	) ([]*worker.WorkerCredential, error)
}
