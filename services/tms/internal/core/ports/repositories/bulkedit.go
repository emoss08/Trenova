package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBulkEditRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListBulkEditsRequest struct {
	TenantInfo pagination.TenantInfo
	Resource   string
	Limit      int
}

type BulkEditRepository interface {
	Create(ctx context.Context, entity *bulkedit.BulkEdit) (*bulkedit.BulkEdit, error)
	GetByID(ctx context.Context, req *GetBulkEditRequest) (*bulkedit.BulkEdit, error)
	Update(ctx context.Context, entity *bulkedit.BulkEdit) (*bulkedit.BulkEdit, error)
	ListForUser(ctx context.Context, req *ListBulkEditsRequest) ([]*bulkedit.BulkEdit, error)
}
