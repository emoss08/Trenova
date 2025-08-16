package holdreason

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/holdreasonvalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.HoldReasonRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *holdreasonvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.HoldReasonRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *holdreasonvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "hold_reason").
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
	req *repositories.ListHoldReasonRequest,
) (*ports.ListResult[*shipment.HoldReason], error) {
	log := s.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.Filter.TenantOpts.UserID,
			OrganizationID: req.Filter.TenantOpts.OrgID,
			BusinessUnitID: req.Filter.TenantOpts.BuID,
			Resource:       permission.ResourceHoldReason,
			Action:         permission.ActionRead,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read hold reasons")
	}

	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetHoldReasonByIDRequest,
) (*shipment.HoldReason, error) {
	log := s.l.With().
		Str("operation", "get").
		Str("holdReasonID", req.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			Resource:       permission.ResourceHoldReason,
			Action:         permission.ActionRead,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this hold reason",
		)
	}

	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	hr *shipment.HoldReason,
	userID pulid.ID,
) (*shipment.HoldReason, error) {
	log := s.l.With().
		Str("operation", "create").
		Str("code", hr.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			OrganizationID: hr.OrganizationID,
			BusinessUnitID: hr.BusinessUnitID,
			Resource:       permission.ResourceHoldReason,
			Action:         permission.ActionCreate,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a hold reason",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hr); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hr)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHoldReason,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Hold reason created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log hold reason creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	hr *shipment.HoldReason,
	userID pulid.ID,
) (*shipment.HoldReason, error) {
	log := s.l.With().
		Str("operation", "update").
		Str("code", hr.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			OrganizationID: hr.OrganizationID,
			BusinessUnitID: hr.BusinessUnitID,
			Resource:       permission.ResourceHoldReason,
			Action:         permission.ActionUpdate,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this hold reason",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, hr); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetHoldReasonByIDRequest{
		ID:    hr.ID,
		OrgID: hr.OrganizationID,
		BuID:  hr.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, hr)
	if err != nil {
		log.Error().Err(err).Msg("failed to update hold reason")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHoldReason,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Hold reason updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log hold reason update")
	}

	return updatedEntity, nil
}
