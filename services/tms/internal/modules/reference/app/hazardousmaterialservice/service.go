package hazardousmaterialservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         hazardousmaterialports.HazardousMaterialRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Transformer  accessorialchargeports.DataTransformer
}

type Service struct {
	l            *zap.Logger
	repo         hazardousmaterialports.HazardousMaterialRepository
	validator    *Validator
	auditService auditports.AuditService
	transformer  accessorialchargeports.DataTransformer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.hazardousmaterial"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		transformer:  p.Transformer,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *hazardousmaterialports.ListHazardousMaterialsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *hazardousmaterialports.ListHazardousMaterialConnectionRequest,
) (*pagination.CursorListResult[*hazardousmaterial.HazardousMaterial], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req hazardousmaterialports.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *hazardousmaterialports.HazardousMaterialSelectOptionsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	changes, err := s.PlanBulkUpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}
	originalEntities := kernelports.Befores(changes)

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update hazardous material status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*hazardousmaterial.HazardousMaterial]{
			Resource:  permission.ResourceHazardousMaterial,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Hazardous material status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
	actor *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	auditActor := actor.AuditActor()
	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, planned)
	if err != nil {
		s.l.Error("failed to create hazardous material", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceHazardousMaterial,
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
		auditservice.WithComment("Hazardous material created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
	actor *permissionports.RequestActor,
) (*hazardousmaterial.HazardousMaterial, error) {
	auditActor := actor.AuditActor()
	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}
	original := change.Before

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update hazardous material", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceHazardousMaterial,
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
		auditservice.WithComment("Hazardous material updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
