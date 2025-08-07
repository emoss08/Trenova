/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package accessorialcharge

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/accessorialchargevalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.AccessorialChargeRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *accessorialchargevalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.AccessorialChargeRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *accessorialchargevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "accessorialcharge").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*accessorialcharge.AccessorialCharge], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceAccessorialCharge,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read accessorial charges",
		)
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list accessorial charges")
		return nil, err
	}

	return &ports.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("accessorialChargeID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceAccessorialCharge,
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
			"You do not have permission to read this accessorial charge",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get accessorial charge")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	acc *accessorialcharge.AccessorialCharge,
	userID pulid.ID,
) (*accessorialcharge.AccessorialCharge, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("accessorialChargeID", acc.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceAccessorialCharge,
				Action:         permission.ActionCreate,
				BusinessUnitID: acc.BusinessUnitID,
				OrganizationID: acc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create an accessorial charge",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, acc); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, acc)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceAccessorialCharge,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Accessorial Charge created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	acc *accessorialcharge.AccessorialCharge,
	userID pulid.ID,
) (*accessorialcharge.AccessorialCharge, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("accessorialChargeID", acc.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceAccessorialCharge,
				Action:         permission.ActionUpdate,
				BusinessUnitID: acc.BusinessUnitID,
				OrganizationID: acc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this accessorial charge",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, acc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:    acc.ID,
		OrgID: acc.OrganizationID,
		BuID:  acc.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, acc)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceAccessorialCharge,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Accessorial Charge updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}
