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
	ListWithScheduledPeriodClose(
		ctx context.Context,
	) ([]*accountingcontrol.AccountingControl, error)
	// ListAll returns one row per tenant that keeps books, regardless of how
	// that tenant has configured period close. Work that every tenant needs —
	// keeping a fiscal calendar ahead of itself, for one — reads this rather
	// than the scheduled-close opt-in.
	ListAll(ctx context.Context) ([]*accountingcontrol.AccountingControl, error)
	Update(
		ctx context.Context,
		ac *accountingcontrol.AccountingControl,
	) (*accountingcontrol.AccountingControl, error)
}
