/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package stop

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.StopRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *shipmentvalidator.StopValidator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.StopRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *shipmentvalidator.StopValidator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "stop").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetStopByIDRequest,
) (*shipment.Stop, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("stopID", req.StopID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceStop,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read stop permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this stop")
	}

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get stop")
		return nil, eris.Wrap(err, "get stop")
	}

	return entity, nil
}
