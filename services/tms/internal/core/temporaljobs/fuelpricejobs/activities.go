package fuelpricejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const reRateBatchLimit = 200

type ListFuelPriceTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListFuelPriceTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type RefreshFuelPricesTenantPayload struct {
	temporaljobs.TenantWorkItem
}

type RefreshFuelPricesTenantResult struct {
	Skipped bool `json:"skipped"`
	NewRows int  `json:"newRows"`
}

type ReRateFallbackShipmentsPayload struct {
	temporaljobs.TenantWorkItem
}

type ReRateFallbackShipmentsResult struct {
	ShipmentsReRated int `json:"shipmentsReRated"`
}

type RefreshFuelPricesResult struct {
	temporaljobs.TenantRunResult
	NewRows          int `json:"newRows"`
	ShipmentsReRated int `json:"shipmentsReRated"`
}

type ActivitiesParams struct {
	fx.In

	FuelSurchargeService *fuelsurchargeservice.Service
	Commercial           *shipmentcommercial.Calculator
	IntegrationRepo      repositories.IntegrationRepository
	ProgramRepo          repositories.FuelSurchargeProgramRepository
	ShipmentRepo         repositories.ShipmentRepository
	ShipmentControlRepo  repositories.ShipmentControlRepository
	Logger               *zap.Logger
}

type Activities struct {
	fuelSurchargeSvc    *fuelsurchargeservice.Service
	commercial          *shipmentcommercial.Calculator
	integrationRepo     repositories.IntegrationRepository
	programRepo         repositories.FuelSurchargeProgramRepository
	shipmentRepo        repositories.ShipmentRepository
	shipmentControlRepo repositories.ShipmentControlRepository
	logger              *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		fuelSurchargeSvc:    p.FuelSurchargeService,
		commercial:          p.Commercial,
		integrationRepo:     p.IntegrationRepo,
		programRepo:         p.ProgramRepo,
		shipmentRepo:        p.ShipmentRepo,
		shipmentControlRepo: p.ShipmentControlRepo,
		logger:              p.Logger.Named("temporal.fuel-price"),
	}
}

func (a *Activities) ListFuelPriceTenantsActivity(
	ctx context.Context,
	payload *ListFuelPriceTenantsPayload,
) (*ListFuelPriceTenantsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)
	integrations, err := a.integrationRepo.ListEnabledByType(ctx, integration.TypeEIAFuelPrices)
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

	return &ListFuelPriceTenantsResult{
		Tenants: temporaljobs.BuildTenantWorkItems(tenants, 1),
	}, nil
}

func (a *Activities) RefreshFuelPricesForTenantActivity(
	ctx context.Context,
	payload *RefreshFuelPricesTenantPayload,
) (*RefreshFuelPricesTenantResult, error) {
	tenantInfo := payload.TenantInfo()
	recordActivityHeartbeat(ctx, "refreshing-fuel-prices", tenantInfo.OrgID.String())

	result, err := a.fuelSurchargeSvc.RefreshEIAPrices(ctx, tenantInfo)
	if err != nil {
		a.logger.Error("Failed to refresh fuel prices for tenant",
			zap.String("orgID", tenantInfo.OrgID.String()),
			zap.String("buID", tenantInfo.BuID.String()),
			zap.Error(err))
		return nil, err
	}

	a.logger.Info("Refreshed fuel prices for tenant",
		zap.String("orgID", tenantInfo.OrgID.String()),
		zap.Bool("skipped", result.Skipped),
		zap.Int("newRows", result.NewRows))

	return &RefreshFuelPricesTenantResult{
		Skipped: result.Skipped,
		NewRows: result.NewRows,
	}, nil
}

func (a *Activities) ReRateFallbackShipmentsActivity(
	ctx context.Context,
	payload *ReRateFallbackShipmentsPayload,
) (*ReRateFallbackShipmentsResult, error) {
	tenantInfo := payload.TenantInfo()
	log := a.logger.With(
		zap.String("orgID", tenantInfo.OrgID.String()),
		zap.String("buID", tenantInfo.BuID.String()),
	)

	shipmentIDs, err := a.programRepo.ListFallbackShipmentIDs(ctx, tenantInfo, reRateBatchLimit)
	if err != nil {
		return nil, err
	}

	if len(shipmentIDs) == 0 {
		return &ReRateFallbackShipmentsResult{}, nil
	}

	control, err := a.shipmentControlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		log.Warn("failed to load shipment control for fallback re-rate", zap.Error(err))
	}

	reRated := 0
	for idx, shipmentID := range shipmentIDs {
		recordActivityHeartbeat(ctx, "re-rating-fallback-shipments", idx+1, len(shipmentIDs))

		if rErr := a.reRateShipment(ctx, tenantInfo, shipmentID, control); rErr != nil {
			log.Warn("failed to re-rate fallback shipment",
				zap.String("shipmentID", shipmentID.String()),
				zap.Error(rErr))
			continue
		}
		reRated++
	}

	log.Info("Re-rated fuel surcharge fallback shipments",
		zap.Int("candidates", len(shipmentIDs)),
		zap.Int("reRated", reRated))

	return &ReRateFallbackShipmentsResult{ShipmentsReRated: reRated}, nil
}

func (a *Activities) reRateShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	control *tenant.ShipmentControl,
) error {
	entity, err := a.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	if err = a.commercial.Recalculate(ctx, entity, control, pulid.Nil); err != nil {
		return err
	}

	_, err = a.shipmentRepo.UpdateDerivedState(ctx, entity)
	return err
}

func recordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		_ = recover()
	}()

	activity.RecordHeartbeat(ctx, details...)
}
