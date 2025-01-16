package hazardousmaterial

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/hazardousmaterialvalidator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.HazardousMaterialRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *hazardousmaterialvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.HazardousMaterialRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *hazardousmaterialvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "hazardousmaterial").
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
		return nil, eris.Wrap(err, "select hazardous materials")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, hm := range result.Items {
		options[i] = &types.SelectOption{
			Value: hm.GetID(),
			Label: hm.Code,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceHazardousMaterial,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read hazardous materials")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list hazardous materials")
		return nil, eris.Wrap(err, "list hazardous materials")
	}

	return &ports.ListResult[*hazardousmaterial.HazardousMaterial]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetHazardousMaterialByIDOptions) (*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("hmID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceHazardousMaterial,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read hazardous material permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this hazardous material")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get hazardous material")
		return nil, eris.Wrap(err, "get hazardous material")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, hm *hazardousmaterial.HazardousMaterial, userID pulid.ID) (*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", hm.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceHazardousMaterial,
				Action:         permission.ActionCreate,
				BusinessUnitID: hm.BusinessUnitID,
				OrganizationID: hm.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create hazardous material permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a hazardous material")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hm); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hm)
	if err != nil {
		return nil, eris.Wrap(err, "create hazardous material")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazardousMaterial,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Hazardous Material created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log hazardous material creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, hm *hazardousmaterial.HazardousMaterial, userID pulid.ID) (*hazardousmaterial.HazardousMaterial, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", hm.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceHazardousMaterial,
				Action:         permission.ActionUpdate,
				BusinessUnitID: hm.BusinessUnitID,
				OrganizationID: hm.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update hazardous material permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this hazardous material")
	}

	// Validate the hazardous material
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hm); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetHazardousMaterialByIDOptions{
		ID:    hm.ID,
		OrgID: hm.OrganizationID,
		BuID:  hm.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get hazardous material")
	}

	updatedEntity, err := s.repo.Update(ctx, hm)
	if err != nil {
		log.Error().Err(err).Msg("failed to update hazardous material")
		return nil, eris.Wrap(err, "update hazardous material")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHazardousMaterial,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
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
		log.Error().Err(err).Msg("failed to log hazardous material update")
	}

	return updatedEntity, nil
}
