package trailer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/trailervalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.TrailerRepository
	AuditService services.AuditService
	Validator    *trailervalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.TrailerRepository
	as   services.AuditService
	v    *trailervalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.trailer"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTrailerRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *trailer.Trailer,
	userID pulid.ID,
) (*trailer.Trailer, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
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
			Resource:       permission.ResourceTrailer,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Trailer created"),
	)
	if err != nil {
		log.Error("failed to log trailer creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *trailer.Trailer,
	userID pulid.ID,
) (*trailer.Trailer, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
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

	original, err := s.repo.GetByID(ctx, &repositories.GetTrailerByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update trailer", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTrailer,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Trailer updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log trailer update", zap.Error(err))
	}

	return updatedEntity, nil
}
