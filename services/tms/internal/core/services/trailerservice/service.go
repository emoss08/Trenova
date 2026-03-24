package trailerservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                    *zap.Logger
	DB                        ports.DBConnection
	Repo                      repositories.TrailerRepository
	AssignmentRepo            repositories.AssignmentRepository
	ContinuityRepo            repositories.EquipmentContinuityRepository
	ShipmentRepo              repositories.ShipmentRepository
	UserRepo                  repositories.UserRepository
	ShipmentCommentRepo       repositories.ShipmentCommentRepository
	ControlRepo               repositories.ShipmentControlRepository
	LocationRepo              repositories.LocationRepository
	Validator                 *Validator
	AuditService              services.AuditService
	Realtime                  services.RealtimeService
	CustomFieldsValuesService *customfieldservice.ValuesService
	ShipmentValidator         *shipmentservice.Validator
	Coordinator               *shipmentstate.Coordinator
	Commercial                *shipmentcommercial.Calculator
}

type Service struct {
	l                         *zap.Logger
	db                        ports.DBConnection
	repo                      repositories.TrailerRepository
	assignmentRepo            repositories.AssignmentRepository
	continuityRepo            repositories.EquipmentContinuityRepository
	shipmentRepo              repositories.ShipmentRepository
	userRepo                  repositories.UserRepository
	shipmentCommentRepo       repositories.ShipmentCommentRepository
	controlRepo               repositories.ShipmentControlRepository
	locationRepo              repositories.LocationRepository
	validator                 *Validator
	auditService              services.AuditService
	realtime                  services.RealtimeService
	customFieldsValuesService *customfieldservice.ValuesService
	shipmentValidator         *shipmentservice.Validator
	coordinator               *shipmentstate.Coordinator
	commercial                *shipmentcommercial.Calculator
}

func New(p Params) *Service {
	return &Service{
		l:                         p.Logger.Named("service.trailer"),
		db:                        p.DB,
		repo:                      p.Repo,
		assignmentRepo:            p.AssignmentRepo,
		continuityRepo:            p.ContinuityRepo,
		shipmentRepo:              p.ShipmentRepo,
		userRepo:                  p.UserRepo,
		shipmentCommentRepo:       p.ShipmentCommentRepo,
		controlRepo:               p.ControlRepo,
		locationRepo:              p.LocationRepo,
		validator:                 p.Validator,
		auditService:              p.AuditService,
		realtime:                  p.Realtime,
		customFieldsValuesService: p.CustomFieldsValuesService,
		shipmentValidator:         p.ShipmentValidator,
		coordinator:               p.Coordinator,
		commercial:                p.Commercial,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTrailersRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	log := s.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(result.Items) > 0 {
		resourceIDs := make([]string, 0, len(result.Items))
		for _, t := range result.Items {
			resourceIDs = append(resourceIDs, t.GetResourceID())
		}

		customFieldsMap, cfErr := s.customFieldsValuesService.GetForResources(
			ctx,
			req.Filter.TenantInfo,
			"trailer",
			resourceIDs,
		)
		if cfErr != nil {
			log.Warn("failed to load custom fields for trailers", zap.Error(cfErr))
		} else {
			for _, t := range result.Items {
				if fields, ok := customFieldsMap[t.GetResourceID()]; ok {
					t.CustomFields = fields
				}
			}
		}
	}

	return result, nil
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	customFields, cfErr := s.customFieldsValuesService.GetForResource(
		ctx,
		req.TenantInfo,
		entity.GetResourceType(),
		entity.GetResourceID(),
	)
	if cfErr != nil {
		s.l.Warn("failed to load custom fields for trailer", zap.Error(cfErr))
	} else {
		entity.CustomFields = customFields
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *trailer.Trailer,
	actor *services.RequestActor,
) (*trailer.Trailer, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create trailer", zap.Error(err))
		return nil, err
	}

	if len(entity.CustomFields) > 0 {
		if cfErr := s.customFieldsValuesService.ValidateAndSave(
			ctx,
			pagination.TenantInfo{
				OrgID: createdEntity.OrganizationID,
				BuID:  createdEntity.BusinessUnitID,
			},
			createdEntity.GetResourceType(),
			createdEntity.GetResourceID(),
			entity.CustomFields,
		); cfErr != nil {
			log.Warn("failed to save custom fields for trailer", zap.Error(cfErr))
			return nil, cfErr
		}
		createdEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceTrailer,
		ResourceID:     createdEntity.GetResourceID(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("Trailer created")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "trailers",
		Action:         "created",
		RecordID:       createdEntity.GetID(),
		Entity:         createdEntity,
	}); err != nil {
		log.Warn("failed to publish trailer invalidation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *trailer.Trailer,
	actor *services.RequestActor,
) (*trailer.Trailer, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetTrailerByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original trailer", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update trailer", zap.Error(err))
		return nil, err
	}

	if entity.CustomFields != nil {
		if cfErr := s.customFieldsValuesService.ValidateAndSave(
			ctx,
			pagination.TenantInfo{
				OrgID: updatedEntity.OrganizationID,
				BuID:  updatedEntity.BusinessUnitID,
			},
			updatedEntity.GetResourceType(),
			updatedEntity.GetResourceID(),
			entity.CustomFields,
		); cfErr != nil {
			log.Warn("failed to save custom fields for trailer", zap.Error(cfErr))
			return nil, cfErr
		}
		updatedEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceTrailer,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Trailer updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updatedEntity.GetOrganizationID(),
		BusinessUnitID: updatedEntity.GetBusinessUnitID(),
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "trailers",
		Action:         "updated",
		RecordID:       updatedEntity.GetID(),
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish trailer invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateTrailerStatusRequest,
) ([]*trailer.Trailer, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetTrailersByIDsRequest{
		TenantInfo: req.TenantInfo,
		TrailerIDs: req.TrailerIDs,
	})
	if err != nil {
		log.Error("failed to get original trailers", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update trailer status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*trailer.Trailer]{
			Resource:  permission.ResourceTrailer,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Trailer status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       "trailers",
		Action:         "bulk_updated",
	}); err != nil {
		log.Warn("failed to publish trailer invalidation", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Locate(
	ctx context.Context,
	req *repositories.LocateTrailerRequest,
	actor *services.RequestActor,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var result *equipmentcontinuity.EquipmentContinuity
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		actorUserID := actor.AuditActor().UserID

		trailerEntity, err := s.repo.GetByID(txCtx, repositories.GetTrailerByIDRequest{
			ID: req.TrailerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: req.TenantInfo.OrgID,
				BuID:  req.TenantInfo.BuID,
			},
		})
		if err != nil {
			return err
		}
		inProgress, err := s.assignmentRepo.FindInProgressByTrailerID(
			txCtx,
			req.TenantInfo,
			req.TrailerID,
			pulid.Nil,
		)
		if err != nil {
			return err
		}
		if inProgress != nil {
			return errortypes.NewBusinessError("Trailer is currently in progress on another move").
				WithParam("trailerId", req.TrailerID.String()).
				WithParam("shipmentMoveId", inProgress.ShipmentMoveID.String())
		}

		current, err := s.continuityRepo.GetEffectiveCurrent(
			txCtx,
			repositories.GetCurrentEquipmentContinuityRequest{
				TenantInfo:    req.TenantInfo,
				EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
				EquipmentID:   req.TrailerID,
			},
		)
		if err != nil {
			return err
		}
		if current == nil {
			return errortypes.NewBusinessError(
				"Trailer has no continuity history and does not need manual locate before dispatch",
			).WithParam("trailerId", req.TrailerID.String())
		}
		if current.SourceShipmentID.IsNil() {
			return errortypes.NewBusinessError(
				"Trailer continuity is missing the previous shipment association required for locate",
			).WithParam("trailerId", req.TrailerID.String())
		}
		if current.CurrentLocationID == req.NewLocationID {
			return errortypes.NewBusinessError("Trailer is already located at the requested location").
				WithParam("trailerId", req.TrailerID.String())
		}

		previousShipment, err := s.shipmentRepo.GetByID(txCtx, &repositories.GetShipmentByIDRequest{
			ID: current.SourceShipmentID,
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

		updatedShipment := cloneShipment(previousShipment)
		timing := locateMoveTiming()
		appendLocateMove(updatedShipment, current, req, timing)

		control, err := s.controlRepo.Get(txCtx, repositories.GetShipmentControlRequest{
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}

		if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
			previousShipment,
			updatedShipment,
			resolveDelayThresholdMinutes(control),
		); multiErr != nil {
			return multiErr
		}

		if err = s.commercial.Recalculate(txCtx, updatedShipment, control, actorUserID); err != nil {
			return err
		}
		if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(txCtx, previousShipment, updatedShipment); multiErr != nil {
			return multiErr
		}
		if _, err = s.shipmentRepo.Update(txCtx, updatedShipment); err != nil {
			return dberror.MapRetryableTransactionError(err, "Shipment is busy. Retry the request.")
		}

		newMove := updatedShipment.Moves[len(updatedShipment.Moves)-1]
		newAssignment, err := s.assignmentRepo.Create(txCtx, &shipment.Assignment{
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			ShipmentMoveID:  newMove.ID,
			TrailerID:       &req.TrailerID,
			Status:          shipment.AssignmentStatusCompleted,
			PrimaryWorkerID: nil,
			TractorID:       nil,
		})
		if err != nil {
			return err
		}
		systemUser, err := s.userRepo.GetSystemUser(txCtx, "id")
		if err != nil {
			return err
		}
		if err = s.createLocateComment(
			txCtx,
			req,
			systemUser.ID,
			trailerEntity,
			previousShipment,
			current.CurrentLocationID,
			req.NewLocationID,
		); err != nil {
			return err
		}
		result, err = s.continuityRepo.Advance(txCtx, repositories.CreateEquipmentContinuityRequest{
			TenantInfo:           req.TenantInfo,
			EquipmentType:        equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:          req.TrailerID,
			CurrentLocationID:    req.NewLocationID,
			SourceType:           equipmentcontinuity.SourceTypeManualLocate,
			SourceShipmentID:     updatedShipment.ID,
			SourceShipmentMoveID: newMove.ID,
			SourceAssignmentID:   newAssignment.ID,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func appendLocateMove(
	entity *shipment.Shipment,
	current *equipmentcontinuity.EquipmentContinuity,
	req *repositories.LocateTrailerRequest,
	timing locateTiming,
) {
	entity.Moves = append(entity.Moves, &shipment.ShipmentMove{
		ShipmentID:     entity.ID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Status:         shipment.MoveStatusCompleted,
		Loaded:         false,
		Sequence:       int64(len(entity.Moves)),
		Stops: []*shipment.Stop{
			{
				OrganizationID:       entity.OrganizationID,
				BusinessUnitID:       entity.BusinessUnitID,
				LocationID:           current.CurrentLocationID,
				Status:               shipment.StopStatusCompleted,
				Type:                 shipment.StopTypePickup,
				ScheduleType:         shipment.StopScheduleTypeOpen,
				Sequence:             0,
				ScheduledWindowStart: timing.pickupArrival,
				ScheduledWindowEnd:   &timing.pickupDeparture,
				ActualArrival:        &timing.pickupArrival,
				ActualDeparture:      &timing.pickupDeparture,
			},
			{
				OrganizationID:       entity.OrganizationID,
				BusinessUnitID:       entity.BusinessUnitID,
				LocationID:           req.NewLocationID,
				Status:               shipment.StopStatusCompleted,
				Type:                 shipment.StopTypeDelivery,
				ScheduleType:         shipment.StopScheduleTypeOpen,
				Sequence:             1,
				ScheduledWindowStart: timing.deliveryArrival,
				ScheduledWindowEnd:   &timing.deliveryDeparture,
				ActualArrival:        &timing.deliveryArrival,
				ActualDeparture:      &timing.deliveryDeparture,
			},
		},
	})
}

type locateTiming struct {
	pickupArrival     int64
	pickupDeparture   int64
	deliveryArrival   int64
	deliveryDeparture int64
}

func locateMoveTiming() locateTiming {
	current := timeutils.NowUnix()

	return locateTiming{
		pickupArrival:     current - 180,
		pickupDeparture:   current - 120,
		deliveryArrival:   current - 60,
		deliveryDeparture: current,
	}
}

func (s *Service) createLocateComment(
	ctx context.Context,
	req *repositories.LocateTrailerRequest,
	actorUserID pulid.ID,
	trailerEntity *trailer.Trailer,
	shipmentEntity *shipment.Shipment,
	fromLocationID pulid.ID,
	toLocationID pulid.ID,
) error {
	fromLocation, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID: fromLocationID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return err
	}

	toLocation, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID: toLocationID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return err
	}

	_, err = s.shipmentCommentRepo.Create(ctx, &shipment.ShipmentComment{
		ShipmentID:     shipmentEntity.ID,
		OrganizationID: shipmentEntity.OrganizationID,
		BusinessUnitID: shipmentEntity.BusinessUnitID,
		UserID:         actorUserID,
		Type:           shipment.CommentTypeDispatch,
		Visibility:     shipment.CommentVisibilityOperations,
		Priority:       shipment.CommentPriorityNormal,
		Source:         shipment.CommentSourceSystem,
		Metadata: map[string]any{
			"trailerId":      trailerEntity.ID,
			"fromLocationId": fromLocationID,
			"toLocationId":   toLocationID,
		},
		Comment: fmt.Sprintf(
			"System-generated empty reposition move created from trailer locate for trailer %s: %s -> %s.",
			trailerEntity.Code,
			locationLabel(fromLocation),
			locationLabel(toLocation),
		),
		MentionedUserIDs: []pulid.ID{},
	})
	return err
}

func locationLabel(entity *location.Location) string {
	if entity == nil {
		return ""
	}
	if entity.Name != "" {
		return entity.Name
	}
	return entity.ID.String()
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

func resolveDelayThresholdMinutes(control *tenant.ShipmentControl) int16 {
	if control == nil || !control.AutoDelayShipments {
		return shipmentstate.DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return shipmentstate.ResolveDelayThresholdMinutes(0)
	}

	return shipmentstate.ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}
