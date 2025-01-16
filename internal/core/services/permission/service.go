package permission

import (
	"context"
	"fmt"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger *logger.Logger
	Repo   repositories.PermissionRepository
}

type Service struct {
	repo  repositories.PermissionRepository
	l     *zerolog.Logger
	clock Clock
}

func NewService(p ServiceParams) services.PermissionService {
	log := p.Logger.With().
		Str("service", "permission").
		Logger()

	return &Service{
		repo:  p.Repo,
		l:     &log,
		clock: &realClock{},
	}
}

// CheckFieldModification checks if a user is allowed to modify a specific field of a resource.
// It evaluates the user's permissions and roles to determine field-level access.
func (s *Service) CheckFieldModification(ctx context.Context, userID pulid.ID, resource permission.Resource, field string) services.FieldPermissionCheck {
	permissions, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user permissions"),
		}
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user roles from cache"),
		}
	}

	permCtx := &services.PermissionContext{
		UserID:     userID,
		Roles:      roles,
		Time:       s.clock.Now(),
		CustomData: map[string]any{},
	}

	for _, perm := range permissions {
		if perm.Resource != resource {
			continue
		}

		if check := evaluateFieldPermission(perm, field, permCtx); check.Allowed {
			return check
		}
	}

	return services.FieldPermissionCheck{
		Allowed: false,
		Error:   eris.New("no valid permission found for field modification"),
	}
}

// HasPermission checks if a user has permission to perform a specific action on a resource.
// It verifies the user's roles and permissions and evaluates conditions and scopes.
func (s *Service) HasPermission(ctx context.Context, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	if check.UserID.IsNil() {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.New("user ID is required"),
		}, nil
	}

	rnp, err := s.repo.GetRolesAndPermissions(ctx, check.UserID)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user roles and permissions"),
		}, nil
	}

	permCTX := &services.PermissionContext{
		UserID:     check.UserID,
		BuID:       check.BusinessUnitID,
		OrgID:      check.OrganizationID,
		Roles:      rnp.Roles,
		Time:       time.Now().UTC(),
		CustomData: check.CustomData,
	}

	for _, permission := range rnp.Permissions {
		if s.matchesPermission(permission, check, permCTX) {
			return services.PermissionCheckResult{
				Allowed: true,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: false,
		Error:   nil,
	}, nil
}

func (s *Service) checkMultiPermissions(ctx context.Context, checks []*services.PermissionCheck) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, check := range checks {
		key := fmt.Sprintf("%s:%s", check.Resource, check.Action)
		hasPermission, err := s.HasPermission(ctx, check)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to check permission for %s", key)
		}
		results[key] = hasPermission.Allowed
	}

	return results, nil
}

// HasAllPermissions checks if a user has all the specified permissions.
// This method evaluates specific actions as well as the "manage" action for each resource.
func (s *Service) HasFieldPermission(ctx context.Context, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	if check.Field == "" {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   ErrFieldRequired,
		}, nil
	}

	results, err := s.HasPermission(ctx, check)
	if err != nil || !results.Allowed {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "check field permission"),
		}, nil
	}

	rnp, err := s.repo.GetRolesAndPermissions(ctx, check.UserID)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user roles and permissions"),
		}, nil
	}

	permCTX := &services.PermissionContext{
		UserID:     check.UserID,
		BuID:       check.BusinessUnitID,
		OrgID:      check.OrganizationID,
		Roles:      rnp.Roles,
		Time:       time.Now().UTC(),
		CustomData: check.CustomData,
	}

	for _, permission := range rnp.Permissions {
		if s.matchesFieldPermission(permission, check, permCTX) {
			return services.PermissionCheckResult{
				Allowed: true,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: false,
		Error:   nil,
	}, nil
}

// HasAllPermissions checks if a user has all the specified permissions.
// This method evaluates specific actions as well as the "manage" action for each resource.
func (s *Service) HasAllPermissions(ctx context.Context, checks []*services.PermissionCheck) (services.PermissionCheckResult, error) {
	allChecks := make([]*services.PermissionCheck, 0, len(checks)*2)

	for _, check := range checks {
		allChecks = append(allChecks, check)

		manageCheck := services.PermissionCheck{
			UserID:         check.UserID,
			Resource:       check.Resource,
			Action:         permission.ActionManage,
			BusinessUnitID: check.BusinessUnitID,
			OrganizationID: check.OrganizationID,
			ResourceID:     check.ResourceID,
			CustomData:     check.CustomData,
		}
		allChecks = append(allChecks, &manageCheck)
	}

	results, err := s.checkMultiPermissions(ctx, allChecks)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "check multiple permissions"),
		}, nil
	}

	for _, check := range checks {
		specificKey := fmt.Sprintf("%s:%s", check.Resource, check.Action)
		manageKey := fmt.Sprintf("%s:%s", check.Resource, permission.ActionManage)

		if !results[specificKey] && !results[manageKey] {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: true,
		Error:   nil,
	}, nil
}

// HasAnyPermissions checks if a user has at least one of the specified permissions.
// This method evaluates both specific actions and the "manage" action for each resource.
func (s *Service) HasAnyPermissions(ctx context.Context, checks []*services.PermissionCheck) (services.PermissionCheckResult, error) {
	allChecks := make([]*services.PermissionCheck, 0, len(checks)*2)

	for _, check := range checks {
		allChecks = append(allChecks, check)

		manageCheck := services.PermissionCheck{
			UserID:         check.UserID,
			Resource:       check.Resource,
			Action:         permission.ActionManage,
			BusinessUnitID: check.BusinessUnitID,
			OrganizationID: check.OrganizationID,
			ResourceID:     check.ResourceID,
			CustomData:     check.CustomData,
		}
		allChecks = append(allChecks, &manageCheck)
	}

	results, err := s.checkMultiPermissions(ctx, allChecks)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "check multiple permissions"),
		}, nil
	}

	for _, check := range checks {
		specificKey := fmt.Sprintf("%s:%s", check.Resource, check.Action)
		manageKey := fmt.Sprintf("%s:%s", check.Resource, permission.ActionManage)

		if results[specificKey] || results[manageKey] {
			return services.PermissionCheckResult{
				Allowed: true,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: false,
		Error:   nil,
	}, nil
}

func (s *Service) HasAnyFieldPermissions(ctx context.Context, fields []string, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	for _, field := range fields {
		fieldCheck := check
		fieldCheck.Field = field

		result, err := s.HasFieldPermission(ctx, fieldCheck)
		if err != nil {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   eris.Wrapf(err, "check field permission for %s", field),
			}, nil
		}
		if result.Allowed {
			return services.PermissionCheckResult{
				Allowed: true,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: false,
		Error:   nil,
	}, nil
}

func (s *Service) HasAllFieldPermissions(ctx context.Context, fields []string, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	for _, field := range fields {
		fieldCheck := check
		fieldCheck.Field = field

		result, err := s.HasFieldPermission(ctx, fieldCheck)
		if err != nil {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   eris.Wrapf(err, "check field permission for %s", field),
			}, nil
		}
		if !result.Allowed {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: true,
		Error:   nil,
	}, nil
}

func (s *Service) HasScopedPermission(ctx context.Context, check *services.PermissionCheck, requiredScope permission.Scope) (services.PermissionCheckResult, error) {
	permissions, err := s.repo.GetUserPermissions(ctx, check.UserID)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user permissions"),
		}, nil
	}

	roles, err := s.repo.GetUserRoles(ctx, check.UserID)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user roles"),
		}, nil
	}

	permCTX := &services.PermissionContext{
		UserID:     check.UserID,
		BuID:       check.BusinessUnitID,
		OrgID:      check.OrganizationID,
		Roles:      roles,
		Time:       time.Now().UTC(),
		CustomData: check.CustomData,
	}

	for _, permission := range permissions {
		if permission.Resource == check.Resource &&
			permission.Scope == requiredScope &&
			s.matchesPermission(permission, check, permCTX) {
			return services.PermissionCheckResult{
				Allowed: true,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: false,
		Error:   nil,
	}, nil
}

func (s *Service) HasDependentPermissions(ctx context.Context, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	hasMainPerm, err := s.HasPermission(ctx, check)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "check main permission"),
		}, nil
	}
	if !hasMainPerm.Allowed {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   nil,
		}, nil
	}

	permissions, err := s.repo.GetUserPermissions(ctx, check.UserID)
	if err != nil {
		return services.PermissionCheckResult{
			Allowed: false,
			Error:   eris.Wrap(err, "get user permissions"),
		}, nil
	}

	var mainPermission *permission.Permission
	for _, p := range permissions {
		if p.Resource == check.Resource && p.Action == check.Action {
			mainPermission = p
			break
		}
	}

	if mainPermission == nil || len(mainPermission.Dependencies) == 0 {
		return services.PermissionCheckResult{
			Allowed: true,
			Error:   nil,
		}, nil
	}

	permMap := make(map[string]*permission.Permission)
	for _, p := range permissions {
		permMap[p.ID.String()] = p
	}

	for _, depID := range mainPermission.Dependencies {
		depPerm, exists := permMap[depID.String()]
		if !exists {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   nil,
			}, nil
		}

		depCheck := &services.PermissionCheck{
			UserID:         check.UserID,
			Resource:       depPerm.Resource,
			Action:         depPerm.Action,
			BusinessUnitID: check.BusinessUnitID,
			OrganizationID: check.OrganizationID,
			ResourceID:     check.ResourceID,
			CustomData:     check.CustomData,
		}

		hasDepPerm, hdpErr := s.HasPermission(ctx, depCheck)
		if hdpErr != nil {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   eris.Wrapf(hdpErr, "check dependent permission %s", depID),
			}, nil
		}
		if !hasDepPerm.Allowed {
			return services.PermissionCheckResult{
				Allowed: false,
				Error:   nil,
			}, nil
		}
	}

	return services.PermissionCheckResult{
		Allowed: true,
		Error:   nil,
	}, nil
}

// HasTemporalPermission checks if the user has a permission that is time-based
// This is useful for checking if a user has permission to view a user, but only if they have permission to view the user's organization
func (s *Service) HasTemporalPermission(ctx context.Context, check *services.PermissionCheck) (services.PermissionCheckResult, error) {
	if check.CustomData == nil {
		check.CustomData = make(map[string]any)
	}

	check.CustomData["currentTime"] = time.Now().UTC()

	return s.HasPermission(ctx, check)
}

func (s *Service) GetEffectivePermissions(ctx context.Context, userID pulid.ID, resource permission.Resource) ([]permission.Action, error) {
	permissions, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, eris.Wrap(err, "get user permissions")
	}

	effectiveActions := make(map[permission.Action]bool)

	for _, perm := range permissions {
		if perm.Resource == resource {
			effectiveActions[perm.Action] = true
			if perm.Action == permission.ActionManage {
				return getAllResourceActions(resource), nil
			}
		}
	}

	result := make([]permission.Action, 0, len(effectiveActions))
	for action := range effectiveActions {
		result = append(result, action)
	}

	return result, nil
}

func (s *Service) matchesPermission(perm *permission.Permission, check *services.PermissionCheck, ctx *services.PermissionContext) bool {
	if !supportsAction(check.Resource, check.Action) {
		return false
	}

	if perm.Resource != check.Resource {
		return false
	}

	if !checkScope(perm.Scope, check) {
		return false
	}

	for _, condition := range perm.Conditions {
		if !evaluateCondition(condition, ctx) {
			return false
		}
	}

	return true
}

func (s *Service) matchesFieldPermission(perm *permission.Permission, check *services.PermissionCheck, ctx *services.PermissionContext) bool {
	if !s.matchesPermission(perm, check, ctx) {
		return false
	}

	return canModifyField(check.Field, ctx, perm.FieldPermissions)
}

func (s *Service) CheckFieldAccess(ctx context.Context, userID pulid.ID, resource permission.Resource, field string) services.FieldAccess {
	access := services.FieldAccess{
		CanModify: false,
		CanView:   false,
		Errors:    make([]error, 0),
	}

	modifyCheck := s.CheckFieldModification(ctx, userID, resource, field)
	if modifyCheck.Error != nil {
		access.Errors = append(access.Errors, modifyCheck.Error)
	}
	access.CanModify = modifyCheck.Allowed

	viewCheck := s.CheckFieldView(ctx, userID, resource, field)
	if viewCheck.Error != nil {
		access.Errors = append(access.Errors, viewCheck.Error)
	}
	access.CanView = viewCheck.Allowed

	return access
}

func (s *Service) CheckFieldView(ctx context.Context, userID pulid.ID, resource permission.Resource, field string) services.FieldPermissionCheck {
	permissions, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user permissions from cache"),
		}
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   eris.Wrap(err, "failed to get user roles from cache"),
		}
	}

	permCtx := &services.PermissionContext{
		UserID:     userID,
		Roles:      roles,
		Time:       s.clock.Now(),
		CustomData: map[string]any{},
	}

	for _, perm := range permissions {
		if perm.Resource != resource {
			continue
		}

		if check := evaluateFieldViewPermission(perm, field, permCtx); check.Allowed {
			return check
		}
	}

	return services.FieldPermissionCheck{
		Allowed: false,
		Error:   eris.New("no valid permission found for field view"),
	}
}
