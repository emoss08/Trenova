package permitservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const PermitRequiredHoldCode = "PERMIT_REQUIRED"

type ServiceParams struct {
	fx.In

	PermitRepo       repositories.PermitRepository
	JurisdictionRepo repositories.JurisdictionRuleRepository
	HoldRepo         repositories.ShipmentHoldRepository
	Logger           *zap.Logger
}

type service struct {
	permitRepo       repositories.PermitRepository
	jurisdictionRepo repositories.JurisdictionRuleRepository
	holdRepo         repositories.ShipmentHoldRepository
	l                *zap.Logger
}

func NewService(p ServiceParams) services.PermitService {
	return &service{
		permitRepo:       p.PermitRepo,
		jurisdictionRepo: p.JurisdictionRepo,
		holdRepo:         p.HoldRepo,
		l:                p.Logger.Named("permit-service"),
	}
}

func tenantOf(entity *shipment.Shipment) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

// orderedStops flattens the shipment into the sequence a truck actually drives,
// which is what determines the jurisdiction order on the permit set.
func orderedStops(entity *shipment.Shipment) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, 4)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop != nil && stop.LocationID.IsNotNil() {
				stops = append(stops, stop)
			}
		}
	}

	return stops
}

// routeJurisdictions reduces the stop sequence to distinct jurisdictions in
// travel order. A route that re-enters a state does not need a second permit,
// so the first appearance wins.
func (s *service) routeJurisdictions(
	ctx context.Context,
	entity *shipment.Shipment,
) ([]permit.RouteJurisdiction, error) {
	stops := orderedStops(entity)
	if len(stops) == 0 {
		return nil, nil
	}

	locationIDs := make([]pulid.ID, 0, len(stops))
	for _, stop := range stops {
		locationIDs = append(locationIDs, stop.LocationID)
	}

	states, err := s.permitRepo.ResolveRouteStates(
		ctx,
		&repositories.ResolveRouteStatesRequest{
			TenantInfo:  tenantOf(entity),
			LocationIDs: locationIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	seen := make(map[pulid.ID]struct{}, len(states))
	route := make([]permit.RouteJurisdiction, 0, len(states))

	for _, stop := range stops {
		stateID, ok := states[stop.LocationID]
		if !ok || stateID.IsNil() {
			continue
		}
		if _, duplicate := seen[stateID]; duplicate {
			continue
		}

		seen[stateID] = struct{}{}
		route = append(route, permit.RouteJurisdiction{
			StateID:  stateID,
			Sequence: int16(len(route)),
		})
	}

	return route, nil
}

func (s *service) resolveRules(
	ctx context.Context,
	entity *shipment.Shipment,
	route []permit.RouteJurisdiction,
) (map[pulid.ID]*jurisdictionrule.Resolved, error) {
	stateIDs := make([]pulid.ID, 0, len(route))
	for _, jurisdiction := range route {
		stateIDs = append(stateIDs, jurisdiction.StateID)
	}

	tenantInfo := tenantOf(entity)

	rules, err := s.jurisdictionRepo.GetActiveByStateIDs(
		ctx,
		&repositories.GetJurisdictionRulesRequest{
			TenantInfo: tenantInfo,
			StateIDs:   stateIDs,
			At:         timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, err
	}

	overrides, err := s.jurisdictionRepo.GetOverrides(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	overrideByState := make(map[pulid.ID]*jurisdictionrule.Override, len(overrides))
	for _, override := range overrides {
		if override != nil {
			overrideByState[override.StateID] = override
		}
	}

	now := timeutils.NowUnix()
	resolved := make(map[pulid.ID]*jurisdictionrule.Resolved, len(rules))

	for _, rule := range rules {
		if rule == nil || !rule.IsEffectiveAt(now) {
			continue
		}
		resolved[rule.StateID] = rule.Resolve(overrideByState[rule.StateID])
	}

	return resolved, nil
}

func measurementsFor(entity *shipment.Shipment) jurisdictionrule.Measurements {
	envelope := entity.CurrentEnvelope()

	m := jurisdictionrule.Measurements{
		WidthFeet:  envelope.WidthFeet,
		LengthFeet: envelope.LengthFeet,
		HeightFeet: envelope.OverallHeightFeet,
	}

	if m.HeightFeet == 0 {
		m.HeightFeet = envelope.HeightFeet
	}
	if entity.Weight != nil {
		m.WeightPounds = *entity.Weight
	}

	return m
}

// deliveryAt is the instant a permit must still cover. Using the last scheduled
// arrival rather than now is what catches a permit that is valid at dispatch
// and lapses before the load is delivered.
func deliveryAt(entity *shipment.Shipment) int64 {
	stops := orderedStops(entity)
	if len(stops) == 0 {
		return timeutils.NowUnix()
	}

	last := stops[len(stops)-1]
	if last.ScheduledWindowEnd != nil && *last.ScheduledWindowEnd > 0 {
		return *last.ScheduledWindowEnd
	}
	if last.ScheduledWindowStart > 0 {
		return last.ScheduledWindowStart
	}

	return timeutils.NowUnix()
}

func FirstPickupAt(entity *shipment.Shipment) int64 {
	stops := orderedStops(entity)
	if len(stops) == 0 {
		return 0
	}

	return stops[0].ScheduledWindowStart
}

func (s *service) Assess(
	ctx context.Context,
	entity *shipment.Shipment,
) (*services.PermitAssessment, error) {
	assessment := &services.PermitAssessment{FeeIsBaseOnly: true}

	if entity == nil || !entity.HasEnvelope() {
		return assessment, nil
	}

	route, err := s.routeJurisdictions(ctx, entity)
	if err != nil || len(route) == 0 {
		return assessment, err
	}

	rules, err := s.resolveRules(ctx, entity, route)
	if err != nil {
		return nil, err
	}

	requirements := permit.Derive(permit.DeriveInput{
		Jurisdictions: route,
		Rules:         rules,
		Measurements:  measurementsFor(entity),
		DerivedAt:     timeutils.NowUnix(),
	})

	if len(requirements) == 0 {
		return assessment, nil
	}

	permits, err := s.permitRepo.ListByShipment(ctx, &repositories.ListPermitsRequest{
		ShipmentID: entity.ID,
	})
	if err != nil {
		return nil, err
	}

	coversAt := deliveryAt(entity)
	permit.MatchPermits(requirements, permits, coversAt)

	assessment.Requirements = requirements
	assessment.Open = permit.UnsatisfiedRequirements(requirements)
	assessment.ExpiringBeforeDelivery = permit.ExpiringBefore(requirements, permits, coversAt)
	assessment.TotalEscorts = permit.TotalEscorts(requirements)
	assessment.TotalEstimatedFee = permit.TotalEstimatedFee(requirements)
	assessment.MaxLeadTimeDays = permit.MaxLeadTimeDays(requirements)
	assessment.EarliestPickup = permit.EarliestFeasiblePickup(
		requirements, timeutils.NowUnix(),
	)

	return assessment, nil
}

func (s *service) Sync(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) (*services.PermitAssessment, error) {
	assessment, err := s.Assess(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.permitRepo.ReplaceForShipment(ctx, &repositories.ReplaceRequirementsRequest{
		TenantInfo:   tenantOf(entity),
		ShipmentID:   entity.ID,
		Requirements: assessment.Requirements,
	}); err != nil {
		return nil, err
	}

	s.reconcileHold(ctx, entity, assessment, actor)

	return assessment, nil
}

func (s *service) ListPermits(
	ctx context.Context,
	shipmentID pulid.ID,
	_ pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return s.permitRepo.ListByShipment(ctx, &repositories.ListPermitsRequest{
		ShipmentID: shipmentID,
	})
}

func (s *service) CreatePermit(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.permitRepo.Create(ctx, entity)
}

func (s *service) UpdatePermit(
	ctx context.Context,
	entity *permit.Permit,
) (*permit.Permit, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.permitRepo.Update(ctx, entity)
}

func (s *service) WaiveRequirement(
	ctx context.Context,
	req *services.WaiveRequirementRequest,
) (*permit.Requirement, error) {
	multiErr := errortypes.NewMultiError()

	if len(req.Reason) < permit.MinWaiverReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid,
			"Provide a reason of at least 10 characters explaining the waiver")
	}
	if req.WaivedByID.IsNil() {
		multiErr.Add("waivedById", errortypes.ErrRequired, "A waiving user is required")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.permitRepo.WaiveRequirement(ctx, &repositories.WaiveRequirementRepoRequest{
		TenantInfo:    req.TenantInfo,
		RequirementID: req.RequirementID,
		WaivedByID:    req.WaivedByID,
		Reason:        req.Reason,
	})
}
