package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBillingControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BillingControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		orgID pulid.ID,
	) (*tenant.BillingControl, error)
	Update(
		ctx context.Context,
		bc *tenant.BillingControl,
	) (*tenant.BillingControl, error)
}
