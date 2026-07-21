package recurringshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	patternLookbackDays = 90
	patternMinShipments = 3
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.RecurringShipmentRepository
	Validator    *Validator
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RecurringShipmentRepository
	validator    *Validator
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.recurring-shipment"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRecurringShipmentsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRecurringShipmentConnectionRequest,
) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetRecurringShipmentByIDRequest,
) (*recurringshipment.RecurringShipment, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.RecurringShipmentSelectOptionsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipment], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if entity.Status == "" {
		entity.Status = recurringshipment.StatusActive
	}

	if entity.ExceptionPolicy == "" {
		entity.ExceptionPolicy = recurringshipment.ExceptionPolicySkip
	}

	entity.EnteredByID = userID

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create recurring shipment", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRecurringShipment,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Recurring shipment created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, createdEntity, userID, "created")

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetRecurringShipmentByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original recurring shipment", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update recurring shipment", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRecurringShipment,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Recurring shipment updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, updatedEntity, userID, "updated")

	return updatedEntity, nil
}

func (s *Service) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateRecurringShipmentStatusRequest,
	userID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	log := s.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("userID", userID.String()),
	)

	updatedEntity, err := s.repo.UpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to update recurring shipment status", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRecurringShipment,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Recurring shipment status changed to "+string(req.Status)),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, updatedEntity, userID, "updated")

	return updatedEntity, nil
}

func (s *Service) Match(
	ctx context.Context,
	req *repositories.MatchRecurringShipmentsRequest,
) (*repositories.MatchRecurringShipmentsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	matches, err := s.repo.Match(ctx, req)
	if err != nil {
		return nil, err
	}

	response := &repositories.MatchRecurringShipmentsResponse{Matches: matches}

	// Only hunt for a historical lane pattern when no active series covers the
	// lane — the pattern hint exists to prompt series creation.
	if len(matches) == 0 {
		pattern, patternErr := s.repo.DetectLanePattern(ctx, &repositories.DetectLanePatternRequest{
			TenantInfo:            req.TenantInfo,
			CustomerID:            req.CustomerID,
			OriginLocationID:      req.OriginLocationID,
			DestinationLocationID: req.DestinationLocationID,
			LookbackDays:          patternLookbackDays,
			MinShipments:          patternMinShipments,
		})
		if patternErr != nil {
			s.l.Warn("failed to detect lane pattern", zap.Error(patternErr))
		} else {
			response.Pattern = pattern
		}
	}

	return response, nil
}

func (s *Service) Generate(
	ctx context.Context,
	req *repositories.GenerateRecurringShipmentRequest,
) (*repositories.GenerateRecurringShipmentResult, error) {
	log := s.l.With(
		zap.String("operation", "Generate"),
		zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
	)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	if req.Trigger == "" {
		req.Trigger = recurringshipment.RunTriggerManual
	}

	result, err := s.repo.Generate(ctx, req)
	if err != nil {
		log.Error("failed to generate shipment from recurring series", zap.Error(err))
		return nil, err
	}

	if result.Shipment != nil {
		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     result.Shipment.GetID().String(),
			Operation:      permission.OpCreate,
			UserID:         req.RequestedBy,
			CurrentState:   jsonutils.MustToJSON(result.Shipment),
			OrganizationID: result.Shipment.OrganizationID,
			BusinessUnitID: result.Shipment.BusinessUnitID,
		},
			auditservice.WithComment("Shipment generated from recurring series"),
		); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}

		if s.realtime != nil {
			if publishErr := realtimeinvalidation.Publish(
				ctx,
				s.realtime,
				&realtimeinvalidation.PublishParams{
					OrganizationID: result.Shipment.OrganizationID,
					BusinessUnitID: result.Shipment.BusinessUnitID,
					ActorUserID:    req.RequestedBy,
					ActorType:      services.PrincipalTypeUser,
					ActorID:        req.RequestedBy,
					Resource:       "shipments",
					Action:         "created",
					RecordID:       result.Shipment.ID,
					Entity:         result.Shipment,
				},
			); publishErr != nil {
				log.Warn("failed to publish generated shipment invalidation", zap.Error(publishErr))
			}
		}
	}

	if result.Series != nil {
		s.publishInvalidation(ctx, result.Series, req.RequestedBy, "updated")
	}

	return result, nil
}

func (s *Service) ListRuns(
	ctx context.Context,
	req *repositories.ListRecurringShipmentRunsRequest,
) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error) {
	return s.repo.ListRuns(ctx, req)
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
	userID pulid.ID,
	action string,
) {
	if s.realtime == nil {
		return
	}

	if err := realtimeinvalidation.Publish(
		ctx,
		s.realtime,
		&realtimeinvalidation.PublishParams{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			ActorUserID:    userID,
			ActorType:      services.PrincipalTypeUser,
			ActorID:        userID,
			Resource:       "recurring-shipments",
			Action:         action,
			RecordID:       entity.ID,
			Entity:         entity,
		},
	); err != nil {
		s.l.Warn("failed to publish recurring shipment invalidation", zap.Error(err))
	}
}
