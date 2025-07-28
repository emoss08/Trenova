package shipmentcomment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger      *logger.Logger
	Repo        repositories.ShipmentCommentRepository
	PermService services.PermissionService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentCommentRepository
	ps   services.PermissionService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmentcomment").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
	}
}

func (s *Service) ListByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) (*ports.ListResult[*shipment.ShipmentComment], error) {
	log := s.l.With().
		Str("operation", "ListByShipmentID").
		Str("shipmentID", req.ShipmentID.String()).
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.Filter.TenantOpts.UserID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionRead,
			BusinessUnitID: req.Filter.TenantOpts.BuID,
			OrganizationID: req.Filter.TenantOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read shipment comments",
		)
	}

	return s.repo.ListByShipmentID(ctx, req)
}
