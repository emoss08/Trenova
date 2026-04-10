package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetInvoiceAdjustmentControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type InvoiceAdjustmentControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		orgID pulid.ID,
	) (*tenant.InvoiceAdjustmentControl, error)
	Update(
		ctx context.Context,
		control *tenant.InvoiceAdjustmentControl,
	) (*tenant.InvoiceAdjustmentControl, error)
}
