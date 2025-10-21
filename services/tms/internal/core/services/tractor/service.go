package tractor

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/tractorvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.TractorRepository
	AuditService services.AuditService
	Validator    *tractorvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.TractorRepository
	as   services.AuditService
	v    *tractorvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.tractor"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *tractor.Tractor,
	userID pulid.ID,
) (*tractor.Tractor, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTractor,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Tractor created"),
	)
	if err != nil {
		log.Error("failed to log tractor creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *tractor.Tractor,
	userID pulid.ID,
) (*tractor.Tractor, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("code", entity.Code),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetTractorByIDRequest{
		TractorID: entity.ID,
		OrgID:     entity.OrganizationID,
		BuID:      entity.BusinessUnitID,
		FilterOptions: repositories.TractorFilterOptions{
			IncludeWorkerDetails:    true,
			IncludeEquipmentDetails: true,
			IncludeFleetDetails:     true,
		},
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update tractor", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTractor,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Tractor updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log tractor update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Assignment(
	ctx context.Context,
	req repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	log := s.l.With(
		zap.String("operation", "Assignment"),
		zap.String("tractorID", req.TractorID.String()),
	)

	assignment, err := s.repo.Assignment(ctx, req)
	if err != nil {
		log.Error("failed to get tractor assignment", zap.Error(err))
		return nil, err
	}

	return assignment, nil
}
