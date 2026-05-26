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
	ErrCannotEscalatePrivileges = errors.New("cannot grant permissions you don't have")
	ErrCannotModifySystemRole   = errors.New("system roles cannot be modified")
	ErrCannotDeleteSystemRole   = errors.New("system roles cannot be deleted")
	ErrCircularInheritance      = errors.New("role inheritance would create a cycle")
	ErrRoleAlreadyAssigned      = errors.New("role is already assigned to this user")
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	RoleRepo         repositories.RoleRepository
	RBACRepo         repositories.RBACRepository
	PermissionCache  repositories.PermissionCacheRepository
	PermissionEngine services.PermissionEngine
	Validator        *Validator
	Registry         *permission.Registry
}

type Service struct {
	l          *zap.Logger
	roleRepo   repositories.RoleRepository
	rbacRepo   repositories.RBACRepository
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
		rbacRepo:   p.RBACRepo,
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

type UpsertHierarchyEdgeRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	SeniorRoleID   pulid.ID
	JuniorRoleID   pulid.ID
}

type SaveConstraintRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Constraint     *permission.RoleConstraint
	RoleIDs        []pulid.ID
}

func (s *Service) CreateRole(ctx context.Context, req CreateRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "CreateRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("roleName", req.Role.Name),
	)

	if validateErr := s.validateNoEscalation(
		ctx,
		req.ActorID,
		req.OrganizationID,
		req.Role,
	); validateErr != nil {
		return validateErr
	}

	if inheritanceErr := s.validateNoCircularInheritance(
		ctx,
		req.Role.ID,
		req.Role.ParentRoleIDs,
	); inheritanceErr != nil {
		return inheritanceErr
	}

	if valErr := s.validator.ValidateCreate(ctx, req.Role); valErr != nil {
		return valErr
	}

	req.Role.OrganizationID = req.OrganizationID
	req.Role.CreatedBy = req.ActorID
	req.Role.BusinessUnitID = req.BusinessUnitID

	if err := s.roleRepo.Create(ctx, req.Role); err != nil {
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

	if validateErr := s.validateNoEscalation(
		ctx,
		req.ActorID,
		req.OrganizationID,
		req.Role,
	); validateErr != nil {
		return validateErr
	}

	if inheritanceErr := s.validateNoCircularInheritance(
		ctx,
		req.Role.ID,
		req.Role.ParentRoleIDs,
	); inheritanceErr != nil {
		return inheritanceErr
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

	if err := s.validateAssignmentPrivileges(ctx, &req, log); err != nil {
		return err
	}

	req.Assignment.OrganizationID = req.OrganizationID
	req.Assignment.AssignedBy = req.ActorID

	if err := s.validateStaticSeparationOfDutyForAssignment(ctx, &req); err != nil {
		return err
	}

	if err := s.roleRepo.CreateAssignment(ctx, req.Assignment); err != nil {
		log.Error("failed to create assignment", zap.Error(err))
		return err
	}

	if err := s.permEngine.InvalidateUser(
		ctx,
		req.Assignment.UserID,
		req.OrganizationID,
	); err != nil {
		log.Warn("failed to invalidate user permission cache", zap.Error(err))
	}

	return nil
}

func (s *Service) validateAssignmentPrivileges(
	ctx context.Context,
	req *AssignRoleRequest,
	log *zap.Logger,
) error {
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

	return s.validateNoEscalation(ctx, req.ActorID, req.OrganizationID, &permission.Role{
		Permissions: role.Permissions,
	})
}

func (s *Service) validateStaticSeparationOfDutyForAssignment(
	ctx context.Context,
	req *AssignRoleRequest,
) error {
	existingAssignments, err := s.roleRepo.GetUserRoleAssignments(
		ctx,
		req.Assignment.UserID,
		req.OrganizationID,
	)
	if err != nil {
		return err
	}

	roleIDs := make([]pulid.ID, 0, len(existingAssignments)+1)
	for _, assignment := range existingAssignments {
		if assignment.IsExpired() {
			continue
		}
		if assignment.RoleID == req.Assignment.RoleID {
			return ErrRoleAlreadyAssigned
		}
		roleIDs = append(roleIDs, assignment.RoleID)
	}
	roleIDs = append(roleIDs, req.Assignment.RoleID)

	violations, err := s.rbacRepo.ValidateStaticSeparationOfDuty(
		ctx,
		req.Assignment.UserID,
		req.OrganizationID,
		roleIDs,
	)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		return separationOfDutyError("role assignment violates static separation of duty")
	}
	return nil
}

func (s *Service) ListRoleHierarchyEdges(
	ctx context.Context,
	orgID pulid.ID,
) ([]*permission.RoleHierarchyEdge, error) {
	return s.rbacRepo.ListRoleHierarchyEdges(ctx, orgID)
}

func (s *Service) UpsertRoleHierarchyEdge(
	ctx context.Context,
	req *UpsertHierarchyEdgeRequest,
) error {
	if err := s.rbacRepo.UpsertRoleHierarchyEdge(ctx, &repositories.UpsertRoleHierarchyEdgeRequest{
		ActorID:        req.ActorID,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		SeniorRoleID:   req.SeniorRoleID,
		JuniorRoleID:   req.JuniorRoleID,
	}); err != nil {
		if errors.Is(err, repositories.ErrCircularRoleHierarchy) {
			return ErrCircularInheritance
		}
		return err
	}

	if err := s.permCache.InvalidateByRole(ctx, req.SeniorRoleID, s.roleRepo); err != nil {
		s.l.Warn("failed to invalidate permission cache", zap.Error(err))
	}
	return nil
}

func (s *Service) DeleteRoleHierarchyEdge(
	ctx context.Context,
	orgID pulid.ID,
	edgeID pulid.ID,
) error {
	return s.rbacRepo.DeleteRoleHierarchyEdge(ctx, repositories.DeleteRoleHierarchyEdgeRequest{
		OrganizationID: orgID,
		EdgeID:         edgeID,
	})
}

func (s *Service) ListRoleConstraints(
	ctx context.Context,
	req repositories.ListRoleConstraintsRequest,
) ([]*permission.RoleConstraint, error) {
	return s.rbacRepo.ListRoleConstraints(ctx, req)
}

func (s *Service) SaveRoleConstraint(ctx context.Context, req *SaveConstraintRequest) error {
	if req.Constraint.MaxRoles < 1 {
		return errortypes.NewValidationError(
			"maxRoles",
			errortypes.ErrInvalid,
			"Max roles must be at least 1",
		)
	}
	if len(req.RoleIDs) <= req.Constraint.MaxRoles {
		return errortypes.NewValidationError(
			"roleIds",
			errortypes.ErrInvalid,
			"Constraint role set must contain more roles than the max roles limit",
		)
	}
	if err := s.validateConstraintRoleIDs(ctx, req.OrganizationID, req.RoleIDs); err != nil {
		return err
	}

	req.Constraint.OrganizationID = req.OrganizationID
	req.Constraint.BusinessUnitID = req.BusinessUnitID
	req.Constraint.CreatedBy = req.ActorID

	if err := s.rbacRepo.SaveRoleConstraint(ctx, &repositories.SaveRoleConstraintRequest{
		Constraint: req.Constraint,
		RoleIDs:    req.RoleIDs,
	}); err != nil {
		return err
	}

	if err := s.permCache.InvalidateOrganization(ctx, req.OrganizationID); err != nil {
		s.l.Warn("failed to invalidate permission cache", zap.Error(err))
	}
	return nil
}

func (s *Service) validateConstraintRoleIDs(
	ctx context.Context,
	orgID pulid.ID,
	roleIDs []pulid.ID,
) error {
	seen := make(map[pulid.ID]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID.IsNil() {
			return errortypes.NewValidationError(
				"roleIds",
				errortypes.ErrInvalid,
				"Constraint role IDs must not be empty",
			)
		}
		if _, ok := seen[roleID]; ok {
			return errortypes.NewValidationError(
				"roleIds",
				errortypes.ErrInvalid,
				"Constraint role IDs must be unique",
			)
		}
		seen[roleID] = struct{}{}
	}

	for _, roleID := range roleIDs {
		if _, err := s.roleRepo.GetByID(ctx, repositories.GetRoleByIDRequest{
			ID: roleID,
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteRoleConstraint(
	ctx context.Context,
	orgID pulid.ID,
	constraintID pulid.ID,
) error {
	if err := s.rbacRepo.DeleteRoleConstraint(ctx, orgID, constraintID); err != nil {
		return err
	}
	if err := s.permCache.InvalidateOrganization(ctx, orgID); err != nil {
		s.l.Warn("failed to invalidate permission cache", zap.Error(err))
	}
	return nil
}

func (s *Service) RunRBACPreflight(
	ctx context.Context,
	orgID pulid.ID,
) (*repositories.RBACPreflightReport, error) {
	return s.rbacRepo.RunPreflight(ctx, orgID)
}

func separationOfDutyError(message string) error {
	me := errortypes.NewMultiError()
	me.Add("roleIds", errortypes.ErrForbidden, message)
	return me
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

	tempRole := &permission.Role{
		Permissions: []*permission.ResourcePermission{rp},
	}
	if validateErr := s.validateNoEscalation(ctx, actorID, orgID, tempRole); validateErr != nil {
		return validateErr
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

	tempRole := &permission.Role{
		Permissions: []*permission.ResourcePermission{rp},
	}
	if validateErr := s.validateNoEscalation(ctx, actorID, orgID, tempRole); validateErr != nil {
		return validateErr
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
		CreatedBy:      creatorID,
		Permissions:    s.allResourcePermissions(),
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

func (s *Service) allResourcePermissions() []*permission.ResourcePermission {
	definitions := s.registry.All()
	permissions := make([]*permission.ResourcePermission, 0, len(definitions))
	for _, def := range definitions {
		operations := make([]permission.Operation, len(def.Operations))
		for i, op := range def.Operations {
			operations[i] = op.Operation
		}

		permissions = append(permissions, &permission.ResourcePermission{
			Resource:   def.Resource,
			Operations: operations,
			DataScope:  permission.DataScopeAll,
		})
	}
	return permissions
}
