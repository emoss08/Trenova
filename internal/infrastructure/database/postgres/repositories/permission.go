package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permission/permissiongrant"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type PermissionRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	Cache  repositories.PermissionCacheRepository
}

type permissionRepository struct {
	db    db.Connection
	l     *zerolog.Logger
	cache repositories.PermissionCacheRepository
}

func NewPermissionRepository(p PermissionRepositoryParams) repositories.PermissionRepository {
	log := p.Logger.With().
		Str("repository", "permission").
		Str("component", "database").
		Logger()

	return &permissionRepository{
		db:    p.DB,
		l:     &log,
		cache: p.Cache,
	}
}

func (pr *permissionRepository) addRoleFilter(
	q *bun.SelectQuery,
	req repositories.RolesQueryOptions,
) *bun.SelectQuery {
	if req.IncludeChildren {
		q = q.Relation("ChildRoles")
	}

	if req.IncludeParent {
		q = q.Relation("ParentRole")
	}

	if req.IncludePermissions {
		q = q.Relation("Permissions")
	}

	return q
}

func (pr *permissionRepository) filterRolesQuery(
	q *bun.SelectQuery,
	req *repositories.ListRolesRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "r",
		Filter:     req.Filter,
	})

	q = pr.addRoleFilter(q, req.QueryOptions)
	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (pr *permissionRepository) ListRoles(
	ctx context.Context,
	req *repositories.ListRolesRequest,
) (*ports.ListResult[*permission.Role], error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "ListRoles").
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	entities := make([]*permission.Role, 0)

	q := dba.NewSelect().Model(&entities)
	q = pr.filterRolesQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan roles")
		return nil, oops.In("permission_repository").
			Tags("crud", "list").
			Time(time.Now()).
			Wrapf(err, "get roles")
	}

	return &ports.ListResult[*permission.Role]{
		Items: entities,
		Total: total,
	}, nil
}

func (pr *permissionRepository) GetRoleByID(
	ctx context.Context,
	req *repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "GetRoleByID").
		Str("roleID", req.RoleID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Str("userID", req.UserID.String()).
		Logger()

	entity := new(permission.Role)

	q := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("r.id = ?", req.RoleID).
				Where("r.organization_id = ?", req.OrgID).
				Where("r.business_unit_id = ?", req.BuID)
		})

	q = pr.addRoleFilter(q, req.QueryOptions)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Role not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get role by id")
		return nil, eris.Wrap(err, "failed to get role by id")
	}

	return entity, nil
}

func (pr *permissionRepository) GetUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) ([]*permission.Permission, error) {
	log := pr.l.With().
		Str("operation", "GetUserPermissions").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	permissions, err := pr.cache.GetUserPermissions(ctx, userID)
	if err == nil && len(permissions) > 0 {
		log.Debug().Int("count", len(permissions)).Msg("got permissions from cache")
		return permissions, nil
	}

	// On cache miss, get from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get permissions from database with role information
	var dbPermissions []*permission.Permission
	err = dba.NewSelect().
		Model(&dbPermissions).
		Join("JOIN role_permissions rp ON rp.permission_id = perm.id").
		Join("JOIN user_roles ur ON ur.role_id = rp.role_id").
		ColumnExpr("perm.*").
		ColumnExpr("rp.role_id AS role_id").
		Where("ur.user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permissions")
	}

	// Get permission grants
	grants := make([]*permissiongrant.Grant, 0)
	err = dba.NewSelect().
		Model(&grants).
		Relation("Permission").
		Where("pg.user_id = ?", userID).
		Where("pg.status = ?", permission.StatusActive).
		Where("pg.expires_at IS NULL OR pg.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permission grants")
	}

	// Combine permissions from roles and grants
	allPermissions := make([]*permission.Permission, 0, len(dbPermissions)+len(grants))
	allPermissions = append(allPermissions, dbPermissions...)

	for _, grant := range grants {
		if grant.Permission != nil {
			if len(grant.FieldOverrides) > 0 {
				grant.Permission.FieldPermissions = grant.FieldOverrides
			}
			allPermissions = append(allPermissions, grant.Permission)
		}
	}

	// Cache the permissions
	if len(allPermissions) > 0 {
		if err = pr.cache.SetUserPermissions(ctx, userID, allPermissions); err != nil {
			log.Warn().Err(err).Msg("failed to cache user permissions")
		}
	}

	log.Debug().Int("count", len(allPermissions)).Msg("got permissions from database")
	return allPermissions, nil
}

func (pr *permissionRepository) GetUserRoles(
	ctx context.Context,
	userID pulid.ID,
) ([]*string, error) {
	log := pr.l.With().
		Str("operation", "GetUserRoles").
		Str("userId", userID.String()).
		Logger()

	// Try to get from cache first
	roles, err := pr.cache.GetUserRoles(ctx, userID)
	if err == nil && len(roles) > 0 {
		log.Debug().Int("count", len(roles)).Msg("got roles from cache")
		return roles, nil
	}

	// On cache miss, get from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get roles from database
	dbRoles := make([]*permission.Role, 0)
	err = dba.NewSelect().
		Model(&dbRoles).
		Join("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("r.status = ?", permission.StatusActive).
		Where("r.expires_at IS NULL OR r.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user roles")
	}

	// Extract just the role names for the cache format
	roleNames := make([]*string, len(dbRoles))
	for i, role := range dbRoles {
		roleNames[i] = &role.Name
	}

	// Cache the roles
	if len(roleNames) > 0 {
		if err = pr.cache.SetUserRoles(ctx, userID, roleNames); err != nil {
			log.Warn().Err(err).Msg("failed to cache user roles")
		}
	}

	log.Debug().Int("count", len(roleNames)).Msg("got roles from database")
	return roleNames, nil
}

func (pr *permissionRepository) InvalidateUserPermissions(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := pr.l.With().
		Str("operation", "InvalidateUserPermissions").
		Str("userId", userID.String()).
		Logger()

	if err := pr.cache.InvalidateAllUserData(ctx, userID); err != nil {
		log.Error().Err(err).Msg("failed to invalidate user cache")
		return eris.Wrap(err, "invalidate user cache")
	}

	log.Debug().Msg("invalidated user permissions and roles")
	return nil
}

func (pr *permissionRepository) GetRolesAndPermissions(
	ctx context.Context,
	userID pulid.ID,
) (*permission.RolesAndPermissions, error) {
	log := pr.l.With().
		Str("operation", "GetRolesAndPermissions").
		Str("userId", userID.String()).
		Logger()

	// First try to get from cache
	permissions, permErr := pr.cache.GetUserPermissions(ctx, userID)
	roles, rolesErr := pr.cache.GetUserRoles(ctx, userID)

	// If cache hit for both, return early
	if permErr == nil && rolesErr == nil && len(roles) > 0 {
		// log.Debug().
		// 	Int("roleCount", len(roles)).
		// 	Int("permissionCount", len(permissions)).
		// 	Msg("got roles and permissions from cache")

		return &permission.RolesAndPermissions{
			Roles:       roles,
			Permissions: permissions,
		}, nil
	}

	// If cache miss, load from database
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	// Get roles with their permissions in one query
	var dbRoles []*permission.Role
	err = dba.NewSelect().
		Model(&dbRoles).
		Relation("Permissions").
		Join("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("r.status = ?", permission.StatusActive).
		Where("r.expires_at IS NULL OR r.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user roles with permissions")
	}

	// Get permission grants
	grants := make([]*permissiongrant.Grant, 0)
	err = dba.NewSelect().
		Model(&grants).
		Relation("Permission").
		Where("pg.user_id = ?", userID).
		Where("pg.status = ?", permission.StatusActive).
		Where("pg.expires_at IS NULL OR pg.expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get user permission grants")
	}

	// Extract role names for cache and collect all permissions
	roleNames := make([]*string, len(dbRoles))
	allPermissions := make([]*permission.Permission, 0)

	for i, role := range dbRoles {
		roleNames[i] = &role.Name
		allPermissions = append(allPermissions, role.Permissions...)
	}

	// Add grant permissions
	for _, grant := range grants {
		if grant.Permission != nil {
			if len(grant.FieldOverrides) > 0 {
				grant.Permission.FieldPermissions = grant.FieldOverrides
			}
			allPermissions = append(allPermissions, grant.Permission)
		}
	}

	// Cache the data for future requests
	if len(roleNames) > 0 {
		if err = pr.cache.SetUserRoles(ctx, userID, roleNames); err != nil {
			log.Warn().Err(err).Msg("failed to cache user roles")
		}
	}

	if len(allPermissions) > 0 {
		if err = pr.cache.SetUserPermissions(ctx, userID, allPermissions); err != nil {
			log.Warn().Err(err).Msg("failed to cache user permissions")
		}
	}

	log.Debug().
		Int("roleCount", len(dbRoles)).
		Int("permissionCount", len(allPermissions)).
		Msg("got roles and permissions from database")

	return &permission.RolesAndPermissions{
		Roles:         roleNames,
		Permissions:   allPermissions,
		CompleteRoles: dbRoles,
	}, nil
}

// CreateRole creates a new role with associated permissions
func (pr *permissionRepository) CreateRole(
	ctx context.Context,
	req *repositories.CreateRoleRequest,
) (*permission.Role, error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "CreateRole").
		Str("roleName", req.Role.Name).
		Str("orgID", req.OrganizationID.String()).
		Str("buID", req.BusinessUnitID.String()).
		Logger()

	// Set the organization and business unit IDs
	req.Role.OrganizationID = req.OrganizationID
	req.Role.BusinessUnitID = req.BusinessUnitID

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Insert the role
		if _, iErr := tx.NewInsert().Model(req.Role).Exec(c); iErr != nil {
			log.Error().Err(iErr).Msg("failed to insert role")
			return iErr
		}

		// Handle role permissions
		if err = pr.handleRolePermissions(c, tx, req.Role, req.PermissionIDs, true); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to create role")
		return nil, err
	}

	log.Debug().Msg("role created successfully")
	return req.Role, nil
}

// UpdateRole updates an existing role and its permissions
func (pr *permissionRepository) UpdateRole(
	ctx context.Context,
	req *repositories.UpdateRoleRequest,
) (*permission.Role, error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "UpdateRole").
		Str("roleID", req.Role.ID.String()).
		Str("roleName", req.Role.Name).
		Str("orgID", req.OrganizationID.String()).
		Str("buID", req.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Update the role
		result, uErr := tx.NewUpdate().
			Model(req.Role).
			Where("r.id = ?", req.Role.ID).
			OmitZero().
			Where("r.organization_id = ?", req.OrganizationID).
			Where("r.business_unit_id = ?", req.BusinessUnitID).
			Where("r.is_system = false"). // Prevent updating system roles
			Exec(c)
		if uErr != nil {
			log.Error().Err(uErr).Msg("failed to update role")
			return uErr
		}

		rowsAffected, raErr := result.RowsAffected()
		if raErr != nil {
			return raErr
		}

		if rowsAffected == 0 {
			return errors.NewNotFoundError("Role not found or is a system role")
		}

		// Handle role permissions
		if err = pr.handleRolePermissions(c, tx, req.Role, req.PermissionIDs, false); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to update role")
		return nil, err
	}

	log.Debug().Msg("role updated successfully")
	return req.Role, nil
}

// DeleteRole deletes a role and its associated permissions
func (pr *permissionRepository) DeleteRole(
	ctx context.Context,
	req *repositories.DeleteRoleRequest,
) error {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "DeleteRole").
		Str("roleID", req.RoleID.String()).
		Str("orgID", req.OrganizationID.String()).
		Str("buID", req.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// First delete role permissions
		_, dpErr := tx.NewDelete().
			Model((*permission.RolePermission)(nil)).
			Where("role_id = ?", req.RoleID).
			Where("organization_id = ?", req.OrganizationID).
			Where("business_unit_id = ?", req.BusinessUnitID).
			Exec(c)
		if dpErr != nil {
			log.Error().Err(dpErr).Msg("failed to delete role permissions")
			return dpErr
		}

		// Then delete user roles
		_, durErr := tx.NewDelete().
			Model((*user.UserRole)(nil)).
			Where("role_id = ?", req.RoleID).
			Where("organization_id = ?", req.OrganizationID).
			Where("business_unit_id = ?", req.BusinessUnitID).
			Exec(c)
		if durErr != nil {
			log.Error().Err(durErr).Msg("failed to delete user roles")
			return durErr
		}

		// Finally delete the role
		result, drErr := tx.NewDelete().
			Model((*permission.Role)(nil)).
			Where("r.id = ?", req.RoleID).
			Where("r.organization_id = ?", req.OrganizationID).
			Where("r.business_unit_id = ?", req.BusinessUnitID).
			Where("r.is_system = false"). // Prevent deleting system roles
			Exec(c)
		if drErr != nil {
			log.Error().Err(drErr).Msg("failed to delete role")
			return drErr
		}

		rowsAffected, raErr := result.RowsAffected()
		if raErr != nil {
			return raErr
		}

		if rowsAffected == 0 {
			return errors.NewNotFoundError("Role not found or is a system role")
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to delete role")
		return err
	}

	log.Debug().Msg("role deleted successfully")
	return nil
}

// ListPermissions lists all available permissions
func (pr *permissionRepository) ListPermissions(
	ctx context.Context,
	req *repositories.ListPermissionsRequest,
) (*ports.ListResult[*permission.Permission], error) {
	dba, err := pr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := pr.l.With().
		Str("operation", "ListPermissions").
		Str("orgID", req.OrganizationID.String()).
		Str("buID", req.BusinessUnitID.String()).
		Logger()

	permissions := make([]*permission.Permission, 0)

	q := dba.NewSelect().Model(&permissions).
		Order("perm.resource ASC", "perm.action ASC")

	// Apply pagination
	if req.Filter != nil {
		q = q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan permissions")
		return nil, eris.Wrap(err, "scan permissions")
	}

	return &ports.ListResult[*permission.Permission]{
		Items: permissions,
		Total: total,
	}, nil
}

// handleRolePermissions manages the role-permission associations
func (pr *permissionRepository) handleRolePermissions(
	ctx context.Context,
	tx bun.IDB,
	role *permission.Role,
	permissionIDs []pulid.ID,
	isCreate bool,
) error {
	// Early return if no permissions to assign
	if len(permissionIDs) == 0 && isCreate {
		return nil
	}

	// Get existing role permissions for update operations
	existingPermissionMap := make(map[pulid.ID]*permission.RolePermission)
	if !isCreate {
		if err := pr.loadExistingRolePermissionsMap(ctx, tx, role, existingPermissionMap); err != nil {
			return err
		}
	}

	// Categorize permissions
	newRolePermissions, updatedPermissionIDs := pr.categorizeRolePermissions(
		role, permissionIDs, existingPermissionMap, isCreate)

	// Process database operations
	if err := pr.processRolePermissionOperations(ctx, tx, newRolePermissions); err != nil {
		return err
	}

	// Handle deletions for update operations
	if !isCreate {
		rolePermissionsToDelete := make([]*permission.RolePermission, 0)
		if err := pr.handleRolePermissionDeletions(ctx, tx, existingPermissionMap, updatedPermissionIDs, &rolePermissionsToDelete); err != nil {
			pr.l.Error().Err(err).Msg("failed to handle role permission deletions")
			return err
		}

		pr.l.Debug().Int("newPermissions", len(newRolePermissions)).
			Int("deletedPermissions", len(rolePermissionsToDelete)).
			Msg("role permission operations completed")
	} else {
		pr.l.Debug().Int("newPermissions", len(newRolePermissions)).
			Msg("role permission operations completed")
	}

	return nil
}

// loadExistingRolePermissionsMap loads existing role permissions into a map
func (pr *permissionRepository) loadExistingRolePermissionsMap(
	ctx context.Context,
	tx bun.IDB,
	role *permission.Role,
	permissionMap map[pulid.ID]*permission.RolePermission,
) error {
	existingPermissions, err := pr.getExistingRolePermissions(ctx, tx, role)
	if err != nil {
		pr.l.Error().Err(err).Msg("failed to get existing role permissions")
		return err
	}

	for _, rolePermission := range existingPermissions {
		permissionMap[rolePermission.PermissionID] = rolePermission
	}

	return nil
}

// categorizeRolePermissions categorizes permissions for different operations
func (pr *permissionRepository) categorizeRolePermissions(
	role *permission.Role,
	permissionIDs []pulid.ID,
	existingPermissionMap map[pulid.ID]*permission.RolePermission,
	isCreate bool,
) (newRolePermissions []*permission.RolePermission, updatedPermissionIDs map[pulid.ID]struct{}) {
	newRolePermissions = make([]*permission.RolePermission, 0)
	updatedPermissionIDs = make(map[pulid.ID]struct{})

	for _, permissionID := range permissionIDs {
		// Check if this permission assignment already exists
		if _, exists := existingPermissionMap[permissionID]; !exists || isCreate {
			// Create new RolePermission assignment
			rolePermission := &permission.RolePermission{
				BusinessUnitID: role.BusinessUnitID,
				OrganizationID: role.OrganizationID,
				RoleID:         role.ID,
				PermissionID:   permissionID,
			}
			newRolePermissions = append(newRolePermissions, rolePermission)
		} else {
			// Mark as updated (exists and should remain)
			updatedPermissionIDs[permissionID] = struct{}{}
		}
	}

	return newRolePermissions, updatedPermissionIDs
}

// processRolePermissionOperations handles database insert operations
func (pr *permissionRepository) processRolePermissionOperations(
	ctx context.Context,
	tx bun.IDB,
	newRolePermissions []*permission.RolePermission,
) error {
	// Handle bulk insert of new role permission assignments
	if len(newRolePermissions) > 0 {
		if _, err := tx.NewInsert().Model(&newRolePermissions).Exec(ctx); err != nil {
			pr.l.Error().Err(err).Msg("failed to bulk insert new role permissions")
			return err
		}
	}

	return nil
}

// getExistingRolePermissions gets the existing role permission assignments
func (pr *permissionRepository) getExistingRolePermissions(
	ctx context.Context,
	tx bun.IDB,
	role *permission.Role,
) ([]*permission.RolePermission, error) {
	rolePermissions := make([]*permission.RolePermission, 0)

	// Fetch the existing role permission assignments
	if err := tx.NewSelect().
		Model(&rolePermissions).
		Where("role_id = ?", role.ID).
		Where("organization_id = ?", role.OrganizationID).
		Where("business_unit_id = ?", role.BusinessUnitID).
		Scan(ctx); err != nil {
		pr.l.Error().Err(err).Msg("failed to fetch existing role permissions")
		return nil, err
	}

	return rolePermissions, nil
}

// handleRolePermissionDeletions handles deletion of permissions that are no longer assigned
func (pr *permissionRepository) handleRolePermissionDeletions(
	ctx context.Context,
	tx bun.IDB,
	existingPermissionMap map[pulid.ID]*permission.RolePermission,
	updatedPermissionIDs map[pulid.ID]struct{},
	rolePermissionsToDelete *[]*permission.RolePermission,
) error {
	// For each existing permission assignment, check if it should remain
	for permissionID, rolePermission := range existingPermissionMap {
		if _, exists := updatedPermissionIDs[permissionID]; !exists {
			*rolePermissionsToDelete = append(*rolePermissionsToDelete, rolePermission)
		}
	}

	// If there are any permission assignments to delete, delete them
	if len(*rolePermissionsToDelete) > 0 {
		_, err := tx.NewDelete().
			Model(rolePermissionsToDelete).
			WherePK().
			Exec(ctx)
		if err != nil {
			pr.l.Error().Err(err).Msg("failed to bulk delete role permissions")
			return err
		}
	}

	return nil
}
