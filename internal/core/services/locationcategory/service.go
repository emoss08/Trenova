package locationcategory

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/location"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/locationvalidator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.LocationCategoryRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *locationvalidator.LocationCategoryValidator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.LocationCategoryRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *locationvalidator.LocationCategoryValidator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "locationcategory").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(ctx context.Context, opts *ports.LimitOffsetQueryOptions) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "select location categories")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, lc := range result.Items {
		options[i] = &types.SelectOption{
			Value: lc.GetID(),
			Label: lc.Name,
			Color: lc.Color,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*location.LocationCategory], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceLocationCategory,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read location categories")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list location categories")
		return nil, eris.Wrap(err, "list location categories")
	}

	return &ports.ListResult[*location.LocationCategory]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetLocationCategoryByIDOptions) (*location.LocationCategory, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("locationCategoryID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceLocationCategory,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read location category permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this location category")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get location category")
		return nil, eris.Wrap(err, "get location category")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, lc *location.LocationCategory, userID pulid.ID) (*location.LocationCategory, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", lc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceLocationCategory,
				Action:         permission.ActionCreate,
				BusinessUnitID: lc.BusinessUnitID,
				OrganizationID: lc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create location category permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a location category")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, lc); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, lc)
	if err != nil {
		return nil, eris.Wrap(err, "create location category")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceLocationCategory,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Location category created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log location category creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, lc *location.LocationCategory, userID pulid.ID) (*location.LocationCategory, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", lc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceLocationCategory,
				Action:         permission.ActionUpdate,
				BusinessUnitID: lc.BusinessUnitID,
				OrganizationID: lc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update location category permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this location category")
	}

	// Validate the location category
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, lc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetLocationCategoryByIDOptions{
		ID:    lc.ID,
		OrgID: lc.OrganizationID,
		BuID:  lc.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get location category")
	}

	updatedEntity, err := s.repo.Update(ctx, lc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update location category")
		return nil, eris.Wrap(err, "update location category")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceLocationCategory,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Location category updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log location category update")
	}

	return updatedEntity, nil
}
