package permitservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
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
	ShipmentRepo     repositories.ShipmentRepository
	AuditService     services.AuditService
	Logger           *zap.Logger
}

type service struct {
	permitRepo       repositories.PermitRepository
	jurisdictionRepo repositories.JurisdictionRuleRepository
	holdRepo         repositories.ShipmentHoldRepository
	shipmentRepo     repositories.ShipmentRepository
	auditService     services.AuditService
	l                *zap.Logger
}

func NewService(p ServiceParams) services.PermitService {
	return &service{
		permitRepo:       p.PermitRepo,
		jurisdictionRepo: p.JurisdictionRepo,
		holdRepo:         p.HoldRepo,
		shipmentRepo:     p.ShipmentRepo,
		auditService:     p.AuditService,
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

// assessJurisdictions projects the resolved rules back onto the route in travel
// order, carrying the headroom for every state rather than only the exceeded
// ones. A jurisdiction with no rule row is skipped rather than reported as
// unlimited, because silence about a state is not permission to enter it.
func assessJurisdictions(
	route []permit.RouteJurisdiction,
	rules map[pulid.ID]*jurisdictionrule.Resolved,
	measurements jurisdictionrule.Measurements,
) []services.AssessedJurisdiction {
	assessed := make([]services.AssessedJurisdiction, 0, len(route))

	for _, jurisdiction := range route {
		resolved, ok := rules[jurisdiction.StateID]
		if !ok || resolved == nil {
			continue
		}

		assessed = append(assessed, services.AssessedJurisdiction{
			StateID:              jurisdiction.StateID,
			StateCode:            resolved.StateCode,
			StateName:            resolved.StateName,
			Sequence:             jurisdiction.Sequence,
			VerificationState:    resolved.VerificationState,
			SourceNote:           resolved.SourceNote,
			SourceURL:            resolved.SourceURL,
			MaxWidthFeet:         resolved.MaxWidthFeet,
			MaxHeightFeet:        resolved.MaxHeightFeet,
			MaxLengthFeet:        resolved.MaxLengthFeet,
			MaxWeightPounds:      resolved.MaxWeightPounds,
			WidthHeadroomFeet:    resolved.MaxWidthFeet - measurements.WidthFeet,
			HeightHeadroomFeet:   resolved.MaxHeightFeet - measurements.HeightFeet,
			LengthHeadroomFeet:   resolved.MaxLengthFeet - measurements.LengthFeet,
			WeightHeadroomPounds: resolved.MaxWeightPounds - measurements.WeightPounds,
			RequiresPermit:       resolved.RequiresPermit(measurements),
			Restrictions:         resolved.Restrictions(),
		})
	}

	return assessed
}

func (s *service) Assess(
	ctx context.Context,
	entity *shipment.Shipment,
) (*services.PermitAssessment, error) {
	assessment := &services.PermitAssessment{FeeIsBaseOnly: true}

	if entity == nil {
		return assessment, nil
	}

	measurements := measurementsFor(entity)
	assessment.Measurements = services.AssessedMeasurements{
		WidthFeet:    measurements.WidthFeet,
		HeightFeet:   measurements.HeightFeet,
		LengthFeet:   measurements.LengthFeet,
		WeightPounds: measurements.WeightPounds,
	}

	if !entity.HasEnvelope() {
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

	assessment.Jurisdictions = assessJurisdictions(route, rules, measurements)
	assessment.RouteResolved = len(assessment.Jurisdictions) > 0

	requirements := permit.Derive(permit.DeriveInput{
		Jurisdictions: route,
		Rules:         rules,
		Measurements:  measurements,
		DerivedAt:     timeutils.NowUnix(),
	})

	if len(requirements) == 0 {
		return assessment, nil
	}

	permits, err := s.permitRepo.ListByShipment(ctx, &repositories.ListPermitsRequest{
		TenantInfo: tenantOf(entity),
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

// carryWaiversForward copies a live waiver onto the re-derived requirement for
// the same jurisdiction. Re-deriving rather than preserving the old row is
// deliberate: the exceedance that justified the waiver can change between saves,
// and an operator needs to see the waiver attached to the load's current
// numbers rather than to the ones it was granted against.
//
// Satisfaction is not carried forward. MatchPermits recomputes it from the
// permit rows on every assessment, so copying a stale SatisfiedByPermitID would
// let a permit that has since expired keep discharging a requirement.
func carryWaiversForward(derived, existing []*permit.Requirement) {
	waived := make(map[pulid.ID]*permit.Requirement, len(existing))

	for _, requirement := range existing {
		if requirement != nil && requirement.Status == permit.RequirementWaived {
			waived[requirement.StateID] = requirement
		}
	}

	if len(waived) == 0 {
		return
	}

	for _, requirement := range derived {
		if requirement == nil {
			continue
		}

		prior, ok := waived[requirement.StateID]
		if !ok {
			continue
		}

		requirement.Status = permit.RequirementWaived
		requirement.WaivedByID = prior.WaivedByID
		requirement.WaiverReason = prior.WaiverReason
	}
}

func (s *service) Sync(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) (*services.PermitAssessment, error) {
	tenantInfo := tenantOf(entity)

	existing, err := s.permitRepo.ListRequirements(ctx, &repositories.ListRequirementsRequest{
		TenantInfo: tenantInfo,
		ShipmentID: entity.ID,
	})
	if err != nil {
		return nil, err
	}

	assessment, err := s.Assess(ctx, entity)
	if err != nil {
		return nil, err
	}

	carryWaiversForward(assessment.Requirements, existing)

	// Recompute after the carry-forward: a waived requirement no longer blocks
	// dispatch, so the open set the hold is built from has to reflect it.
	assessment.Open = permit.UnsatisfiedRequirements(assessment.Requirements)

	if err = s.permitRepo.ReplaceForShipment(ctx, &repositories.ReplaceRequirementsRequest{
		TenantInfo:   tenantInfo,
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
	tenantInfo pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return s.permitRepo.ListByShipment(ctx, &repositories.ListPermitsRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
}

func (s *service) ListRequirements(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	return s.permitRepo.ListRequirements(ctx, &repositories.ListRequirementsRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
}

// logPermitAction records a permit write in the audit trail.
//
// This is not optional bookkeeping for this resource. A permit is the evidence
// that an oversize movement was legal, and a waiver is a named person accepting
// the compliance risk of moving without one. When a load is stopped at a scale
// or a claim is worked months later, the audit entry is what establishes who
// decided what and when.
//
// Marked Critical for the same reason: these entries must survive retention
// trimming that ordinary edits are subject to.
func (s *service) logPermitAction(
	params *permitAuditParams,
) {
	if s.auditService == nil {
		return
	}

	logParams := &services.LogActionParams{
		Resource:       permission.ResourcePermit,
		ResourceID:     params.ResourceID,
		Operation:      params.Operation,
		UserID:         params.Actor.UserID,
		APIKeyID:       params.Actor.APIKeyID,
		PrincipalType:  params.Actor.PrincipalType,
		PrincipalID:    params.Actor.PrincipalID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Critical:       true,
	}
	if params.Current != nil {
		logParams.CurrentState = jsonutils.MustToJSON(params.Current)
	}
	if params.Previous != nil {
		logParams.PreviousState = jsonutils.MustToJSON(params.Previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(params.Comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId": params.ShipmentID.String(),
		}),
	}
	if params.Previous != nil && params.Current != nil {
		opts = append(opts, auditservice.WithDiff(params.Previous, params.Current))
	}

	if err := s.auditService.LogAction(logParams, opts...); err != nil {
		s.l.Error("failed to log permit action",
			zap.String("resourceId", params.ResourceID),
			zap.Error(err),
		)
	}
}

type permitAuditParams struct {
	ResourceID     string
	ShipmentID     pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Operation      permission.Operation
	Actor          services.AuditActor
	Previous       any
	Current        any
	Comment        string
}

// resync re-derives and reconciles the hold after a permit write. Recording the
// permit that discharges a requirement has to clear the block in the same
// request; leaving it until the next shipment save would mean an operator files
// a permit, sees nothing change, and files it again.
//
// A failure is logged rather than returned: the permit itself is already
// written, and reporting an error would invite the operator to record it twice.
func (s *service) resync(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) {
	if s.shipmentRepo == nil {
		return
	}

	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		s.l.Error("failed to reload shipment after permit write",
			zap.String("shipmentId", shipmentID.String()), zap.Error(err))
		return
	}

	if _, err = s.Sync(ctx, entity, actor); err != nil {
		s.l.Error("failed to resync permits after permit write",
			zap.String("shipmentId", shipmentID.String()), zap.Error(err))
	}
}

func (s *service) CreatePermit(
	ctx context.Context,
	entity *permit.Permit,
	actor *services.RequestActor,
) (*permit.Permit, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.permitRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logPermitAction(&permitAuditParams{
		ResourceID:     created.ID.String(),
		ShipmentID:     created.ShipmentID,
		OrganizationID: created.OrganizationID,
		BusinessUnitID: created.BusinessUnitID,
		Operation:      permission.OpCreate,
		Actor:          actor.AuditActorOrSystem(),
		Current:        created,
		Comment:        "Permit recorded",
	})

	s.resync(ctx, created.ShipmentID, pagination.TenantInfo{
		OrgID: created.OrganizationID,
		BuID:  created.BusinessUnitID,
	}, actor)

	return created, nil
}

func (s *service) UpdatePermit(
	ctx context.Context,
	entity *permit.Permit,
	actor *services.RequestActor,
) (*permit.Permit, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Read the row before overwriting it so the audit entry carries a real diff.
	// "Permit updated" without a before-state is close to useless when the
	// question months later is whether an expiry was extended after the fact.
	// A read failure loses the diff, not the write or the audit entry.
	var previous *permit.Permit
	if existing, readErr := s.permitRepo.GetByID(ctx, &repositories.GetPermitByIDRequest{
		PermitID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}); readErr != nil {
		s.l.Warn("failed to read permit before update; audit entry will carry no diff",
			zap.String("permitId", entity.ID.String()), zap.Error(readErr))
	} else {
		previous = existing
	}

	updated, err := s.permitRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logPermitAction(&permitAuditParams{
		ResourceID:     updated.ID.String(),
		ShipmentID:     updated.ShipmentID,
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		Operation:      permission.OpUpdate,
		Actor:          actor.AuditActorOrSystem(),
		Previous:       previous,
		Current:        updated,
		Comment:        "Permit updated",
	})

	s.resync(ctx, updated.ShipmentID, pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}, actor)

	return updated, nil
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

	waived, err := s.permitRepo.WaiveRequirement(ctx, &repositories.WaiveRequirementRepoRequest{
		TenantInfo:    req.TenantInfo,
		RequirementID: req.RequirementID,
		WaivedByID:    req.WaivedByID,
		Reason:        req.Reason,
	})
	if err != nil {
		return nil, err
	}

	// OpApprove, not OpUpdate: the waiver is an authorization act, and the
	// reason is the record of why the risk was accepted.
	s.logPermitAction(&permitAuditParams{
		ResourceID:     waived.ID.String(),
		ShipmentID:     waived.ShipmentID,
		OrganizationID: waived.OrganizationID,
		BusinessUnitID: waived.BusinessUnitID,
		Operation:      permission.OpApprove,
		Actor:          req.Actor.AuditActorOrSystem(),
		Current:        waived,
		Comment:        "Permit requirement waived: " + req.Reason,
	})

	s.resync(ctx, waived.ShipmentID, req.TenantInfo, req.Actor)

	return waived, nil
}
