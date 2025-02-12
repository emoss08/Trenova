package stop

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
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

func (s *Service) Get(ctx context.Context, req repositories.GetStopByIDRequest) (*shipment.Stop, error) {
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

func (s *Service) Update(ctx context.Context, stp *shipment.Stop, userID pulid.ID) (*shipment.Stop, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("stopID", stp.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceStop,
				Action:         permission.ActionUpdate,
				BusinessUnitID: stp.BusinessUnitID,
				OrganizationID: stp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this stop")
	}

	// Validate the stop
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, stp); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetStopByIDRequest{
		StopID:            stp.ID,
		OrgID:             stp.OrganizationID,
		BuID:              stp.BusinessUnitID,
		UserID:            userID,
		ExpandStopDetails: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get original stop")
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, stp)
	if err != nil {
		log.Error().Err(err).Msg("failed to update stop")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceStop,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Stop updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log stop update")
	}

	return updatedEntity, nil
}
