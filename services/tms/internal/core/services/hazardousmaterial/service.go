package hazardousmaterial

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/hazardousmaterialvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.HazardousMaterialRepository
	AuditService services.AuditService
	Validator    *hazardousmaterialvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.HazardousMaterialRepository
	as   services.AuditService
	v    *hazardousmaterialvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.hazardousmaterial"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListHazardousMaterialRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	return s.repo.GetByID(ctx, opts)
}

func (s *Service) Create(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
	userID pulid.ID,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("code", hm.Code),
		zap.String("buID", hm.BusinessUnitID.String()),
		zap.String("orgID", hm.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hm); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hm)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazardousMaterial,
			ResourceID:     createdEntity.GetID(),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
			UserID:         userID,
			Operation:      permission.OpCreate,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
		},
		audit.WithComment("Hazardous Material created"),
	)
	if err != nil {
		log.Error("failed to log hazardous material creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
	userID pulid.ID,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("code", hm.Code),
		zap.String("buID", hm.BusinessUnitID.String()),
		zap.String("orgID", hm.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hm); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetHazardousMaterialByIDRequest{
		ID:    hm.ID,
		OrgID: hm.OrganizationID,
		BuID:  hm.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, hm)
	if err != nil {
		log.Error("failed to update hazardous material", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazardousMaterial,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Hazardous Material updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log hazardous material update", zap.Error(err))
	}

	return updatedEntity, nil
}
