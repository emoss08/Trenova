package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
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

type AuditRepository interface {
	InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*audit.Entry], error)
	ListByResourceID(
		ctx context.Context,
		opts ListByResourceIDRequest,
	) (*ports.ListResult[*audit.Entry], error)
	GetByID(ctx context.Context, opts GetAuditEntryByIDOptions) (*audit.Entry, error)
}
