/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package assignment

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
	"github.com/emoss08/trenova/internal/pkg/validator/assignmentvalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
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

func (s *Service) List(
	ctx context.Context,
	req repositories.ListAssignmentsRequest,
) (*ports.ListResult[*shipment.Assignment], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceAssignment,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read assignments")
	}

	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetAssignmentByIDOptions,
) (*shipment.Assignment, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("hmID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceAssignment,
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
			"You do not have permission to read this assignment",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get assignment")
		return nil, err
	}

	return entity, nil
}

func (s *Service) BulkAssign(
	ctx context.Context,
	req *repositories.AssignmentRequest,
) (*ports.ListResult[*shipment.Assignment], error) {
	log := s.l.With().
		Str("operation", "BulkAssign").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceAssignment,
				Action:         permission.ActionAssign,
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
		return nil, errors.NewAuthorizationError("You do not have permission to assign")
	}

	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.BulkAssign(ctx, req)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceAssignment,
			ResourceID:     "",
			Action:         permission.ActionCreate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Assignments created"),
		audit.WithDiff(req, createdEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log assignment creation")
	}

	return &ports.ListResult[*shipment.Assignment]{
		Items: createdEntity,
		Total: len(createdEntity),
	}, nil
}

func (s *Service) SingleAssign(
	ctx context.Context,
	a *shipment.Assignment,
	userID pulid.ID,
) (*shipment.Assignment, error) {
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
			Resource:       permission.ResourceAssignment,
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

func (s *Service) Reassign(
	ctx context.Context,
	a *shipment.Assignment,
	userID pulid.ID,
) (*shipment.Assignment, error) {
	log := s.l.With().
		Str("operation", "Reassign").
		Str("id", a.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceAssignment,
				Action:         permission.ActionReassign,
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
		return nil, errors.NewAuthorizationError("You do not have permission to reassign")
	}

	if err := s.v.Validate(ctx, a); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Reassign(ctx, a)
	if err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetAssignmentByIDOptions{
		ID:    a.ID,
		OrgID: a.OrganizationID,
		BuID:  a.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceAssignment,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionReassign,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Assignment re-assigned"),
		audit.WithDiff(original, createdEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log assignment re-assignment")
	}

	return createdEntity, nil
}
