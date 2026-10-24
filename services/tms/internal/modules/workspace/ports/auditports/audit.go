package auditports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
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

type DeleteAuditEntriesRequest struct {
	OrgID          pulid.ID
	BuID           pulid.ID
	Before         int64
	CriticalBefore int64
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
	DeleteAuditEntries(ctx context.Context, req DeleteAuditEntriesRequest) (int64, error)
}
