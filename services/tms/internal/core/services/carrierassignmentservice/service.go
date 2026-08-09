package carrierassignmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                ports.DBConnection
	Repo              repositories.CarrierAssignmentRepository
	AssignmentRepo    repositories.AssignmentRepository
	CarrierRepo       repositories.CarrierRepository
	ShipmentRepo      repositories.ShipmentRepository
	HoldRepo          repositories.ShipmentHoldRepository
	ControlRepo       repositories.ShipmentControlRepository
	ShipmentValidator *shipmentservice.Validator
	Coordinator       *shipmentstate.Coordinator
	EventService      portservices.ShipmentEventService
	Realtime          portservices.RealtimeService
}

type Service struct {
	l                 *zap.Logger
	db                ports.DBConnection
	repo              repositories.CarrierAssignmentRepository
	assignmentRepo    repositories.AssignmentRepository
	carrierRepo       repositories.CarrierRepository
	shipmentRepo      repositories.ShipmentRepository
	holdRepo          repositories.ShipmentHoldRepository
	controlRepo       repositories.ShipmentControlRepository
	shipmentValidator *shipmentservice.Validator
	coordinator       *shipmentstate.Coordinator
	eventService      portservices.ShipmentEventService
	realtime          portservices.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.carrier-assignment"),
		db:                p.DB,
		repo:              p.Repo,
		assignmentRepo:    p.AssignmentRepo,
		carrierRepo:       p.CarrierRepo,
		shipmentRepo:      p.ShipmentRepo,
		holdRepo:          p.HoldRepo,
		controlRepo:       p.ControlRepo,
		shipmentValidator: p.ShipmentValidator,
		coordinator:       p.Coordinator,
		eventService:      p.EventService,
		realtime:          p.Realtime,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetCarrierAssignmentByIDRequest,
) (*shipment.CarrierAssignment, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetActiveByMoveID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.CarrierAssignment, error) {
	return s.repo.GetActiveByMoveID(ctx, tenantInfo, moveID)
}

// PreviewEligibility surfaces the block/warn outcome for assigning the given
// carrier without mutating anything, so dispatch can preflight the decision.
func (s *Service) PreviewEligibility(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (*EligibilityResult, error) {
	carrierEntity, err := s.loadCarrier(ctx, tenantInfo, carrierID)
	if err != nil {
		return nil, err
	}

	result := EvaluateCarrierEligibility(carrierEntity, timeutils.NowUnix())
	return &result, nil
}

func (s *Service) AssignToMove(
	ctx context.Context,
	req *repositories.AssignMoveToCarrierRequest,
) (*shipment.CarrierAssignment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var result *shipment.CarrierAssignment
	var carrierEntity *carrier.Carrier

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, txErr := s.assignmentRepo.GetMoveByID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if txErr != nil {
			return txErr
		}
		if txErr = ensureCoverableMove(move); txErr != nil {
			return txErr
		}
		if txErr = s.ensureNoDispatchHold(txCtx, move.ShipmentID, req.TenantInfo); txErr != nil {
			return txErr
		}

		driverAssignment, txErr := s.assignmentRepo.GetByMoveID(
			txCtx, req.TenantInfo, req.ShipmentMoveID,
		)
		if txErr != nil {
			return txErr
		}
		if driverAssignment != nil {
			return errortypes.NewBusinessError("Shipment move already has a driver assignment. Unassign the driver before assigning a carrier").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}

		existing, txErr := s.repo.GetActiveByMoveID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if txErr != nil {
			return txErr
		}
		if existing != nil && !req.Replace {
			return errortypes.NewBusinessError("Shipment move already has a carrier assignment").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}

		carrierEntity, txErr = s.loadCarrier(txCtx, req.TenantInfo, req.CarrierID)
		if txErr != nil {
			return txErr
		}
		if txErr = enforceEligibility(carrierEntity, req.OverrideInsuranceWarning); txErr != nil {
			return txErr
		}

		original, txErr := s.loadShipment(txCtx, req.TenantInfo, move.ShipmentID)
		if txErr != nil {
			return txErr
		}

		if existing != nil {
			now := timeutils.NowUnix()
			existing.Status = shipment.CarrierAssignmentStatusCanceled
			existing.CanceledAt = &now
			existing.CancellationReason = "Replaced by a new carrier assignment"
			if _, txErr = s.repo.Update(txCtx, existing); txErr != nil {
				return txErr
			}
		}

		entity := s.buildAssignment(req, move)
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		saved, txErr := s.repo.Create(txCtx, entity)
		if txErr != nil {
			if dberror.IsUniqueConstraintViolation(txErr) {
				return errortypes.NewBusinessError("Shipment move already has a carrier assignment").
					WithParam("shipmentMoveId", req.ShipmentMoveID.String())
			}
			return txErr
		}

		if txErr = s.applyCoverageChange(
			txCtx, req.TenantInfo, original, req.ShipmentMoveID, saved,
		); txErr != nil {
			return txErr
		}

		result, txErr = s.repo.GetByID(txCtx, &repositories.GetCarrierAssignmentByIDRequest{
			TenantInfo:          req.TenantInfo,
			CarrierAssignmentID: saved.ID,
		})
		return txErr
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Carrier assignment is busy. Retry the request.",
		)
	}

	s.recordEvent(ctx, shipmenteventservice.BuildCarrierAssigned(
		tenantRef(req.TenantInfo),
		assignmentRef(result),
		result,
		carrierEntity.Name,
		actorFromTenant(req.TenantInfo),
	))
	s.publishInvalidation(ctx, req.TenantInfo, shipmentIDOf(result), "carrier_assigned")

	return result, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	var canceled *shipment.CarrierAssignment
	var carrierName string

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, txErr := s.repo.GetActiveByMoveID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if txErr != nil {
			return txErr
		}
		if existing == nil {
			return errortypes.NewNotFoundError(
				"Carrier assignment not found within your organization",
			)
		}

		move, txErr := s.assignmentRepo.GetMoveByID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if txErr != nil {
			return txErr
		}
		if move.Status != shipment.MoveStatusAssigned {
			return errortypes.NewBusinessError("Only fresh assigned shipment moves can have their carrier assignment canceled").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}

		original, txErr := s.loadShipment(txCtx, req.TenantInfo, move.ShipmentID)
		if txErr != nil {
			return txErr
		}

		if existing.Carrier != nil {
			carrierName = existing.Carrier.Name
		}

		now := timeutils.NowUnix()
		existing.Status = shipment.CarrierAssignmentStatusCanceled
		existing.CanceledAt = &now
		existing.CancellationReason = req.Reason
		if canceled, txErr = s.repo.Update(txCtx, existing); txErr != nil {
			return txErr
		}

		return s.applyCoverageChange(txCtx, req.TenantInfo, original, req.ShipmentMoveID, nil)
	})
	if err != nil {
		return dberror.MapRetryableTransactionError(
			err,
			"Carrier assignment is busy. Retry the request.",
		)
	}

	s.recordEvent(ctx, shipmenteventservice.BuildCarrierUnassigned(
		tenantRef(req.TenantInfo),
		assignmentRef(canceled),
		carrierName,
		req.Reason,
		actorFromTenant(req.TenantInfo),
	))
	s.publishInvalidation(ctx, req.TenantInfo, shipmentIDOf(canceled), "carrier_unassigned")

	return nil
}

// applyCoverageChange re-derives the move and shipment statuses through the
// shared coordinator after carrier coverage is added (assignment non-nil) or
// removed (assignment nil), then persists the shipment.
func (s *Service) applyCoverageChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
	moveID pulid.ID,
	assignment *shipment.CarrierAssignment,
) error {
	updated := cloneShipment(original)
	targetMove := findMove(updated, moveID)
	if targetMove == nil {
		return errortypes.NewBusinessError("Shipment does not contain the target move").
			WithParam("shipmentMoveId", moveID.String())
	}

	targetMove.CarrierAssignment = assignment
	if assignment != nil {
		targetMove.CoverageType = shipment.MoveCoverageTypeCarrier
	} else {
		targetMove.CoverageType = shipment.MoveCoverageTypeDriver
	}

	control, err := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		original,
		updated,
		resolveDelayThresholdMinutes(control),
	); multiErr != nil {
		return multiErr
	}

	if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(
		ctx,
		original,
		updated,
	); multiErr != nil {
		return multiErr
	}

	_, err = s.shipmentRepo.Update(ctx, updated)
	return err
}

func (s *Service) buildAssignment(
	req *repositories.AssignMoveToCarrierRequest,
	move *shipment.ShipmentMove,
) *shipment.CarrierAssignment {
	accessorials := make([]*shipment.CarrierAssignmentAccessorial, 0, len(req.Accessorials))
	for _, acc := range req.Accessorials {
		accessorials = append(accessorials, &shipment.CarrierAssignmentAccessorial{
			OrganizationID:      req.TenantInfo.OrgID,
			BusinessUnitID:      req.TenantInfo.BuID,
			AccessorialChargeID: acc.AccessorialChargeID,
			Description:         acc.Description,
			Amount:              acc.Amount,
		})
	}

	var assignedBy *pulid.ID
	if !req.TenantInfo.UserID.IsNil() {
		userID := req.TenantInfo.UserID
		assignedBy = &userID
	}

	entity := &shipment.CarrierAssignment{
		OrganizationID:        req.TenantInfo.OrgID,
		BusinessUnitID:        req.TenantInfo.BuID,
		ShipmentMoveID:        req.ShipmentMoveID,
		CarrierID:             req.CarrierID,
		Status:                shipment.CarrierAssignmentStatusPending,
		RateMethod:            req.RateMethod,
		BaseRate:              req.BaseRate,
		FuelSurcharge:         req.FuelSurcharge,
		ProNumber:             req.ProNumber,
		ExternalDriverName:    req.ExternalDriverName,
		ExternalDriverPhone:   req.ExternalDriverPhone,
		ExternalTractorNumber: req.ExternalTractorNumber,
		ExternalTrailerNumber: req.ExternalTrailerNumber,
		AssignedByID:          assignedBy,
		Accessorials:          accessorials,
	}
	entity.SyncTotals(move.Distance)

	return entity
}

func (s *Service) loadCarrier(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (*carrier.Carrier, error) {
	return s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
}

func (s *Service) loadShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	return s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
}

func (s *Service) ensureNoDispatchHold(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	hasHold, err := s.holdRepo.HasActiveDispatchHold(ctx, &repositories.ActiveShipmentHoldRequest{
		ShipmentID: shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if hasHold {
		return errortypes.NewBusinessError("Shipment has an active dispatch-blocking hold").
			WithParam("shipmentId", shipmentID.String())
	}

	return nil
}

func (s *Service) recordEvent(
	ctx context.Context,
	params *portservices.RecordShipmentEventParams,
) {
	if params == nil || params.ShipmentID.IsNil() {
		return
	}

	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record carrier assignment event", zap.Error(err))
	}
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	action string,
) {
	if s.realtime == nil || shipmentID.IsNil() {
		return
	}

	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		ActorType:      portservices.PrincipalTypeUser,
		ActorID:        tenantInfo.UserID,
		Resource:       "shipments",
		Action:         action,
		RecordID:       shipmentID,
	})
	if err != nil {
		s.l.Warn("failed to publish carrier assignment invalidation", zap.Error(err))
	}
}

func enforceEligibility(entity *carrier.Carrier, overrideWarnings bool) error {
	result := EvaluateCarrierEligibility(entity, timeutils.NowUnix())

	if result.IsBlocked() {
		return errortypes.NewBusinessError(
			"Carrier is not eligible for assignment: "+strings.Join(result.Blockers, "; "),
		).WithParam("carrierId", entity.ID.String())
	}

	if result.HasWarnings() && !overrideWarnings {
		return errortypes.NewBusinessError(
			"Carrier has insurance warnings: "+strings.Join(result.Warnings, "; ")+
				". Confirm the override to proceed",
		).WithParam("carrierId", entity.ID.String()).
			WithParam("overridable", "true")
	}

	return nil
}

func ensureCoverableMove(move *shipment.ShipmentMove) error {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch move.Status {
	case shipment.MoveStatusCompleted:
		return errortypes.NewBusinessError("Completed shipment moves cannot be assigned")
	case shipment.MoveStatusCanceled:
		return errortypes.NewBusinessError("Canceled shipment moves cannot be assigned")
	default:
		return nil
	}
}

func tenantRef(tenantInfo pagination.TenantInfo) shipmenteventservice.TenantRef {
	return shipmenteventservice.TenantRef{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func actorFromTenant(tenantInfo pagination.TenantInfo) portservices.AuditActor {
	if tenantInfo.UserID.IsNil() {
		return portservices.AuditActor{}
	}
	return portservices.AuditActor{
		PrincipalType: portservices.PrincipalTypeUser,
		PrincipalID:   tenantInfo.UserID,
		UserID:        tenantInfo.UserID,
	}
}

func assignmentRef(assignment *shipment.CarrierAssignment) shipmenteventservice.AssignmentRef {
	if assignment == nil {
		return shipmenteventservice.AssignmentRef{}
	}

	ref := shipmenteventservice.AssignmentRef{
		MoveID: assignment.ShipmentMoveID,
	}
	if assignment.ShipmentMove != nil {
		ref.ShipmentID = assignment.ShipmentMove.ShipmentID
	}
	return ref
}

func shipmentIDOf(assignment *shipment.CarrierAssignment) pulid.ID {
	if assignment == nil || assignment.ShipmentMove == nil {
		return pulid.Nil
	}
	return assignment.ShipmentMove.ShipmentID
}

func cloneShipment(source *shipment.Shipment) *shipment.Shipment {
	if source == nil {
		return nil
	}

	clone := *source
	clone.Moves = make([]*shipment.ShipmentMove, 0, len(source.Moves))

	for _, move := range source.Moves {
		if move == nil {
			clone.Moves = append(clone.Moves, nil)
			continue
		}

		moveClone := *move
		if move.Assignment != nil {
			assignmentClone := *move.Assignment
			moveClone.Assignment = &assignmentClone
		}
		if move.CarrierAssignment != nil {
			carrierAssignmentClone := *move.CarrierAssignment
			moveClone.CarrierAssignment = &carrierAssignmentClone
		}
		moveClone.Stops = make([]*shipment.Stop, 0, len(move.Stops))

		for _, stop := range move.Stops {
			if stop == nil {
				moveClone.Stops = append(moveClone.Stops, nil)
				continue
			}

			stopClone := *stop
			moveClone.Stops = append(moveClone.Stops, &stopClone)
		}

		clone.Moves = append(clone.Moves, &moveClone)
	}

	return &clone
}

func findMove(entity *shipment.Shipment, moveID pulid.ID) *shipment.ShipmentMove {
	for _, move := range entity.Moves {
		if move != nil && move.ID == moveID {
			return move
		}
	}

	return nil
}

func resolveDelayThresholdMinutes(control *tenant.ShipmentControl) int16 {
	if control == nil {
		return shipmentstate.DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return shipmentstate.ResolveDelayThresholdMinutes(0)
	}
	return shipmentstate.ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}
