package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablelayout"
	"github.com/emoss08/trenova/pkg/pagination"
)

type TableLayoutRequest struct {
	TenantInfo pagination.TenantInfo
	Resource   string
}

type TableLayoutRepository interface {
	Get(ctx context.Context, req *TableLayoutRequest) (*tablelayout.TableLayout, bool, error)
	CountForUser(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
	Upsert(ctx context.Context, entity *tablelayout.TableLayout) (*tablelayout.TableLayout, error)
	Delete(ctx context.Context, req *TableLayoutRequest) error
}
