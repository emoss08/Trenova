package ratetableservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.RateTableRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RateTableRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.rate-table"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateTablesRequest,
) (*pagination.ListResult[*ratetable.RateTable], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRateTableConnectionRequest,
) (*pagination.CursorListResult[*ratetable.RateTable], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateTableByIDRequest,
) (*ratetable.RateTable, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.RateTableSelectOptionsRequest,
) (*pagination.ListResult[*ratetable.RateTable], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *ratetable.RateTable,
	userID pulid.ID,
) (*ratetable.RateTable, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create rate table", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRateTable,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Rate table created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *ratetable.RateTable,
	userID pulid.ID,
) (*ratetable.RateTable, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetRateTableByIDRequest{
		RateTableID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original rate table", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update rate table", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRateTable,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Rate table updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.GetRateTableByIDRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.RateTableID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to get rate table for delete", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete rate table", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceRateTable,
		ResourceID:     existing.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: existing.OrganizationID,
		BusinessUnitID: existing.BusinessUnitID,
	},
		auditservice.WithComment("Rate table deleted"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}
