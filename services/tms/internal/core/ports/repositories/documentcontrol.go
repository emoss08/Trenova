package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDocumentControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DocumentControlRepository interface {
	Get(
		ctx context.Context,
		req GetDocumentControlRequest,
	) (*tenant.DocumentControl, error)
	Create(
		ctx context.Context,
		entity *tenant.DocumentControl,
	) (*tenant.DocumentControl, error)
	Update(
		ctx context.Context,
		entity *tenant.DocumentControl,
	) (*tenant.DocumentControl, error)
	GetOrCreate(
		ctx context.Context,
		orgID, buID pulid.ID,
	) (*tenant.DocumentControl, error)
}
