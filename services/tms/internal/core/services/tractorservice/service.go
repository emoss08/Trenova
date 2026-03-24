package tractorservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                    *zap.Logger
	Repo                      repositories.TractorRepository
	Validator                 *Validator
	AuditService              services.AuditService
	Realtime                  services.RealtimeService
	CustomFieldsValuesService *customfieldservice.ValuesService
}

type Service struct {
	l                         *zap.Logger
	repo                      repositories.TractorRepository
	validator                 *Validator
	auditService              services.AuditService
	realtime                  services.RealtimeService
	customFieldsValuesService *customfieldservice.ValuesService
}

func New(p Params) *Service {
	return &Service{
		l:                         p.Logger.Named("service.tractor"),
		repo:                      p.Repo,
		validator:                 p.Validator,
		auditService:              p.AuditService,
		realtime:                  p.Realtime,
		customFieldsValuesService: p.CustomFieldsValuesService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTractorsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	log := s.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	log.Info("listing tractors")
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
			"tractor",
			resourceIDs,
		)
		if cfErr != nil {
			log.Warn("failed to load custom fields for tractors", zap.Error(cfErr))
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
	req *repositories.TractorSelectOptionsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
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
		s.l.Warn("failed to load custom fields for tractor", zap.Error(cfErr))
	} else {
		entity.CustomFields = customFields
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *tractor.Tractor,
	actor *services.RequestActor,
) (*tractor.Tractor, error) {
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
		log.Error("failed to create tractor", zap.Error(err))
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
			log.Warn("failed to save custom fields for tractor", zap.Error(cfErr))
			return nil, cfErr
		}
		createdEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceTractor,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	}, auditservice.WithComment("Tractor created")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "tractors",
		Action:         "created",
		RecordID:       createdEntity.GetID(),
		Entity:         createdEntity,
	}); err != nil {
		log.Warn("failed to publish tractor invalidation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *tractor.Tractor,
	actor *services.RequestActor,
) (*tractor.Tractor, error) {
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

	original, err := s.repo.GetByID(ctx, repositories.GetTractorByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original tractor", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update tractor", zap.Error(err))
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
			log.Warn("failed to save custom fields for tractor", zap.Error(cfErr))
			return nil, cfErr
		}
		updatedEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceTractor,
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
		auditservice.WithComment("Tractor updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       "tractors",
		Action:         "updated",
		RecordID:       updatedEntity.GetID(),
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish tractor invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateTractorStatusRequest,
) ([]*tractor.Tractor, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetTractorsByIDsRequest{
		TenantInfo: req.TenantInfo,
		TractorIDs: req.TractorIDs,
	})
	if err != nil {
		log.Error("failed to get original tractors", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update tractor status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*tractor.Tractor]{
			Resource:  permission.ResourceTractor,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Tractor status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       permission.ResourceTractor.String(),
		Action:         "bulk_updated",
	}); err != nil {
		log.Warn("failed to publish tractor invalidation", zap.Error(err))
	}

	return entities, nil
}
