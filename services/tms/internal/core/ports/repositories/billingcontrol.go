package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetBillingControlRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type BillingControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.BillingControl, error)
	Update(ctx context.Context, bc *tenant.BillingControl) (*tenant.BillingControl, error)
}

type BillingControlCacheRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.BillingControl, error)
	Set(ctx context.Context, bc *tenant.BillingControl) error
	Invalidate(ctx context.Context, orgID pulid.ID) error
}
