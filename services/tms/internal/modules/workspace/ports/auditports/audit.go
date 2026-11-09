package auditports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type EntryAccess struct {
	AllResources    bool
	Resources       []permission.Resource
	IncludeSecurity bool
}

type GetAuditEntryByIDOptions struct {
	EntryID    pulid.ID              `json:"entryId"`
	TenantInfo pagination.TenantInfo `json:"-"`
	Access     EntryAccess           `json:"-"`
}

type ListAuditEntriesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	Cursor         pagination.CursorInfo    `json:"-"`
	Access         EntryAccess              `json:"-"`
	ResourceID     string                   `json:"resourceId"`
	ChangedField   string                   `json:"changedField"`
	IncludeDetail  bool                     `json:"-"`
	IncludeChanges bool                     `json:"-"`
	OnePerRequest  bool                     `json:"-"`
}

type CountByRequestRequest struct {
	TenantInfo     pagination.TenantInfo
	Access         EntryAccess
	CorrelationIDs []string
	From           int64
	To             int64
}

type DeleteExpiredAuditEntriesRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	Class  audit.RetentionClass
	Before int64
	Limit  int
}

type ExportPageRequest struct {
	TenantInfo pagination.TenantInfo
	Export     *audit.Export
	AfterTS    int64
	AfterID    pulid.ID
	Limit      int
}

type TenantKey struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type AuditRepository interface {
	Insert(ctx context.Context, entries []*audit.Entry) error
	ListConnection(
		ctx context.Context,
		req *ListAuditEntriesRequest,
	) (*pagination.CursorListResult[*audit.Entry], error)
	GetByID(ctx context.Context, req *GetAuditEntryByIDOptions) (*audit.Entry, error)
	ExportPage(ctx context.Context, req *ExportPageRequest) ([]*audit.Entry, error)
	DeleteExpired(ctx context.Context, req DeleteExpiredAuditEntriesRequest) (int64, error)
	PartitionHorizon(ctx context.Context) (int64, error)
	TenantsWithEntries(ctx context.Context, from, to int64) ([]TenantKey, error)
	CountByRequest(ctx context.Context, req *CountByRequestRequest) (map[string]int64, error)
}

type LegalHoldRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListLegalHoldsRequest,
	) (*pagination.CursorListResult[*audit.LegalHold], error)
	ListActive(ctx context.Context, tenant pagination.TenantInfo) ([]*audit.LegalHold, error)
	GetByID(ctx context.Context, req GetLegalHoldRequest) (*audit.LegalHold, error)
	Create(ctx context.Context, hold *audit.LegalHold) (*audit.LegalHold, error)
	Release(ctx context.Context, hold *audit.LegalHold) (*audit.LegalHold, error)
}

type ListLegalHoldsRequest struct {
	Filter     *pagination.QueryOptions
	Cursor     pagination.CursorInfo
	ActiveOnly bool
}

type GetLegalHoldRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}
