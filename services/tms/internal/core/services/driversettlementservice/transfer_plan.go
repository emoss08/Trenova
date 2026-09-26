package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/shopspring/decimal"
)

func errHoldReasonRequired() error {
	return errortypes.NewValidationError(
		"reason",
		errortypes.ErrRequired,
		"A hold reason is required so the driver knows why pay was deferred",
	)
}

func PlanHoldPayEvent(event *driversettlement.PayEvent, reason string) (bool, error) {
	if reason == "" {
		return false, errHoldReasonRequired()
	}
	if event.Status != driversettlement.PayEventStatusAccrued {
		return false, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only accrued, unsettled pay events can be held",
		)
	}
	if event.OnHold {
		return false, nil
	}
	event.OnHold = true
	event.HoldReason = reason
	return true, nil
}

func PlanReleasePayEvent(event *driversettlement.PayEvent) bool {
	if !event.OnHold {
		return false
	}
	event.OnHold = false
	event.HoldReason = ""
	return true
}

func checkAttachable(
	entity *driversettlement.Settlement,
	event *driversettlement.PayEvent,
) error {
	if event.Status != driversettlement.PayEventStatusAccrued {
		return errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Pay event {0} is not accrued; only unsettled events can be added", event.ProNumber,
		)
	}
	if event.WorkerID != entity.WorkerID {
		return errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Pay event {0} belongs to a different driver", event.ProNumber,
		)
	}
	return nil
}

func AppendPayEventLines(entity *driversettlement.Settlement, event *driversettlement.PayEvent) {
	eventRef := event.ID
	shipmentID := event.ShipmentID
	for _, comp := range event.Components {
		entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
			Category:      driversettlement.LineCategoryEarning,
			ComponentKind: comp.Kind,
			Method:        comp.Method,
			Description:   comp.Description,
			Quantity:      comp.Quantity,
			Rate:          comp.Rate,
			AmountMinor:   comp.AmountMinor,
			ShipmentID:    &shipmentID,
			MoveID:        event.MoveID,
			PayEventID:    &eventRef,
			ProNumber:     event.ProNumber,
		})
	}
	entity.TotalMiles = entity.TotalMiles.Add(event.TotalMiles)
}

func FinishAttach(entity *driversettlement.Settlement) {
	entity.ShipmentCount = countSettlementShipments(entity.Lines)
	entity.SyncTotals()
}

func checkDraftForEvents(entity *driversettlement.Settlement, verb string) error {
	if entity.Status != driversettlement.StatusDraft {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Pay events can only be "+verb+" draft settlements",
		)
	}
	return nil
}

func PlanDetachPayEvent(
	entity *driversettlement.Settlement,
	event *driversettlement.PayEvent,
) error {
	if err := checkDraftForEvents(entity, "removed from"); err != nil {
		return err
	}
	found := false
	remaining := make([]*driversettlement.SettlementLine, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.PayEventID != nil && *line.PayEventID == event.ID {
			found = true
			continue
		}
		remaining = append(remaining, line)
	}
	if !found {
		return errortypes.NewValidationError(
			"payEventId",
			errortypes.ErrInvalid,
			"Pay event is not part of this settlement",
		)
	}

	entity.Lines = remaining
	entity.TotalMiles = entity.TotalMiles.Sub(event.TotalMiles)
	if entity.TotalMiles.IsNegative() {
		entity.TotalMiles = decimal.Zero
	}
	entity.ShipmentCount = countSettlementShipments(remaining)
	entity.SyncTotals()
	return nil
}

type PayEventPlan struct {
	Before     *driversettlement.PayEvent
	After      *driversettlement.PayEvent
	Changed    bool
	AutoAttach bool
	Refusal    error
}

func (s *Service) PlanPayEventHold(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventID pulid.ID,
	reason string,
	hold bool,
) (*PayEventPlan, error) {
	event, err := s.payEventRepo.GetByID(ctx, repositories.GetPayEventByIDRequest{
		ID:         eventID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	after := *event
	plan := &PayEventPlan{Before: event, After: &after}
	if hold {
		plan.Changed, plan.Refusal = PlanHoldPayEvent(plan.After, reason)
		return plan, nil
	}

	plan.Changed = PlanReleasePayEvent(plan.After)
	if !plan.Changed {
		return plan, nil
	}
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	plan.AutoAttach = control.AutoAttachAccruals
	return plan, nil
}

type EventTransferRequest struct {
	TenantInfo   pagination.TenantInfo
	SettlementID pulid.ID
	EventIDs     []pulid.ID
	Detach       bool
}

func (s *Service) PlanEventTransfer(
	ctx context.Context,
	req *EventTransferRequest,
) (*ActionPlan, error) {
	entity, err := s.getForUpdate(ctx, req.TenantInfo, req.SettlementID)
	if err != nil {
		return nil, err
	}
	plan := &ActionPlan{Before: entity, After: CloneSettlement(entity)}
	if len(req.EventIDs) == 0 {
		plan.Refusal = errNoPayEventsSelected()
		return plan, nil
	}

	for _, eventID := range sliceutils.Dedupe(req.EventIDs) {
		event, getErr := s.payEventRepo.GetByID(
			ctx,
			repositories.GetPayEventByIDRequest{ID: eventID, TenantInfo: req.TenantInfo},
		)
		if getErr != nil {
			if settlementshared.IsRefusal(getErr) {
				plan.Refusal = getErr
				return plan, nil
			}
			return nil, getErr
		}
		if req.Detach {
			if plan.Refusal = PlanDetachPayEvent(plan.After, event); plan.Refused() {
				return plan, nil
			}
			continue
		}
		if plan.Refusal = checkDraftForEvents(plan.After, "added to"); plan.Refused() {
			return plan, nil
		}
		if plan.Refusal = checkAttachable(plan.After, event); plan.Refused() {
			return plan, nil
		}
		AppendPayEventLines(plan.After, event)
	}
	if !req.Detach {
		FinishAttach(plan.After)
	}
	return plan, nil
}

func errNoPayEventsSelected() error {
	return errortypes.NewValidationError(
		"payEventIds",
		errortypes.ErrRequired,
		"Select at least one pay event to add",
	)
}
