package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetAccountingControlRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type AccountingControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*accounting.AccountingControl, error)
	Update(
		ctx context.Context,
		ac *accounting.AccountingControl,
	) (*accounting.AccountingControl, error)
}
type AccountingControlCacheRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*accounting.AccountingControl, error)
	Set(ctx context.Context, ac *accounting.AccountingControl) error
	Invalidate(ctx context.Context, orgID pulid.ID) error
}
