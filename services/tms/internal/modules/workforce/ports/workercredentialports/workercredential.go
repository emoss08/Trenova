package workercredentialports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
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

type GetCredentialTypesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TypeIDs    []pulid.ID            `json:"typeIds"`
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
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerIDs are the workers whose credentials are read, in one query.
	WorkerIDs       []pulid.ID `json:"workerIds"`
	IncludeArchived bool       `json:"includeArchived"`
	IncludeType     bool       `json:"includeType"`
	IncludeDocument bool       `json:"includeDocument"`
}

type GetWorkerCredentialByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeType     bool                  `json:"includeType"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
}

type CreateWorkerCredentialRequest struct {
	Entity *workercredential.WorkerCredential `json:"entity"`
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
	// AsOf, when set, is the start of the caller's day, and the window is then
	// whole days: from GraceDays before it through the end of the HorizonDays-th
	// day after it. Unset, the window runs from this instant, which is what an
	// interactive page wants and what a question about days does not.
	AsOf    int64    `json:"asOf"`
	AfterID pulid.ID `json:"afterId"`
	Limit   int      `json:"limit"`
	// RequiredOnly restricts the walk to credential types flagged required.
	RequiredOnly bool `json:"requiredOnly"`
	// CredentialTypeCodes narrows the walk to those credential types, by code.
	// It belongs in the query rather than in the caller: taking a page of every
	// expiring credential and keeping the medical cards among them reports two
	// when there are forty, and nothing in the result says so.
	CredentialTypeCodes []string `json:"credentialTypeCodes"`
	// WorkerIDs narrows the walk to these workers' credentials.
	WorkerIDs    []pulid.ID `json:"workerIds"`
	SoonestFirst bool       `json:"soonestFirst"`
	Offset       int        `json:"offset"`
	// AnyExpiry drops the window: every active credential, whatever its
	// expiry or none, which is who holds a type. A road test that never
	// expires is outside every window.
	AnyExpiry bool `json:"anyExpiry"`
}

type WorkerCredentialRepository interface {
	ListTypes(
		ctx context.Context,
		req *ListCredentialTypesRequest,
	) (*pagination.CursorListResult[*workercredential.WorkerCredentialType], error)
	ListActiveTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*workercredential.WorkerCredentialType, error)
	TypeSelectOptions(
		ctx context.Context,
		req *WorkerCredentialTypeSelectOptionsRequest,
	) (*pagination.ListResult[*workercredential.WorkerCredentialType], error)
	GetTypeByID(
		ctx context.Context,
		req *GetCredentialTypeByIDRequest,
	) (*workercredential.WorkerCredentialType, error)
	GetTypesByIDs(
		ctx context.Context,
		req *GetCredentialTypesByIDsRequest,
	) ([]*workercredential.WorkerCredentialType, error)
	TypeCodeExists(ctx context.Context, req *CredentialTypeCodeExistsRequest) (bool, error)
	CreateType(
		ctx context.Context,
		entity *workercredential.WorkerCredentialType,
	) (*workercredential.WorkerCredentialType, error)
	UpdateType(
		ctx context.Context,
		entity *workercredential.WorkerCredentialType,
	) (*workercredential.WorkerCredentialType, error)
	EnsureSystemTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		types []*workercredential.WorkerCredentialType,
	) (int, error)
	CountCredentialsByType(ctx context.Context, req *CountCredentialsByTypeRequest) (int, error)
	CountCredentialsByTypeIDs(
		ctx context.Context,
		req *CountCredentialsByTypeIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerCredentialsRequest,
	) ([]*workercredential.WorkerCredential, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerCredentialByIDRequest,
	) (*workercredential.WorkerCredential, error)
	Create(
		ctx context.Context,
		req *CreateWorkerCredentialRequest,
	) (*workercredential.WorkerCredential, error)
	Update(
		ctx context.Context,
		entity *workercredential.WorkerCredential,
	) (*workercredential.WorkerCredential, error)
	ListExpiring(
		ctx context.Context,
		req *ListExpiringWorkerCredentialsRequest,
	) ([]*workercredential.WorkerCredential, error)
}
