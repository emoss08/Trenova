package trailer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/trailervalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.TrailerRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *trailervalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.TrailerRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *trailervalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "trailer").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *repositories.ListTrailerOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, t := range result.Items {
		options = append(options, &types.SelectOption{
			Value: t.GetID(),
			Label: t.Code,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListTrailerOptions,
) (*ports.ListResult[*trailer.Trailer], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceTrailer,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read trailers")
	}

	return s.repo.List(ctx, opts)
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetTrailerByIDOptions,
) (*trailer.Trailer, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("trailerID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceTrailer,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this trailer")
	}

	return s.repo.GetByID(ctx, opts)
}

func (s *Service) Create(
	ctx context.Context,
	tr *trailer.Trailer,
	userID pulid.ID,
) (*trailer.Trailer, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", tr.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceTrailer,
				Action:         permission.ActionCreate,
				BusinessUnitID: tr.BusinessUnitID,
				OrganizationID: tr.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a trailer")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, tr); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, tr)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTrailer,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Trailer created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log trailer creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	t *trailer.Trailer,
	userID pulid.ID,
) (*trailer.Trailer, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", t.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceTrailer,
				Action:         permission.ActionUpdate,
				BusinessUnitID: t.BusinessUnitID,
				OrganizationID: t.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this trailer",
		)
	}

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, t); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetTrailerByIDOptions{
		ID:    t.ID,
		OrgID: t.OrganizationID,
		BuID:  t.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, t)
	if err != nil {
		log.Error().Err(err).Msg("failed to update trailer")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTrailer,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
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
		log.Error().Err(err).Msg("failed to log trailer update")
	}

	return updatedEntity, nil
}
