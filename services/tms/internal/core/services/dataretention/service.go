package dataretention

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.DataRetentionRepository
	PermService  services.PermissionService
	AuditService services.AuditService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.DataRetentionRepository
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.
		With().
		Str("service", "dataretention").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*organization.DataRetention, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceDataRetention,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this data retention",
		)
	}

	return s.repo.Get(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	dr *organization.DataRetention,
	userID pulid.ID,
) (*organization.DataRetention, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("orgID", dr.OrganizationID.String()).
		Str("buID", dr.BusinessUnitID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDataRetention,
				Action:         permission.ActionUpdate,
				BusinessUnitID: dr.BusinessUnitID,
				OrganizationID: dr.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this data retention",
		)
	}

	return s.repo.Update(ctx, dr)
}
