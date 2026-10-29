package carrierassignmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.Transactor
	Repo              carrierassignmentports.CarrierAssignmentRepository
	AssignmentRepo    assignmentports.AssignmentRepository
	CarrierRepo       carrierports.CarrierRepository
	ShipmentRepo      shipmentports.ShipmentRepository
	HoldRepo          shipmentholdports.ShipmentHoldRepository
	ControlRepo       tenantports.ShipmentControlRepository
	RateConRepo       rateconfirmationports.RateConfirmationRepository
	ShipmentValidator *shipmentvalidator.Validator
	Coordinator       *shipmentstate.Coordinator
	EventService      shipmenteventports.ShipmentEventService
	AuditService      auditports.AuditService
	Invalidator       shipmentinvalidationports.ShipmentInvalidator
	CostAccrual       carrierports.CarrierCostAccrual `optional:"true"`
	TenderGuard       tenderports.TenderGuard         `optional:"true"`
	RateEngine        ratequoteports.RateEngine       `optional:"true"`
	AgreementRepo     rateagreementports.RateAgreementRepository
	BillingCtrlRepo   tenantports.BillingControlRepository
	AccessorialRepo   accessorialchargeports.AccessorialChargeRepository
	FuelResolver      fuelsurchargeports.FuelSurchargeResolver
}

type Service struct {
	l                 *zap.Logger
	db                kernelports.Transactor
	repo              carrierassignmentports.CarrierAssignmentRepository
	assignmentRepo    assignmentports.AssignmentRepository
	carrierRepo       carrierports.CarrierRepository
	shipmentRepo      shipmentports.ShipmentRepository
	holdRepo          shipmentholdports.ShipmentHoldRepository
	controlRepo       tenantports.ShipmentControlRepository
	rateConRepo       rateconfirmationports.RateConfirmationRepository
	shipmentValidator *shipmentvalidator.Validator
	coordinator       *shipmentstate.Coordinator
	eventService      shipmenteventports.ShipmentEventService
	auditService      auditports.AuditService
	invalidator       shipmentinvalidationports.ShipmentInvalidator
	costAccrual       carrierports.CarrierCostAccrual
	tenderGuard       tenderports.TenderGuard
	rateEngine        ratequoteports.RateEngine
	agreementRepo     rateagreementports.RateAgreementRepository
	billingCtrlRepo   tenantports.BillingControlRepository
	intelGate         carrierintelports.CarrierIntelGate
	lifecycle         carrierintelports.CarrierLifecycleObserver
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	fuelResolver      fuelsurchargeports.FuelSurchargeResolver
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
		invalidator:       p.Invalidator,
		costAccrual:       p.CostAccrual,
		tenderGuard:       p.TenderGuard,
		rateEngine:        p.RateEngine,
		agreementRepo:     p.AgreementRepo,
		billingCtrlRepo:   p.BillingCtrlRepo,
		accessorialRepo:   p.AccessorialRepo,
		fuelResolver:      p.FuelResolver,
	}
}

var _ tenderports.CarrierMoveAssigner = (*Service)(nil)

func (s *Service) Get(
	ctx context.Context,
	req *carrierassignmentports.GetCarrierAssignmentByIDRequest,
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

	result := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: carrierEntity,
		Now:     timeutils.NowUnix(),
		Intel:   s.intelGateFor(ctx, tenantInfo, carrierID),
	})
	return &result, nil
}

func (s *Service) SetIntelGate(gate carrierintelports.CarrierIntelGate) {
	s.intelGate = gate
}

func (s *Service) SetLifecycleObserver(observer carrierintelports.CarrierLifecycleObserver) {
	s.lifecycle = observer
}

func (s *Service) intelGateFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) *carrier.IntelGate {
	if s.intelGate == nil {
		return nil
	}
	gates, err := s.intelGate.GateFor(ctx, tenantInfo, []pulid.ID{carrierID})
	if err != nil {
		s.l.Warn("failed to evaluate carrier intelligence gate", zap.Error(err))
		return nil
	}
	return gates[carrierID]
}

func (s *Service) refreshIntel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) *carrier.IntelGate {
	if s.intelGate == nil {
		return nil
	}
	s.intelGate.EnsureFresh(ctx, &carrierintelports.CarrierIntelEnsureFreshRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: []pulid.ID{carrierID},
	})
	return s.intelGateFor(ctx, tenantInfo, carrierID)
}

func (s *Service) AssignToMove(
	ctx context.Context,
	req *carrierassignmentports.AssignMoveToCarrierRequest,
) (*shipment.CarrierAssignment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var result *shipment.CarrierAssignment
	var carrierEntity *carrier.Carrier
	var replaced bool

	intelGate := s.refreshIntel(ctx, req.TenantInfo, req.CarrierID)

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		plan, txErr := s.planAssignToMove(txCtx, req, intelGate)
		if txErr != nil {
			return txErr
		}
		carrierEntity = plan.carrier
		original := plan.original

		if existing := plan.existing; existing != nil {
			existing.Cancel(timeutils.NowUnix(), replacedAssignmentReason)
			if _, txErr = s.repo.Update(txCtx, existing); txErr != nil {
				return txErr
			}
			if txErr = s.voidActiveRateConfirmation(
				txCtx, req.TenantInfo, existing.ID, replacedRateConReason,
			); txErr != nil {
				return txErr
			}
			replaced = true
		}

		if txErr = s.rateBuySide(txCtx, req, original); txErr != nil {
			return txErr
		}

		entity := s.buildAssignment(req, plan.move)
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

		result, txErr = s.repo.GetByID(
			txCtx,
			&carrierassignmentports.GetCarrierAssignmentByIDRequest{
				TenantInfo:          req.TenantInfo,
				CarrierAssignmentID: saved.ID,
			},
		)
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
	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			req.TenantInfo,
			shipmentIDOf(result),
			"carrier_assigned",
		),
	)
	if replaced {
		s.reaccrueMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	}
	s.withdrawLiveTender(ctx, req.TenantInfo, req.ShipmentMoveID)
	if s.lifecycle != nil {
		s.lifecycle.CarriersUsed(ctx, req.TenantInfo, []pulid.ID{req.CarrierID})
	}

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
	entity, err := s.repo.GetByID(ctx, &carrierassignmentports.GetCarrierAssignmentByIDRequest{
		TenantInfo:          tenantInfo,
		CarrierAssignmentID: assignmentID,
	})
	if err != nil {
		return err
	}
	if entity.Status != shipmentvocab.CarrierAssignmentStatusPending {
		if entity.Status == shipmentvocab.CarrierAssignmentStatusConfirmed {
			return nil
		}
		return errortypes.NewBusinessError(
			"Only pending carrier assignments can be confirmed",
		).WithParam("carrierAssignmentId", assignmentID.String())
	}

	previous := *entity
	now := timeutils.NowUnix()
	entity.Status = shipmentvocab.CarrierAssignmentStatusConfirmed
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
	req *carrierassignmentports.CancelCarrierAssignmentRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	var carrierName string
	var shipmentID pulid.ID
	var canceled *shipment.CarrierAssignment

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		plan, txErr := s.planCancel(txCtx, req)
		if txErr != nil {
			return txErr
		}
		existing := plan.existing
		shipmentID = plan.move.ShipmentID

		if existing.Carrier != nil {
			carrierName = existing.Carrier.Name
		}

		existing.Cancel(timeutils.NowUnix(), req.Reason)
		if _, txErr = s.repo.Update(txCtx, existing); txErr != nil {
			return txErr
		}
		if txErr = s.voidActiveRateConfirmation(
			txCtx, req.TenantInfo, existing.ID, canceledRateConReason,
		); txErr != nil {
			return txErr
		}
		canceled = existing

		return s.applyCoverageChange(
			txCtx, req.TenantInfo, plan.original, req.ShipmentMoveID, nil,
		)
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
	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			req.TenantInfo,
			shipmentID,
			"carrier_unassigned",
		),
	)
	s.reaccrueMove(ctx, req.TenantInfo, req.ShipmentMoveID)

	return nil
}

// applyCoverageChange persists the shipment as projectCoverageChange derives
// it after carrier coverage is added (assignment non-nil) or removed
// (assignment nil).
func (s *Service) applyCoverageChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
	moveID pulid.ID,
	assignment *shipment.CarrierAssignment,
) error {
	updated, err := s.projectCoverageChange(ctx, tenantInfo, original, moveID, assignment)
	if err != nil {
		return err
	}

	_, err = s.shipmentRepo.Update(ctx, updated)
	return err
}

func (s *Service) buildAssignment(
	req *carrierassignmentports.AssignMoveToCarrierRequest,
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
		Status:                shipmentvocab.CarrierAssignmentStatusPending,
		RateMethod:            req.RateMethod,
		BaseRate:              req.BaseRate,
		FuelSurcharge:         req.FuelSurcharge,
		ProNumber:             req.ProNumber,
		ExternalDriverName:    req.ExternalDriverName,
		ExternalDriverPhone:   req.ExternalDriverPhone,
		ExternalTractorNumber: req.ExternalTractorNumber,
		ExternalTrailerNumber: req.ExternalTrailerNumber,
		AssignedByID:          assignedBy,
		RateQuoteID:           req.RateQuoteID,
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
	return s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
}

func (s *Service) loadShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	return s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
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
	active.Void(now, reason)
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

func validatePerMileDistance(rateMethod shipmentvocab.CarrierRateMethod, distance *float64) error {
	if rateMethod != shipmentvocab.CarrierRateMethodPerMile {
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
	params *shipmenteventports.RecordShipmentEventParams,
) {
	if params == nil || params.ShipmentID.IsNil() {
		return
	}

	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record carrier assignment event", zap.Error(err))
	}
}

// enforceEligibility refuses a carrier that is blocked, or that carries
// warnings nobody overrode, and returns the warnings an override let through.
func enforceEligibility(
	entity *carrier.Carrier,
	intelGate *carrier.IntelGate,
	overrideWarnings bool,
) ([]string, error) {
	result := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: entity,
		Now:     timeutils.NowUnix(),
		Intel:   intelGate,
	})

	if result.IsBlocked() {
		return nil, errortypes.NewBusinessError(
			"Carrier is not eligible for assignment: {0}", strings.Join(result.Blockers, "; "),
		).WithParam("carrierId", entity.ID.String())
	}

	if result.HasWarnings() && !overrideWarnings {
		return nil, errortypes.NewBusinessError(
			"Carrier has insurance warnings: {0}. Confirm the override to proceed",
			strings.Join(result.Warnings, "; "),
		).WithParam("carrierId", entity.ID.String()).
			WithParam("overridable", "true")
	}

	return result.Warnings, nil
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
