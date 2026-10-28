//go:build integration

package roleservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/rolerepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/permchangetest"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/rbactest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type integrationPermCacheRepo struct {
	client *redis.Client
}

func (p *integrationPermCacheRepo) Get(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
) (*permissionports.CachedPermissions, error) {
	redisKey := fmt.Sprintf("perms:%s:%s", key.UserID.String(), key.OrgID.String())
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
	return &perms, nil
}

func (p *integrationPermCacheRepo) Set(
	ctx context.Context,
	key permissionports.PermissionCacheKey,
	perms *permissionports.CachedPermissions,
	ttl time.Duration,
) error {
	redisKey := fmt.Sprintf("perms:%s:%s", key.UserID.String(), key.OrgID.String())
	data, err := sonic.Marshal(perms)
	if err != nil {
		return err
	}
	if err := p.client.HSet(ctx, redisKey, key.Variant, data).Err(); err != nil {
		return err
	}
	return p.client.Expire(ctx, redisKey, ttl).Err()
}

func (p *integrationPermCacheRepo) Delete(ctx context.Context, userID, orgID pulid.ID) error {
	key := fmt.Sprintf("perms:%s:%s", userID.String(), orgID.String())
	return p.client.Del(ctx, key).Err()
}

func (p *integrationPermCacheRepo) InvalidateByRole(
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

func (p *integrationPermCacheRepo) Epoch(
	context.Context,
	pulid.ID,
	pulid.ID,
) (permissionports.CacheEpoch, error) {
	return permissionports.CacheEpoch{}, nil
}

func (p *integrationPermCacheRepo) InvalidateOrganization(
	ctx context.Context,
	orgID pulid.ID,
) error {
	pattern := fmt.Sprintf("perms:*:%s", orgID.String())
	keys, err := p.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return p.client.Del(ctx, keys...).Err()
	}
	return nil
}

type integrationPermEngine struct {
	roleRepo  permissionports.RoleRepository
	cacheRepo permissionports.PermissionCacheRepository
	registry  *permission.Registry
}

func (e *integrationPermEngine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}

	if len(roleIDs) == 0 {
		return &permissionports.PermissionCheckResult{Allowed: false, Reason: "no_permission"}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		for _, perm := range role.Permissions {
			if string(perm.Resource) == string(req.Resource) {
				for _, op := range perm.Operations {
					if op == req.Operation {
						return &permissionports.PermissionCheckResult{
							Allowed:   true,
							Reason:    "allowed",
							DataScope: perm.DataScope,
						}, nil
					}
				}
			}
		}
	}

	return &permissionports.PermissionCheckResult{Allowed: false, Reason: "no_permission"}, nil
}

func (e *integrationPermEngine) CheckBatch(
	_ context.Context,
	_ *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	return nil, nil
}

func (e *integrationPermEngine) GetLightManifest(
	_ context.Context,
	_, _ pulid.ID,
) (*permissionports.LightPermissionManifest, error) {
	return nil, nil
}

func (e *integrationPermEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	_ string,
) (*permissionports.ResourcePermissionDetail, error) {
	return nil, nil
}

func (e *integrationPermEngine) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, e, userID, orgID, resources)
}

func (e *integrationPermEngine) InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error {
	return e.cacheRepo.Delete(ctx, userID, orgID)
}

func (e *integrationPermEngine) GetEffectivePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		roleIDs = append(roleIDs, a.RoleID)
	}

	if len(roleIDs) == 0 {
		return &permissionports.EffectivePermissions{
			UserID:         userID,
			OrganizationID: orgID,
			Resources:      make(map[string]permissionports.EffectiveResourcePermission),
		}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	resources := make(map[string]permissionports.EffectiveResourcePermission)
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

	return &permissionports.EffectivePermissions{
		UserID:         userID,
		OrganizationID: orgID,
		MaxSensitivity: maxSensitivity,
		Resources:      resources,
	}, nil
}

func (e *integrationPermEngine) SimulatePermissions(
	_ context.Context,
	_ *permissionports.SimulatePermissionsRequest,
) (*permissionports.EffectivePermissions, error) {
	return nil, nil
}

func (e *integrationPermEngine) AgentsUsable(
	_ context.Context,
	_ *permissionports.RequestActor,
	_ permission.Operation,
) (*permissionports.UsableAgents, error) {
	return &permissionports.UsableAgents{Assistant: true}, nil
}

func (e *integrationPermEngine) MayUseAgent(
	_ context.Context,
	_ *permissionports.RequestActor,
	_ *agentdefinition.Definition,
) (bool, error) {
	return true, nil
}

func (e *integrationPermEngine) RoleCoverage(
	_ context.Context,
	_ *permissionports.RoleCoverageRequest,
) ([]permissionports.RoleCoverage, error) {
	return []permissionports.RoleCoverage{}, nil
}

type integrationFixture struct {
	svc  *Service
	db   *keel.DB
	repo permissionports.RoleRepository
	data *seedtest.TestData
	ctx  context.Context
}

func (f *integrationFixture) orgID() pulid.ID { return f.data.Organization.ID }

func (f *integrationFixture) actorID() pulid.ID { return f.data.User.ID }

func (f *integrationFixture) addUser(t *testing.T, username string) pulid.ID {
	t.Helper()

	return seedtest.NewUser(f.orgID(), f.data.BusinessUnit.ID).
		WithUsername(username).
		WithEmail(username+"@example.com").
		Build(t, f.ctx, f.db).ID
}

func (f *integrationFixture) addRole(t *testing.T, role *permission.Role) *permission.Role {
	t.Helper()

	if role.MaxSensitivity == "" {
		role.MaxSensitivity = permission.SensitivityInternal
	}
	require.NoError(t, permissiondb.Roles.Create(f.ctx, f.db, role))
	return role
}

func setupIntegrationService(t *testing.T) *integrationFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	redisClient := sharedtestutil.SetupTestRedis(t)

	roleRepo := rolerepository.New(rolerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	cacheRepo := &integrationPermCacheRepo{client: redisClient}

	permEngine := &integrationPermEngine{
		roleRepo:  roleRepo,
		cacheRepo: cacheRepo,
		registry:  permission.NewRegistry(),
	}

	svc := &Service{
		l:          zap.NewNop().Named("test.role-service"),
		roleRepo:   roleRepo,
		rbacRepo:   &rbactest.Repository{},
		notifier:   &permchangetest.Recorder{},
		permEngine: permEngine,
		validator: &Validator{
			validator: validationframework.
				NewTenantedValidatorBuilder[*permission.Role]().
				WithModelName("Role").
				Build(),
		},
		registry: permission.NewRegistry(),
	}

	return &integrationFixture{
		svc:  svc,
		db:   db,
		repo: roleRepo,
		data: data,
		ctx:  data.Ctx(ctx),
	}
}

func TestIntegration_CreateRole_CircularInheritance(t *testing.T) {
	f := setupIntegrationService(t)

	roleID := pulid.MustNew("rol_")
	role := &permission.Role{
		ID:            roleID,
		Name:          "Self Referencing",
		ParentRoleIDs: []pulid.ID{roleID},
	}

	err := f.svc.CreateRole(f.ctx, CreateRoleRequest{
		ActorID:        f.actorID(),
		OrganizationID: f.orgID(),
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCircularInheritance, err)
}

// seedActorRole gives the actor a role carrying the sensitivity and resource
// grants the service's no-escalation guard requires. Without it an actor can
// only ever grant permissions it already holds, which is nothing.
func seedActorRole(
	t *testing.T,
	f *integrationFixture,
	sensitivity permission.FieldSensitivity,
	resources ...string,
) pulid.ID {
	t.Helper()

	actorRole := f.addRole(t, &permission.Role{Name: "Actor Role", MaxSensitivity: sensitivity})

	grants := make([]*permission.ResourcePermission, 0, len(resources))
	for _, resource := range resources {
		grants = append(grants, &permission.ResourcePermission{
			RoleID:   actorRole.ID,
			Resource: resource,
			Operations: []permission.Operation{
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
				permission.OpDelete,
			},
			DataScope: permission.DataScopeOrganization,
		})
	}
	if len(grants) > 0 {
		require.NoError(t, permissiondb.ResourcePermissions.CreateMany(f.ctx, f.db, grants))
	}

	require.NoError(
		t,
		permissiondb.UserRoleAssignments.Create(f.ctx, f.db, &permission.UserRoleAssignment{
			UserID:         f.actorID(),
			OrganizationID: f.orgID(),
			RoleID:         actorRole.ID,
			AssignedAt:     timeutils.NowUnix(),
		}),
	)

	return actorRole.ID
}

func TestIntegration_UpdateRole_Success(t *testing.T) {
	f := setupIntegrationService(t)

	existing := f.addRole(t, &permission.Role{Name: "Original"})
	seedActorRole(t, f, permission.SensitivityRestricted)

	role := &permission.Role{
		ID:             existing.ID,
		Name:           "Updated Role",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	err := f.svc.UpdateRole(f.ctx, UpdateRoleRequest{
		ActorID:        f.actorID(),
		OrganizationID: f.orgID(),
		Role:           role,
	})
	require.NoError(t, err)

	stored, err := f.repo.GetByID(f.ctx, permissionports.GetRoleByIDRequest{
		ID:         existing.ID,
		TenantInfo: f.data.Info(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Role", stored.Name)
}

func TestIntegration_UpdateRole_SystemRole_Fails(t *testing.T) {
	f := setupIntegrationService(t)

	existing := f.addRole(t, &permission.Role{Name: "System Role", IsSystem: true})

	err := f.svc.UpdateRole(f.ctx, UpdateRoleRequest{
		ActorID:        f.actorID(),
		OrganizationID: f.orgID(),
		Role:           &permission.Role{ID: existing.ID, Name: "Updated"},
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)
}

func TestIntegration_AssignRole_Success(t *testing.T) {
	f := setupIntegrationService(t)

	targetUserID := f.addUser(t, "target")
	seedActorRole(t, f, permission.SensitivityInternal)
	role := f.addRole(t, &permission.Role{Name: "Test Role"})

	assignment := &permission.UserRoleAssignment{
		UserID: targetUserID,
		RoleID: role.ID,
	}

	err := f.svc.AssignRole(f.ctx, AssignRoleRequest{
		ActorID:        f.actorID(),
		OrganizationID: f.orgID(),
		Assignment:     assignment,
	})

	require.NoError(t, err)
	assert.Equal(t, f.orgID(), assignment.OrganizationID)
	assert.Equal(t, f.actorID(), assignment.AssignedBy)

	assignments, err := f.repo.GetUserRoleAssignments(f.ctx, targetUserID, f.orgID())
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, role.ID, assignments[0].RoleID)
}

func TestIntegration_InitializeOrganizationRoles(t *testing.T) {
	f := setupIntegrationService(t)

	creatorID := f.addUser(t, "creator")

	require.NoError(t, f.svc.InitializeOrganizationRoles(f.ctx, f.orgID(), creatorID))

	assignments, err := f.repo.GetUserRoleAssignments(f.ctx, creatorID, f.orgID())
	require.NoError(t, err)
	assert.NotEmpty(t, assignments)
}

func TestIntegration_CreateResourcePermission_Success(t *testing.T) {
	f := setupIntegrationService(t)

	role := f.addRole(t, &permission.Role{Name: "Test Role"})
	seedActorRole(t, f, permission.SensitivityRestricted, "shipment")

	rp := &permission.ResourcePermission{
		RoleID:     role.ID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	err := f.svc.CreateResourcePermission(f.ctx, f.actorID(), f.orgID(), rp)

	require.NoError(t, err)
	assert.NotEmpty(t, rp.ID)

	stored, err := f.repo.GetResourcePermissionsByRoleID(f.ctx, role.ID)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "shipment", stored[0].Resource)
}
