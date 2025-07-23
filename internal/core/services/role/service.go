// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package role

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger      *logger.Logger
	Repo        repositories.PermissionRepository
	PermService services.PermissionService
}

type Service struct {
	repo repositories.PermissionRepository
	ps   services.PermissionService
	l    *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "role").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		l:    &log,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRolesRequest,
) (*ports.ListResult[*permission.Role], error) {
	log := s.l.With().
		Str("operation", "ListRoles").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceRole,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read roles")
	}

	return s.repo.ListRoles(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	if err := s.checkPermission(
		ctx,
		permission.ActionRead,
		req.UserID,
		req.BuID,
		req.OrgID,
	); err != nil {
		return nil, err
	}

	return s.repo.GetRoleByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	req *services.CreateRoleRequest,
) (*permission.Role, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("roleName", req.Name).
		Logger()

	if err := s.checkPermission(
		ctx,
		permission.ActionCreate,
		req.UserID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	role := &permission.Role{
		Name:           req.Name,
		Description:    req.Description,
		RoleType:       req.RoleType,
		Priority:       req.Priority,
		ParentRoleID:   req.ParentRoleID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Status:         domain.StatusActive,
	}

	if err := role.Validate(); err != nil {
		log.Error().Err(err).Msg("role validation failed")
		return nil, errors.NewValidationError("role", errors.ErrInvalid, err.Error())
	}

	repoReq := &repositories.CreateRoleRequest{
		Role:           role,
		PermissionIDs:  req.PermissionIDs,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
	}

	createdRole, err := s.repo.CreateRole(ctx, repoReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to create role")
		return nil, err
	}

	log.Debug().Str("roleID", createdRole.ID.String()).Msg("role created successfully")
	return createdRole, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateRoleRequest,
) (*permission.Role, error) {
	log := s.l.With().
		Str("operation", "UpdateRole").
		Str("roleID", req.ID.String()).
		Str("roleName", req.Name).
		Logger()

	if err := s.checkPermission(
		ctx,
		permission.ActionUpdate,
		req.UserID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return nil, err
	}

	role := &permission.Role{
		ID:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		RoleType:       req.RoleType,
		Priority:       req.Priority,
		ParentRoleID:   req.ParentRoleID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
	}

	// Validate the role
	if err := role.Validate(); err != nil {
		log.Error().Err(err).Msg("role validation failed")
		return nil, errors.NewValidationError("role", errors.ErrInvalid, err.Error())
	}

	repoReq := &repositories.UpdateRoleRequest{
		Role:           role,
		PermissionIDs:  req.PermissionIDs,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
	}

	updatedRole, err := s.repo.UpdateRole(ctx, repoReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to update role")
		return nil, err
	}

	log.Debug().Str("roleID", updatedRole.ID.String()).Msg("role updated successfully")
	return updatedRole, nil
}

// Delete deletes a role and its associated permissions
func (s *Service) Delete(
	ctx context.Context,
	req *services.DeleteRoleRequest,
) error {
	log := s.l.With().
		Str("operation", "DeleteRole").
		Str("roleID", req.ID.String()).
		Logger()

	if err := s.checkPermission(
		ctx,
		permission.ActionDelete,
		req.UserID,
		req.BusinessUnitID,
		req.OrganizationID,
	); err != nil {
		return err
	}

	repoReq := &repositories.DeleteRoleRequest{
		RoleID:         req.ID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
	}

	err := s.repo.DeleteRole(ctx, repoReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete role")
		return err
	}

	log.Debug().Str("roleID", req.ID.String()).Msg("role deleted successfully")
	return nil
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
				Resource:       permission.ResourceRole,
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
				"You do not have permission to %s this role",
				strings.ToLower(string(action)),
			),
		)
	}

	return nil
}
