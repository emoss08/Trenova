package equipmentmanufacturer

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/core/services/audit"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/jsonutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/equipmentmanufacturervalidator"
	"github.com/trenova-app/transport/pkg/types"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.EquipmentManufacturerRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *equipmentmanufacturervalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.EquipmentManufacturerRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *equipmentmanufacturervalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "equipmentmanufacturer").
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
		return nil, eris.Wrap(err, "select equipment manufacturers")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, em := range result.Items {
		options[i] = &types.SelectOption{
			Value: em.ID.String(),
			Label: em.Name,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceEquipmentManufacturer,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read equipment manufacturers")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list equipment manufacturers")
		return nil, eris.Wrap(err, "failed to list equipment manufacturers")
	}

	return &ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetEquipManufacturerByIDOptions) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("equipManuID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceEquipmentManufacturer,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check read equipment manufacturer permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this equipment manufacturer")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get equipment manufacturer")
		return nil, eris.Wrap(err, "failed to get equipment manufacturer")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, et *equipmentmanufacturer.EquipmentManufacturer, userID pulid.ID) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", et.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceEquipmentManufacturer,
				Action:         permission.ActionCreate,
				BusinessUnitID: et.BusinessUnitID,
				OrganizationID: et.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create equipment manufacturer permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a equipment manufacturer")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, et); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, et)
	if err != nil {
		return nil, eris.Wrap(err, "create equipment manufacturer")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEquipmentManufacturer,
			ResourceID:     createdEntity.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Equipment Manufacturer created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log equipment manufacturer creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, et *equipmentmanufacturer.EquipmentManufacturer, userID pulid.ID) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", et.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceEquipmentManufacturer,
				Action:         permission.ActionUpdate,
				BusinessUnitID: et.BusinessUnitID,
				OrganizationID: et.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update equipment manufacturer permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this equipment manufacturer")
	}

	// Validate the equipment manufacturer
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, et); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetEquipManufacturerByIDOptions{
		ID:    et.ID,
		OrgID: et.OrganizationID,
		BuID:  et.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get equipment manufacturer")
	}

	updatedEntity, err := s.repo.Update(ctx, et)
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment manufacturer")
		return nil, eris.Wrap(err, "update equipment manufacturer")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEquipmentManufacturer,
			ResourceID:     updatedEntity.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Equipment Manufacturer updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log equipment manufacturer update")
	}

	return updatedEntity, nil
}
