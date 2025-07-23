// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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

type GetAuditByResourceRequest struct {
	Resource       permission.Resource
	ResourceID     string
	Action         permission.Action
	OrganizationID pulid.ID
	Limit          int
}

type GetRecentEntriesRequest struct {
	SinceTimestamp int64
	Action         permission.Action
	Limit          int
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
	GetByResourceAndAction(
		ctx context.Context,
		req *GetAuditByResourceRequest,
	) ([]*audit.Entry, error)
	GetRecentEntries(ctx context.Context, req *GetRecentEntriesRequest) ([]*audit.Entry, error)
}
