package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
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
	) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error)
	Update(
		ctx context.Context,
		control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
	) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error)
}
