package exchangeratejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/exchangerateservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ListExchangeRateTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListExchangeRateTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type RefreshExchangeRateTenantPayload struct {
	temporaljobs.TenantWorkItem
	BaseCurrency string `json:"baseCurrency"`
}

type RefreshExchangeRatesResult struct {
	temporaljobs.TenantRunResult
}

type ActivitiesParams struct {
	fx.In

	ExchangeRateService   *exchangerateservice.Service
	AccountingControlRepo repositories.AccountingControlRepository
	IntegrationRepo       repositories.IntegrationRepository
	Logger                *zap.Logger
}

type Activities struct {
	exchangeRateSvc       *exchangerateservice.Service
	accountingControlRepo repositories.AccountingControlRepository
	integrationRepo       repositories.IntegrationRepository
	logger                *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		exchangeRateSvc:       p.ExchangeRateService,
		accountingControlRepo: p.AccountingControlRepo,
		integrationRepo:       p.IntegrationRepo,
		logger:                p.Logger.Named("temporal.exchange-rate"),
	}
}

func (a *Activities) RefreshExchangeRatesActivity(ctx context.Context) error {
	a.logger.Info("Starting exchange rate refresh activity")
	recordActivityHeartbeat(ctx, "refreshing-exchange-rates")

	integrations, err := a.integrationRepo.ListEnabledByType(ctx, integration.TypeOANDAExchangeRates)
	if err != nil {
		a.logger.Error("Failed to list enabled OANDA exchange-rate integrations", zap.Error(err))
		return err
	}

	if len(integrations) == 0 {
		a.logger.Info("No enabled OANDA exchange-rate integrations found, skipping refresh")
		return nil
	}

	for idx := range integrations {
		recordActivityHeartbeat(ctx, "refreshing", idx+1, len(integrations))

		integ := integrations[idx]
		tenantInfo := pagination.TenantInfo{
			OrgID: integ.OrganizationID,
			BuID:  integ.BusinessUnitID,
		}

		baseCurrency, currencyErr := a.functionalCurrency(ctx, tenantInfo)
		if currencyErr != nil {
			a.logger.Error("Failed to resolve functional currency for tenant",
				zap.String("orgID", integ.OrganizationID.String()),
				zap.Error(currencyErr))
			continue
		}

		if refreshErr := a.exchangeRateSvc.RefreshRates(ctx, tenantInfo, baseCurrency); refreshErr != nil {
			a.logger.Error("Failed to refresh rates for tenant",
				zap.String("orgID", integ.OrganizationID.String()),
				zap.String("baseCurrency", baseCurrency),
				zap.Error(refreshErr))
			continue
		}

		a.logger.Info("Refreshed exchange rates for tenant",
			zap.String("orgID", integ.OrganizationID.String()))
	}

	a.logger.Info("Exchange rate refresh activity completed",
		zap.Int("tenantsProcessed", len(integrations)))
	return nil
}

func (a *Activities) ListExchangeRateTenantsActivity(
	ctx context.Context,
	payload *ListExchangeRateTenantsPayload,
) (*ListExchangeRateTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)
	integrations, err := a.integrationRepo.ListEnabledByType(ctx, integration.TypeOANDAExchangeRates)
	if err != nil {
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, min(len(integrations), limit))
	for idx := range integrations {
		if len(tenants) >= limit {
			break
		}
		integ := integrations[idx]
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: integ.OrganizationID,
			BuID:  integ.BusinessUnitID,
		})
	}

	return &ListExchangeRateTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) RefreshExchangeRatesForTenantActivity(
	ctx context.Context,
	payload *RefreshExchangeRateTenantPayload,
) error {
	baseCurrency := payload.BaseCurrency
	if baseCurrency == "" {
		var err error
		baseCurrency, err = a.functionalCurrency(ctx, payload.TenantInfo())
		if err != nil {
			return err
		}
	}

	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "refreshing-exchange-rates", tenantInfo.OrgID.String())
	if err := a.exchangeRateSvc.RefreshRates(ctx, tenantInfo, baseCurrency); err != nil {
		a.logger.Error("Failed to refresh rates for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		return err
	}

	a.logger.Info("Refreshed exchange rates for tenant",
		zap.String("orgID", tenantInfo.OrgID.String()),
		zap.String("buID", tenantInfo.BuID.String()))
	return nil
}

func (a *Activities) functionalCurrency(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	accountingControl, err := a.accountingControlRepo.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return "", err
	}
	return accountingControl.FunctionalCurrencyCode, nil
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
