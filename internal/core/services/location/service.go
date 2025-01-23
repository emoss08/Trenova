package location

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
	Repo         repositories.LocationRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *locationvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.LocationRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *locationvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "location").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(ctx context.Context, opts *repositories.ListLocationOptions) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "select locations")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, loc := range result.Items {
		options[i] = &types.SelectOption{
			Value: loc.GetID(),
			Label: loc.Name,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *repositories.ListLocationOptions) (*ports.ListResult[*location.Location], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceLocation,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read locations")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list locations")
		return nil, eris.Wrap(err, "list locations")
	}

	return &ports.ListResult[*location.Location]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetLocationByIDOptions) (*location.Location, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("hmID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceLocation,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read location permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this location")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get location")
		return nil, eris.Wrap(err, "get location")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, loc *location.Location, userID pulid.ID) (*location.Location, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", loc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceLocation,
				Action:         permission.ActionCreate,
				BusinessUnitID: loc.BusinessUnitID,
				OrganizationID: loc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create location permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a location")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, loc); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, loc)
	if err != nil {
		return nil, eris.Wrap(err, "create location")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceLocation,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Location created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log location creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, loc *location.Location, userID pulid.ID) (*location.Location, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", loc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceLocation,
				Action:         permission.ActionUpdate,
				BusinessUnitID: loc.BusinessUnitID,
				OrganizationID: loc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update location permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this location")
	}

	// Validate the location
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, loc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetLocationByIDOptions{
		ID:    loc.ID,
		OrgID: loc.OrganizationID,
		BuID:  loc.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get location")
	}

	updatedEntity, err := s.repo.Update(ctx, loc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update location")
		return nil, eris.Wrap(err, "update location")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceLocation,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Location updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log location update")
	}

	return updatedEntity, nil
}
