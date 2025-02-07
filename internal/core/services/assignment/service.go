package assignment

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
	"github.com/emoss08/trenova/internal/pkg/validator/assignmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.AssignmentRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *assignmentvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.AssignmentRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *assignmentvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "assignment").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SingleAssign(ctx context.Context, a *shipment.Assignment, userID pulid.ID) (*shipment.Assignment, error) {
	log := s.l.With().
		Str("operation", "SingleAssign").
		Str("id", a.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceAssignment,
				Action:         permission.ActionAssign,
				BusinessUnitID: a.BusinessUnitID,
				OrganizationID: a.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to assign")
	}

	// * Validate the assignment
	if err := s.v.Validate(ctx, a); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.SingleAssign(ctx, a)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Assignment created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log assignment creation")
	}

	return createdEntity, nil
}
