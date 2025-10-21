package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetDispatchControlRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type DispatchControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.DispatchControl, error)
	Update(ctx context.Context, dc *tenant.DispatchControl) (*tenant.DispatchControl, error)
}

type DispatchControlCacheRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.DispatchControl, error)
	Set(ctx context.Context, dc *tenant.DispatchControl) error
	Invalidate(ctx context.Context, orgID pulid.ID) error
}
