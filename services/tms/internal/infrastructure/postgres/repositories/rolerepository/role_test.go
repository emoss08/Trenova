//go:build integration

package rolerepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type testDBConnection struct {
	db *bun.DB
}

func (t *testDBConnection) DB() *bun.DB {
	return t.db
}

func (t *testDBConnection) HealthCheck(_ context.Context) error {
	return nil
}

func (t *testDBConnection) IsHealthy(_ context.Context) bool {
	return true
}

func (t *testDBConnection) Close() error {
	return nil
}

func setupTestRepository(t *testing.T, db *bun.DB) *testRepository {
	t.Helper()
	logger := zap.NewNop()
	return newTestRepository(db, logger)
}

func newTestRepository(db *bun.DB, logger *zap.Logger) *testRepository {
	return &testRepository{
		db: db,
		l:  logger.Named("test.role-repository"),
	}
}

type testRepository struct {
	db *bun.DB
	l  *zap.Logger
}

func (r *testRepository) Create(ctx context.Context, role *permission.Role) error {
	_, err := r.db.NewInsert().Model(role).Returning("*").Exec(ctx)
	if err != nil {
		return err
	}

	for _, rp := range role.Permissions {
		rp.RoleID = role.ID
		if _, err := r.db.NewInsert().Model(rp).Returning("*").Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *testRepository) Update(ctx context.Context, role *permission.Role) error {
	_, err := r.db.
		NewUpdate().
		Model(role).
		WherePK().
		OmitZero().
		Returning("*").
		Exec(ctx)
	return err
}

func (r *testRepository) GetByID(
	ctx context.Context,
	req repositories.GetRoleByIDRequest,
) (*permission.Role, error) {
	role := new(permission.Role)
	err := r.db.
		NewSelect().
		Model(role).
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

func (r *testRepository) List(
	ctx context.Context,
	req *repositories.ListRolesRequest,
) (*pagination.ListResult[*permission.Role], error) {
	roles := make([]*permission.Role, 0)
	q := r.db.
		NewSelect().
		Model(&roles).
		Relation("Permissions").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("r.organization_id = ?", req.Filter.TenantInfo.OrgID).
				WhereOr("r.organization_id IS NULL")
		}).
		Order("r.name ASC")

	if req.Filter != nil && req.Filter.Pagination.Limit > 0 {
		q = q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*permission.Role]{
		Items: roles,
		Total: total,
	}, nil
}

func (r *testRepository) GetRolesWithInheritance(
	ctx context.Context,
	roleIDs []pulid.ID,
) ([]*permission.Role, error) {
	if len(roleIDs) == 0 {
		return []*permission.Role{}, nil
	}

	roles := make([]*permission.Role, 0, len(roleIDs))
	err := r.db.
		NewSelect().
		Model(&roles).
		Relation("Permissions").
		WithRecursive("role_closure", r.db.NewRaw(roleClosureQuery, bun.List(roleIDs))).
		Where("r.id IN (SELECT c.id FROM role_closure c)").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (r *testRepository) GetUsersWithRole(
	ctx context.Context,
	roleID pulid.ID,
) ([]repositories.ImpactedUser, error) {
	var results []repositories.ImpactedUser
	err := r.db.NewSelect().
		TableExpr("user_role_assignments AS ura").
		ColumnExpr("ura.user_id").
		ColumnExpr("u.name AS user_name").
		ColumnExpr("ura.organization_id").
		ColumnExpr("o.name AS org_name").
		ColumnExpr("'direct' AS assignment_type").
		Join("JOIN users AS u ON u.id = ura.user_id").
		Join("JOIN organizations AS o ON o.id = ura.organization_id").
		Where("ura.role_id = ?", roleID).
		Scan(ctx, &results)
	return results, err
}

func (r *testRepository) GetUserRoleAssignments(
	ctx context.Context,
	userID, orgID pulid.ID,
) ([]*permission.UserRoleAssignment, error) {
	assignments := make([]*permission.UserRoleAssignment, 0)
	err := r.db.
		NewSelect().
		Model(&assignments).
		Where("ura.user_id = ?", userID).
		Where("ura.organization_id = ?", orgID).
		Scan(ctx)
	return assignments, err
}

func (r *testRepository) CreateAssignment(
	ctx context.Context,
	assignment *permission.UserRoleAssignment,
) error {
	_, err := r.db.NewInsert().Model(assignment).Returning("*").Exec(ctx)
	return err
}

func (r *testRepository) DeleteAssignment(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.UserRoleAssignment)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *testRepository) CreateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	_, err := r.db.NewInsert().Model(rp).Returning("*").Exec(ctx)
	return err
}

func (r *testRepository) UpdateResourcePermission(
	ctx context.Context,
	rp *permission.ResourcePermission,
) error {
	_, err := r.db.
		NewUpdate().
		Model(rp).
		WherePK().
		OmitZero().
		Returning("*").
		Exec(ctx)
	return err
}

func (r *testRepository) DeleteResourcePermission(ctx context.Context, id pulid.ID) error {
	_, err := r.db.NewDelete().
		Model((*permission.ResourcePermission)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *testRepository) GetResourcePermissionsByRoleID(
	ctx context.Context,
	roleID pulid.ID,
) ([]*permission.ResourcePermission, error) {
	permissions := make([]*permission.ResourcePermission, 0)
	err := r.db.
		NewSelect().
		Model(&permissions).
		Where("rp.role_id = ?", roleID).
		Scan(ctx)
	return permissions, err
}

func createTestSchema(t *testing.T, db *bun.DB, ctx testutil.TestContext) {
	t.Helper()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			organization_id VARCHAR(100) REFERENCES organizations(id)
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
		_, err := db.ExecContext(ctx.Ctx, q)
		require.NoError(t, err)
	}
}

func createTestOrg(t *testing.T, db *bun.DB, ctx testutil.TestContext) pulid.ID {
	t.Helper()
	orgID := pulid.MustNew("org_")
	_, err := db.ExecContext(ctx.Ctx,
		`INSERT INTO organizations (id, name) VALUES (?, ?)`,
		orgID.String(), "Test Org",
	)
	require.NoError(t, err)
	return orgID
}

func createTestUser(t *testing.T, db *bun.DB, ctx testutil.TestContext, orgID pulid.ID) pulid.ID {
	t.Helper()
	userID := pulid.MustNew("usr_")
	_, err := db.ExecContext(ctx.Ctx,
		`INSERT INTO users (id, name, organization_id) VALUES (?, ?, ?)`,
		userID.String(), "Test User", orgID.String(),
	)
	require.NoError(t, err)
	return userID
}

func TestRoleRepository_Create_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Test Role",
		Description:    "A test role",
		MaxSensitivity: permission.SensitivityInternal,
		IsSystem:       false,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	err := repo.Create(tc.Ctx, role)
	require.NoError(t, err)

	var count int
	err = db.NewSelect().
		TableExpr("roles").
		ColumnExpr("COUNT(*)").
		Where("id = ?", role.ID).
		Scan(tc.Ctx, &count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var permCount int
	err = db.NewSelect().
		TableExpr("resource_permissions").
		ColumnExpr("COUNT(*)").
		Where("role_id = ?", role.ID).
		Scan(tc.Ctx, &permCount)
	require.NoError(t, err)
	assert.Equal(t, 1, permCount)
}

func TestRoleRepository_GetByID_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "GetByID Role",
		Description:    "Test role for GetByID",
		MaxSensitivity: permission.SensitivityRestricted,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				Resource:   "driver",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	err := repo.Create(tc.Ctx, role)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(tc.Ctx, repositories.GetRoleByIDRequest{
		ID:         role.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})
	require.NoError(t, err)
	assert.Equal(t, role.ID, retrieved.ID)
	assert.Equal(t, "GetByID Role", retrieved.Name)
	assert.Len(t, retrieved.Permissions, 1)
	assert.Equal(t, "driver", retrieved.Permissions[0].Resource)
}

func TestRoleRepository_List_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	roles := []*permission.Role{
		{
			ID:             pulid.MustNew("rol_"),
			OrganizationID: orgID,
			Name:           "Alpha Role",
			MaxSensitivity: permission.SensitivityInternal,
			IsSystem:       false,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             pulid.MustNew("rol_"),
			OrganizationID: orgID,
			Name:           "Beta Role",
			MaxSensitivity: permission.SensitivityInternal,
			IsSystem:       false,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             pulid.MustNew("rol_"),
			OrganizationID: orgID,
			Name:           "System Role",
			MaxSensitivity: permission.SensitivityInternal,
			IsSystem:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	for _, role := range roles {
		err := repo.Create(tc.Ctx, role)
		require.NoError(t, err)
	}

	t.Run("list roles", func(t *testing.T) {
		result, err := repo.List(tc.Ctx, &repositories.ListRolesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: orgID},
				Pagination: pagination.Info{Limit: 100},
			},
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 3)
		assert.Equal(t, "Alpha Role", result.Items[0].Name)
		assert.Equal(t, "Beta Role", result.Items[1].Name)
	})
}

func TestRoleRepository_Update_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Original Name",
		Description:    "Original description",
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.Create(tc.Ctx, role)
	require.NoError(t, err)

	role.Name = "Updated Name"
	role.Description = "Updated description"
	role.UpdatedAt = timeutils.NowUnix()

	err = repo.Update(tc.Ctx, role)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(tc.Ctx, repositories.GetRoleByIDRequest{
		ID:         role.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Updated description", retrieved.Description)
}

func TestRoleRepository_RoleAssignments_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)
	userID := createTestUser(t, db, *tc, orgID)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Assignable Role",
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.Create(tc.Ctx, role)
	require.NoError(t, err)

	assignment := &permission.UserRoleAssignment{
		ID:             pulid.MustNew("ura_"),
		UserID:         userID,
		OrganizationID: orgID,
		RoleID:         role.ID,
		AssignedAt:     now,
	}

	t.Run("create assignment", func(t *testing.T) {
		err := repo.CreateAssignment(tc.Ctx, assignment)
		require.NoError(t, err)
	})

	t.Run("get user role assignments", func(t *testing.T) {
		assignments, err := repo.GetUserRoleAssignments(tc.Ctx, userID, orgID)
		require.NoError(t, err)
		assert.Len(t, assignments, 1)
		assert.Equal(t, role.ID, assignments[0].RoleID)
	})

	t.Run("get users with role", func(t *testing.T) {
		users, err := repo.GetUsersWithRole(tc.Ctx, role.ID)
		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, userID, users[0].UserID)
	})

	t.Run("delete assignment", func(t *testing.T) {
		err := repo.DeleteAssignment(tc.Ctx, assignment.ID)
		require.NoError(t, err)

		assignments, err := repo.GetUserRoleAssignments(tc.Ctx, userID, orgID)
		require.NoError(t, err)
		assert.Len(t, assignments, 0)
	})
}

func TestRoleRepository_GetRolesWithInheritance_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()

	parentRole := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Parent Role",
		MaxSensitivity: permission.SensitivityRestricted,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	err := repo.Create(tc.Ctx, parentRole)
	require.NoError(t, err)

	childRole := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Child Role",
		ParentRoleIDs:  []pulid.ID{parentRole.ID},
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				Resource:   "driver",
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	err = repo.Create(tc.Ctx, childRole)
	require.NoError(t, err)

	roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{childRole.ID})
	require.NoError(t, err)
	assert.Len(t, roles, 2)

	roleNames := make(map[string]bool)
	for _, r := range roles {
		roleNames[r.Name] = true
	}
	assert.True(t, roleNames["Parent Role"])
	assert.True(t, roleNames["Child Role"])
}

func roleNameSet(roles []*permission.Role) map[string]bool {
	names := make(map[string]bool, len(roles))
	for _, role := range roles {
		names[role.Name] = true
	}
	return names
}

func newInheritanceTestRole(
	orgID pulid.ID,
	name string,
	parents []pulid.ID,
	resource string,
) *permission.Role {
	now := timeutils.NowUnix()
	return &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           name,
		ParentRoleIDs:  parents,
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
		Permissions: []*permission.ResourcePermission{
			{
				ID:         pulid.MustNew("rp_"),
				Resource:   resource,
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}
}

func TestRoleRepository_GetRolesWithInheritance_Depth_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	grandparent := newInheritanceTestRole(orgID, "Grandparent", nil, "shipment")
	require.NoError(t, repo.Create(tc.Ctx, grandparent))

	parent := newInheritanceTestRole(
		orgID, "Parent", []pulid.ID{grandparent.ID}, "driver",
	)
	require.NoError(t, repo.Create(tc.Ctx, parent))

	child := newInheritanceTestRole(orgID, "Child", []pulid.ID{parent.ID}, "customer")
	require.NoError(t, repo.Create(tc.Ctx, child))

	unrelated := newInheritanceTestRole(orgID, "Unrelated", nil, "invoice")
	require.NoError(t, repo.Create(tc.Ctx, unrelated))

	t.Run("three deep chain resolves in full", func(t *testing.T) {
		roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{child.ID})
		require.NoError(t, err)

		names := roleNameSet(roles)
		assert.Len(t, roles, 3)
		assert.True(t, names["Child"])
		assert.True(t, names["Parent"])
		assert.True(t, names["Grandparent"])
		assert.False(t, names["Unrelated"])
	})

	t.Run("permissions are loaded for inherited roles", func(t *testing.T) {
		roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{child.ID})
		require.NoError(t, err)

		resources := make(map[string]bool, len(roles))
		for _, role := range roles {
			require.Len(t, role.Permissions, 1, "role %s lost its permissions", role.Name)
			resources[role.Permissions[0].Resource] = true
		}
		assert.True(t, resources["shipment"])
		assert.True(t, resources["driver"])
		assert.True(t, resources["customer"])
	})

	t.Run("a role reached by two paths is returned once", func(t *testing.T) {
		diamond := newInheritanceTestRole(
			orgID, "Diamond", []pulid.ID{parent.ID, grandparent.ID}, "worker",
		)
		require.NoError(t, repo.Create(tc.Ctx, diamond))

		roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{diamond.ID})
		require.NoError(t, err)
		assert.Len(t, roles, 3)
	})

	t.Run("multiple roots share one query", func(t *testing.T) {
		roles, err := repo.GetRolesWithInheritance(
			tc.Ctx, []pulid.ID{child.ID, unrelated.ID},
		)
		require.NoError(t, err)

		names := roleNameSet(roles)
		assert.True(t, names["Child"])
		assert.True(t, names["Grandparent"])
		assert.True(t, names["Unrelated"])
	})
}

func TestRoleRepository_GetRolesWithInheritance_Cycle_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	first := newInheritanceTestRole(orgID, "Cycle First", nil, "shipment")
	require.NoError(t, repo.Create(tc.Ctx, first))

	second := newInheritanceTestRole(orgID, "Cycle Second", []pulid.ID{first.ID}, "driver")
	require.NoError(t, repo.Create(tc.Ctx, second))

	_, err := db.ExecContext(tc.Ctx,
		`UPDATE roles SET parent_role_ids = ARRAY[?]::text[] WHERE id = ?`,
		second.ID.String(), first.ID.String(),
	)
	require.NoError(t, err)

	roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{second.ID})
	require.NoError(t, err)

	names := roleNameSet(roles)
	assert.Len(t, roles, 2)
	assert.True(t, names["Cycle First"])
	assert.True(t, names["Cycle Second"])
}

func TestRoleRepository_ResourcePermissions_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)
	orgID := createTestOrg(t, db, *tc)

	repo := setupTestRepository(t, db)

	now := timeutils.NowUnix()
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Permission Test Role",
		MaxSensitivity: permission.SensitivityInternal,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.Create(tc.Ctx, role)
	require.NoError(t, err)

	rp := &permission.ResourcePermission{
		ID:         pulid.MustNew("rp_"),
		RoleID:     role.ID,
		Resource:   "customer",
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeOrganization,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	t.Run("create resource permission", func(t *testing.T) {
		err := repo.CreateResourcePermission(tc.Ctx, rp)
		require.NoError(t, err)
	})

	t.Run("get resource permissions by role id", func(t *testing.T) {
		permissions, err := repo.GetResourcePermissionsByRoleID(tc.Ctx, role.ID)
		require.NoError(t, err)
		assert.Len(t, permissions, 1)
		assert.Equal(t, "customer", permissions[0].Resource)
	})

	t.Run("update resource permission", func(t *testing.T) {
		rp.Operations = append(rp.Operations, permission.OpExport)
		rp.UpdatedAt = timeutils.NowUnix()
		err := repo.UpdateResourcePermission(tc.Ctx, rp)
		require.NoError(t, err)

		permissions, err := repo.GetResourcePermissionsByRoleID(tc.Ctx, role.ID)
		require.NoError(t, err)
		assert.Contains(t, permissions[0].Operations, permission.OpExport)
	})

	t.Run("delete resource permission", func(t *testing.T) {
		err := repo.DeleteResourcePermission(tc.Ctx, rp.ID)
		require.NoError(t, err)

		permissions, err := repo.GetResourcePermissionsByRoleID(tc.Ctx, role.ID)
		require.NoError(t, err)
		assert.Len(t, permissions, 0)
	})
}

func TestRoleRepository_EmptyRoleIDs_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)
	createTestSchema(t, db, *tc)

	repo := setupTestRepository(t, db)

	roles, err := repo.GetRolesWithInheritance(tc.Ctx, []pulid.ID{})
	require.NoError(t, err)
	assert.Len(t, roles, 0)
}
