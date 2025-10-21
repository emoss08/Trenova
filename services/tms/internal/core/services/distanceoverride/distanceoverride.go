package distanceoverride

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/distanceoverridevalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DistanceOverrideRepository
	AuditService services.AuditService
	Validator    *distanceoverridevalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.DistanceOverrideRepository
	as   services.AuditService
	v    *distanceoverridevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.distanceoverride"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDistanceOverrideRequest,
) (*pagination.ListResult[*distanceoverride.Override], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetDistanceOverrideRequest,
) (*distanceoverride.Override, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *distanceoverride.Override,
	userID pulid.ID,
) (*distanceoverride.Override, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("id", entity.GetID()),
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
			Resource:       permission.ResourceDistanceOverride,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Distance override created"),
	)
	if err != nil {
		log.Error("failed to log distance override creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *distanceoverride.Override,
	userID pulid.ID,
) (*distanceoverride.Override, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.GetID()),
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

	original, err := s.repo.GetByID(ctx, &repositories.GetDistanceOverrideRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update distance override", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDistanceOverride,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Distance override updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log distance override update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteDistanceOverrideRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	existing, err := s.repo.GetByID(ctx, &repositories.GetDistanceOverrideRequest{
		ID:    req.ID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error("failed to get distance override", zap.Error(err))
		return err
	}

	err = s.repo.Delete(ctx, req)
	if err != nil {
		log.Error("failed to delete distance override", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDistanceOverride,
			ResourceID:     req.ID.String(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Distance override deleted"),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log distance override deletion", zap.Error(err))
		return err
	}

	return nil
}
