package user

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.UserRepository
	AuditService services.AuditService
	PermService  services.PermissionService
}

type Service struct {
	repo repositories.UserRepository
	l    *zerolog.Logger
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "user").
		Logger()

	return &Service{
		repo: p.Repo,
		l:    &log,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, repositories.ListUserRequest{
		Filter: opts,
		// IncludeRoles: true,
	})
	if err != nil {
		return nil, eris.Wrap(err, "select users")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, u := range result.Items {
		options[i] = &types.SelectOption{
			Value: u.ID.String(),
			Label: u.Name,
		}
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts repositories.ListUserRequest,
) (*ports.ListResult[*user.User], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read users")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list users")
		return nil, err
	}

	return &ports.ListResult[*user.User]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetUserByIDOptions,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.UserID.String()).
		Logger()

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Update(
	ctx context.Context,
	u *user.User,
	userID pulid.ID,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("userID", u.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceUser,
			Action:         permission.ActionUpdate,
			BusinessUnitID: u.BusinessUnitID,
			OrganizationID: u.CurrentOrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this user",
		)
	}

	original, err := s.repo.GetByID(ctx, repositories.GetUserByIDOptions{
		OrgID:        u.CurrentOrganizationID,
		BuID:         u.BusinessUnitID,
		UserID:       u.ID,
		IncludeRoles: true,
	})
	if err != nil {
		log.Error().Err(err).Str("userID", u.ID.String()).Msg("failed to get user")
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, u)
	if err != nil {
		log.Error().Err(err).Interface("user", u).Msg("failed to update user")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.CurrentOrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("User updated"),
		audit.WithCritical(),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log user update")
	}

	return updatedEntity, nil

}
