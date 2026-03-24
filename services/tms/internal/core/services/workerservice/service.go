package workerservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                    *zap.Logger
	Repo                      repositories.WorkerRepository
	CacheRepo                 repositories.WorkerCacheRepository
	UserRepo                  repositories.UserRepository
	TemporalClient            client.Client
	Validator                 *Validator
	AuditService              services.AuditService
	Realtime                  services.RealtimeService
	CustomFieldsValuesService *customfieldservice.ValuesService
}

type Service struct {
	l                         *zap.Logger
	repo                      repositories.WorkerRepository
	cacheRepo                 repositories.WorkerCacheRepository
	userRepo                  repositories.UserRepository
	temporalClient            client.Client
	auditService              services.AuditService
	realtime                  services.RealtimeService
	customFieldsValuesService *customfieldservice.ValuesService
	validator                 *Validator
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                         p.Logger.Named("service.worker"),
		repo:                      p.Repo,
		cacheRepo:                 p.CacheRepo,
		userRepo:                  p.UserRepo,
		temporalClient:            p.TemporalClient,
		auditService:              p.AuditService,
		realtime:                  p.Realtime,
		customFieldsValuesService: p.CustomFieldsValuesService,
		validator:                 p.Validator,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.WorkerSelectOptionsRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.ListResult[*worker.Worker], error) {
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
		for _, w := range result.Items {
			resourceIDs = append(resourceIDs, w.GetResourceID())
		}

		customFieldsMap, cfErr := s.customFieldsValuesService.GetForResources(
			ctx,
			req.Filter.TenantInfo,
			"worker",
			resourceIDs,
		)
		if cfErr != nil {
			log.Warn("failed to load custom fields for workers", zap.Error(cfErr))
		} else {
			for _, w := range result.Items {
				if fields, ok := customFieldsMap[w.GetResourceID()]; ok {
					w.CustomFields = fields
				}
			}
		}
	}

	return result, nil
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	log := s.l.With(
		zap.String("operation", "Get"),
		zap.String("workerID", req.ID.String()),
		zap.String("buID", req.TenantInfo.BuID.String()),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	var (
		entity *worker.Worker
		err    error
	)

	entity, err = s.cacheRepo.GetByID(ctx, req)
	if err != nil {
		if !errors.Is(err, repositories.ErrCacheMiss) {
			log.Warn("failed to load worker from cache", zap.Error(err))
		}
		entity, err = s.repo.GetByID(ctx, req)
	}
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
		log.Warn("failed to load custom fields for worker", zap.Error(cfErr))
	} else {
		entity.CustomFields = customFields
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *worker.Worker,
	actor *services.RequestActor,
) (*worker.Worker, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create worker", zap.Error(err))
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
			log.Warn("failed to save custom fields for worker", zap.Error(cfErr))
			return nil, cfErr
		}

		createdEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceWorker,
		ResourceID:     createdEntity.GetResourceID(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("Worker created")); err != nil {
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
		Resource:       "workers",
		Action:         "created",
		RecordID:       createdEntity.GetID(),
		Entity:         createdEntity,
	}); err != nil {
		log.Warn("failed to publish worker invalidation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.Worker,
	actor *services.RequestActor,
) (*worker.Worker, error) {
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

	original, err := s.repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original worker", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update worker", zap.Error(err))
		return nil, err
	}

	if entity.CustomFields != nil {
		if cfErr := s.customFieldsValuesService.ValidateAndSave(
			ctx,
			pagination.TenantInfo{
				OrgID: updatedEntity.GetOrganizationID(),
				BuID:  updatedEntity.GetBusinessUnitID(),
			},
			updatedEntity.GetResourceType(),
			updatedEntity.GetResourceID(),
			entity.CustomFields,
		); cfErr != nil {
			log.Warn("failed to save custom fields for worker", zap.Error(cfErr))
			return nil, cfErr
		}

		updatedEntity.CustomFields = entity.CustomFields
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceWorker,
		ResourceID:     updatedEntity.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.GetOrganizationID(),
		BusinessUnitID: updatedEntity.GetBusinessUnitID(),
	},
		auditservice.WithComment("Worker updated"),
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
		Resource:       permission.ResourceWorker.String(),
		Action:         "updated",
		RecordID:       updatedEntity.GetID(),
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish worker invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}
