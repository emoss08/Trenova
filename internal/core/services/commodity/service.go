package commodity

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/commodity"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/commodityvalidator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.CommodityRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *commodityvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.CommodityRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *commodityvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "commodity").
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
		return nil, eris.Wrap(err, "select commodities")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, com := range result.Items {
		options[i] = &types.SelectOption{
			Value: com.GetID(),
			Label: com.Name,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*commodity.Commodity], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceCommodity,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read commodities")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list commodities")
		return nil, eris.Wrap(err, "list commodities")
	}

	return &ports.ListResult[*commodity.Commodity]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetCommodityByIDOptions) (*commodity.Commodity, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("hmID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceCommodity,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read commodity permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this commodity")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get commodity")
		return nil, eris.Wrap(err, "get commodity")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, com *commodity.Commodity, userID pulid.ID) (*commodity.Commodity, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", com.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceCommodity,
				Action:         permission.ActionCreate,
				BusinessUnitID: com.BusinessUnitID,
				OrganizationID: com.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create commodity permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a commodity")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, com); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, com)
	if err != nil {
		return nil, eris.Wrap(err, "create commodity")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCommodity,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Commodity created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log commodity creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, com *commodity.Commodity, userID pulid.ID) (*commodity.Commodity, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", com.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceCommodity,
				Action:         permission.ActionUpdate,
				BusinessUnitID: com.BusinessUnitID,
				OrganizationID: com.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update commodity permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this commodity")
	}

	// Validate the commodity
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, com); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCommodityByIDOptions{
		ID:    com.ID,
		OrgID: com.OrganizationID,
		BuID:  com.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get commodity")
	}

	updatedEntity, err := s.repo.Update(ctx, com)
	if err != nil {
		log.Error().Err(err).Msg("failed to update commodity")
		return nil, eris.Wrap(err, "update commodity")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCommodity,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Commodity updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log commodity update")
	}

	return updatedEntity, nil
}
