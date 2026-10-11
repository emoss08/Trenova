package commodityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         commodityports.CommodityRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Transformer  accessorialchargeports.DataTransformer
}

type Service struct {
	l            *zap.Logger
	repo         commodityports.CommodityRepository
	validator    *Validator
	auditService auditports.AuditService
	transformer  accessorialchargeports.DataTransformer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.commodity"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		transformer:  p.Transformer,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *commodityports.ListCommodityRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *commodityports.ListCommodityConnectionRequest,
) (*pagination.CursorListResult[*commodity.Commodity], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req commodityports.GetCommodityByIDRequest,
) (*commodity.Commodity, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *commodityports.CommoditySelectOptionsRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *commodityports.BulkUpdateCommodityStatusRequest,
) ([]*commodity.Commodity, error) {
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
		log.Error("failed to bulk update commodity status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*commodity.Commodity]{
			Resource:  permission.ResourceCommodity,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Commodity status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *commodity.Commodity,
	actor *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	planned, err := s.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, planned)
	if err != nil {
		log.Error("failed to create commodity", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCommodity,
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
		auditservice.WithComment("Commodity created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *commodity.Commodity,
	actor *permissionports.RequestActor,
) (*commodity.Commodity, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	change, err := s.PlanUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}
	original := change.Before

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update commodity", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCommodity,
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
		auditservice.WithComment("Commodity updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
