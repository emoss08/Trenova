package detentionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	DetentionService    *detentionservice.Service
	ShipmentControlRepo repositories.ShipmentControlRepository
	Logger              *zap.Logger
}

type Activities struct {
	detentionService    *detentionservice.Service
	shipmentControlRepo repositories.ShipmentControlRepository
	logger              *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		detentionService:    p.DetentionService,
		shipmentControlRepo: p.ShipmentControlRepo,
		logger:              p.Logger.Named("temporal.detention"),
	}
}

func (a *Activities) ListDetentionTenantsActivity(
	ctx context.Context,
	payload *ListDetentionTenantsPayload,
) (*ListDetentionTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)

	tenants, err := a.shipmentControlRepo.ListDetentionEngineTenants(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &ListDetentionTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) SweepTenantNoticesActivity(
	ctx context.Context,
	payload *SweepTenantNoticesPayload,
) (*SweepTenantNoticesResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "sweeping-detention-notices", tenantInfo.OrgID.String())

	result, err := a.detentionService.SweepNoticesDue(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("Failed to sweep detention notices for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		return nil, err
	}

	return &SweepTenantNoticesResult{
		Due:     result.Due,
		Sent:    result.Sent,
		Skipped: result.Skipped,
		Failed:  result.Failed,
	}, nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, details...)
	}
}
