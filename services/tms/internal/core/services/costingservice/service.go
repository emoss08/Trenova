package costingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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
	Repo         repositories.CostingControlRepository
	ActualsRepo  repositories.CostingActualsRepository
	PriceRepo    repositories.FuelIndexPriceRepository
	ShipmentRepo repositories.ShipmentRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.CostingControlRepository
	actualsRepo  repositories.CostingActualsRepository
	priceRepo    repositories.FuelIndexPriceRepository
	shipmentRepo repositories.ShipmentRepository
	validator    *Validator
	auditService services.AuditService
	now          func() time.Time
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:            p.Logger.Named("service.costing"),
		repo:         p.Repo,
		actualsRepo:  p.ActualsRepo,
		priceRepo:    p.PriceRepo,
		shipmentRepo: p.ShipmentRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
		now:          time.Now,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetCostingControlRequest,
) (*costingcontrol.CostingControl, error) {
	return s.repo.GetByOrgID(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	entity *costingcontrol.CostingControl,
	userID pulid.ID,
) (*costingcontrol.CostingControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByOrgID(ctx, &repositories.GetCostingControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		log.Error("failed to get original costing control", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update costing control", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCostingControl,
			ResourceID:     updatedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		auditservice.WithComment("Costing control updated"),
		auditservice.WithDiff(original, updatedEntity),
		auditservice.WithCritical(),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

type UpdateCategoryParams struct {
	Category     *costingcontrol.CostCategory
	GLAccountIDs []pulid.ID
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
}

func (s *Service) UpdateCategory(
	ctx context.Context,
	params *UpdateCategoryParams,
) (*costingcontrol.CostCategory, error) {
	log := s.l.With(
		zap.String("operation", "UpdateCategory"),
		zap.String("categoryID", params.Category.ID.String()),
	)

	if multiErr := s.validator.ValidateCategoryUpdate(ctx, params.Category); multiErr != nil {
		return nil, multiErr
	}

	updatedCategory, err := s.repo.UpdateCategory(ctx, params.Category)
	if err != nil {
		log.Error("failed to update cost category", zap.Error(err))
		return nil, err
	}

	if err = s.repo.ReplaceCategoryGLAccounts(
		ctx,
		&repositories.ReplaceCategoryGLAccountsRequest{
			TenantInfo:     params.TenantInfo,
			CostCategoryID: updatedCategory.ID,
			GLAccountIDs:   params.GLAccountIDs,
		},
	); err != nil {
		log.Error("failed to replace cost category GL accounts", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCostingControl,
			ResourceID:     updatedCategory.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         params.UserID,
			CurrentState:   jsonutils.MustToJSON(updatedCategory),
			OrganizationID: updatedCategory.OrganizationID,
			BusinessUnitID: updatedCategory.BusinessUnitID,
		},
		auditservice.WithComment("Cost category updated"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedCategory, nil
}
