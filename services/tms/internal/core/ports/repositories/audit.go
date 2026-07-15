package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAuditEntryByIDOptions struct {
	EntryID    pulid.ID              `json:"entryId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type ListAuditEntriesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListAuditEntriesConnectionRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	Cursor     pagination.CursorInfo    `json:"-"`
	ResourceID pulid.ID                 `json:"resourceId"`
}

type ListByResourceIDRequest struct {
	ResourceID pulid.ID                 `json:"resourceId"`
	Filter     *pagination.QueryOptions `json:"filter"`
}

type GetAuditByResourceRequest struct {
	Resource       permission.Resource
	ResourceID     string
	Operation      uint32
	OrganizationID pulid.ID
	Limit          int
}

type GetRecentEntriesRequest struct {
	SinceTimestamp int64
	Operation      uint32
	Limit          int
}

type DeleteAuditEntriesRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	Before int64
}

type AuditRepository interface {
	InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error
	List(
		ctx context.Context,
		req *ListAuditEntriesRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	ListConnection(
		ctx context.Context,
		req *ListAuditEntriesConnectionRequest,
	) (*pagination.CursorListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		req *ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, req GetAuditEntryByIDOptions) (*audit.Entry, error)
	GetByResourceAndOperation(
		ctx context.Context,
		req *GetAuditByResourceRequest,
	) ([]*audit.Entry, error)
	GetRecentEntries(ctx context.Context, req *GetRecentEntriesRequest) ([]*audit.Entry, error)
	DeleteAuditEntries(ctx context.Context, req DeleteAuditEntriesRequest) (int64, error)
}
