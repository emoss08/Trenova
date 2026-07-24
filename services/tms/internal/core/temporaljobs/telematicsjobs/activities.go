package telematicsjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	eventRetentionSeconds     = int64(90 * 86400)
	violationRetentionSeconds = int64(365 * 86400)
)

type ListTelematicsTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListTelematicsTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type TenantPayload struct {
	temporaljobs.TenantWorkItem
}

type PollTenantResult struct {
	telematicsservice.TenantPollResult
}

type SweepTenantResult struct {
	telematicsservice.TenantSweepResult
}

type RetentionResult struct {
	RowsDeleted int64 `json:"rowsDeleted"`
}

type ActivitiesParams struct {
	fx.In

	IntegrationRepo repositories.IntegrationRepository
	TelematicsRepo  repositories.TelematicsRepository
	Service         *telematicsservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	integrationRepo repositories.IntegrationRepository
	telematicsRepo  repositories.TelematicsRepository
	service         *telematicsservice.Service
	logger          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		integrationRepo: p.IntegrationRepo,
		telematicsRepo:  p.TelematicsRepo,
		service:         p.Service,
		logger:          p.Logger.Named("temporal.telematics"),
	}
}

func (a *Activities) CleanupTelematicsActivity(
	ctx context.Context,
) (*RetentionResult, error) {
	recordActivityHeartbeat(ctx, "telematics-retention")

	now := timeutils.NowUnix()
	deleted, err := a.telematicsRepo.CleanupExpired(
		ctx,
		now-eventRetentionSeconds,
		now-violationRetentionSeconds,
	)
	if err != nil {
		return nil, err
	}
	return &RetentionResult{RowsDeleted: deleted}, nil
}

func (a *Activities) ListTelematicsTenantsActivity(
	ctx context.Context,
	payload *ListTelematicsTenantsPayload,
) (*ListTelematicsTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)
	integrations, err := a.integrationRepo.ListEnabledByType(ctx, integration.TypeSamsara)
	if err != nil {
		return nil, err
	}

	spec := integration.ConfigSpecs[integration.TypeSamsara]
	tenants := make([]pagination.TenantInfo, 0, min(len(integrations), limit))
	for idx := range integrations {
		if len(tenants) >= limit {
			break
		}
		integ := integrations[idx]
		if !integration.HasRequiredConfiguration(integ.Configuration, spec) {
			continue
		}
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: integ.OrganizationID,
			BuID:  integ.BusinessUnitID,
		})
	}

	return &ListTelematicsTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) PollTenantTelematicsActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*PollTenantResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "polling-telematics", tenantInfo.OrgID.String())

	result, err := a.service.PollTenant(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("failed to poll telematics for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		if result == nil {
			return nil, err
		}
	}
	if result == nil {
		result = new(telematicsservice.TenantPollResult)
	}
	return &PollTenantResult{TenantPollResult: *result}, nil
}

func (a *Activities) SweepTenantTelematicsActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*SweepTenantResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "sweeping-telematics", tenantInfo.OrgID.String())

	result, err := a.service.SweepTenant(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("failed to sweep telematics for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		if result == nil {
			return nil, err
		}
	}
	if result == nil {
		result = new(telematicsservice.TenantSweepResult)
	}
	return &SweepTenantResult{TenantSweepResult: *result}, nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
