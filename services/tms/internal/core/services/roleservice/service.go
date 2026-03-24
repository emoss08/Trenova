package roleservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrCannotEscalatePrivileges      = errors.New("cannot grant permissions you don't have")
	ErrCannotCreateOrgAdmin          = errors.New("only platform admins can create org admin roles")
	ErrCannotCreateBusinessUnitAdmin = errors.New(
		"only platform admins can create business unit admin roles",
	)
	ErrCannotModifySystemRole = errors.New("system roles cannot be modified")
	ErrCannotDeleteSystemRole = errors.New("system roles cannot be deleted")
	ErrCircularInheritance    = errors.New("role inheritance would create a cycle")
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	RoleRepo         repositories.RoleRepository
	UserRepo         repositories.UserRepository
	PermissionCache  repositories.PermissionCacheRepository
	PermissionEngine services.PermissionEngine
	Validator        *Validator
	Registry         *permission.Registry
}

type Service struct {
	l          *zap.Logger
	roleRepo   repositories.RoleRepository
	userRepo   repositories.UserRepository
	permCache  repositories.PermissionCacheRepository
	permEngine services.PermissionEngine
	validator  *Validator
	registry   *permission.Registry
}

//nolint:gocritic // dependencies injection
func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.role"),
		roleRepo:   p.RoleRepo,
		userRepo:   p.UserRepo,
		permCache:  p.PermissionCache,
		permEngine: p.PermissionEngine,
		validator:  p.Validator,
		registry:   p.Registry,
	}
}

type CreateRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Role           *permission.Role
}

type UpdateRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	Role           *permission.Role
}

type AssignRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	Assignment     *permission.UserRoleAssignment
}

type UnassignRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	AssignmentID   pulid.ID
}

func (s *Service) CreateRole(ctx context.Context, req CreateRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "CreateRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("roleName", req.Role.Name),
	)

	isPlatformAdmin, err := s.userRepo.IsPlatformAdmin(ctx, req.ActorID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return err
	}

	if req.Role.IsOrgAdmin && !isPlatformAdmin {
		return ErrCannotCreateOrgAdmin
	}
	if req.Role.IsBusinessUnitAdmin && !isPlatformAdmin {
		return ErrCannotCreateBusinessUnitAdmin
	}

	if !isPlatformAdmin {
		if err = s.validateNoEscalation(
			ctx,
			req.ActorID,
			req.OrganizationID,
			req.Role,
		); err != nil {
			return err
		}
	}

	if err = s.validateNoCircularInheritance(ctx, req.Role.ID, req.Role.ParentRoleIDs); err != nil {
		return err
	}

	if valErr := s.validator.ValidateCreate(ctx, req.Role); valErr != nil {
		return valErr
	}

	req.Role.OrganizationID = req.OrganizationID
	req.Role.CreatedBy = req.ActorID
	req.Role.BusinessUnitID = req.BusinessUnitID

	if err = s.roleRepo.Create(ctx, req.Role); err != nil {
		log.Error("failed to create role", zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) UpdateRole(ctx context.Context, req UpdateRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "UpdateRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("roleID", req.Role.ID.String()),
	)

	existingRole, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID: req.Role.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
		},
	})
	if err != nil {
		log.Error("failed to get existing role", zap.Error(err))
		return err
	}

	if existingRole.IsSystem {
		return ErrCannotModifySystemRole
	}

	isPlatformAdmin, err := s.userRepo.IsPlatformAdmin(ctx, req.ActorID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return err
	}

	if req.Role.IsOrgAdmin && !isPlatformAdmin {
		return ErrCannotCreateOrgAdmin
	}
	if req.Role.IsBusinessUnitAdmin && !isPlatformAdmin {
		return ErrCannotCreateBusinessUnitAdmin
	}

	if !isPlatformAdmin {
		if err = s.validateNoEscalation(
			ctx,
			req.ActorID,
			req.OrganizationID,
			req.Role,
		); err != nil {
			return err
		}
	}

	if err = s.validateNoCircularInheritance(ctx, req.Role.ID, req.Role.ParentRoleIDs); err != nil {
		return err
	}

	if valErr := s.validator.ValidateUpdate(ctx, req.Role); valErr != nil {
		return valErr
	}

	if err = s.roleRepo.Update(ctx, req.Role); err != nil {
		log.Error("failed to update role", zap.Error(err))
		return err
	}

	if err = s.permCache.InvalidateByRole(ctx, req.Role.ID, s.roleRepo); err != nil {
		log.Warn("failed to invalidate permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) ListRoles(
	ctx context.Context,
	req *repositories.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return s.roleRepo.List(ctx, req)
}

func (s *Service) SelectRoleOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return s.roleRepo.SelectOptions(ctx, req)
}

func (s *Service) GetRoleByID(
	ctx context.Context,
	req repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	return s.roleRepo.GetByID(ctx, req)
}

func (s *Service) GetImpactedUsers(
	ctx context.Context,
	roleID pulid.ID,
) ([]repositories.ImpactedUser, error) {
	return s.roleRepo.GetUsersWithRole(ctx, roleID)
}

func (s *Service) GetUserRoleAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*permission.UserRoleAssignment, error) {
	return s.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
}

func (s *Service) AssignRole(ctx context.Context, req AssignRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "AssignRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("userID", req.Assignment.UserID.String()),
		zap.String("roleID", req.Assignment.RoleID.String()),
	)

	role, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID: req.Assignment.RoleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
		},
	})
	if err != nil {
		log.Error("failed to get role", zap.Error(err))
		return err
	}

	isPlatformAdmin, err := s.userRepo.IsPlatformAdmin(ctx, req.ActorID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return err
	}

	if role.IsOrgAdmin && !isPlatformAdmin {
		actorPerms, aErr := s.permEngine.GetEffectivePermissions(
			ctx,
			req.ActorID,
			req.OrganizationID,
		)
		if aErr != nil {
			return aErr
		}
		isActorOrgAdmin := false
		for _, r := range actorPerms.Roles {
			if r.IsOrgAdmin {
				isActorOrgAdmin = true
				break
			}
		}
		if !isActorOrgAdmin {
			return ErrCannotEscalatePrivileges
		}
	}

	if role.IsBusinessUnitAdmin && !isPlatformAdmin {
		actorPerms, aErr := s.permEngine.GetEffectivePermissions(
			ctx,
			req.ActorID,
			req.OrganizationID,
		)
		if aErr != nil {
			return aErr
		}
		isActorBUAdmin := false
		for _, r := range actorPerms.Roles {
			if r.IsBusinessUnitAdmin {
				isActorBUAdmin = true
				break
			}
		}
		if !isActorBUAdmin {
			return ErrCannotCreateBusinessUnitAdmin
		}
	}

	if !isPlatformAdmin {
		tempRole := &permission.Role{
			Permissions: role.Permissions,
		}
		if err = s.validateNoEscalation(
			ctx,
			req.ActorID,
			req.OrganizationID,
			tempRole,
		); err != nil {
			return err
		}
	}

	req.Assignment.OrganizationID = req.OrganizationID
	req.Assignment.AssignedBy = req.ActorID

	if err = s.roleRepo.CreateAssignment(ctx, req.Assignment); err != nil {
		log.Error("failed to create assignment", zap.Error(err))
		return err
	}

	if err = s.permEngine.InvalidateUser(
		ctx,
		req.Assignment.UserID,
		req.OrganizationID,
	); err != nil {
		log.Warn("failed to invalidate user permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) UnassignRole(ctx context.Context, req UnassignRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "UnassignRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("assignmentID", req.AssignmentID.String()),
	)

	if err := s.roleRepo.DeleteAssignment(ctx, req.AssignmentID); err != nil {
		log.Error("failed to delete assignment", zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) validateNoEscalation(
	ctx context.Context,
	actorID, orgID pulid.ID,
	targetRole *permission.Role,
) error {
	actorPerms, err := s.permEngine.GetEffectivePermissions(ctx, actorID, orgID)
	if err != nil {
		return err
	}

	if targetRole.MaxSensitivity.Level() > actorPerms.MaxSensitivity.Level() {
		me := errortypes.NewMultiError()
		me.Add(
			"maxSensitivity",
			errortypes.ErrForbidden,
			"cannot grant sensitivity level higher than your own",
		)
		return me
	}

	for _, rp := range targetRole.Permissions {
		actorResourcePerm, ok := actorPerms.Resources[rp.Resource]
		if !ok {
			me := errortypes.NewMultiError()
			me.Add(
				"permissions",
				errortypes.ErrForbidden,
				"cannot grant permissions for resource '"+rp.Resource+"' that you don't have access to",
			)
			return me
		}

		actorOps := permission.NewOperationSet(actorResourcePerm.Operations...)
		for _, op := range rp.Operations {
			if !actorOps.Has(op) {
				me := errortypes.NewMultiError()
				me.Add(
					"permissions",
					errortypes.ErrForbidden,
					"cannot grant '"+string(
						op,
					)+"' operation on '"+rp.Resource+"' that you don't have",
				)
				return me
			}
		}

		if rp.DataScope.IsMorePermissive(actorResourcePerm.DataScope) {
			me := errortypes.NewMultiError()
			me.Add(
				"permissions",
				errortypes.ErrForbidden,
				"cannot grant data scope '"+string(
					rp.DataScope,
				)+"' on '"+rp.Resource+"' that is more permissive than your own",
			)
			return me
		}
	}

	return nil
}

func (s *Service) validateNoCircularInheritance(
	ctx context.Context,
	roleID pulid.ID,
	parentRoleIDs []pulid.ID,
) error {
	if len(parentRoleIDs) == 0 {
		return nil
	}

	visited := make(map[pulid.ID]bool)
	stack := make(map[pulid.ID]bool)

	var dfs func(id pulid.ID) bool
	dfs = func(id pulid.ID) bool {
		visited[id] = true
		stack[id] = true

		if id == roleID {
			return true
		}

		role, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{ID: id})
		if err != nil {
			return false
		}

		for _, parentID := range role.ParentRoleIDs {
			if !visited[parentID] {
				if dfs(parentID) {
					return true
				}
			} else if stack[parentID] {
				return true
			}
		}

		stack[id] = false
		return false
	}

	for _, parentID := range parentRoleIDs {
		if parentID == roleID {
			return ErrCircularInheritance
		}
		if dfs(parentID) {
			return ErrCircularInheritance
		}
	}

	return nil
}

func (s *Service) CreateResourcePermission(
	ctx context.Context,
	actorID, orgID pulid.ID,
	rp *permission.ResourcePermission,
) error {
	log := s.l.With(
		zap.String("operation", "CreateResourcePermission"),
		zap.String("roleID", rp.RoleID.String()),
		zap.String("resource", rp.Resource),
	)

	role, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID: rp.RoleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
		},
	})
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrCannotModifySystemRole
	}

	isPlatformAdmin, err := s.userRepo.IsPlatformAdmin(ctx, actorID)
	if err != nil {
		return err
	}

	if !isPlatformAdmin {
		tempRole := &permission.Role{
			Permissions: []*permission.ResourcePermission{rp},
		}
		if err = s.validateNoEscalation(ctx, actorID, orgID, tempRole); err != nil {
			return err
		}
	}

	if err = s.roleRepo.CreateResourcePermission(ctx, rp); err != nil {
		log.Error("failed to create resource permission", zap.Error(err))
		return err
	}

	if err = s.permCache.InvalidateByRole(ctx, rp.RoleID, s.roleRepo); err != nil {
		log.Warn("failed to invalidate permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) UpdateResourcePermission(
	ctx context.Context,
	actorID, orgID pulid.ID,
	rp *permission.ResourcePermission,
) error {
	log := s.l.With(
		zap.String("operation", "UpdateResourcePermission"),
		zap.String("id", rp.ID.String()),
	)

	role, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID: rp.RoleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
		},
	})
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrCannotModifySystemRole
	}

	isPlatformAdmin, err := s.userRepo.IsPlatformAdmin(ctx, actorID)
	if err != nil {
		return err
	}

	if !isPlatformAdmin {
		tempRole := &permission.Role{
			Permissions: []*permission.ResourcePermission{rp},
		}
		if err = s.validateNoEscalation(ctx, actorID, orgID, tempRole); err != nil {
			return err
		}
	}

	if err = s.roleRepo.UpdateResourcePermission(ctx, rp); err != nil {
		log.Error("failed to update resource permission", zap.Error(err))
		return err
	}

	if err = s.permCache.InvalidateByRole(ctx, rp.RoleID, s.roleRepo); err != nil {
		log.Warn("failed to invalidate permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) DeleteResourcePermission(
	ctx context.Context,
	orgID, permID, roleID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteResourcePermission"),
		zap.String("id", permID.String()),
	)

	role, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID: roleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
		},
	})
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrCannotModifySystemRole
	}

	if err = s.roleRepo.DeleteResourcePermission(ctx, permID); err != nil {
		log.Error("failed to delete resource permission", zap.Error(err))
		return err
	}

	if err = s.permCache.InvalidateByRole(ctx, roleID, s.roleRepo); err != nil {
		log.Warn("failed to invalidate permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) InitializeOrganizationRoles(
	ctx context.Context,
	orgID, creatorID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "InitializeOrganizationRoles"),
		zap.String("orgID", orgID.String()),
	)

	adminRole := &permission.Role{
		OrganizationID: orgID,
		Name:           "Organization Administrator",
		Description:    "Full access to all resources within the organization",
		MaxSensitivity: permission.SensitivityConfidential,
		IsSystem:       true,
		IsOrgAdmin:     true,
		CreatedBy:      creatorID,
	}

	if err := s.roleRepo.Create(ctx, adminRole); err != nil {
		log.Error("failed to create admin role", zap.Error(err))
		return err
	}

	assignment := &permission.UserRoleAssignment{
		UserID:         creatorID,
		OrganizationID: orgID,
		RoleID:         adminRole.ID,
		AssignedBy:     creatorID,
	}

	if err := s.roleRepo.CreateAssignment(ctx, assignment); err != nil {
		log.Error("failed to assign admin role to creator", zap.Error(err))
		return err
	}

	log.Info("initialized organization roles", zap.String("adminRoleID", adminRole.ID.String()))
	return nil
}
