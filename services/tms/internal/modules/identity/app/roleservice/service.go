package roleservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
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
	RoleRepo         permissionports.RoleRepository
	RBACRepo         permissionports.RBACRepository
	PermissionEngine permissionports.PermissionEngine
	Validator        *Validator
	Registry         *permission.Registry
	Auditor          permissionports.SecurityAuditor
	Notifier         permissionports.PermissionChangeNotifier
	Scopes           permissionports.ScopePolicy
	RoleActivation   iamports.RoleActivationPolicy `optional:"true"`
}

type scopePolicy = permissionports.ScopePolicy

type Service struct {
	l          *zap.Logger
	roleRepo   permissionports.RoleRepository
	rbacRepo   permissionports.RBACRepository
	permEngine permissionports.PermissionEngine
	validator  *Validator
	registry   *permission.Registry
	auditor    permissionports.SecurityAuditor
	scopes     permissionports.ScopePolicy
	activation iamports.RoleActivationPolicy
	notifier   permissionports.PermissionChangeNotifier
}

//nolint:gocritic // dependencies injection
func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.role"),
		roleRepo:   p.RoleRepo,
		rbacRepo:   p.RBACRepo,
		permEngine: p.PermissionEngine,
		validator:  p.Validator,
		registry:   p.Registry,
		auditor:    p.Auditor,
		notifier:   p.Notifier,
		scopes:     p.Scopes,
		activation: p.RoleActivation,
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

type DeleteRoleRequest struct {
	Tenant pagination.TenantInfo
	RoleID pulid.ID
	Force  bool
}

type AssignRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	Assignment     *permission.UserRoleAssignment
}

type UnassignRoleRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	AssignmentID   pulid.ID
}

type RevokeOrganizationRolesRequest struct {
	ActorID        pulid.ID
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
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

	req.Role.IsSystem = false
	req.Role.Permissions = compactPermissions(req.Role.Permissions)

	if scopeErr := unenforcedScopes(
		s.scopes,
		s.registry,
		req.Role.Permissions,
		nil,
		permissionField,
	); scopeErr != nil {
		return scopeErr
	}

	if inheritanceErr := s.validateNoCircularInheritance(
		ctx,
		req.Role.ID,
		req.Role.ParentRoleIDs,
	); inheritanceErr != nil {
		return inheritanceErr
	}

	if validateErr := s.validateNoEscalationWithInheritance(
		ctx,
		&escalationCheck{
			actorID:   req.ActorID,
			orgID:     req.OrganizationID,
			role:      req.Role,
			parentIDs: req.Role.ParentRoleIDs,
		},
	); validateErr != nil {
		return validateErr
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

	s.recordChange(ctx, &roleChange{
		actorID:        req.ActorID,
		organizationID: req.OrganizationID,
		businessUnitID: req.BusinessUnitID,
		resourceID:     req.Role.ID.String(),
		operation:      permission.OpCreate,
		after:          req.Role,
		comment:        "Role created",
	})

	return nil
}

func (s *Service) UpdateRole(ctx context.Context, req UpdateRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "UpdateRole"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("roleID", req.Role.ID.String()),
	)

	existingRole, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID: req.Role.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
		},
	})
	if err != nil {
		log.Error("failed to get existing role", zap.Error(err))
		return err
	}

	if existingRole.IsSystem || existingRole.OrganizationID != req.OrganizationID {
		return ErrCannotModifySystemRole
	}

	updated := editableRoleUpdate(existingRole, req.Role)

	if scopeErr := unenforcedScopes(
		s.scopes,
		s.registry,
		updated.Permissions,
		existingRole.Permissions,
		permissionField,
	); scopeErr != nil {
		return scopeErr
	}

	if validateErr := s.validateNoEscalation(
		ctx,
		req.ActorID,
		req.OrganizationID,
		updated,
	); validateErr != nil {
		return validateErr
	}

	if valErr := s.validator.ValidateUpdate(ctx, updated); valErr != nil {
		return valErr
	}

	if err = s.roleRepo.Update(ctx, updated); err != nil {
		log.Error("failed to update role", zap.Error(err))
		return err
	}

	if updated.Permissions == nil {
		updated.Permissions = existingRole.Permissions
	}
	*req.Role = *updated

	s.recordChange(ctx, &roleChange{
		actorID:        req.ActorID,
		organizationID: req.OrganizationID,
		businessUnitID: existingRole.BusinessUnitID,
		resourceID:     updated.ID.String(),
		operation:      permission.OpUpdate,
		before:         existingRole,
		after:          req.Role,
		comment:        "Role updated",
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   existingRole.BusinessUnitID,
		UserID: req.ActorID,
	})

	return nil
}

func (s *Service) DeleteRole(ctx context.Context, req DeleteRoleRequest) error {
	log := s.l.With(
		zap.String("operation", "DeleteRole"),
		zap.String("actorID", req.Tenant.UserID.String()),
		zap.String("roleID", req.RoleID.String()),
	)

	role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID:         req.RoleID,
		TenantInfo: pagination.TenantInfo{OrgID: req.Tenant.OrgID},
	})
	if err != nil {
		return err
	}
	if role.IsSystem || role.OrganizationID != req.Tenant.OrgID {
		return ErrCannotDeleteSystemRole
	}

	holders, err := s.roleRepo.GetUsersWithRole(ctx, role.ID)
	if err != nil {
		return err
	}
	if len(holders) > 0 {
		if !req.Force {
			return errortypes.NewValidationError(
				"roleId",
				errortypes.ErrInvalidOperation,
				"This role is assigned to {0, plural, one {# person} other {# people}}. Unassign it first, or delete it anyway to take it from them",
				len(holders),
			)
		}
		if err = s.requireUnassign(ctx, req.Tenant); err != nil {
			return err
		}
	}

	if err = s.roleRepo.Delete(ctx, permissionports.DeleteRoleRequest{
		RoleID:         role.ID,
		OrganizationID: req.Tenant.OrgID,
	}); err != nil {
		log.Error("failed to delete role", zap.Error(err))
		return err
	}

	for _, holder := range holders {
		s.recordChange(ctx, &roleChange{
			actorID:        req.Tenant.UserID,
			organizationID: req.Tenant.OrgID,
			businessUnitID: role.BusinessUnitID,
			resourceID:     role.ID.String(),
			operation:      permission.OpUnassign,
			comment:        "Role unassigned because the role was deleted",
			metadata:       map[string]any{metadataUserID: holder.UserID.String()},
		})
	}
	s.recordChange(ctx, &roleChange{
		actorID:        req.Tenant.UserID,
		organizationID: req.Tenant.OrgID,
		businessUnitID: role.BusinessUnitID,
		resourceID:     role.ID.String(),
		operation:      permission.OpDelete,
		before:         role,
		comment:        "Role deleted",
		metadata:       map[string]any{"holderCount": len(holders)},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  req.Tenant.OrgID,
		BuID:   role.BusinessUnitID,
		UserID: req.Tenant.UserID,
	})

	return nil
}

func (s *Service) requireUnassign(ctx context.Context, tenant pagination.TenantInfo) error {
	held, err := s.permEngine.GetEffectivePermissions(ctx, tenant.UserID, tenant.OrgID)
	if err != nil {
		return err
	}

	violation, violated := held.DelegationViolation(
		permission.ResourceUser.String(),
		[]permission.Operation{permission.OpUnassign},
		permission.DataScopeOrganization,
	)
	if violated {
		return violation.FieldError("roleId")
	}

	return nil
}

func editableRoleUpdate(existing, requested *permission.Role) *permission.Role {
	maxSensitivity := requested.MaxSensitivity
	if maxSensitivity == "" {
		maxSensitivity = existing.MaxSensitivity
	}

	return &permission.Role{
		ID:                 existing.ID,
		BusinessUnitID:     existing.BusinessUnitID,
		OrganizationID:     existing.OrganizationID,
		Name:               requested.Name,
		Description:        requested.Description,
		CoreResponsibility: requested.CoreResponsibility,
		ParentRoleIDs:      existing.ParentRoleIDs,
		MaxSensitivity:     maxSensitivity,
		IsSystem:           existing.IsSystem,
		CreatedBy:          existing.CreatedBy,
		CreatedAt:          existing.CreatedAt,
		UpdatedAt:          existing.UpdatedAt,
		Permissions:        compactPermissions(requested.Permissions),
	}
}

func compactPermissions(
	permissions []*permission.ResourcePermission,
) []*permission.ResourcePermission {
	if permissions == nil {
		return nil
	}

	compacted := make([]*permission.ResourcePermission, 0, len(permissions))
	for _, rp := range permissions {
		if rp != nil {
			compacted = append(compacted, rp)
		}
	}

	return compacted
}

func (s *Service) ListRoles(
	ctx context.Context,
	req *permissionports.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return s.roleRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *permissionports.ListRoleConnectionRequest,
) (*pagination.CursorListResult[*permission.Role], error) {
	return s.roleRepo.ListConnection(ctx, req)
}

func (s *Service) SelectRoleOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return s.roleRepo.SelectOptions(ctx, req)
}

func (s *Service) GetRoleByID(
	ctx context.Context,
	req permissionports.GetRoleByIDRequest,
) (*permission.Role, error) {
	return s.roleRepo.GetByID(ctx, req)
}

func (s *Service) GetImpactedUsers(
	ctx context.Context,
	roleID pulid.ID,
) ([]permissionports.ImpactedUser, error) {
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

	role, err := s.validateAssignmentPrivileges(ctx, &req, log)
	if err != nil {
		return err
	}

	req.Assignment.OrganizationID = req.OrganizationID
	req.Assignment.AssignedBy = req.ActorID

	if err = s.validateStaticSeparationOfDutyForAssignment(ctx, &req); err != nil {
		return err
	}

	if err = s.roleRepo.CreateAssignment(ctx, req.Assignment); err != nil {
		log.Error("failed to create assignment", zap.Error(err))
		return err
	}

	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		s.recordChange(committedCtx, &roleChange{
			actorID:        req.ActorID,
			organizationID: req.OrganizationID,
			businessUnitID: role.BusinessUnitID,
			resourceID:     req.Assignment.RoleID.String(),
			operation:      permission.OpAssign,
			after:          req.Assignment,
			comment:        "Role assigned",
			metadata: map[string]any{
				metadataAssignmentID: req.Assignment.ID.String(),
				metadataUserID:       req.Assignment.UserID.String(),
			},
		})

		s.notifier.UserChanged(committedCtx, pagination.TenantInfo{
			OrgID:  req.OrganizationID,
			BuID:   role.BusinessUnitID,
			UserID: req.ActorID,
		}, req.Assignment.UserID)
	})

	return nil
}

func (s *Service) validateAssignmentPrivileges(
	ctx context.Context,
	req *AssignRoleRequest,
	log *zap.Logger,
) (*permission.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID: req.Assignment.RoleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
		},
	})
	if err != nil {
		log.Error("failed to get role", zap.Error(err))
		return nil, err
	}

	return role, s.validateNoEscalationWithInheritance(ctx, &escalationCheck{
		actorID: req.ActorID,
		orgID:   req.OrganizationID,
		role: &permission.Role{
			MaxSensitivity: role.MaxSensitivity,
			Permissions:    role.Permissions,
		},
		parentIDs: role.ParentRoleIDs,
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
	if err := s.validateHierarchyEdge(ctx, req); err != nil {
		return err
	}

	if err := s.rbacRepo.UpsertRoleHierarchyEdge(
		ctx,
		&permissionports.UpsertRoleHierarchyEdgeRequest{
			ActorID:        req.ActorID,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
			SeniorRoleID:   req.SeniorRoleID,
			JuniorRoleID:   req.JuniorRoleID,
		},
	); err != nil {
		if errors.Is(err, permissionports.ErrCircularRoleHierarchy) {
			return ErrCircularInheritance
		}
		return err
	}

	s.recordChange(ctx, &roleChange{
		actorID:        req.ActorID,
		organizationID: req.OrganizationID,
		businessUnitID: req.BusinessUnitID,
		resourceID:     req.SeniorRoleID.String(),
		operation:      permission.OpUpdate,
		comment:        "Role now inherits another role",
		metadata: map[string]any{
			"seniorRoleId": req.SeniorRoleID.String(),
			"juniorRoleId": req.JuniorRoleID.String(),
		},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ActorID,
	})
	return nil
}

func (s *Service) validateHierarchyEdge(
	ctx context.Context,
	req *UpsertHierarchyEdgeRequest,
) error {
	if req.SeniorRoleID == req.JuniorRoleID {
		return ErrCircularInheritance
	}

	tenantInfo := pagination.TenantInfo{OrgID: req.OrganizationID}
	senior, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID:         req.SeniorRoleID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if senior.IsSystem {
		return ErrCannotModifySystemRole
	}

	junior, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID:         req.JuniorRoleID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	return s.validateNoEscalationWithInheritance(ctx, &escalationCheck{
		actorID: req.ActorID,
		orgID:   req.OrganizationID,
		role: &permission.Role{
			MaxSensitivity: junior.MaxSensitivity,
			Permissions:    junior.Permissions,
		},
		parentIDs: junior.ParentRoleIDs,
	})
}

func (s *Service) DeleteRoleHierarchyEdge(
	ctx context.Context,
	tenant pagination.TenantInfo,
	edgeID pulid.ID,
) error {
	if err := s.rbacRepo.DeleteRoleHierarchyEdge(
		ctx,
		permissionports.DeleteRoleHierarchyEdgeRequest{
			OrganizationID: tenant.OrgID,
			EdgeID:         edgeID,
		},
	); err != nil {
		return err
	}

	s.recordChange(ctx, &roleChange{
		actorID:        tenant.UserID,
		organizationID: tenant.OrgID,
		businessUnitID: tenant.BuID,
		resourceID:     edgeID.String(),
		operation:      permission.OpUpdate,
		comment:        "Role inheritance removed",
		metadata:       map[string]any{"edgeId": edgeID.String()},
	})

	s.notifier.OrganizationChanged(ctx, tenant)
	return nil
}

func (s *Service) ListRoleConstraints(
	ctx context.Context,
	req permissionports.ListRoleConstraintsRequest,
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
	if err := s.requireActivationForDSD(ctx, req); err != nil {
		return err
	}

	req.Constraint.OrganizationID = req.OrganizationID
	req.Constraint.BusinessUnitID = req.BusinessUnitID
	req.Constraint.CreatedBy = req.ActorID

	if err := s.rbacRepo.SaveRoleConstraint(ctx, &permissionports.SaveRoleConstraintRequest{
		Constraint: req.Constraint,
		RoleIDs:    req.RoleIDs,
	}); err != nil {
		return err
	}

	roleIDs := make([]string, 0, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		roleIDs = append(roleIDs, roleID.String())
	}
	s.recordChange(ctx, &roleChange{
		actorID:        req.ActorID,
		organizationID: req.OrganizationID,
		businessUnitID: req.BusinessUnitID,
		resourceID:     req.Constraint.ID.String(),
		operation:      permission.OpUpdate,
		after:          req.Constraint,
		comment:        "Separation of duty constraint saved",
		metadata: map[string]any{
			"constraintId": req.Constraint.ID.String(),
			"roleIds":      roleIDs,
		},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ActorID,
	})
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
		if _, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
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
	tenant pagination.TenantInfo,
	constraintID pulid.ID,
) error {
	if err := s.rbacRepo.DeleteRoleConstraint(ctx, tenant.OrgID, constraintID); err != nil {
		return err
	}
	s.recordChange(ctx, &roleChange{
		actorID:        tenant.UserID,
		organizationID: tenant.OrgID,
		businessUnitID: tenant.BuID,
		resourceID:     constraintID.String(),
		operation:      permission.OpUpdate,
		comment:        "Separation of duty constraint removed",
		metadata:       map[string]any{"constraintId": constraintID.String()},
	})
	s.notifier.OrganizationChanged(ctx, tenant)
	return nil
}

func (s *Service) RunRBACPreflight(
	ctx context.Context,
	orgID pulid.ID,
) (*permissionports.RBACPreflightReport, error) {
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

	deleted, err := s.roleRepo.DeleteAssignment(ctx, permissionports.DeleteRoleAssignmentRequest{
		AssignmentID:   req.AssignmentID,
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		log.Error("failed to delete assignment", zap.Error(err))
		return err
	}

	s.recordChange(ctx, &roleChange{
		actorID:        req.ActorID,
		organizationID: req.OrganizationID,
		businessUnitID: req.BusinessUnitID,
		resourceID:     deleted.RoleID.String(),
		operation:      permission.OpUnassign,
		before:         deleted,
		comment:        "Role unassigned",
		metadata: map[string]any{
			metadataAssignmentID: req.AssignmentID.String(),
			metadataUserID:       deleted.UserID.String(),
		},
	})

	s.notifier.UserChanged(ctx, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ActorID,
	}, deleted.UserID)

	return nil
}

type escalationCheck struct {
	actorID   pulid.ID
	orgID     pulid.ID
	role      *permission.Role
	parentIDs []pulid.ID
}

func (s *Service) validateNoEscalationWithInheritance(
	ctx context.Context,
	check *escalationCheck,
) error {
	if len(check.parentIDs) == 0 {
		return s.validateNoEscalation(ctx, check.actorID, check.orgID, check.role)
	}

	inherited, err := s.roleRepo.GetRolesWithInheritance(ctx, check.parentIDs)
	if err != nil {
		return err
	}

	combined := &permission.Role{
		MaxSensitivity: check.role.MaxSensitivity,
		Permissions: make(
			[]*permission.ResourcePermission,
			0,
			len(check.role.Permissions)+len(inherited),
		),
	}
	combined.Permissions = append(combined.Permissions, check.role.Permissions...)
	for _, role := range inherited {
		if role.OrganizationID.IsNotNil() && role.OrganizationID != check.orgID {
			return errortypes.NewValidationError(
				"parentRoleIds",
				errortypes.ErrInvalid,
				"Inherited roles must belong to your organization",
			)
		}
		if role.MaxSensitivity.Level() > combined.MaxSensitivity.Level() {
			combined.MaxSensitivity = role.MaxSensitivity
		}
		combined.Permissions = append(combined.Permissions, role.Permissions...)
	}

	return s.validateNoEscalation(ctx, check.actorID, check.orgID, combined)
}

func (s *Service) RevokeOrganizationRoles(
	ctx context.Context,
	req RevokeOrganizationRolesRequest,
) error {
	log := s.l.With(
		zap.String("operation", "RevokeOrganizationRoles"),
		zap.String("actorID", req.ActorID.String()),
		zap.String("userID", req.UserID.String()),
		zap.String("orgID", req.OrganizationID.String()),
	)

	scoped := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		UserID:         req.ActorID,
	})

	deleted, err := s.roleRepo.DeleteUserAssignments(scoped, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to delete role assignments", zap.Error(err))
		return err
	}

	for _, assignment := range deleted {
		s.recordChange(scoped, &roleChange{
			actorID:        req.ActorID,
			organizationID: req.OrganizationID,
			businessUnitID: req.BusinessUnitID,
			resourceID:     assignment.RoleID.String(),
			operation:      permission.OpUnassign,
			before:         assignment,
			comment:        "Role unassigned because the user left the organization",
			metadata: map[string]any{
				metadataAssignmentID: assignment.ID.String(),
				metadataUserID:       assignment.UserID.String(),
			},
		})
	}

	s.notifier.UserChanged(scoped, pagination.TenantInfo{
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		UserID: req.ActorID,
	}, req.UserID)

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
		s.recordRefusedGrant(ctx, actorID, orgID, targetRole, map[string]any{
			"refused":          "maxSensitivity",
			"maxSensitivity":   string(targetRole.MaxSensitivity),
			"actorSensitivity": string(actorPerms.MaxSensitivity),
		})
		me := errortypes.NewMultiError()
		me.Add(
			"maxSensitivity",
			errortypes.ErrForbidden,
			"cannot grant sensitivity level higher than your own",
		)
		return me
	}

	for _, rp := range targetRole.Permissions {
		if rp == nil {
			continue
		}
		violation, violated := actorPerms.DelegationViolation(
			rp.Resource,
			rp.Operations,
			rp.DataScope,
		)
		if violated {
			s.recordRefusedGrant(ctx, actorID, orgID, targetRole, map[string]any{
				"refused":   "permissions",
				"resource":  violation.Resource,
				"operation": string(violation.Operation),
				"dataScope": string(violation.Scope),
			})
			return violation.FieldError("permissions")
		}
	}

	return nil
}

func (s *Service) recordRefusedGrant(
	ctx context.Context,
	actorID, orgID pulid.ID,
	targetRole *permission.Role,
	metadata map[string]any,
) {
	s.recordChange(ctx, &roleChange{
		actorID:        actorID,
		organizationID: orgID,
		businessUnitID: targetRole.BusinessUnitID,
		resourceID:     targetRole.ID.String(),
		operation:      permission.OpUpdate,
		comment:        "Refused a grant beyond the actor's own permissions",
		metadata:       metadata,
	})
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

		role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{ID: id})
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

	role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
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

	if scopeErr := unenforcedScopes(
		s.scopes,
		s.registry,
		[]*permission.ResourcePermission{rp},
		role.Permissions,
		singleField,
	); scopeErr != nil {
		return scopeErr
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

	s.recordChange(ctx, &roleChange{
		actorID:        actorID,
		organizationID: orgID,
		businessUnitID: role.BusinessUnitID,
		resourceID:     role.ID.String(),
		operation:      permission.OpUpdate,
		after:          rp,
		comment:        "Permission granted to role",
		metadata:       map[string]any{"resource": rp.Resource},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  orgID,
		BuID:   role.BusinessUnitID,
		UserID: actorID,
	})

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

	role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
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

	previous := findResourcePermission(role.Permissions, rp.ID)

	if scopeErr := unenforcedScopes(
		s.scopes,
		s.registry,
		[]*permission.ResourcePermission{rp},
		role.Permissions,
		singleField,
	); scopeErr != nil {
		return scopeErr
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

	s.recordChange(ctx, &roleChange{
		actorID:        actorID,
		organizationID: orgID,
		businessUnitID: role.BusinessUnitID,
		resourceID:     role.ID.String(),
		operation:      permission.OpUpdate,
		before:         previous,
		after:          rp,
		comment:        "Role permission changed",
		metadata:       map[string]any{"resource": rp.Resource},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  orgID,
		BuID:   role.BusinessUnitID,
		UserID: actorID,
	})

	return nil
}

func (s *Service) DeleteResourcePermission(
	ctx context.Context,
	tenant pagination.TenantInfo,
	permID, roleID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteResourcePermission"),
		zap.String("id", permID.String()),
	)

	role, err := s.roleRepo.GetByID(ctx, permissionports.GetRoleByIDRequest{
		ID: roleID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenant.OrgID,
		},
	})
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrCannotModifySystemRole
	}

	if err = s.roleRepo.DeleteResourcePermission(
		ctx,
		permissionports.DeleteResourcePermissionRequest{
			PermissionID: permID,
			RoleID:       role.ID,
		},
	); err != nil {
		log.Error("failed to delete resource permission", zap.Error(err))
		return err
	}

	s.recordChange(ctx, &roleChange{
		actorID:        tenant.UserID,
		organizationID: tenant.OrgID,
		businessUnitID: role.BusinessUnitID,
		resourceID:     role.ID.String(),
		operation:      permission.OpUpdate,
		before:         findResourcePermission(role.Permissions, permID),
		comment:        "Permission removed from role",
		metadata:       map[string]any{"permissionId": permID.String()},
	})

	s.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
		OrgID:  tenant.OrgID,
		BuID:   role.BusinessUnitID,
		UserID: tenant.UserID,
	})

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

	s.recordChange(ctx, &roleChange{
		actorID:        creatorID,
		organizationID: orgID,
		resourceID:     adminRole.ID.String(),
		operation:      permission.OpAssign,
		after:          assignment,
		comment:        "Organization administrator role created and assigned to the organization's creator",
		metadata:       map[string]any{metadataUserID: creatorID.String()},
	})

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

func (s *Service) requireActivationForDSD(ctx context.Context, req *SaveConstraintRequest) error {
	if req.Constraint.Type != permission.RoleConstraintTypeDSD || s.activation == nil {
		return nil
	}
	required, err := s.activation.RoleActivationRequired(ctx, pagination.TenantInfo{
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
	})
	if err != nil {
		return err
	}
	if !required {
		return errortypes.NewValidationError(
			"type",
			errortypes.ErrInvalid,
			"Dynamic separation of duty is enforced when people activate their roles. Turn on role activation in the security settings first.",
		)
	}
	return nil
}
