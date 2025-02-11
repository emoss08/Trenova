package shipmentmove

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.ShipmentMoveRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	// Validator     *shipmentvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentMoveRepository
	ps   services.PermissionService
	as   services.AuditService
	// v    *shipmentvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmentmove").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) Split(ctx context.Context, req *repositories.SplitMoveRequest, userID pulid.ID) (*repositories.SplitMoveResponse, error) {
	log := s.l.With().
		Str("operation", "Split").
		Str("moveID", req.MoveID.String()).
		Str("splitLocationID", req.SplitLocationID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipmentMove,
				Action:         permission.ActionSplit,
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
		return nil, errors.NewAuthorizationError("You do not have permission to create a shipment")
	}

	original, err := s.repo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            req.MoveID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	// TODO(Wolfred): Add validation
	if err := req.Validate(ctx, original); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.SplitMove(ctx, req)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentMove,
			ResourceID:     createdEntity.NewMove.GetID(),
			Action:         permission.ActionSplit,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: createdEntity.NewMove.OrganizationID,
			BusinessUnitID: createdEntity.NewMove.BusinessUnitID,
		},
		audit.WithDiff(original, createdEntity),
		audit.WithComment("Shipment move split"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment move split")
	}

	return createdEntity, nil
}
