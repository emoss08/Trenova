package exchangeratejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/exchangerateservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	ExchangeRateService *exchangerateservice.Service
	IntegrationRepo     repositories.IntegrationRepository
	Logger              *zap.Logger
}

type Activities struct {
	exchangeRateSvc  *exchangerateservice.Service
	integrationRepo  repositories.IntegrationRepository
	logger           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		exchangeRateSvc: p.ExchangeRateService,
		integrationRepo: p.IntegrationRepo,
		logger:          p.Logger.Named("temporal.exchange-rate"),
	}
}

func (a *Activities) RefreshExchangeRatesActivity(ctx context.Context) error {
	a.logger.Info("Starting exchange rate refresh activity")
	recordActivityHeartbeat(ctx, "refreshing-exchange-rates")

	integrations, err := a.integrationRepo.ListEnabledByType(ctx, integration.TypeExchangeRateAPI)
	if err != nil {
		a.logger.Error("Failed to list enabled ExchangeRateAPI integrations", zap.Error(err))
		return err
	}

	if len(integrations) == 0 {
		a.logger.Info("No enabled ExchangeRateAPI integrations found, skipping refresh")
		return nil
	}

	for idx := range integrations {
		recordActivityHeartbeat(ctx, "refreshing", idx+1, len(integrations))

		integ := integrations[idx]
		tenantInfo := pagination.TenantInfo{
			OrgID:  integ.OrganizationID,
			BuID:   integ.BusinessUnitID,
		}

		if err := a.exchangeRateSvc.RefreshRates(ctx, tenantInfo, "USD"); err != nil {
			a.logger.Error("Failed to refresh rates for tenant",
				zap.String("orgID", integ.OrganizationID.String()),
				zap.Error(err))
			continue
		}

		a.logger.Info("Refreshed exchange rates for tenant",
			zap.String("orgID", integ.OrganizationID.String()))
	}

	a.logger.Info("Exchange rate refresh activity completed",
		zap.Int("tenantsProcessed", len(integrations)))
	return nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
