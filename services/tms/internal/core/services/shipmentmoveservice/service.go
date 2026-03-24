package shipmentmoveservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/equipmentavailabilityhelper"
	"github.com/emoss08/trenova/internal/core/services/equipmentcontinuityhelper"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             ports.DBConnection
	Repo           repositories.ShipmentMoveRepository
	AssignmentRepo repositories.AssignmentRepository
	ShipmentRepo   repositories.ShipmentRepository
	HoldRepo       repositories.ShipmentHoldRepository
	ControlRepo    repositories.ShipmentControlRepository
	ContinuityRepo repositories.EquipmentContinuityRepository
	Coordinator    *shipmentstate.Coordinator
	Commercial     *shipmentcommercial.Calculator
}

type service struct {
	l              *zap.Logger
	db             ports.DBConnection
	repo           repositories.ShipmentMoveRepository
	assignmentRepo repositories.AssignmentRepository
	shipmentRepo   repositories.ShipmentRepository
	holdRepo       repositories.ShipmentHoldRepository
	controlRepo    repositories.ShipmentControlRepository
	continuityRepo repositories.EquipmentContinuityRepository
	coordinator    *shipmentstate.Coordinator
	commercial     *shipmentcommercial.Calculator
}

//nolint:gocritic // service constructor
func New(p Params) portservices.ShipmentMoveService {
	return &service{
		l:              p.Logger.Named("service.shipment-move"),
		db:             p.DB,
		repo:           p.Repo,
		assignmentRepo: p.AssignmentRepo,
		shipmentRepo:   p.ShipmentRepo,
		holdRepo:       p.HoldRepo,
		controlRepo:    p.ControlRepo,
		continuityRepo: p.ContinuityRepo,
		coordinator:    p.Coordinator,
		commercial:     p.Commercial,
	}
}

func (s *service) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var updatedMove *shipment.ShipmentMove
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, err := s.repo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: false,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}

		if !shipmentstate.CanTransitionMoveStatus(move.Status, req.Status) {
			return errortypes.NewBusinessError(
				fmt.Sprintf(
					"Move status transition from %s to %s is not allowed",
					move.Status,
					req.Status,
				),
			).WithParam("moveId", req.MoveID.String())
		}
		if req.Status == shipment.MoveStatusInTransit {
			if err = s.ensureEquipmentAvailableForProgress(
				txCtx,
				req.TenantInfo,
				move.ID,
			); err != nil {
				return err
			}
		}
		if err = s.ensureNoDeliveryHold(txCtx, move.ShipmentID, req.TenantInfo, req.Status); err != nil {
			return err
		}

		updatedMove, err = s.repo.UpdateStatus(txCtx, req)
		if err != nil {
			return err
		}
		if err = s.advanceEquipmentContinuityForMove(txCtx, req.TenantInfo, updatedMove, req.Status); err != nil {
			return err
		}

		return s.refreshShipmentState(txCtx, move.ShipmentID, req.TenantInfo)
	})
	if err != nil {
		return nil, err
	}

	return updatedMove, nil
}

func (s *service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	seen := make(map[pulid.ID]struct{}, len(req.MoveIDs))
	for _, moveID := range req.MoveIDs {
		if _, ok := seen[moveID]; ok {
			return nil, errortypes.NewBusinessError("Move IDs must be unique").
				WithParam("moveId", moveID.String())
		}
		seen[moveID] = struct{}{}
	}

	var updatedMoves []*shipment.ShipmentMove
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		shipmentIDs := make(map[pulid.ID]struct{}, len(req.MoveIDs))
		seenTractors := make(map[pulid.ID]pulid.ID, len(req.MoveIDs))
		seenTrailers := make(map[pulid.ID]pulid.ID, len(req.MoveIDs))

		for _, moveID := range req.MoveIDs {
			move, err := s.repo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
				MoveID:            moveID,
				TenantInfo:        req.TenantInfo,
				ExpandMoveDetails: false,
				ForUpdate:         true,
			})
			if err != nil {
				return err
			}

			if !shipmentstate.CanTransitionMoveStatus(move.Status, req.Status) {
				return errortypes.NewBusinessError(
					fmt.Sprintf(
						"Move status transition from %s to %s is not allowed",
						move.Status,
						req.Status,
					),
				).WithParam("moveId", moveID.String())
			}
			if err = s.ensureEquipmentAvailableForProgressBulk(
				txCtx,
				req.TenantInfo,
				move.ID,
				req.Status,
				seenTractors,
				seenTrailers,
			); err != nil {
				return err
			}
			if err = s.ensureNoDeliveryHold(txCtx, move.ShipmentID, req.TenantInfo, req.Status); err != nil {
				return err
			}

			shipmentIDs[move.ShipmentID] = struct{}{}
		}

		var err error
		updatedMoves, err = s.repo.BulkUpdateStatus(txCtx, req)
		if err != nil {
			return err
		}
		for _, updatedMove := range updatedMoves {
			if err = s.advanceEquipmentContinuityForMove(txCtx, req.TenantInfo, updatedMove, req.Status); err != nil {
				return err
			}
		}

		for shipmentID := range shipmentIDs {
			if err = s.refreshShipmentState(txCtx, shipmentID, req.TenantInfo); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMoves, nil
}

func (s *service) SplitMove(
	ctx context.Context,
	req *repositories.SplitMoveRequest,
) (*repositories.SplitMoveResponse, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var response *repositories.SplitMoveResponse
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		lockedMove, err := s.repo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: false,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}

		move, err := s.repo.GetByID(txCtx, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
		})
		if err != nil {
			return err
		}

		if err = validateSplitRequest(req, move); err != nil {
			return err
		}

		response, err = s.repo.SplitMove(txCtx, req)
		if err != nil {
			return err
		}

		return s.refreshShipmentState(txCtx, lockedMove.ShipmentID, req.TenantInfo)
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func validateSplitRequest(req *repositories.SplitMoveRequest, move *shipment.ShipmentMove) error {
	if len(move.Stops) != 2 {
		return errortypes.NewBusinessError("Only simple two-stop moves can be split").
			WithParam("moveId", move.ID.String())
	}

	if move.Status != shipment.MoveStatusNew && move.Status != shipment.MoveStatusAssigned {
		return errortypes.NewBusinessError("Only new or assigned moves can be split").
			WithParam("moveId", move.ID.String())
	}

	originStop := move.Stops[0]
	deliveryStop := move.Stops[1]
	if originStop.Type != shipment.StopTypePickup ||
		deliveryStop.Type != shipment.StopTypeDelivery {
		return errortypes.NewBusinessError("Only pickup-to-delivery moves can be split").
			WithParam("moveId", move.ID.String())
	}

	if deliveryStop.LocationID == req.NewDeliveryLocationID {
		return errortypes.NewBusinessError("New delivery location must differ from the current delivery location").
			WithParam("moveId", move.ID.String())
	}

	splitPickupDeadline := req.SplitPickupTimes.ScheduledWindowStart
	if req.SplitPickupTimes.ScheduledWindowEnd != nil {
		splitPickupDeadline = *req.SplitPickupTimes.ScheduledWindowEnd
		if splitPickupDeadline < req.SplitPickupTimes.ScheduledWindowStart {
			return errortypes.NewBusinessError(
				"Split pickup scheduled window end must be greater than or equal to the scheduled window start",
			).WithParam("moveId", move.ID.String())
		}
	}

	newDeliveryDeadline := req.NewDeliveryTimes.ScheduledWindowStart
	if req.NewDeliveryTimes.ScheduledWindowEnd != nil {
		newDeliveryDeadline = *req.NewDeliveryTimes.ScheduledWindowEnd
		if newDeliveryDeadline < req.NewDeliveryTimes.ScheduledWindowStart {
			return errortypes.NewBusinessError(
				"New delivery scheduled window end must be greater than or equal to the scheduled window start",
			).WithParam("moveId", move.ID.String())
		}
	}

	if req.SplitPickupTimes.ScheduledWindowStart < deliveryStop.EffectiveScheduledWindowEnd() {
		return errortypes.NewBusinessError(
			"Split pickup scheduled window start must occur after the original delivery window ends",
		).WithParam("moveId", move.ID.String())
	}

	if req.NewDeliveryTimes.ScheduledWindowStart < splitPickupDeadline {
		return errortypes.NewBusinessError(
			"New delivery scheduled window start must occur after the split pickup window ends",
		).WithParam("moveId", move.ID.String())
	}

	return nil
}

func (s *service) refreshShipmentState(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	control, err := s.controlRepo.Get(
		ctx,
		repositories.GetShipmentControlRequest{TenantInfo: tenantInfo},
	)
	if err != nil {
		return err
	}

	s.coordinator.RefreshShipmentStateWithDelayThreshold(
		entity,
		resolveDelayThresholdMinutes(control),
	)
	if err = s.commercial.Recalculate(ctx, entity, control, pulid.Nil); err != nil {
		return err
	}
	_, err = s.shipmentRepo.UpdateDerivedState(ctx, entity)
	return err
}

func resolveDelayThresholdMinutes(control *tenant.ShipmentControl) int16 {
	if control == nil || !control.AutoDelayShipments {
		return shipmentstate.DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return shipmentstate.ResolveDelayThresholdMinutes(0)
	}

	return shipmentstate.ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}

func (s *service) ensureNoDeliveryHold(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipment.MoveStatus,
) error {
	if status != shipment.MoveStatusCompleted {
		return nil
	}

	hasHold, err := s.holdRepo.HasActiveDeliveryHold(ctx, &repositories.ActiveShipmentHoldRequest{
		ShipmentID: shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if hasHold {
		return errortypes.NewBusinessError("Shipment has an active delivery-blocking hold").
			WithParam("shipmentId", shipmentID.String())
	}

	return nil
}

func (s *service) ensureEquipmentAvailableForProgress(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) error {
	assignment, err := s.assignmentRepo.GetByMoveID(ctx, tenantInfo, moveID)
	if err != nil || assignment == nil {
		return err
	}

	return s.ensureAssignmentEquipmentAvailable(ctx, tenantInfo, assignment, moveID)
}

func (s *service) ensureEquipmentAvailableForProgressBulk(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	status shipment.MoveStatus,
	seenTractors map[pulid.ID]pulid.ID,
	seenTrailers map[pulid.ID]pulid.ID,
) error {
	if status != shipment.MoveStatusInTransit {
		return nil
	}

	assignment, err := s.assignmentRepo.GetByMoveID(ctx, tenantInfo, moveID)
	if err != nil || assignment == nil {
		return err
	}

	if assignment.TractorID != nil {
		if priorMoveID, ok := seenTractors[*assignment.TractorID]; ok {
			return errortypes.NewBusinessError("Tractor is currently in progress on another move").
				WithParam("tractorId", assignment.TractorID.String()).
				WithParam("shipmentMoveId", priorMoveID.String())
		}
		seenTractors[*assignment.TractorID] = moveID
	}

	if assignment.TrailerID != nil {
		if priorMoveID, ok := seenTrailers[*assignment.TrailerID]; ok {
			return errortypes.NewBusinessError("Trailer is currently in progress on another move").
				WithParam("trailerId", assignment.TrailerID.String()).
				WithParam("shipmentMoveId", priorMoveID.String())
		}
		seenTrailers[*assignment.TrailerID] = moveID
	}

	return s.ensureAssignmentEquipmentAvailable(ctx, tenantInfo, assignment, moveID)
}

func (s *service) ensureAssignmentEquipmentAvailable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignment *shipment.Assignment,
	moveID pulid.ID,
) error {
	return equipmentavailabilityhelper.EnsureAssignmentEquipmentAvailable(
		ctx,
		s.assignmentRepo,
		tenantInfo,
		assignment,
		moveID,
	)
}

func (s *service) advanceEquipmentContinuityForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	status shipment.MoveStatus,
) error {
	if status != shipment.MoveStatusCompleted {
		return nil
	}

	return equipmentcontinuityhelper.AdvanceForCompletedMove(
		ctx,
		s.continuityRepo,
		tenantInfo,
		move,
	)
}
