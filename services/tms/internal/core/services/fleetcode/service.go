/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package fleetcode

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/fleetcodevalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.FleetCodeRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *fleetcodevalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.FleetCodeRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *fleetcodevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "fleetcode").
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
	opts *repositories.ListFleetCodeOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, fc := range result.Items {
		options[i] = &types.SelectOption{
			Value: fc.ID.String(),
			Label: fc.Name,
			Color: fc.Color,
		}
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListFleetCodeOptions,
) (*ports.ListResult[*fleetcode.FleetCode], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceFleetCode,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read fleet codes")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list fleet codes")
		return nil, err
	}

	return &ports.ListResult[*fleetcode.FleetCode]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetFleetCodeByIDOptions,
) (*fleetcode.FleetCode, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("fleetCodeID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceFleetCode,
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
			"You do not have permission to read this fleet code",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get fleet code")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	fc *fleetcode.FleetCode,
	userID pulid.ID,
) (*fleetcode.FleetCode, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", fc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFleetCode,
				Action:         permission.ActionCreate,
				BusinessUnitID: fc.BusinessUnitID,
				OrganizationID: fc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create fleet code permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a fleet code",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, fc); err != nil {
		return nil, err
	}

	createdFleetCode, err := s.repo.Create(ctx, fc)
	if err != nil {
		return nil, eris.Wrap(err, "create fleet code")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFleetCode,
			ResourceID:     createdFleetCode.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdFleetCode),
			OrganizationID: createdFleetCode.OrganizationID,
			BusinessUnitID: createdFleetCode.BusinessUnitID,
		},
		audit.WithComment("Fleet code created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log fleet code creation")
	}

	return createdFleetCode, nil
}

func (s *Service) Update(
	ctx context.Context,
	fc *fleetcode.FleetCode,
	userID pulid.ID,
) (*fleetcode.FleetCode, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", fc.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceFleetCode,
				Action:         permission.ActionUpdate,
				BusinessUnitID: fc.BusinessUnitID,
				OrganizationID: fc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this fleet code",
		)
	}

	// Validate the fleet code
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, fc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetFleetCodeByIDOptions{
		ID:    fc.ID,
		OrgID: fc.OrganizationID,
		BuID:  fc.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedFleetCode, err := s.repo.Update(ctx, fc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update fleet code")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceFleetCode,
			ResourceID:     updatedFleetCode.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedFleetCode),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedFleetCode.OrganizationID,
			BusinessUnitID: updatedFleetCode.BusinessUnitID,
		},
		audit.WithComment("Fleet code updated"),
		audit.WithDiff(original, updatedFleetCode),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log fleet code update")
	}

	return updatedFleetCode, nil
}
