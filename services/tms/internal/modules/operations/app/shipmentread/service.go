package shipmentread

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l                   *zap.Logger
	repo                shipmentports.ShipmentRepository
	cacheRepo           customerports.ShipmentCacheRepository
	modeProfileService  modeprofileports.ModeProfileService
	quickFilters        shipmentports.ShipmentQuickFilterBasisResolver
	commercial          *shipmentcommercial.Calculator
	distanceCalculation shipmentdistanceports.DistanceCalculationService
	effects             *shipmenteffects.Effects
}

type Params struct {
	fx.In

	Logger              *zap.Logger
	Repo                shipmentports.ShipmentRepository
	CacheRepo           customerports.ShipmentCacheRepository
	ModeProfileService  modeprofileports.ModeProfileService
	QuickFilters        shipmentports.ShipmentQuickFilterBasisResolver
	Commercial          *shipmentcommercial.Calculator
	DistanceCalculation shipmentdistanceports.DistanceCalculationService `optional:"true"`
	Effects             *shipmenteffects.Effects
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.shipment"),
		repo:                p.Repo,
		cacheRepo:           p.CacheRepo,
		modeProfileService:  p.ModeProfileService,
		quickFilters:        p.QuickFilters,
		commercial:          p.Commercial,
		distanceCalculation: p.DistanceCalculation,
		effects:             p.Effects,
	}
}

func (s *Service) PreviewRecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentdistanceports.ShipmentDistancePreview, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}

	return s.distanceCalculation.PreviewRecalculateShipment(ctx, shipmentID, tenantInfo)
}

func (s *Service) List(
	ctx context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	if req.ShipmentOptions.HasQuickFilters() {
		if s.quickFilters == nil {
			return nil, errQuickFiltersUnavailable
		}
		if err := s.quickFilters.Prepare(
			ctx,
			req.Filter.TenantInfo,
			&req.ShipmentOptions,
		); err != nil {
			return nil, err
		}
	}

	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	entity, err := s.cacheRepo.GetByID(ctx, req)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, customerports.ErrCacheMiss) {
		s.l.Warn(
			"failed to load shipment from cache",
			zap.Error(err),
			zap.String("shipmentID", req.ID.String()),
		)
	}

	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetUnassigned(
	ctx context.Context,
	req *shipmentports.GetUnassignedShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return s.repo.GetUnassigned(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req *shipmentports.GetShipmentsByIDsRequest,
) ([]*shipment.Shipment, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *shipmentports.ShipmentSelectOptionsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetUIPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentserviceports.ShipmentUIPolicy, error) {
	control, err := s.effects.GetShipmentControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	policy := &shipmentserviceports.ShipmentUIPolicy{
		AllowMoveRemovals:      control.AllowMoveRemovals,
		CheckForDuplicateBOLs:  control.CheckForDuplicateBOLs,
		CheckHazmatSegregation: control.CheckHazmatSegregation,
		MaxShipmentWeightLimit: control.MaxShipmentWeightLimit,
	}

	if s.modeProfileService == nil {
		return policy, nil
	}

	resolved, err := s.modeProfileService.Resolve(ctx, &modeprofileports.ResolveModeProfileRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to resolve mode profile for ui policy", zap.Error(err))
		return policy, nil
	}

	policy.Profile = resolved

	return policy, nil
}

func (s *Service) GetPreviousRates(
	ctx context.Context,
	req *shipmentports.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.repo.GetPreviousRates(ctx, req)
}

func (s *Service) CheckForDuplicateBOLs(
	ctx context.Context,
	req *shipmentports.DuplicateBOLCheckRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	return s.effects.CheckDuplicateBOLsWithControl(ctx, control, req)
}

func (s *Service) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipmentports.ShipmentTotalsResponse, error) {
	if entity == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("shipment", errortypes.ErrRequired, "Shipment is required")
		return nil, multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if err = s.effects.HydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}

	s.effects.ApplyShipmentEnvelope(ctx, entity)

	if s.distanceCalculation != nil {
		if _, err = s.distanceCalculation.ResolveForShipment(ctx, entity); err != nil {
			return nil, err
		}
	}

	resp, err := s.commercial.CalculateTotals(ctx, entity, control, userID)
	if err != nil {
		s.l.Error("failed to calculate shipment totals", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func (s *Service) CalculateDistance(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipmentdistanceports.DistanceCalculationResponse, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}
	if err := s.effects.HydrateShipmentCommodityDetails(ctx, entity); err != nil {
		return nil, err
	}
	return s.distanceCalculation.ResolveForShipment(ctx, entity)
}

func (s *Service) RecalculateDistance(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentdistanceports.DistanceCalculationResponse, error) {
	if s.distanceCalculation == nil {
		return nil, errortypes.NewBusinessError("distance calculation service is not configured")
	}
	return s.distanceCalculation.RecalculateShipment(ctx, shipmentID, tenantInfo)
}

var _ shipmentserviceports.ShipmentReader = (*Service)(nil)
