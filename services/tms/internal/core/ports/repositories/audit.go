package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetAuditEntryByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ListByResourceIDRequest struct {
	ResourceID pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
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

type AuditRepository interface {
	InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error
	List(
		ctx context.Context,
		opts *pagination.QueryOptions,
	) (*pagination.ListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		opts ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, opts GetAuditEntryByIDOptions) (*audit.Entry, error)
	GetByResourceAndOperation(
		ctx context.Context,
		req *GetAuditByResourceRequest,
	) ([]*audit.Entry, error)
	GetRecentEntries(ctx context.Context, req *GetRecentEntriesRequest) ([]*audit.Entry, error)
	DeleteAuditEntries(ctx context.Context, timestamp int64) (int64, error)
}
