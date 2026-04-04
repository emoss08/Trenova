//go:build integration

package roleservice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type integrationRoleRepo struct {
	db *bun.DB
	*mocks.MockRoleRepository
}

func (r *integrationRoleRepo) Create(ctx context.Context, role *permission.Role) error {
	now := timeutils.NowUnix()
	if role.CreatedAt == 0 {
		role.CreatedAt = now
	}
	if role.UpdatedAt == 0 {
		role.UpdatedAt = now
	}
	_, err := r.db.NewInsert().Model(role).Exec(ctx)
	if err != nil {
		return err
	}
	for _, rp := range role.Permissions {
		rp.RoleID = role.ID
		if rp.CreatedAt == 0 {
			rp.CreatedAt = now
		}
		if rp.UpdatedAt == 0 {
			rp.UpdatedAt = now
		}
		if _, err := r.db.NewInsert().Model(rp).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *integrationRoleRepo) Update(ctx context.Context, role *permission.Role) error {
	role.UpdatedAt = timeutils.NowUnix()
	_, err := r.db.NewUpdate().Model(role).WherePK().OmitZero().Exec(ctx)
	return err
}

func (r *integrationRoleRepo) GetByID(
	ctx context.Context,
	req repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	role := new(permission.Role)
	err := r.db.NewSelect().Model(role).
		Relation("Permissions").
		Where("r.id = ?", req.ID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("r.organization_id = ?", req.TenantInfo.OrgID).
				WhereOr("r.organization_id IS NULL")
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *integrationRoleRepo) List(
	_ context.Context,
	_ *repositories.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return nil, nil
}

func (r *integrationRoleRepo) GetRolesWithInheritance(
	ctx context.Context,
	roleIDs []pulid.ID,
) ([]*permission.Role, error) {
	roles := make([]*permission.Role, 0)
	err := r.db.NewSelect().Model(&roles).
		Relation("Permissions").
		Where("r.id IN (?)", bun.List(roleIDs)).
		Scan(ctx)
	return roles, err
}

func (r *integrationRoleRepo) GetUsersWithRole(
	ctx context.Context,
	roleID pulid.ID,
) ([]repositories.ImpactedUser, error) {
	var users []repositories.ImpactedUser
	err := r.db.NewSelect().
		TableExpr("user_role_assignments AS ura").
		Join("JOIN users AS u ON u.id = ura.user_id").
		ColumnExpr("ura.user_id").
		ColumnExpr("ura.organization_id").
		ColumnExpr("u.name AS user_name").
		Where("ura.role_id = ?", roleID).
		Scan(ctx, &users)
	return users, err
}

func (r *integrationRoleRepo) GetUserRoleAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*permission.UserRoleAssignment, error) {
	assignments := make([]*permission.UserRoleAssignment, 0)
	err := r.db.NewSelect().Model(&assignments).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("user_id = ?", userID).
				Where("organization_id = ?", orgID)
		}).
		Scan(ctx)
	return assignments, err
}

func (r *integrationRoleRepo) CreateAssignment(
	ctx context.Context,
	assignment *permission.UserRoleAssignment,
) error {
	if assignment.AssignedAt == 0 {
		assignment.AssignedAt = timeutils.NowUnix()
	}
	_, err := r.db.NewInsert().Model(assignment).Exec(ctx)
	return err
}

func (r *integrationRoleRepo) DeleteAssignment(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.UserRoleAssignment)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *integrationRoleRepo) CreateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	now := timeutils.NowUnix()
	if rp.CreatedAt == 0 {
		rp.CreatedAt = now
	}
	if rp.UpdatedAt == 0 {
		rp.UpdatedAt = now
	}
	_, err := r.db.NewInsert().Model(rp).Exec(ctx)
	return err
}

func (r *integrationRoleRepo) UpdateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	rp.UpdatedAt = timeutils.NowUnix()
	_, err := r.db.NewUpdate().Model(rp).WherePK().Exec(ctx)
	return err
}

func (r *integrationRoleRepo) DeleteResourcePermission(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.ResourcePermission)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *integrationRoleRepo) GetResourcePermissionsByRoleID(
	ctx context.Context,
	roleID pulid.ID,
) ([]*permission.ResourcePermission, error) {
	perms := make([]*permission.ResourcePermission, 0)
	err := r.db.NewSelect().Model(&perms).Where("role_id = ?", roleID).Scan(ctx)
	return perms, err
}

type integrationUserRepo struct {
	db *bun.DB
	*mocks.MockUserRepository
}

func (u *integrationUserRepo) GetByID(
	_ context.Context,
	_ repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return nil, nil
}

func (u *integrationUserRepo) SelectOptions(
	_ context.Context,
	_ *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return nil, nil
}

func (u *integrationUserRepo) FindByEmail(_ context.Context, _ string) (*tenant.User, error) {
	return nil, nil
}

func (u *integrationUserRepo) UpdateLastLoginAt(_ context.Context, _ pulid.ID) error {
	return nil
}

func (u *integrationUserRepo) GetOrganizations(
	_ context.Context,
	_ pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	return nil, nil
}

func (u *integrationUserRepo) List(
	_ context.Context,
	_ *repositories.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return nil, nil
}

func (u *integrationUserRepo) UpdateCurrentOrganization(_ context.Context, _, _, _ pulid.ID) error {
	return nil
}

func (u *integrationUserRepo) IsPlatformAdmin(ctx context.Context, userID pulid.ID) (bool, error) {
	var isPlatformAdmin bool
	err := u.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.is_platform_admin").
		Where("u.id = ?", userID).
		Scan(ctx, &isPlatformAdmin)
	return isPlatformAdmin, err
}

func (u *integrationUserRepo) GetUserOrganizationSummaries(
	ctx context.Context,
	userID pulid.ID,
) ([]repositories.OrgSummary, error) {
	var summaries []repositories.OrgSummary
	err := u.db.NewSelect().
		TableExpr("organizations AS o").
		Join("JOIN users AS u ON u.organization_id = o.id").
		ColumnExpr("o.id").
		ColumnExpr("o.name").
		Where("u.id = ?", userID).
		Scan(ctx, &summaries)
	return summaries, err
}

func (u *integrationUserRepo) Update(_ context.Context, _ *tenant.User) (*tenant.User, error) {
	return nil, nil
}

func (u *integrationUserRepo) BulkUpdateStatus(
	_ context.Context,
	_ *repositories.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	return nil, nil
}

func (u *integrationUserRepo) GetByIDs(
	_ context.Context,
	_ repositories.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	return nil, nil
}

type integrationPermCacheRepo struct {
	client *redis.Client
}

func (p *integrationPermCacheRepo) Get(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, error) {
	key := "perms:" + userID.String() + ":" + orgID.String()
	data, err := p.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var perms repositories.CachedPermissions
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, err
	}
	return &perms, nil
}

func (p *integrationPermCacheRepo) Set(
	ctx context.Context,
	userID, orgID pulid.ID,
	perms *repositories.CachedPermissions,
	ttl time.Duration,
) error {
	key := "perms:" + userID.String() + ":" + orgID.String()
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return p.client.Set(ctx, key, data, ttl).Err()
}

func (p *integrationPermCacheRepo) Delete(ctx context.Context, userID, orgID pulid.ID) error {
	key := "perms:" + userID.String() + ":" + orgID.String()
	return p.client.Del(ctx, key).Err()
}

func (p *integrationPermCacheRepo) InvalidateByRole(
	ctx context.Context,
	roleID pulid.ID,
	roleRepo repositories.RoleRepository,
) error {
	users, err := roleRepo.GetUsersWithRole(ctx, roleID)
	if err != nil {
		return err
	}
	for _, u := range users {
		if err := p.Delete(ctx, u.UserID, u.OrganizationID); err != nil {
			return err
		}
	}
	return nil
}

func (p *integrationPermCacheRepo) InvalidateOrganization(
	ctx context.Context,
	orgID pulid.ID,
) error {
	pattern := "perms:*:" + orgID.String()
	keys, err := p.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return p.client.Del(ctx, keys...).Err()
	}
	return nil
}

func createRoleServiceSchema(t *testing.T, db *bun.DB) {
	t.Helper()
	ctx := t.Context()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			organization_id VARCHAR(100) REFERENCES organizations(id),
			is_platform_admin BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id VARCHAR(100) PRIMARY KEY,
			business_unit_id VARCHAR(100),
			organization_id VARCHAR(100) REFERENCES organizations(id),
			name VARCHAR(255) NOT NULL,
			description TEXT,
			parent_role_ids TEXT[],
			max_sensitivity VARCHAR(50) NOT NULL DEFAULT 'internal',
			is_system BOOLEAN DEFAULT FALSE,
			is_org_admin BOOLEAN DEFAULT FALSE,
			is_business_unit_admin BOOLEAN DEFAULT FALSE,
			created_by VARCHAR(100),
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS resource_permissions (
			id VARCHAR(100) PRIMARY KEY,
			role_id VARCHAR(100) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			resource VARCHAR(100) NOT NULL,
			operations TEXT[] NOT NULL,
			data_scope VARCHAR(50) NOT NULL DEFAULT 'organization',
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			UNIQUE(role_id, resource)
		)`,
		`CREATE TABLE IF NOT EXISTS user_role_assignments (
			id VARCHAR(100) PRIMARY KEY,
			user_id VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			organization_id VARCHAR(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			role_id VARCHAR(100) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			expires_at BIGINT,
			assigned_by VARCHAR(100),
			assigned_at BIGINT NOT NULL,
			UNIQUE(user_id, organization_id, role_id)
		)`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err)
	}
}

type integrationPermEngine struct {
	roleRepo  repositories.RoleRepository
	cacheRepo repositories.PermissionCacheRepository
	userRepo  repositories.UserRepository
	registry  *permission.Registry
}

func (e *integrationPermEngine) Check(
	ctx context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	isPlatformAdmin, err := e.userRepo.IsPlatformAdmin(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if isPlatformAdmin {
		return &services.PermissionCheckResult{
			Allowed:   true,
			Reason:    "platform_admin",
			DataScope: permission.DataScopeAll,
		}, nil
	}

	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}

	if len(roleIDs) == 0 {
		return &services.PermissionCheckResult{Allowed: false, Reason: "no_permission"}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.IsOrgAdmin {
			return &services.PermissionCheckResult{
				Allowed:   true,
				Reason:    "org_admin",
				DataScope: permission.DataScopeOrganization,
			}, nil
		}
		for _, perm := range role.Permissions {
			if string(perm.Resource) == string(req.Resource) {
				for _, op := range perm.Operations {
					if op == req.Operation {
						return &services.PermissionCheckResult{
							Allowed:   true,
							Reason:    "allowed",
							DataScope: perm.DataScope,
						}, nil
					}
				}
			}
		}
	}

	return &services.PermissionCheckResult{Allowed: false, Reason: "no_permission"}, nil
}

func (e *integrationPermEngine) CheckBatch(
	_ context.Context,
	_ *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	return nil, nil
}

func (e *integrationPermEngine) GetLightManifest(
	_ context.Context,
	_, _ pulid.ID,
) (*services.LightPermissionManifest, error) {
	return nil, nil
}

func (e *integrationPermEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	_ string,
) (*services.ResourcePermissionDetail, error) {
	return nil, nil
}

func (e *integrationPermEngine) InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error {
	return e.cacheRepo.Delete(ctx, userID, orgID)
}

func (e *integrationPermEngine) GetEffectivePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.EffectivePermissions, error) {
	isPlatformAdmin, err := e.userRepo.IsPlatformAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isPlatformAdmin {
		return &services.EffectivePermissions{
			UserID:         userID,
			OrganizationID: orgID,
			MaxSensitivity: permission.SensitivityConfidential,
			Resources:      make(map[string]services.EffectiveResourcePermission),
		}, nil
	}

	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}

	if len(roleIDs) == 0 {
		return &services.EffectivePermissions{
			UserID:         userID,
			OrganizationID: orgID,
			Resources:      make(map[string]services.EffectiveResourcePermission),
		}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	resources := make(map[string]services.EffectiveResourcePermission)
	var maxSensitivity permission.FieldSensitivity
	for _, role := range roles {
		if role.MaxSensitivity.Level() > maxSensitivity.Level() {
			maxSensitivity = role.MaxSensitivity
		}
		for _, perm := range role.Permissions {
			rp := resources[string(perm.Resource)]
			rp.Operations = append(rp.Operations, perm.Operations...)
			rp.DataScope = perm.DataScope
			resources[string(perm.Resource)] = rp
		}
	}

	return &services.EffectivePermissions{
		UserID:         userID,
		OrganizationID: orgID,
		MaxSensitivity: maxSensitivity,
		Resources:      resources,
	}, nil
}

func (e *integrationPermEngine) SimulatePermissions(
	_ context.Context,
	_ *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	return nil, nil
}

func setupIntegrationService(t *testing.T) (*Service, *bun.DB) {
	t.Helper()

	tc, db := sharedtestutil.SetupTestDB(t)
	_ = tc
	createRoleServiceSchema(t, db)

	redisClient := sharedtestutil.SetupTestRedis(t)

	roleRepo := &integrationRoleRepo{
		db:                 db,
		MockRoleRepository: &mocks.MockRoleRepository{},
	}
	userRepo := &integrationUserRepo{
		db:                 db,
		MockUserRepository: &mocks.MockUserRepository{},
	}
	cacheRepo := &integrationPermCacheRepo{client: redisClient}

	permEngine := &integrationPermEngine{
		roleRepo:  roleRepo,
		cacheRepo: cacheRepo,
		userRepo:  userRepo,
		registry:  permission.NewRegistry(),
	}

	svc := &Service{
		l:          zap.NewNop().Named("test.role-service"),
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		permCache:  cacheRepo,
		permEngine: permEngine,
		validator: &Validator{
			validator: validationframework.
				NewTenantedValidatorBuilder[*permission.Role]().
				WithModelName("Role").
				Build(),
		},
		registry: permission.NewRegistry(),
	}

	return svc, db
}

func TestIntegration_CreateRole_PlatformAdmin(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	role := &permission.Role{
		Name:           "New Role",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	err := svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, role.OrganizationID)
	assert.Equal(t, actorID, role.CreatedBy)
	assert.NotEmpty(t, role.ID)
}

func TestIntegration_CreateRole_CircularInheritance(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	roleID := pulid.MustNew("rol_")
	role := &permission.Role{
		ID:            roleID,
		Name:          "Self Referencing",
		ParentRoleIDs: []pulid.ID{roleID},
	}

	err := svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCircularInheritance, err)
}

func TestIntegration_UpdateRole_Success(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	now := timeutils.NowUnix()
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO roles (id, organization_id, name, max_sensitivity, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		roleID.String(),
		orgID.String(),
		"Original",
		"internal",
		now,
		now,
	)

	role := &permission.Role{
		ID:             roleID,
		Name:           "Updated Role",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	err := svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)
}

func TestIntegration_UpdateRole_SystemRole_Fails(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	now := timeutils.NowUnix()
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO roles (id, organization_id, name, max_sensitivity, is_system, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		roleID.String(),
		orgID.String(),
		"System Role",
		"internal",
		true,
		now,
		now,
	)

	role := &permission.Role{
		ID:   roleID,
		Name: "Updated",
	}

	err := svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)
}

func TestIntegration_AssignRole_Success(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")
	targetUserID := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		targetUserID.String(),
		"User",
		orgID.String(),
		false,
	)

	now := timeutils.NowUnix()
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO roles (id, organization_id, name, max_sensitivity, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		roleID.String(),
		orgID.String(),
		"Test Role",
		"internal",
		now,
		now,
	)

	assignment := &permission.UserRoleAssignment{
		UserID: targetUserID,
		RoleID: roleID,
	}

	err := svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, assignment.OrganizationID)
	assert.Equal(t, actorID, assignment.AssignedBy)
}

func TestIntegration_InitializeOrganizationRoles(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	creatorID := pulid.MustNew("usr_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		creatorID.String(),
		"Creator",
		orgID.String(),
		true,
	)

	err := svc.InitializeOrganizationRoles(ctx, orgID, creatorID)

	require.NoError(t, err)

	roleRepo := &integrationRoleRepo{
		db:                 db,
		MockRoleRepository: &mocks.MockRoleRepository{},
	}
	assignments, err := roleRepo.GetUserRoleAssignments(ctx, creatorID, orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, assignments)
}

func TestIntegration_CreateResourcePermission_Success(t *testing.T) {
	svc, db := setupIntegrationService(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	actorID := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")

	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(),
		"Test Org",
	)
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO users (id, name, organization_id, is_platform_admin) VALUES (?, ?, ?, ?)`,
		actorID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	now := timeutils.NowUnix()
	sharedtestutil.MustExec(
		t,
		db,
		`INSERT INTO roles (id, organization_id, name, max_sensitivity, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		roleID.String(),
		orgID.String(),
		"Test Role",
		"internal",
		now,
		now,
	)

	rp := &permission.ResourcePermission{
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	err := svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.NoError(t, err)
	assert.NotEmpty(t, rp.ID)
}
