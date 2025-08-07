/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dedicatedlane

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.DedicatedLaneRepository
	PermService  services.PermissionService
	AuditService services.AuditService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.DedicatedLaneRepository
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "dedicated_lane").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneRequest,
) (*ports.ListResult[*dedicatedlane.DedicatedLane], error) {
	if err := s.checkPermission(
		ctx,
		permission.ActionRead,
		req.Filter.TenantOpts.UserID,
		req.Filter.TenantOpts.BuID,
		req.Filter.TenantOpts.OrgID,
	); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetDedicatedLaneByIDRequest,
) (*dedicatedlane.DedicatedLane, error) {
	if err := s.checkPermission(
		ctx,
		permission.ActionRead,
		opts.UserID,
		opts.BuID,
		opts.OrgID,
	); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, opts)
}

func (s *Service) FindByShipment(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLane, error) {
	// * we don't care about permissions here, we just want to find the dedicated lane
	return s.repo.FindByShipment(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	req *dedicatedlane.DedicatedLane,
	userID pulid.ID,
) (*dedicatedlane.DedicatedLane, error) {
	log := s.l.With().
		Str("operation", "create").
		Str("id", req.ID.String()).
		Logger()

	if err := s.checkPermission(
		ctx,
		permission.ActionCreate,
		userID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	entity, err := s.repo.Create(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create dedicated lane")
		return nil, eris.Wrap(err, "create dedicated lane")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDedicatedLane,
			ResourceID:     entity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(entity),
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Dedicated lane created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log dedicated lane creation")
	}

	return entity, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *dedicatedlane.DedicatedLane,
	userID pulid.ID,
) (*dedicatedlane.DedicatedLane, error) {
	log := s.l.With().
		Str("operation", "update").
		Str("id", req.ID.String()).
		Logger()

	if err := s.checkPermission(
		ctx,
		permission.ActionUpdate,
		userID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetDedicatedLaneByIDRequest{
		ID:     req.ID,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to update dedicated lane")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDedicatedLane,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Dedicated lane updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log dedicated lane update")
	}

	return updatedEntity, nil
}

func (s *Service) checkPermission(
	ctx context.Context,
	action permission.Action,
	userID, buID, orgID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "checkPermission").
		Str("action", string(action)).
		Str("userID", userID.String()).
		Str("buID", buID.String()).
		Str("orgID", orgID.String()).
		Logger()

	// Check if user has permission to delete roles
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDedicatedLane,
				Action:         action,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			fmt.Sprintf(
				"You do not have permission to %s this dedicated lane",
				strings.ToLower(string(action)),
			),
		)
	}

	return nil
}
