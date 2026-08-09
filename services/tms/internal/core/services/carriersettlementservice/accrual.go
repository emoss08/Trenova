package carriersettlementservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

//nolint:exhaustive // canceled shipments never accrue cost
var shipmentStatusRank = map[shipment.Status]int{
	shipment.StatusNew:                0,
	shipment.StatusPartiallyAssigned:  1,
	shipment.StatusAssigned:           2,
	shipment.StatusInTransit:          3,
	shipment.StatusDelayed:            3,
	shipment.StatusPartiallyCompleted: 4,
	shipment.StatusCompleted:          5,
	shipment.StatusReadyToInvoice:     6,
	shipment.StatusInvoiced:           7,
}

func payTriggerRank(trigger tenant.PayTrigger) int {
	switch trigger {
	case tenant.PayTriggerMoveCompleted, tenant.PayTriggerShipmentDelivered:
		return shipmentStatusRank[shipment.StatusCompleted]
	case tenant.PayTriggerPODReceived:
		return shipmentStatusRank[shipment.StatusReadyToInvoice]
	case tenant.PayTriggerShipmentInvoiced:
		return shipmentStatusRank[shipment.StatusInvoiced]
	default:
		return shipmentStatusRank[shipment.StatusCompleted]
	}
}

func (s *Service) AfterShipmentUpdate(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
	_ *serviceports.RequestActor,
) error {
	if updated == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}

	if updated.Status == shipment.StatusCanceled {
		if err := s.VoidShipmentCostEvents(ctx, tenantInfo, updated.ID,
			"Shipment canceled"); err != nil {
			s.l.Error("failed to void carrier cost events for canceled shipment",
				zap.Error(err), zap.String("shipmentId", updated.ID.String()))
		}
		return nil
	}

	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Error("failed to load carrier settlement control for cost accrual", zap.Error(err))
		return nil
	}

	triggerRank := payTriggerRank(control.PayTrigger)
	updatedRank, ok := shipmentStatusRank[updated.Status]
	if !ok || updatedRank < triggerRank {
		return nil
	}
	if original != nil {
		if originalRank, known := shipmentStatusRank[original.Status]; known &&
			originalRank >= triggerRank {
			return nil
		}
	}

	if err = s.AccrueShipment(ctx, tenantInfo, updated.ID); err != nil {
		s.l.Error("failed to accrue carrier cost for shipment",
			zap.Error(err), zap.String("shipmentId", updated.ID.String()))
	}
	return nil
}

func (s *Service) AfterMoveStatusChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	previous shipment.MoveStatus,
) error {
	if move == nil {
		return nil
	}

	if move.Status != shipment.MoveStatusCompleted &&
		(previous == shipment.MoveStatusCompleted ||
			move.Status == shipment.MoveStatusCanceled) {
		reason := "Move canceled"
		if move.Status != shipment.MoveStatusCanceled {
			reason = "Move status reverted from completed"
		}
		if err := s.voidMoveCostEvents(ctx, tenantInfo, move.ID, reason); err != nil {
			s.l.Error("failed to void carrier cost events for move",
				zap.Error(err), zap.String("moveId", move.ID.String()))
		}
		return nil
	}

	if move.Status != shipment.MoveStatusCompleted ||
		previous == shipment.MoveStatusCompleted {
		return nil
	}

	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Error("failed to load carrier settlement control for move cost accrual",
			zap.Error(err))
		return nil
	}
	if control.PayTrigger != tenant.PayTriggerMoveCompleted {
		return nil
	}

	if err = s.AccrueMove(ctx, tenantInfo, move.ShipmentID, move.ID); err != nil {
		s.l.Error("failed to accrue carrier cost for completed move",
			zap.Error(err), zap.String("moveId", move.ID.String()))
	}
	return nil
}

func (s *Service) AccrueShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	return s.accrueShipmentMoves(ctx, tenantInfo, shipmentID, nil)
}

func (s *Service) AccrueMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID, moveID pulid.ID,
) error {
	return s.accrueShipmentMoves(ctx, tenantInfo, shipmentID, &moveID)
}

func (s *Service) accrueShipmentMoves(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	onlyMoveID *pulid.ID,
) error {
	sp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	eventDate := timeutils.NowUnix()
	if sp.ActualDeliveryDate != nil {
		eventDate = *sp.ActualDeliveryDate
	}

	for _, move := range sp.Moves {
		if move == nil || move.Status == shipment.MoveStatusCanceled ||
			!move.IsCarrierCovered() {
			continue
		}
		if onlyMoveID != nil && move.ID != *onlyMoveID {
			continue
		}
		moveEventDate := eventDate
		if onlyMoveID != nil {
			moveEventDate = timeutils.NowUnix()
		}
		if err = s.accrueCarrierMove(ctx, tenantInfo, sp, move, moveEventDate); err != nil {
			s.l.Error("failed to accrue carrier cost events for move",
				zap.Error(err), zap.String("moveId", move.ID.String()))
		}
	}
	return nil
}

func (s *Service) accrueCarrierMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sp *shipment.Shipment,
	move *shipment.ShipmentMove,
	eventDate int64,
) error {
	assignment, err := s.assignmentRepo.GetActiveByMoveID(ctx, tenantInfo, move.ID)
	if err != nil {
		return err
	}
	if assignment == nil || !assignment.IsActive() {
		return nil
	}

	inputs := BuildAssignmentCostEventInputs(assignment)
	activeKeys := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		activeKeys[input.IdempotencyKey] = struct{}{}
		if err = s.upsertCostEvent(ctx, &upsertCostEventParams{
			TenantInfo: tenantInfo,
			Shipment:   sp,
			Move:       move,
			Assignment: assignment,
			Input:      input,
			EventDate:  eventDate,
		}); err != nil {
			return err
		}
	}

	return s.voidStaleMoveCostEvents(ctx, tenantInfo, move.ID, activeKeys)
}

type upsertCostEventParams struct {
	TenantInfo pagination.TenantInfo
	Shipment   *shipment.Shipment
	Move       *shipment.ShipmentMove
	Assignment *shipment.CarrierAssignment
	Input      CostEventInput
	EventDate  int64
}

func (s *Service) upsertCostEvent(
	ctx context.Context,
	params *upsertCostEventParams,
) error {
	existing, err := s.costEventRepo.GetByIdempotencyKey(
		ctx,
		params.TenantInfo,
		params.Input.IdempotencyKey,
	)
	if err == nil && existing != nil {
		switch existing.Status {
		case carriersettlement.CostEventStatusSettled,
			carriersettlement.CostEventStatusAttached:
			return nil
		case carriersettlement.CostEventStatusPending,
			carriersettlement.CostEventStatusVoided:
			existing.Status = carriersettlement.CostEventStatusPending
			existing.AmountMinor = params.Input.AmountMinor
			existing.Description = params.Input.Description
			existing.ProNumber = costEventProNumber(params.Shipment, params.Assignment)
			existing.AssignmentVersion = params.Assignment.Version
			existing.VoidedAt = nil
			existing.VoidReason = ""
			refreshed, updateErr := s.costEventRepo.Update(ctx, existing)
			if updateErr != nil {
				return updateErr
			}
			s.publishCostEventInvalidation(ctx, refreshed, permission.OpUpdate, pulid.Nil)
			return nil
		}
		return nil
	}

	moveID := params.Move.ID
	shipmentID := params.Shipment.ID
	assignmentID := params.Assignment.ID
	event := &carriersettlement.CostEvent{
		OrganizationID:      params.TenantInfo.OrgID,
		BusinessUnitID:      params.TenantInfo.BuID,
		CarrierID:           params.Assignment.CarrierID,
		CarrierAssignmentID: &assignmentID,
		ShipmentID:          &shipmentID,
		MoveID:              &moveID,
		EventType:           params.Input.EventType,
		Status:              carriersettlement.CostEventStatusPending,
		IdempotencyKey:      params.Input.IdempotencyKey,
		EventDate:           params.EventDate,
		AmountMinor:         params.Input.AmountMinor,
		CurrencyCode:        params.Assignment.CurrencyCode,
		Description:         params.Input.Description,
		ProNumber:           costEventProNumber(params.Shipment, params.Assignment),
		AssignmentVersion:   params.Assignment.Version,
	}

	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	created, err := s.costEventRepo.Create(ctx, event)
	if err != nil {
		return err
	}
	s.publishCostEventInvalidation(ctx, created, permission.OpCreate, pulid.Nil)
	return nil
}

func costEventProNumber(sp *shipment.Shipment, assignment *shipment.CarrierAssignment) string {
	if assignment != nil && assignment.ProNumber != "" {
		return assignment.ProNumber
	}
	if sp != nil {
		return sp.ProNumber
	}
	return ""
}

// voidStaleMoveCostEvents voids pending assignment-derived events whose
// idempotency key is no longer produced by the assignment — e.g. a removed
// accessorial line or a fuel surcharge zeroed out on renegotiation.
func (s *Service) voidStaleMoveCostEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	activeKeys map[string]struct{},
) error {
	events, err := s.costEventRepo.ListByMove(ctx, tenantInfo, moveID)
	if err != nil {
		return err
	}
	now := timeutils.NowUnix()
	for _, event := range events {
		if event.Status != carriersettlement.CostEventStatusPending ||
			event.EventType == carriersettlement.CostEventTypeAdjustment ||
			!strings.HasPrefix(event.IdempotencyKey, "carrier-cost:") {
			continue
		}
		if _, active := activeKeys[event.IdempotencyKey]; active {
			continue
		}
		event.Status = carriersettlement.CostEventStatusVoided
		event.VoidedAt = &now
		event.VoidReason = "Assignment cost line removed"
		if _, err = s.costEventRepo.Update(ctx, event); err != nil {
			return err
		}
		s.publishCostEventInvalidation(ctx, event, permission.OpCancel, pulid.Nil)
	}
	return nil
}

func (s *Service) voidMoveCostEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	reason string,
) error {
	events, err := s.costEventRepo.ListByMove(ctx, tenantInfo, moveID)
	if err != nil {
		return err
	}
	return s.voidPendingEvents(ctx, events, reason)
}

func (s *Service) VoidShipmentCostEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	reason string,
) error {
	events, err := s.costEventRepo.ListByShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return err
	}
	return s.voidPendingEvents(ctx, events, reason)
}

func (s *Service) voidPendingEvents(
	ctx context.Context,
	events []*carriersettlement.CostEvent,
	reason string,
) error {
	now := timeutils.NowUnix()
	for _, event := range events {
		if event.Status != carriersettlement.CostEventStatusPending {
			continue
		}
		event.Status = carriersettlement.CostEventStatusVoided
		event.VoidedAt = &now
		event.VoidReason = reason
		if _, err := s.costEventRepo.Update(ctx, event); err != nil {
			return err
		}
		s.publishCostEventInvalidation(ctx, event, permission.OpCancel, pulid.Nil)
	}
	return nil
}
