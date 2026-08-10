package routingguideservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.RoutingGuideRepository
	CarrierRepo  repositories.CarrierRepository
	Validator    *Validator
	AuditService services.AuditService
	Realtime     services.RealtimeService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RoutingGuideRepository
	carrierRepo  repositories.CarrierRepository
	validator    *Validator
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.routing-guide"),
		repo:         p.Repo,
		carrierRepo:  p.CarrierRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRoutingGuideRequest,
) (*pagination.ListResult[*tender.RoutingGuide], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRoutingGuideConnectionRequest,
) (*pagination.CursorListResult[*tender.RoutingGuide], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetRoutingGuideByIDRequest,
) (*tender.RoutingGuide, error) {
	return s.repo.GetByID(ctx, req)
}

// MatchLane resolves the most specific active guide for a lane, used both by
// tender creation and by the dispatch preview.
func (s *Service) MatchLane(
	ctx context.Context,
	req *repositories.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	return s.repo.MatchLane(ctx, req)
}

// validateEDIEntries rejects EDI-channel entries whose carrier has no default
// EDI channel at save time, so a waterfall never discovers the gap mid-flight.
func (s *Service) validateEDIEntries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entries []*tender.RoutingGuideEntry,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for idx, entry := range entries {
		if entry == nil || entry.Channel != tender.ChannelEDI {
			continue
		}
		channel, err := s.carrierRepo.GetDefaultEDIChannel(
			ctx,
			repositories.GetCarrierEDIChannelRequest{
				TenantInfo: tenantInfo,
				CarrierID:  entry.CarrierID,
			},
		)
		if err != nil {
			s.l.Error("failed to resolve carrier EDI channel", zap.Error(err))
			multiErr.WithIndex("entries", idx).
				Add("channel", errortypes.ErrInvalid, "Could not verify the carrier's EDI channel")
			continue
		}
		if channel == nil {
			multiErr.WithIndex("entries", idx).
				Add(
					"channel",
					errortypes.ErrInvalid,
					"Carrier has no default EDI channel. Link an EDI partner on the carrier or use the Email channel",
				)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *tender.RoutingGuide,
	actor *services.RequestActor,
) (*tender.RoutingGuide, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	if multiErr := s.validateEDIEntries(ctx, tenantInfo, entity.Entries); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create routing guide", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRoutingGuide,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Routing guide created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, createdEntity, auditActor, "created")

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *tender.RoutingGuide,
	actor *services.RequestActor,
) (*tender.RoutingGuide, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.GetID().String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	if multiErr := s.validateEDIEntries(ctx, tenantInfo, entity.Entries); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetRoutingGuideByIDRequest{
		ID:         entity.GetID(),
		TenantInfo: tenantInfo,
		RoutingGuideFilterOptions: repositories.RoutingGuideFilterOptions{
			IncludeEntries: true,
		},
	})
	if err != nil {
		log.Error("failed to get original routing guide", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update routing guide", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRoutingGuide,
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
		auditservice.WithComment("Routing guide updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, updatedEntity, auditActor, "updated")

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteRoutingGuideRequest,
	actor *services.RequestActor,
) error {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	original, err := s.repo.GetByID(ctx, repositories.GetRoutingGuideByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		RoutingGuideFilterOptions: repositories.RoutingGuideFilterOptions{
			IncludeEntries: true,
		},
	})
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete routing guide", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRoutingGuide,
		ResourceID:     req.ID.String(),
		Operation:      permission.OpDelete,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	},
		auditservice.WithComment(fmt.Sprintf("Routing guide %q deleted", original.Name)),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	s.publishInvalidation(ctx, original, auditActor, "deleted")

	return nil
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	entity *tender.RoutingGuide,
	auditActor services.AuditActor,
	action string,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "routing_guides",
		Action:         action,
		RecordID:       entity.GetID(),
	}); err != nil {
		s.l.Warn("failed to publish routing guide invalidation", zap.Error(err))
	}
}
