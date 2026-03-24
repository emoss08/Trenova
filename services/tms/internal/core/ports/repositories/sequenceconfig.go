package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetSequenceConfigRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type SequenceConfigRepository interface {
	GetByTenant(
		ctx context.Context,
		req GetSequenceConfigRequest,
	) (*tenant.SequenceConfigDocument, error)
	UpdateByTenant(
		ctx context.Context,
		doc *tenant.SequenceConfigDocument,
	) (*tenant.SequenceConfigDocument, error)
}
