package tractorservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                    *zap.Logger
	DB                        ports.DBConnection
	Repo                      repositories.TractorRepository
	AssignmentRepo            repositories.AssignmentRepository
	ContinuityRepo            repositories.EquipmentContinuityRepository
	LocationRepo              repositories.LocationRepository
	Validator                 *Validator
	AuditService              services.AuditService
	Realtime                  services.RealtimeService
	CustomFieldsValuesService *customfieldservice.ValuesService
}

type Service struct {
	l                         *zap.Logger
	db                        ports.DBConnection
	repo                      repositories.TractorRepository
	assignmentRepo            repositories.AssignmentRepository
	continuityRepo            repositories.EquipmentContinuityRepository
	locationRepo              repositories.LocationRepository
	validator                 *Validator
	auditService              services.AuditService
	realtime                  services.RealtimeService
	customFieldsValuesService *customfieldservice.ValuesService
}

func New(p Params) *Service {
	return &Service{
		l:                         p.Logger.Named("service.tractor"),
		db:                        p.DB,
		repo:                      p.Repo,
		assignmentRepo:            p.AssignmentRepo,
		continuityRepo:            p.ContinuityRepo,
		locationRepo:              p.LocationRepo,
		validator:                 p.Validator,
		auditService:              p.AuditService,
		realtime:                  p.Realtime,
		customFieldsValuesService: p.CustomFieldsValuesService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	log := s.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	if shouldIncludeTractorCustomFields(req.TractorRelationIncludes) {
		err = s.attachCustomFields(ctx, req.Filter.TenantInfo, result.Items)
	}
	if err != nil {
		log.Warn("failed to load custom fields for tractors", zap.Error(err))
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

	if shouldIncludeTractorCustomFields(req.TractorRelationIncludes) {
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
	}

	return entity, nil
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req repositories.GetTractorsByIDsRequest,
) ([]*tractor.Tractor, error) {
	entities, err := s.repo.GetByIDs(ctx, req)
	if err != nil {
		return nil, err
	}

	if shouldIncludeTractorCustomFields(req.TractorRelationIncludes) {
		err = s.attachCustomFields(ctx, req.TenantInfo, entities)
	}
	if err != nil {
		s.l.Warn("failed to load custom fields for tractors", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) attachCustomFields(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entities []*tractor.Tractor,
) error {
	if len(entities) == 0 {
		return nil
	}

	resourceIDs := make([]string, 0, len(entities))
	for _, t := range entities {
		resourceIDs = append(resourceIDs, t.GetResourceID())
	}

	customFieldsMap, err := s.customFieldsValuesService.GetForResources(
		ctx,
		tenantInfo,
		"tractor",
		resourceIDs,
	)
	if err != nil {
		return err
	}

	for _, t := range entities {
		if fields, ok := customFieldsMap[t.GetResourceID()]; ok {
			t.CustomFields = fields
		}
	}

	return nil
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

	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, planned)
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

	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}
	original := change.Before

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

func (s *Service) Locate(
	ctx context.Context,
	req *repositories.LocateTractorRequest,
) (*equipmentcontinuity.EquipmentContinuity, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	var result *equipmentcontinuity.EquipmentContinuity
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := s.planLocate(txCtx, req); err != nil {
			return err
		}

		var advanceErr error
		result, advanceErr = s.continuityRepo.Advance(
			txCtx,
			repositories.CreateEquipmentContinuityRequest{
				TenantInfo:        req.TenantInfo,
				EquipmentType:     equipmentcontinuity.EquipmentTypeTractor,
				EquipmentID:       req.TractorID,
				CurrentLocationID: req.NewLocationID,
				SourceType:        equipmentcontinuity.SourceTypeManualLocate,
			},
		)
		return advanceErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func shouldIncludeTractorCustomFields(includes repositories.TractorRelationIncludes) bool {
	return includes.IncludeCustomFields || len(includes.TractorColumns) == 0
}
