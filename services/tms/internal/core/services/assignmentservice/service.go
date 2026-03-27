package assignmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	DB                  ports.DBConnection
	Repo                repositories.AssignmentRepository
	ShipmentRepo        repositories.ShipmentRepository
	HoldRepo            repositories.ShipmentHoldRepository
	ControlRepo         repositories.ShipmentControlRepository
	DispatchControlRepo repositories.DispatchControlRepository
	WorkerRepo          repositories.WorkerRepository
	CommodityRepo       repositories.CommodityRepository
	ContinuityRepo      repositories.EquipmentContinuityRepository
	TrailerRepo         repositories.TrailerRepository
	LocationRepo        repositories.LocationRepository
	ShipmentValidator   *shipmentservice.Validator
	Coordinator         *shipmentstate.Coordinator
	Commercial          *shipmentcommercial.Calculator
}

type service struct {
	l                   *zap.Logger
	db                  ports.DBConnection
	repo                repositories.AssignmentRepository
	shipmentRepo        repositories.ShipmentRepository
	holdRepo            repositories.ShipmentHoldRepository
	controlRepo         repositories.ShipmentControlRepository
	dispatchControlRepo repositories.DispatchControlRepository
	workerRepo          repositories.WorkerRepository
	commodityRepo       repositories.CommodityRepository
	continuityRepo      repositories.EquipmentContinuityRepository
	trailerRepo         repositories.TrailerRepository
	locationRepo        repositories.LocationRepository
	shipmentValidator   *shipmentservice.Validator
	coordinator         *shipmentstate.Coordinator
	commercial          *shipmentcommercial.Calculator
}

func New(p Params) portservices.AssignmentService {
	return &service{
		l:                   p.Logger.Named("service.assignment"),
		db:                  p.DB,
		repo:                p.Repo,
		shipmentRepo:        p.ShipmentRepo,
		holdRepo:            p.HoldRepo,
		controlRepo:         p.ControlRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		workerRepo:          p.WorkerRepo,
		commodityRepo:       p.CommodityRepo,
		continuityRepo:      p.ContinuityRepo,
		trailerRepo:         p.TrailerRepo,
		locationRepo:        p.LocationRepo,
		shipmentValidator:   p.ShipmentValidator,
		coordinator:         p.Coordinator,
		commercial:          p.Commercial,
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListAssignmentsRequest,
) (*pagination.ListResult[*shipment.Assignment], error) {
	return s.repo.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req *repositories.GetAssignmentByIDRequest,
) (*shipment.Assignment, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *service) AssignToMove(
	ctx context.Context,
	req *repositories.AssignShipmentMoveRequest,
) (*shipment.Assignment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.upsertAssignment(
		ctx,
		req.TenantInfo,
		req.ShipmentMoveID,
		func(existing *shipment.Assignment) (*shipment.Assignment, error) {
			if existing != nil {
				return nil, errortypes.NewBusinessError("Shipment move already has an assignment").
					WithParam("shipmentMoveId", req.ShipmentMoveID.String())
			}

			return &shipment.Assignment{
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				ShipmentMoveID:    req.ShipmentMoveID,
				PrimaryWorkerID:   &req.PrimaryWorkerID,
				TractorID:         &req.TractorID,
				TrailerID:         req.TrailerID,
				SecondaryWorkerID: req.SecondaryWorkerID,
				Status:            shipment.AssignmentStatusNew,
			}, nil
		},
		func(txCtx context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			return s.repo.Create(txCtx, entity)
		},
	)
}

func (s *service) Reassign(
	ctx context.Context,
	req *repositories.ReassignShipmentMoveRequest,
) (*shipment.Assignment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.upsertAssignment(
		ctx,
		req.TenantInfo,
		req.ShipmentMoveID,
		func(existing *shipment.Assignment) (*shipment.Assignment, error) {
			if existing == nil {
				return nil, errortypes.NewNotFoundError(
					"Assignment not found within your organization",
				)
			}

			updated := *existing
			updated.PrimaryWorkerID = &req.PrimaryWorkerID
			updated.TractorID = &req.TractorID
			updated.TrailerID = req.TrailerID
			updated.SecondaryWorkerID = req.SecondaryWorkerID
			updated.Status = shipment.AssignmentStatusNew

			return &updated, nil
		},
		func(txCtx context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			return s.repo.Update(txCtx, entity)
		},
	)
}

func (s *service) Unassign(
	ctx context.Context,
	req *repositories.UnassignShipmentMoveRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, err := s.repo.GetMoveByID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if err != nil {
			return err
		}

		if move.Status != shipment.MoveStatusAssigned {
			return errortypes.NewBusinessError("Only fresh assigned shipment moves can be unassigned").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}

		original, err := s.shipmentRepo.GetByID(txCtx, &repositories.GetShipmentByIDRequest{
			ID: move.ShipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: req.TenantInfo.OrgID,
				BuID:  req.TenantInfo.BuID,
			},
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return err
		}

		existing, err := s.repo.GetByMoveID(txCtx, req.TenantInfo, req.ShipmentMoveID)
		if err != nil {
			return err
		}
		if existing == nil {
			return errortypes.NewNotFoundError("Assignment not found within your organization")
		}
		if existing.Status != shipment.AssignmentStatusNew {
			return errortypes.NewBusinessError("Only fresh assignments can be unassigned").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}

		if _, err = s.repo.Unassign(txCtx, existing); err != nil {
			return err
		}
		updatedShipment := cloneShipment(original)
		targetMove := findMove(updatedShipment, req.ShipmentMoveID)
		if targetMove == nil {
			return errortypes.NewBusinessError("Shipment does not contain the target move").
				WithParam("shipmentMoveId", req.ShipmentMoveID.String())
		}
		targetMove.Assignment = nil
		targetMove.Status = shipment.MoveStatusNew

		control, err := s.controlRepo.Get(txCtx, repositories.GetShipmentControlRequest{
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}

		if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
			original,
			updatedShipment,
			resolveDelayThresholdMinutes(control),
		); multiErr != nil {
			return multiErr
		}

		if err = s.commercial.Recalculate(txCtx, updatedShipment, control, pulid.Nil); err != nil {
			return err
		}

		if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(txCtx, original, updatedShipment); multiErr != nil {
			return multiErr
		}

		_, err = s.shipmentRepo.Update(txCtx, updatedShipment)
		return err
	})
	if err != nil {
		return dberror.MapRetryableTransactionError(
			err,
			"Assignment is busy. Retry the request.",
		)
	}

	return nil
}

func (s *service) CheckWorkerCompliance(
	ctx context.Context,
	req *repositories.CheckWorkerComplianceRequest,
) error {
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	dc, err := s.dispatchControlRepo.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return err
	}

	if !dc.EnforceDriverQualificationCompliance &&
		!dc.EnforceMedicalCertCompliance &&
		!dc.EnforceDrugAndAlcoholCompliance &&
		!dc.EnforceHazmatCompliance &&
		!dc.EnforceHOSCompliance {
		return nil
	}

	primaryWorker, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             req.PrimaryWorkerID,
		TenantInfo:     req.TenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		return err
	}

	var secondaryWorker *worker.Worker
	if req.SecondaryWorkerID != nil && !req.SecondaryWorkerID.IsNil() {
		secondaryWorker, err = s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
			ID:             *req.SecondaryWorkerID,
			TenantInfo:     req.TenantInfo,
			IncludeProfile: true,
		})
		if err != nil {
			return err
		}
	}

	hasHazmatCommodities := false
	if dc.EnforceHazmatCompliance {
		hasHazmatCommodities, err = s.shipmentMoveHasHazmat(ctx, req.TenantInfo, req.ShipmentMoveID)
		if err != nil {
			return err
		}
	}

	multiErr := errortypes.NewMultiError()

	runWorkerComplianceChecks(primaryWorker, dc, hasHazmatCommodities, "primaryWorker", multiErr)

	if secondaryWorker != nil {
		runWorkerComplianceChecks(secondaryWorker, dc, hasHazmatCommodities, "secondaryWorker", multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) shipmentMoveHasHazmat(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (bool, error) {
	move, err := s.repo.GetMoveByID(ctx, tenantInfo, moveID)
	if err != nil {
		return false, err
	}

	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return false, err
	}

	if len(shp.Commodities) == 0 {
		return false, nil
	}

	commodityIDs := make([]pulid.ID, 0, len(shp.Commodities))
	for _, sc := range shp.Commodities {
		if sc != nil && !sc.CommodityID.IsNil() {
			commodityIDs = append(commodityIDs, sc.CommodityID)
		}
	}

	if len(commodityIDs) == 0 {
		return false, nil
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
		TenantInfo:   tenantInfo,
		CommodityIDs: commodityIDs,
	})
	if err != nil {
		return false, err
	}

	for _, c := range commodities {
		if c != nil && c.HazardousMaterial != nil && !c.HazardousMaterial.ID.IsNil() {
			return true, nil
		}
	}

	return false, nil
}

func (s *service) upsertAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	build func(*shipment.Assignment) (*shipment.Assignment, error),
	persist func(context.Context, *shipment.Assignment) (*shipment.Assignment, error),
) (*shipment.Assignment, error) {
	var result *shipment.Assignment

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		move, err := s.repo.GetMoveByID(txCtx, tenantInfo, moveID)
		if err != nil {
			return err
		}

		if err = ensureAssignableMove(move); err != nil {
			return err
		}
		if err = s.ensureNoDispatchHold(txCtx, move.ShipmentID, tenantInfo); err != nil {
			return err
		}

		original, err := s.shipmentRepo.GetByID(txCtx, &repositories.GetShipmentByIDRequest{
			ID: move.ShipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: tenantInfo.OrgID,
				BuID:  tenantInfo.BuID,
			},
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return err
		}
		targetMove := findMove(original, moveID)
		if targetMove == nil {
			return errortypes.NewBusinessError("Shipment does not contain the target move").
				WithParam("shipmentMoveId", moveID.String())
		}

		existing, err := s.repo.GetByMoveID(txCtx, tenantInfo, moveID)
		if err != nil {
			return err
		}

		entity, err := build(existing)
		if err != nil {
			return err
		}
		if err = s.validateTrailerContinuity(txCtx, tenantInfo, targetMove, entity); err != nil {
			return err
		}

		savedAssignment, err := persist(txCtx, entity)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return errortypes.NewBusinessError("Shipment move already has an assignment").
					WithParam("shipmentMoveId", moveID.String())
			}
			return err
		}

		updatedShipment := cloneShipment(original)
		targetMove = findMove(updatedShipment, moveID)
		if targetMove == nil {
			return errortypes.NewBusinessError("Shipment does not contain the target move").
				WithParam("shipmentMoveId", moveID.String())
		}
		targetMove.Assignment = savedAssignment

		control, err := s.controlRepo.Get(txCtx, repositories.GetShipmentControlRequest{
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return err
		}

		if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
			original,
			updatedShipment,
			resolveDelayThresholdMinutes(control),
		); multiErr != nil {
			return multiErr
		}

		if err = s.commercial.Recalculate(txCtx, updatedShipment, control, pulid.Nil); err != nil {
			return err
		}

		if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(txCtx, original, updatedShipment); multiErr != nil {
			return multiErr
		}

		if _, err = s.shipmentRepo.Update(txCtx, updatedShipment); err != nil {
			return err
		}
		result, err = s.repo.GetByID(txCtx, &repositories.GetAssignmentByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: tenantInfo.OrgID,
				BuID:  tenantInfo.BuID,
			},
			AssignmentID: savedAssignment.ID,
		})
		return err
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Assignment is busy. Retry the request.",
		)
	}

	return result, nil
}

func (s *service) validateTrailerContinuity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	targetMove *shipment.ShipmentMove,
	candidate *shipment.Assignment,
) error {
	if candidate.TrailerID == nil {
		return nil
	}

	dc, err := s.dispatchControlRepo.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return err
	}
	if !dc.EnforceTrailerContinuity {
		return nil
	}

	pickupStop, err := firstPickupStop(targetMove)
	if err != nil {
		return err
	}

	effective, err := s.continuityRepo.GetEffectiveCurrent(ctx, repositories.GetCurrentEquipmentContinuityRequest{
		TenantInfo:    tenantInfo,
		EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
		EquipmentID:   *candidate.TrailerID,
	})
	if err != nil {
		return err
	}
	if effective == nil {
		return nil
	}
	if effective.CurrentLocationID == pickupStop.LocationID {
		return nil
	}

	trailerEntity, locationEntity, err := s.resolveTrailerContinuityMessageParts(ctx, tenantInfo, *candidate.TrailerID, effective.CurrentLocationID)
	if err != nil {
		return err
	}

	return errortypes.NewBusinessError(
		fmt.Sprintf(
			"Trailer %s is currently located at %s which doesn't match this move's current pickup location. Locate the trailer before assigning or assign a different trailer",
			trailerEntity.Code,
			locationEntity.Name,
		),
	).
		WithParam("trailerId", candidate.TrailerID.String()).
		WithParam("trailerCode", trailerEntity.Code).
		WithParam("currentLocationId", effective.CurrentLocationID.String()).
		WithParam("currentLocationName", locationEntity.Name).
		WithParam("pickupLocationId", pickupStop.LocationID.String())
}

func (s *service) resolveTrailerContinuityMessageParts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	trailerID pulid.ID,
	locationID pulid.ID,
) (*trailer.Trailer, *location.Location, error) {
	trailerEntity, err := s.trailerRepo.GetByID(ctx, repositories.GetTrailerByIDRequest{
		ID:         trailerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	locationEntity, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID:         locationID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	return trailerEntity, locationEntity, nil
}

func (s *service) ensureNoDispatchHold(
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

func ensureAssignableMove(move *shipment.ShipmentMove) error {
	switch move.Status {
	case shipment.MoveStatusCompleted:
		return errortypes.NewBusinessError("Completed shipment moves cannot be assigned")
	case shipment.MoveStatusCanceled:
		return errortypes.NewBusinessError("Canceled shipment moves cannot be assigned")
	default:
		return nil
	}
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

func firstPickupStop(move *shipment.ShipmentMove) (*shipment.Stop, error) {
	var candidate *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || !stop.IsOriginStop() {
			continue
		}
		if candidate == nil || stop.Sequence < candidate.Sequence {
			candidate = stop
		}
	}

	if candidate == nil {
		return nil, errortypes.NewBusinessError("Shipment move is missing a pickup stop").
			WithParam("shipmentMoveId", move.ID.String())
	}

	return candidate, nil
}

func lastDeliveryStop(move *shipment.ShipmentMove) (*shipment.Stop, error) {
	var candidate *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || !stop.IsDestinationStop() {
			continue
		}
		if candidate == nil || stop.Sequence > candidate.Sequence {
			candidate = stop
		}
	}

	if candidate == nil {
		return nil, errortypes.NewBusinessError("Shipment move is missing a delivery stop").
			WithParam("shipmentMoveId", move.ID.String())
	}

	return candidate, nil
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
