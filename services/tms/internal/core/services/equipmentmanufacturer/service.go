/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package equipmentmanufacturer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/equipmentmanufacturervalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
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

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, repositories.ListEquipmentManufacturerOptions{
		Filter: opts,
	})
	if err != nil {
		return nil, eris.Wrap(err, "select equipment manufacturers")
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, em := range result.Items {
		options = append(options, &types.SelectOption{
			Value: em.ID.String(),
			Label: em.Name,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts repositories.ListEquipmentManufacturerOptions,
) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceEquipmentManufacturer,
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
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read equipment manufacturers",
		)
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list equipment manufacturers")
		return nil, err
	}

	return &ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetEquipmentManufacturerByIDOptions,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
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
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this equipment manufacturer",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get equipment manufacturer")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	et *equipmentmanufacturer.EquipmentManufacturer,
	userID pulid.ID,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
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
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a equipment manufacturer",
		)
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
		return nil, err
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

func (s *Service) Update(
	ctx context.Context,
	et *equipmentmanufacturer.EquipmentManufacturer,
	userID pulid.ID,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
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
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this equipment manufacturer",
		)
	}

	// Validate the equipment manufacturer
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, et); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetEquipmentManufacturerByIDOptions{
		ID:     et.ID,
		OrgID:  et.OrganizationID,
		BuID:   et.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, et)
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment manufacturer")
		return nil, err
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
