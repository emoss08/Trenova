package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

//nolint:exhaustive // canceled shipments never accrue pay
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
		if err := s.VoidShipmentPayEvents(ctx, tenantInfo, updated.ID,
			"Shipment canceled"); err != nil {
			s.l.Error("failed to void pay events for canceled shipment",
				zap.Error(err), zap.String("shipmentId", updated.ID.String()))
		}
		return nil
	}

	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Error("failed to load settlement control for pay accrual", zap.Error(err))
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
		s.l.Error("failed to accrue driver pay for shipment",
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
		if err := s.voidMovePayEvents(ctx, tenantInfo, move.ID, reason); err != nil {
			s.l.Error("failed to void pay events for move",
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
		s.l.Error("failed to load settlement control for move pay accrual", zap.Error(err))
		return nil
	}
	if control.PayTrigger != tenant.PayTriggerMoveCompleted {
		return nil
	}

	if err = s.AccrueMove(ctx, tenantInfo, move.ShipmentID, move.ID); err != nil {
		s.l.Error("failed to accrue driver pay for completed move",
			zap.Error(err), zap.String("moveId", move.ID.String()))
	}
	return nil
}

func (s *Service) AccrueMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID, moveID pulid.ID,
) error {
	return s.accrueShipmentMoves(ctx, tenantInfo, shipmentID, &moveID)
}

func (s *Service) voidMovePayEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	reason string,
) error {
	events, err := s.payEventRepo.ListByMove(ctx, tenantInfo, moveID)
	if err != nil {
		return err
	}
	now := timeutils.NowUnix()
	for _, event := range events {
		if event.Status != driversettlement.PayEventStatusAccrued {
			continue
		}
		event.Status = driversettlement.PayEventStatusVoided
		event.VoidedAt = &now
		event.VoidReason = reason
		if _, err = s.payEventRepo.Update(ctx, event); err != nil {
			return err
		}
		s.publishPayEventInvalidation(ctx, event, permission.OpCancel, pulid.Nil)
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

	totalDistance := decimal.Zero
	for _, move := range sp.Moves {
		totalDistance = totalDistance.Add(moveDistance(move))
	}
	hasHazmat := shipmentHasHazmat(sp)
	fuelSurcharge := shipmentFuelSurcharge(sp)
	eventDate := timeutils.NowUnix()
	if sp.ActualDeliveryDate != nil {
		eventDate = *sp.ActualDeliveryDate
	}

	accruedWorkers := make(map[pulid.ID]struct{}, 2)
	for _, move := range sp.Moves {
		if move == nil || move.Assignment == nil ||
			move.Status == shipment.MoveStatusCanceled {
			continue
		}
		if onlyMoveID != nil && move.ID != *onlyMoveID {
			continue
		}
		workers := make([]pulid.ID, 0, 2)
		if move.Assignment.PrimaryWorkerID != nil &&
			!move.Assignment.PrimaryWorkerID.IsNil() {
			workers = append(workers, *move.Assignment.PrimaryWorkerID)
		}
		if move.Assignment.SecondaryWorkerID != nil &&
			!move.Assignment.SecondaryWorkerID.IsNil() {
			workers = append(workers, *move.Assignment.SecondaryWorkerID)
		}

		moveEventDate := eventDate
		if onlyMoveID != nil {
			moveEventDate = timeutils.NowUnix()
		}
		for _, workerID := range workers {
			if err = s.accrueWorkerMove(ctx, &accrueWorkerMoveParams{
				TenantInfo:    tenantInfo,
				WorkerID:      workerID,
				Shipment:      sp,
				Move:          move,
				TotalDistance: totalDistance,
				HasHazmat:     hasHazmat,
				FuelSurcharge: fuelSurcharge,
				EventDate:     moveEventDate,
			}); err != nil {
				s.l.Error("failed to accrue pay event for worker move",
					zap.Error(err),
					zap.String("workerId", workerID.String()),
					zap.String("moveId", move.ID.String()))
				continue
			}
			accruedWorkers[workerID] = struct{}{}
		}
	}

	s.autoAttachForWorkers(ctx, tenantInfo, accruedWorkers)
	return nil
}

func (s *Service) autoAttachForWorkers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workers map[pulid.ID]struct{},
) {
	if len(workers) == 0 {
		return
	}
	control, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Error("failed to load settlement control for auto-attach", zap.Error(err))
		return
	}
	if !control.AutoAttachAccruals {
		return
	}
	for workerID := range workers {
		if err = s.refreshOpenDraftForWorker(ctx, tenantInfo, workerID); err != nil {
			s.l.Error("failed to auto-attach accruals to open draft settlement",
				zap.Error(err), zap.String("workerId", workerID.String()))
		}
	}
}

type accrueWorkerMoveParams struct {
	TenantInfo    pagination.TenantInfo
	WorkerID      pulid.ID
	Shipment      *shipment.Shipment
	Move          *shipment.ShipmentMove
	TotalDistance decimal.Decimal
	HasHazmat     bool
	FuelSurcharge decimal.Decimal
	EventDate     int64
}

func (s *Service) accrueWorkerMove(
	ctx context.Context,
	params *accrueWorkerMoveParams,
) error {
	assignment, err := s.assignmentRepo.GetEffectiveForWorker(
		ctx,
		repositories.GetWorkerPayAssignmentRequest{
			TenantInfo: params.TenantInfo,
			WorkerID:   params.WorkerID,
			AsOf:       params.EventDate,
		},
	)
	if err != nil {
		s.l.Warn("worker has no effective pay assignment; skipping pay accrual",
			zap.String("workerId", params.WorkerID.String()),
			zap.String("shipmentId", params.Shipment.ID.String()))
		return nil //nolint:nilerr // a worker without a pay assignment simply doesn't accrue
	}
	if assignment.PayProfile == nil {
		return errortypes.NewValidationError(
			"payProfileId",
			errortypes.ErrInvalid,
			"Pay assignment is missing its pay profile",
		)
	}

	components, gross := computeMovePay(&moveCalcInput{
		Profile:           assignment.PayProfile,
		SplitPercent:      assignment.SplitPercent,
		RateOverrides:     assignment.RateOverrides,
		Shipment:          params.Shipment,
		Move:              params.Move,
		TotalTripDistance: params.TotalDistance,
		MoveCount:         len(params.Shipment.Moves),
		HasHazmat:         params.HasHazmat,
		FuelSurcharge:     params.FuelSurcharge,
	})

	key := payEventIdempotencyKey(params.WorkerID, params.Move.ID)
	existing, err := s.payEventRepo.GetByIdempotencyKey(ctx, params.TenantInfo, key)
	if err == nil && existing != nil {
		switch existing.Status {
		case driversettlement.PayEventStatusSettled:
			return nil
		case driversettlement.PayEventStatusAccrued, driversettlement.PayEventStatusVoided:
			existing.Status = driversettlement.PayEventStatusAccrued
			existing.GrossAmountMinor = gross
			existing.TotalMiles = moveDistance(params.Move)
			existing.Components = components
			existing.VoidedAt = nil
			existing.VoidReason = ""
			refreshed, updateErr := s.payEventRepo.Update(ctx, existing)
			if updateErr != nil {
				return updateErr
			}
			s.publishPayEventInvalidation(ctx, refreshed, permission.OpUpdate, pulid.Nil)
			return nil
		}
		return nil
	}

	profileID := assignment.PayProfileID
	moveID := params.Move.ID
	assignmentID := assignment.ID
	event := &driversettlement.PayEvent{
		OrganizationID:   params.TenantInfo.OrgID,
		BusinessUnitID:   params.TenantInfo.BuID,
		WorkerID:         params.WorkerID,
		ShipmentID:       params.Shipment.ID,
		MoveID:           &moveID,
		AssignmentID:     &assignmentID,
		PayProfileID:     &profileID,
		IdempotencyKey:   key,
		Status:           driversettlement.PayEventStatusAccrued,
		EventDate:        params.EventDate,
		GrossAmountMinor: gross,
		TotalMiles:       moveDistance(params.Move),
		CurrencyCode:     assignment.PayProfile.CurrencyCode,
		Components:       components,
		ProNumber:        params.Shipment.ProNumber,
	}

	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	created, err := s.payEventRepo.Create(ctx, event)
	if err != nil {
		return err
	}
	s.publishPayEventInvalidation(ctx, created, permission.OpCreate, pulid.Nil)
	return nil
}

func (s *Service) VoidShipmentPayEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	reason string,
) error {
	events, err := s.payEventRepo.ListByShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return err
	}
	now := timeutils.NowUnix()
	for _, event := range events {
		if event.Status != driversettlement.PayEventStatusAccrued {
			continue
		}
		event.Status = driversettlement.PayEventStatusVoided
		event.VoidedAt = &now
		event.VoidReason = reason
		if _, err = s.payEventRepo.Update(ctx, event); err != nil {
			return err
		}
		s.publishPayEventInvalidation(ctx, event, permission.OpCancel, pulid.Nil)
	}
	return nil
}

func classificationForWorker(profile *driverpay.PayProfile) driverpay.PayeeClassification {
	if profile != nil {
		return profile.Classification
	}
	return driverpay.PayeeClassificationCompanyDriver
}
