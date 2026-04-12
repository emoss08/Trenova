//go:build integration

package permission

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
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type integrationTestDB struct {
	db *bun.DB
}

func (i *integrationTestDB) roleRepo() *testRoleRepo {
	return newTestRoleRepo(i.db)
}

func (i *integrationTestDB) userRepo() *testUserRepo {
	return newTestUserRepo(i.db)
}

type testRoleRepo struct {
	db *bun.DB
	*mocks.MockRoleRepository
}

func newTestRoleRepo(db *bun.DB) *testRoleRepo {
	mockRoleRepo := &mocks.MockRoleRepository{}
	mockRoleRepo.On(
		"HasBusinessUnitAdminAccess",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(false, nil)

	return &testRoleRepo{
		db:                 db,
		MockRoleRepository: mockRoleRepo,
	}
}

func (r *testRoleRepo) Create(ctx context.Context, role *permission.Role) error {
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

func (r *testRoleRepo) Update(ctx context.Context, role *permission.Role) error {
	_, err := r.db.NewUpdate().Model(role).WherePK().OmitZero().Exec(ctx)
	return err
}

func (r *testRoleRepo) GetByID(
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

func (r *testRoleRepo) List(
	_ context.Context,
	_ *repositories.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	return nil, nil
}

func (r *testRoleRepo) GetRolesWithInheritance(
	ctx context.Context,
	roleIDs []pulid.ID,
) ([]*permission.Role, error) {
	roles := make([]*permission.Role, 0)
	err := r.db.NewSelect().Model(&roles).
		Relation("Permissions").
		Where("r.id IN (?)", bun.List(roleIDs)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *testRoleRepo) GetUsersWithRole(
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

func (r *testRoleRepo) GetUserRoleAssignments(
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
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

func (r *testRoleRepo) CreateAssignment(
	ctx context.Context,
	assignment *permission.UserRoleAssignment,
) error {
	if assignment.AssignedAt == 0 {
		assignment.AssignedAt = timeutils.NowUnix()
	}
	_, err := r.db.NewInsert().Model(assignment).Exec(ctx)
	return err
}

func (r *testRoleRepo) DeleteAssignment(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.UserRoleAssignment)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *testRoleRepo) CreateResourcePermission(
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

func (r *testRoleRepo) UpdateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	_, err := r.db.NewUpdate().Model(rp).WherePK().Exec(ctx)
	return err
}

func (r *testRoleRepo) DeleteResourcePermission(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.ResourcePermission)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *testRoleRepo) GetResourcePermissionsByRoleID(
	ctx context.Context,
	roleID pulid.ID,
) ([]*permission.ResourcePermission, error) {
	perms := make([]*permission.ResourcePermission, 0)
	err := r.db.NewSelect().Model(&perms).Where("role_id = ?", roleID).Scan(ctx)
	return perms, err
}

type testUserRepo struct {
	db *bun.DB
	*mocks.MockUserRepository
}

func newTestUserRepo(db *bun.DB) *testUserRepo {
	return &testUserRepo{
		db:                 db,
		MockUserRepository: &mocks.MockUserRepository{},
	}
}

func (u *testUserRepo) GetByID(
	_ context.Context,
	_ repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return nil, nil
}

func (u *testUserRepo) SelectOptions(
	_ context.Context,
	_ *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return nil, nil
}

func (u *testUserRepo) FindByEmail(_ context.Context, _ string) (*tenant.User, error) {
	return nil, nil
}

func (u *testUserRepo) UpdateLastLoginAt(_ context.Context, _ pulid.ID) error {
	return nil
}

func (u *testUserRepo) GetOrganizations(
	_ context.Context,
	_ pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	return nil, nil
}

func (u *testUserRepo) List(
	_ context.Context,
	_ *repositories.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return nil, nil
}

func (u *testUserRepo) UpdateCurrentOrganization(_ context.Context, _, _, _ pulid.ID) error {
	return nil
}

func (u *testUserRepo) IsPlatformAdmin(ctx context.Context, userID pulid.ID) (bool, error) {
	var isPlatformAdmin bool
	err := u.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.is_platform_admin").
		Where("u.id = ?", userID).
		Scan(ctx, &isPlatformAdmin)
	if err != nil {
		return false, err
	}
	return isPlatformAdmin, nil
}

func (u *testUserRepo) GetUserOrganizationSummaries(
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

func (u *testUserRepo) Update(_ context.Context, _ *tenant.User) (*tenant.User, error) {
	return nil, nil
}

func (u *testUserRepo) BulkUpdateStatus(
	_ context.Context,
	_ *repositories.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	return nil, nil
}

func (u *testUserRepo) GetByIDs(
	_ context.Context,
	_ repositories.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	return nil, nil
}

type testPermCacheRepo struct {
	client *redis.Client
}

func (p *testPermCacheRepo) Get(
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

func (p *testPermCacheRepo) Set(
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

func (p *testPermCacheRepo) Delete(ctx context.Context, userID, orgID pulid.ID) error {
	key := "perms:" + userID.String() + ":" + orgID.String()
	return p.client.Del(ctx, key).Err()
}

func (p *testPermCacheRepo) InvalidateByRole(
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

func (p *testPermCacheRepo) InvalidateOrganization(ctx context.Context, orgID pulid.ID) error {
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

func createIntegrationSchema(t *testing.T, db *bun.DB) {
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
			core_responsibility VARCHAR(50),
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

type integrationTestData struct {
	orgID    pulid.ID
	userID   pulid.ID
	roleID   pulid.ID
	roleRepo *testRoleRepo
	userRepo *testUserRepo
}

func seedIntegrationData(t *testing.T, db *bun.DB) *integrationTestData {
	t.Helper()
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")
	now := timeutils.NowUnix()

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
		userID.String(),
		"Test User",
		orgID.String(),
		false,
	)

	roleRepo := newTestRoleRepo(db)
	role := &permission.Role{
		ID:             roleID,
		OrganizationID: orgID,
		Name:           "Test Role",
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				RoleID:     roleID,
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}
	require.NoError(t, roleRepo.Create(ctx, role))

	assignment := &permission.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         userID,
		OrganizationID: orgID,
		RoleID:         roleID,
		AssignedAt:     now,
	}
	require.NoError(t, roleRepo.CreateAssignment(ctx, assignment))

	return &integrationTestData{
		orgID:    orgID,
		userID:   userID,
		roleID:   roleID,
		roleRepo: roleRepo,
		userRepo: newTestUserRepo(db),
	}
}

func setupIntegrationEngine(t *testing.T) (*engine, *bun.DB, *redis.Client) {
	t.Helper()

	tc, db := sharedtestutil.SetupTestDB(t)
	_ = tc
	createIntegrationSchema(t, db)

	redisClient := sharedtestutil.SetupTestRedis(t)

	roleRepo := newTestRoleRepo(db)
	userRepo := newTestUserRepo(db)
	cacheRepo := &testPermCacheRepo{client: redisClient}

	eng := &engine{
		roleRepo:      roleRepo,
		cacheRepo:     cacheRepo,
		userRepo:      userRepo,
		registry:      permission.NewRegistry(),
		routeRegistry: permission.NewRouteRegistry(),
		l:             zap.NewNop().Named("test.permission-engine"),
	}

	return eng, db, redisClient
}

func TestIntegration_PlatformAdminBypass(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)
	ctx := t.Context()

	orgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
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
		userID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "platform_admin", result.Reason)
	assert.Equal(t, permission.DataScopeAll, result.DataScope)
}

func TestIntegration_RegularUserPermissionCheck(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, db)
	ctx := t.Context()

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "allowed", result.Reason)
	assert.Equal(t, permission.DataScopeOrganization, result.DataScope)
	assert.False(t, result.CacheHit)
}

func TestIntegration_CacheHitPath(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, db)
	ctx := t.Context()

	_, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.True(t, result.CacheHit)
}

func TestIntegration_NoPermissionDenied(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, db)
	ctx := t.Context()

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "billing",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "no_permission", result.Reason)
}

func TestIntegration_BatchPermissionCheck(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	orgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
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
		userID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	ctx := t.Context()

	result, err := eng.CheckBatch(ctx, &services.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []services.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "customer", Operation: permission.OpCreate},
		},
	})

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.True(t, result.Results[0].Allowed)
	assert.True(t, result.Results[1].Allowed)
}

func TestIntegration_InvalidateUser(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, db)
	ctx := t.Context()

	_, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)

	err = eng.InvalidateUser(ctx, data.userID, data.orgID)
	require.NoError(t, err)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.False(t, result.CacheHit)
}

func TestIntegration_LightManifest(t *testing.T) {
	eng, db, _ := setupIntegrationEngine(t)

	orgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
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
		userID.String(),
		"Admin",
		orgID.String(),
		true,
	)

	ctx := t.Context()

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.True(t, manifest.IsPlatformAdmin)
	assert.True(t, manifest.IsOrgAdmin)
	assert.NotEmpty(t, manifest.Checksum)
	assert.NotEmpty(t, manifest.RouteAccess)
}
