package carrierassignmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchguard"
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
	RateConRepo       repositories.RateConfirmationRepository
	ShipmentValidator *shipmentservice.Validator
	Coordinator       *shipmentstate.Coordinator
	EventService      portservices.ShipmentEventService
	AuditService      portservices.AuditService
	Realtime          portservices.RealtimeService
	CostAccrual       portservices.CarrierCostAccrual `optional:"true"`
	TenderGuard       portservices.TenderGuard        `optional:"true"`
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
	rateConRepo       repositories.RateConfirmationRepository
	shipmentValidator *shipmentservice.Validator
	coordinator       *shipmentstate.Coordinator
	eventService      portservices.ShipmentEventService
	auditService      portservices.AuditService
	realtime          portservices.RealtimeService
	costAccrual       portservices.CarrierCostAccrual
	tenderGuard       portservices.TenderGuard
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
		rateConRepo:       p.RateConRepo,
		shipmentValidator: p.ShipmentValidator,
		coordinator:       p.Coordinator,
		eventService:      p.EventService,
		auditService:      p.AuditService,
		realtime:          p.Realtime,
		costAccrual:       p.CostAccrual,
		tenderGuard:       p.TenderGuard,
	}
}

var _ portservices.CarrierMoveAssigner = (*Service)(nil)

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
) (*carrier.EligibilityResult, error) {
	carrierEntity, err := s.loadCarrier(ctx, tenantInfo, carrierID)
	if err != nil {
		return nil, err
	}

	result := carrier.EvaluateCarrierEligibility(carrierEntity, timeutils.NowUnix())
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
	var replaced bool

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, txErr := s.assignmentRepo.GetMoveByID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if txErr != nil {
			return txErr
		}
		if txErr = move.EnsureAssignable(); txErr != nil {
			return txErr
		}
		if txErr = validatePerMileDistance(req.RateMethod, move.Distance); txErr != nil {
			return txErr
		}
		if txErr = dispatchguard.EnsureNoDispatchHold(
			txCtx, s.holdRepo, req.TenantInfo, move.ShipmentID,
		); txErr != nil {
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
			if txErr = s.voidActiveRateConfirmation(
				txCtx, req.TenantInfo, existing.ID, "Carrier assignment replaced",
			); txErr != nil {
				return txErr
			}
			replaced = true
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
		shipmenteventservice.TenantRefFor(req.TenantInfo),
		assignmentRef(result),
		result,
		carrierEntity.Name,
		shipmenteventservice.ActorFor(req.TenantInfo),
	))
	s.logCarrierAssignmentAudit(&carrierAssignmentAuditParams{
		TenantInfo: req.TenantInfo,
		MoveID:     req.ShipmentMoveID,
		Operation:  permission.OpAssign,
		UserID:     req.TenantInfo.UserID,
		Comment:    "Carrier " + carrierEntity.Name + " assigned to move",
		Current:    result,
	})
	s.publishInvalidation(ctx, req.TenantInfo, shipmentIDOf(result), "carrier_assigned")
	if replaced {
		s.reaccrueMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	}
	s.withdrawLiveTender(ctx, req.TenantInfo, req.ShipmentMoveID)

	return result, nil
}

// ConfirmAssignment marks a pending assignment Confirmed, used when a carrier
// explicitly agreed to the rate — a signed rate confirmation or an accepted
// tender offer.
func (s *Service) ConfirmAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
) error {
	entity, err := s.repo.GetByID(ctx, &repositories.GetCarrierAssignmentByIDRequest{
		TenantInfo:          tenantInfo,
		CarrierAssignmentID: assignmentID,
	})
	if err != nil {
		return err
	}
	if entity.Status != shipment.CarrierAssignmentStatusPending {
		if entity.Status == shipment.CarrierAssignmentStatusConfirmed {
			return nil
		}
		return errortypes.NewBusinessError(
			"Only pending carrier assignments can be confirmed",
		).WithParam("carrierAssignmentId", assignmentID.String())
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = shipment.CarrierAssignmentStatusConfirmed
	entity.ConfirmedAt = &now
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return err
	}

	s.logCarrierAssignmentAudit(&carrierAssignmentAuditParams{
		TenantInfo: tenantInfo,
		MoveID:     updated.ShipmentMoveID,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    "Carrier assignment confirmed",
		Previous:   &previous,
		Current:    updated,
	})

	return nil
}

// withdrawLiveTender runs post-transaction: a move covered outside its tender
// leaves that tender pointless, so any live one is withdrawn. The tender
// acceptance flow is unaffected because it moves the tender terminal before
// creating the assignment.
func (s *Service) withdrawLiveTender(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) {
	if s.tenderGuard == nil {
		return
	}
	if err := s.tenderGuard.CancelLiveTenderForMove(
		ctx, tenantInfo, moveID, "Move was covered outside the tender",
	); err != nil {
		s.l.Error("failed to withdraw live tender for covered move",
			zap.Error(err), zap.String("moveId", moveID.String()))
	}
}

func (s *Service) Cancel(
	ctx context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	var carrierName string
	var shipmentID pulid.ID
	var canceled *shipment.CarrierAssignment

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
		shipmentID = move.ShipmentID

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
		if _, txErr = s.repo.Update(txCtx, existing); txErr != nil {
			return txErr
		}
		if txErr = s.voidActiveRateConfirmation(
			txCtx, req.TenantInfo, existing.ID, "Carrier assignment canceled",
		); txErr != nil {
			return txErr
		}
		canceled = existing

		return s.applyCoverageChange(txCtx, req.TenantInfo, original, req.ShipmentMoveID, nil)
	})
	if err != nil {
		return dberror.MapRetryableTransactionError(
			err,
			"Carrier assignment is busy. Retry the request.",
		)
	}

	s.recordEvent(ctx, shipmenteventservice.BuildCarrierUnassigned(
		shipmenteventservice.TenantRefFor(req.TenantInfo),
		shipmenteventservice.AssignmentRef{
			ShipmentID: shipmentID,
			MoveID:     req.ShipmentMoveID,
		},
		carrierName,
		req.Reason,
		shipmenteventservice.ActorFor(req.TenantInfo),
	))
	s.logCarrierAssignmentAudit(&carrierAssignmentAuditParams{
		TenantInfo: req.TenantInfo,
		MoveID:     req.ShipmentMoveID,
		Operation:  permission.OpUnassign,
		UserID:     req.TenantInfo.UserID,
		Comment:    "Carrier assignment canceled: " + req.Reason,
		Current:    canceled,
	})
	s.publishInvalidation(ctx, req.TenantInfo, shipmentID, "carrier_unassigned")
	s.reaccrueMove(ctx, req.TenantInfo, req.ShipmentMoveID)

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
	updated := shipment.CloneForUpdate(original)
	targetMove := updated.FindMove(moveID)
	if targetMove == nil {
		return errortypes.NewBusinessError("Shipment does not contain the target move").
			WithParam("shipmentMoveId", moveID.String())
	}

	targetMove.CarrierAssignment = assignment
	if assignment != nil {
		targetMove.CoverageType = shipment.MoveCoverageTypeCarrier
	} else {
		targetMove.CoverageType = shipment.MoveCoverageTypeUnassigned
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
		shipmentstate.ResolveControlDelayThreshold(control),
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

// voidActiveRateConfirmation retires the assignment's standing agreement inside
// the mutating transaction so a canceled or replaced assignment can never keep
// an active rate confirmation behind it.
func (s *Service) voidActiveRateConfirmation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
	reason string,
) error {
	active, err := s.rateConRepo.GetActiveByAssignmentID(ctx, tenantInfo, assignmentID)
	if err != nil {
		return err
	}
	if active == nil {
		return nil
	}

	now := timeutils.NowUnix()
	active.Status = rateconfirmation.StatusVoided
	active.VoidedAt = &now
	active.VoidReason = reason
	if _, err = s.rateConRepo.Update(ctx, active); err != nil {
		return err
	}

	if revokeErr := s.rateConRepo.RevokeTokensForRateConfirmation(
		ctx, tenantInfo, active.ID, now,
	); revokeErr != nil {
		s.l.Warn("failed to revoke rate confirmation tokens",
			zap.Error(revokeErr),
			zap.String("rateConfirmationId", active.ID.String()))
	}
	return nil
}

// reaccrueMove runs post-transaction, mirroring how accrual hooks fire
// elsewhere; a failure is logged because the sweep re-derives from current
// state on its next run.
func (s *Service) reaccrueMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) {
	if s.costAccrual == nil {
		return
	}
	if err := s.costAccrual.ReaccrueMove(ctx, tenantInfo, moveID); err != nil {
		s.l.Error("failed to re-accrue carrier cost events for move",
			zap.Error(err), zap.String("moveId", moveID.String()))
	}
}

func validatePerMileDistance(rateMethod shipment.CarrierRateMethod, distance *float64) error {
	if rateMethod != shipment.CarrierRateMethodPerMile {
		return nil
	}
	if distance == nil || *distance <= 0 {
		return errortypes.NewValidationError(
			"rateMethod",
			errortypes.ErrInvalid,
			"Per-mile rates require a move with a computed distance. Compute the move distance first or use a Flat rate",
		)
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
	result := carrier.EvaluateCarrierEligibility(entity, timeutils.NowUnix())

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
