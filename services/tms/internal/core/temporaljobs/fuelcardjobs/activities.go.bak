package fuelcardjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ListFuelCardTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListFuelCardTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type TenantPayload struct {
	temporaljobs.TenantWorkItem
}

type SyncTenantResult struct {
	Providers       int `json:"providers"`
	Fetched         int `json:"fetched"`
	Committed       int `json:"committed"`
	Queued          int `json:"queued"`
	CardsDiscovered int `json:"cardsDiscovered"`
}

type ActivitiesParams struct {
	fx.In

	IntegrationRepo repositories.IntegrationRepository
	Service         *fuelpurchaseservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	integrationRepo repositories.IntegrationRepository
	service         *fuelpurchaseservice.Service
	logger          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		integrationRepo: p.IntegrationRepo,
		service:         p.Service,
		logger:          p.Logger.Named("temporal.fuel-card"),
	}
}

// ListFuelCardTenantsActivity finds the organizations with at least one enabled,
// fully configured fuel card connection. A tenant can hold several, so it is
// listed once and its providers are read together.
func (a *Activities) ListFuelCardTenantsActivity(
	ctx context.Context,
	payload *ListFuelCardTenantsPayload,
) (*ListFuelCardTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)

	seen := make(map[pagination.TenantInfo]struct{}, limit)
	tenants := make([]pagination.TenantInfo, 0, limit)

	for _, typ := range fuelCardIntegrationTypes() {
		integrations, err := a.integrationRepo.ListEnabledByType(ctx, typ)
		if err != nil {
			return nil, err
		}

		spec := integration.ConfigSpecs[typ]
		for idx := range integrations {
			if len(tenants) >= limit {
				break
			}

			record := integrations[idx]
			if !integration.HasRequiredConfiguration(record.Configuration, spec) {
				continue
			}

			tenant := pagination.TenantInfo{
				OrgID: record.OrganizationID,
				BuID:  record.BusinessUnitID,
			}
			if _, duplicate := seen[tenant]; duplicate {
				continue
			}

			seen[tenant] = struct{}{}
			tenants = append(tenants, tenant)
		}
	}

	return &ListFuelCardTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) SyncTenantFuelCardsActivity(
	ctx context.Context,
	payload *TenantPayload,
) (*SyncTenantResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "syncing-fuel-cards", tenantInfo.OrgID.String())

	results, err := a.service.SyncAllFeeds(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("failed to sync fuel card feeds for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))

		return nil, err
	}

	summary := &SyncTenantResult{Providers: len(results)}
	for _, result := range results {
		summary.Fetched += result.Fetched
		summary.Committed += result.Committed
		summary.Queued += result.Queued
		summary.CardsDiscovered += result.CardsDiscovered
	}

	return summary, nil
}

func fuelCardIntegrationTypes() []integration.Type {
	return []integration.Type{
		integration.TypeWEXFuel,
		integration.TypeComdataFuel,
		integration.TypeRampFuel,
	}
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
