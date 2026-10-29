package shipmentloading

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazmatsegregationruleports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Service struct {
	commodityRepo     commodityports.CommodityRepository
	hazmatRuleRepo    hazmatsegregationruleports.HazmatSegregationRuleRepository
	equipmentTypeRepo equipmenttypeports.EquipmentTypeRepository
	effects           *shipmenteffects.Effects
}

type Deps struct {
	CommodityRepo     commodityports.CommodityRepository
	HazmatRuleRepo    hazmatsegregationruleports.HazmatSegregationRuleRepository
	EquipmentTypeRepo equipmenttypeports.EquipmentTypeRepository
	Effects           *shipmenteffects.Effects
}

func New(d Deps) *Service {
	return &Service{
		commodityRepo:     d.CommodityRepo,
		hazmatRuleRepo:    d.HazmatRuleRepo,
		equipmentTypeRepo: d.EquipmentTypeRepo,
		effects:           d.Effects,
	}
}

func (s *Service) CalculateLoadingOptimization(
	ctx context.Context,
	req *shipmentserviceports.LoadingOptimizationRequest,
) (*shipmentserviceports.LoadingOptimizationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	commodityIDs := make([]pulid.ID, 0, len(req.Commodities))
	for _, c := range req.Commodities {
		commodityIDs = append(commodityIDs, c.CommodityID)
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, commodityports.GetCommoditiesByIDsRequest{
		TenantInfo:   req.TenantInfo,
		CommodityIDs: commodityIDs,
	})
	if err != nil {
		return nil, err
	}

	trailerLength := defaultTrailerLengthFeet
	deckHeightFeet := req.DeckHeightFeet
	if req.EquipmentTypeID != nil && !req.EquipmentTypeID.IsNil() {
		et, etErr := s.equipmentTypeRepo.GetByID(
			ctx,
			equipmenttypeports.GetEquipmentTypeByIDRequest{
				ID:         *req.EquipmentTypeID,
				TenantInfo: req.TenantInfo,
			},
		)
		if etErr == nil && et.InteriorLength != nil && *et.InteriorLength > 0 {
			trailerLength = *et.InteriorLength
		}
		if etErr == nil {
			deckHeightFeet = et.DeckHeightFeet()
		}
	}

	control, _ := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	maxWeight := defaultMaxWeight
	if control != nil && control.MaxShipmentWeightLimit > 0 {
		maxWeight = int64(control.MaxShipmentWeightLimit)
	}

	checkHazmat := control != nil && control.CheckHazmatSegregation

	var rules []*hazmatsegregationrule.HazmatSegregationRule
	if checkHazmat {
		rules, _ = s.hazmatRuleRepo.ListActiveByTenant(ctx, req.TenantInfo)
	}

	commodityByID := make(map[pulid.ID]*commodity.Commodity, len(commodities))
	for _, c := range commodities {
		commodityByID[c.ID] = c
	}

	return buildLoadingPlan(&loadPlanParams{
		commodityByID:  commodityByID,
		inputs:         req.Commodities,
		stops:          req.Stops,
		rules:          rules,
		trailerLength:  trailerLength,
		deckHeightFeet: deckHeightFeet,
		maxWeight:      maxWeight,
		checkHazmat:    checkHazmat,
	}), nil
}

func (s *Service) CheckHazmatSegregation(
	ctx context.Context,
	req *shipmentports.CheckHazmatSegregationRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	control, err := s.effects.GetShipmentControl(ctx, req.TenantInfo)
	if err != nil {
		return err
	}

	conflicts, err := s.evaluateHazmatSegregationRequest(ctx, control, req)
	if err != nil {
		return err
	}

	multiErr := errortypes.NewMultiError()
	shipmentvalidator.AddHazmatConflictsToMultiError(multiErr, conflicts)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) evaluateHazmatSegregationRequest(
	ctx context.Context,
	control *shipmentcontrol.ShipmentControl,
	req *shipmentports.CheckHazmatSegregationRequest,
) ([]shipmentvalidator.HazmatConflict, error) {
	if !control.CheckHazmatSegregation {
		return nil, nil
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, commodityports.GetCommoditiesByIDsRequest{
		TenantInfo:   req.TenantInfo,
		CommodityIDs: req.CommodityIDs,
	})
	if err != nil {
		return nil, err
	}

	hazmatCommodities := shipmentvalidator.MapHazmatCommodities(commodities)
	if len(hazmatCommodities) < 2 {
		return nil, nil
	}

	rules, err := s.hazmatRuleRepo.ListActiveByTenant(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return shipmentvalidator.EvaluateHazmatConflicts(
		hazmatCandidatesFromCommodityIDs(req.CommodityIDs),
		hazmatCommodities,
		rules,
	), nil
}
