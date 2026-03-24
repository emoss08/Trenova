package repositories

import (
	"context"

	accountingcontrol "github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type AccountingControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*accountingcontrol.AccountingControl, error)
	Update(
		ctx context.Context,
		ac *accountingcontrol.AccountingControl,
	) (*accountingcontrol.AccountingControl, error)
	ListWithAutoClosePeriods(ctx context.Context) ([]*accountingcontrol.AccountingControl, error)
}
