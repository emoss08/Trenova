//go:build integration

package permission

import (
	"context"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/rolerepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testPermCacheRepo struct {
	client *redis.Client
}

func (p *testPermCacheRepo) Get(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
) (*permissionports.CachedPermissions, error) {
	redisKey := "perms:" + key.UserID.String() + ":" + key.OrgID.String()
	data, err := p.client.HGet(ctx, redisKey, key.Variant).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var perms permissionports.CachedPermissions
	if err := sonic.Unmarshal(data, &perms); err != nil {
		return nil, err
	}
	epoch, err := p.client.Get(ctx, p.epochKey(key.OrgID)).Int64()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if perms.Epoch != epoch {
		return nil, nil
	}
	return &perms, nil
}

func (p *testPermCacheRepo) Epoch(ctx context.Context, orgID pulid.ID) (int64, error) {
	key := p.epochKey(orgID)
	epoch, err := p.client.Get(ctx, key).Int64()
	if err == nil {
		return epoch, nil
	}
	if err != redis.Nil {
		return 0, err
	}
	if err = p.client.SetNX(ctx, key, time.Now().UnixNano(), 0).Err(); err != nil {
		return 0, err
	}
	return p.client.Get(ctx, key).Int64()
}

func (p *testPermCacheRepo) epochKey(orgID pulid.ID) string {
	return "perms:epoch:" + orgID.String()
}

func (p *testPermCacheRepo) Set(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
	perms *permissionports.CachedPermissions,
	ttl time.Duration,
) error {
	redisKey := "perms:" + key.UserID.String() + ":" + key.OrgID.String()
	data, err := sonic.Marshal(perms)
	if err != nil {
		return err
	}
	if err := p.client.HSet(ctx, redisKey, key.Variant, data).Err(); err != nil {
		return err
	}
	return p.client.Expire(ctx, redisKey, ttl).Err()
}

func (p *testPermCacheRepo) Delete(ctx context.Context, userID, orgID pulid.ID) error {
	key := "perms:" + userID.String() + ":" + orgID.String()
	return p.client.Del(ctx, key).Err()
}

func (p *testPermCacheRepo) InvalidateByRole(
	ctx context.Context,
	roleID pulid.ID,
	roleRepo permissionports.RoleRepository,
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
	return p.client.Incr(ctx, p.epochKey(orgID)).Err()
}

type integrationTestData struct {
	ctx    context.Context
	orgID  pulid.ID
	userID pulid.ID
	roleID pulid.ID
}

type integrationEnv struct {
	ctx  context.Context
	db   *keel.DB
	data *seedtest.TestData
}

func (env *integrationEnv) addMembership(t *testing.T, userID pulid.ID) {
	t.Helper()

	require.NoError(
		t,
		tenantdb.OrganizationMemberships.Create(env.ctx, env.db, &tenant.OrganizationMembership{
			UserID:         userID,
			BusinessUnitID: env.data.BusinessUnit.ID,
			OrganizationID: env.data.Organization.ID,
			IsDefault:      true,
		}),
	)
}

func seedIntegrationData(t *testing.T, env *integrationEnv) *integrationTestData {
	t.Helper()

	userID := env.data.User.ID
	env.addMembership(t, userID)

	role := &permission.Role{
		Name:           "Test Role",
		MaxSensitivity: permission.SensitivityInternal,
	}
	require.NoError(t, permissiondb.Roles.Create(env.ctx, env.db, role))
	require.NoError(
		t,
		permissiondb.ResourcePermissions.Create(env.ctx, env.db, &permission.ResourcePermission{
			RoleID:     role.ID,
			Resource:   "shipment",
			Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			DataScope:  permission.DataScopeOrganization,
		}),
	)
	require.NoError(
		t,
		permissiondb.UserRoleAssignments.Create(env.ctx, env.db, &permission.UserRoleAssignment{
			UserID:         userID,
			OrganizationID: env.data.Organization.ID,
			RoleID:         role.ID,
			AssignedAt:     timeutils.NowUnix(),
		}),
	)

	return &integrationTestData{
		ctx:    env.ctx,
		orgID:  env.data.Organization.ID,
		userID: userID,
		roleID: role.ID,
	}
}

func setupIntegrationEngine(t *testing.T) (*engine, *integrationEnv, *redis.Client) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	redisClient := sharedtestutil.SetupTestRedis(t)

	conn := postgres.NewTestConnection(db)
	cacheRepo := &testPermCacheRepo{client: redisClient}

	metricsRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	eng := &engine{
		roleRepo:      rolerepository.New(rolerepository.Params{DB: conn, Logger: zap.NewNop()}),
		cacheRepo:     cacheRepo,
		userRepo:      userrepository.New(userrepository.Params{DB: conn, Logger: zap.NewNop()}),
		registry:      permissioncatalog.NewRegistry(),
		routeRegistry: permissioncatalog.NewRouteRegistry(),
		metrics:       metricsRegistry,
		l:             zap.NewNop().Named("test.permission-engine"),
	}

	return eng, &integrationEnv{ctx: data.Ctx(ctx), db: db, data: data}, redisClient
}

func TestIntegration_RegularUserPermissionCheck(t *testing.T) {
	eng, env, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, env)
	ctx := data.ctx

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
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
	eng, env, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, env)
	ctx := data.ctx

	_, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
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
	eng, env, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, env)
	ctx := data.ctx

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
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
	eng, env, _ := setupIntegrationEngine(t)
	data := seedIntegrationData(t, env)

	ctx := data.ctx

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "shipment", Operation: permission.OpCreate},
			{Resource: "shipment", Operation: permission.OpDelete},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 3)
	assert.True(t, result.Results[0].Allowed, "granted read should be allowed")
	assert.True(t, result.Results[1].Allowed, "granted create should be allowed")
	assert.False(t, result.Results[2].Allowed, "ungranted delete should be denied")
}

func TestIntegration_InvalidateUser(t *testing.T) {
	eng, env, _ := setupIntegrationEngine(t)

	data := seedIntegrationData(t, env)
	ctx := data.ctx

	_, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         data.userID,
		OrganizationID: data.orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})
	require.NoError(t, err)

	err = eng.InvalidateUser(ctx, data.userID, data.orgID)
	require.NoError(t, err)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
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
	eng, env, _ := setupIntegrationEngine(t)

	orgID := env.data.Organization.ID
	userID := env.data.User.ID
	env.addMembership(t, userID)
	ctx := env.ctx

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.NotEmpty(t, manifest.Checksum)
	assert.NotEmpty(t, manifest.RouteAccess)
}
