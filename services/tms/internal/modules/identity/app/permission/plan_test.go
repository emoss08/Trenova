package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreach"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var accountingDenial = permissionports.PlanDenial{
	Feature:     platformcatalog.FeatureAccounting,
	FeatureName: "Accounting",
	AddOn:       platformcatalog.AddOnAccounting,
	AddOnName:   "Accounting & Cash",
}

var errPlanUnavailable = errors.New("purchased add-ons unavailable")

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errPlanUnavailable
}

func (r failingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return r.Enabled(ctx, feature)
}

func addOnFeatures() []platformcatalog.FeatureKey {
	keys := make([]platformcatalog.FeatureKey, 0)
	for _, feature := range platformcatalog.Features() {
		if !feature.IsBase() {
			keys = append(keys, feature.Key)
		}
	}

	return keys
}

func baseOnlyPlan() *plantest.Gate {
	return plantest.Without(addOnFeatures()...)
}

func administrator() *permissionports.CachedPermissions {
	registry := permissioncatalog.NewRegistry()
	grants := make(map[string][]string)
	for _, definition := range registry.All() {
		ops := registry.GetOperationsForResource(definition.Resource)
		names := make([]string, 0, len(ops))
		for _, op := range ops {
			names = append(names, string(op))
		}
		grants[definition.Resource] = names
	}

	held := creatorHolds(grants)
	held.MaxSensitivity = string(permission.SensitivityConfidential)

	return held
}

func dispatcher() *permissionports.CachedPermissions {
	held := creatorHolds(map[string][]string{
		permission.ResourceShipment.String():             {"read", "update"},
		permission.ResourceGeneralLedgerAccount.String(): {"read"},
	})
	held.MaxSensitivity = string(permission.SensitivityInternal)

	return held
}

type planCase struct {
	eng    *engine
	roles  *permissionportsmock.MockRoleRepository
	cache  *permissionportsmock.MockPermissionCacheRepository
	users  *tenantportsmock.MockUserRepository
	userID pulid.ID
	orgID  pulid.ID
	buID   pulid.ID
}

func newPlanCase(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
	held *permissionports.CachedPermissions,
) *planCase {
	t.Helper()

	eng, roles, cache, users := setupTestEngine(t)
	eng.withPlan(gate)
	c := &planCase{
		eng:    eng,
		roles:  roles,
		cache:  cache,
		users:  users,
		userID: pulid.MustNew("usr_"),
		orgID:  pulid.MustNew("org_"),
		buID:   pulid.MustNew("bu_"),
	}
	cache.On("Get", mock.Anything, allRolesKey(c.userID, c.orgID)).Return(held, nil).Maybe()
	users.On("GetUserOrganizationSummaries", mock.Anything, c.userID).
		Return([]tenantports.OrgSummary{{ID: c.orgID, Name: "Test Org"}}, nil).
		Maybe()

	return c
}

func (c *planCase) request(
	resource permission.Resource,
	op permission.Operation,
) *permissionports.PermissionCheckRequest {
	return &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    c.userID,
		UserID:         c.userID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Resource:       resource.String(),
		Operation:      op,
	}
}

func (c *planCase) check(
	t *testing.T,
	resource permission.Resource,
	op permission.Operation,
) *permissionports.PermissionCheckResult {
	t.Helper()

	result, err := c.eng.Check(t.Context(), c.request(resource, op))
	require.NoError(t, err)

	return result
}

func TestPlan_CheckRefusesAnAddOnResourceTheBasePlanLacks(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	result := c.check(t, permission.ResourceGeneralLedgerAccount, permission.OpRead)

	assert.False(t, result.Allowed)
	assert.Equal(t, permissionports.ReasonNotOnPlan, result.Reason)
	require.NotNil(t, result.PlanDenial)
	assert.Equal(t, accountingDenial, *result.PlanDenial)
	assert.Empty(t, result.DataScope)
	c.cache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestPlan_CheckDecidesABaseResourceAsBefore(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	allowed := c.check(t, permission.ResourceShipment, permission.OpUpdate)
	assert.True(t, allowed.Allowed)
	assert.Equal(t, "allowed", allowed.Reason)
	assert.Equal(t, permission.DataScopeOrganization, allowed.DataScope)
	assert.Nil(t, allowed.PlanDenial)

	denied := c.check(t, permission.ResourceShipment, permission.OpDelete)
	assert.False(t, denied.Allowed)
	assert.Equal(t, "no_permission", denied.Reason)
	assert.Nil(t, denied.PlanDenial)
}

func TestPlan_AnAdministratorIsStillRefusedAnAddOnThatIsOff(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), administrator())

	for _, feature := range addOnFeatures() {
		for _, definition := range permissioncatalog.NewRegistry().All() {
			if definition.Feature != feature {
				continue
			}
			result := c.check(
				t,
				permission.Resource(definition.Resource),
				permission.OpRead,
			)
			assert.False(t, result.Allowed, definition.Resource)
			assert.Equal(t, permissionports.ReasonNotOnPlan, result.Reason, definition.Resource)
			require.NotNil(t, result.PlanDenial, definition.Resource)
			assert.Equal(t, feature, result.PlanDenial.Feature, definition.Resource)
		}
	}

	base := c.check(t, permission.ResourceShipment, permission.OpRead)
	assert.True(t, base.Allowed)
	assert.Equal(t, "allowed", base.Reason)
	assert.Nil(t, base.PlanDenial)
}

func TestPlan_AnAPIKeyIsRefusedAnAddOnResourceBeforeItsKeyIsRead(t *testing.T) {
	t.Parallel()

	c := newAPIKeyCase(t)
	c.eng.withPlan(baseOnlyPlan())

	result, err := c.eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    c.keyID,
		APIKeyID:       c.keyID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Resource:       permission.ResourceGeneralLedgerAccount.String(),
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, permissionports.ReasonNotOnPlan, result.Reason)
	require.NotNil(t, result.PlanDenial)
	assert.Equal(t, accountingDenial, *result.PlanDenial)
	c.keys.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything, mock.Anything)
}

func TestPlan_AnAgentIsRefusedAnAddOnResource(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	eng.withPlan(baseOnlyPlan())
	actor := unattendedSystemActor()

	result, err := eng.Check(
		t.Context(),
		actor.PermissionCheck(permission.ResourceGeneralLedgerAccount, permission.OpRead),
	)

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, permissionports.ReasonNotOnPlan, result.Reason)
	require.NotNil(t, result.PlanDenial)
	assert.Equal(t, accountingDenial, *result.PlanDenial)
	cacheRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestPlan_CheckBatchAnswersEachItemOnItsOwn(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	batch, err := c.eng.CheckBatch(t.Context(), &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    c.userID,
		UserID:         c.userID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{
				Resource:  permission.ResourceGeneralLedgerAccount.String(),
				Operation: permission.OpRead,
			},
			{Resource: permission.ResourceShipment.String(), Operation: permission.OpRead},
			{Resource: "not_a_resource", Operation: permission.OpRead},
			{Resource: permission.ResourceShipment.String(), Operation: permission.OpDelete},
		},
	})

	require.NoError(t, err)
	require.Len(t, batch.Results, 4)

	offPlan := batch.Results[0]
	assert.False(t, offPlan.Allowed)
	assert.Equal(t, permissionports.ReasonNotOnPlan, offPlan.Reason)
	require.NotNil(t, offPlan.PlanDenial)
	assert.Equal(t, accountingDenial, *offPlan.PlanDenial)

	assert.True(t, batch.Results[1].Allowed)
	assert.Equal(t, "allowed", batch.Results[1].Reason)
	assert.Nil(t, batch.Results[1].PlanDenial)

	assert.False(t, batch.Results[2].Allowed)
	assert.Equal(t, "no_permission", batch.Results[2].Reason)
	assert.Nil(t, batch.Results[2].PlanDenial)

	assert.False(t, batch.Results[3].Allowed)
	assert.Equal(t, "no_permission", batch.Results[3].Reason)
	assert.Nil(t, batch.Results[3].PlanDenial)

	assert.True(t, batch.CacheHit)
}

func TestPlan_CheckBatchOfOnlyOffPlanItemsReadsNoPermissions(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	batch, err := c.eng.CheckBatch(t.Context(), &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    c.userID,
		UserID:         c.userID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{
				Resource:  permission.ResourceGeneralLedgerAccount.String(),
				Operation: permission.OpRead,
			},
			{Resource: permission.ResourceAssistant.String(), Operation: permission.OpCreate},
		},
	})

	require.NoError(t, err)
	require.Len(t, batch.Results, 2)
	assert.Equal(t, permissionports.ReasonNotOnPlan, batch.Results[0].Reason)
	assert.Equal(t, permissionports.ReasonNotOnPlan, batch.Results[1].Reason)
	require.NotNil(t, batch.Results[1].PlanDenial)
	assert.Equal(t, platformcatalog.FeatureAIDesk, batch.Results[1].PlanDenial.Feature)
	assert.Equal(t, platformcatalog.AddOnAI, batch.Results[1].PlanDenial.AddOn)
	assert.Equal(t, "Trenova AI", batch.Results[1].PlanDenial.AddOnName)
	assert.False(t, batch.CacheHit)
	c.cache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestPlan_ManifestOmitsOffPlanResourcesAndListsWhatIsOff(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	manifest, err := c.eng.GetLightManifest(t.Context(), c.userID, c.orgID)

	require.NoError(t, err)
	assert.Contains(t, manifest.Permissions, permission.ResourceShipment.String())
	assert.NotContains(t, manifest.Permissions, permission.ResourceGeneralLedgerAccount.String())
	assert.Equal(t, c.eng.routeRegistry.ComputeAccess(manifest.Permissions), manifest.RouteAccess)

	off := addOnFeatures()
	require.Len(t, manifest.UnavailableFeatures, len(off))
	for i, feature := range off {
		assert.Equal(t, feature, manifest.UnavailableFeatures[i].Feature)
	}
	assert.Equal(t, accountingDenial, manifest.UnavailableFeatures[0])
}

func TestPlan_ManifestChecksumFollowsThePlan(t *testing.T) {
	t.Parallel()

	baseOnly := newPlanCase(t, baseOnlyPlan(), dispatcher())
	first, err := baseOnly.eng.GetLightManifest(t.Context(), baseOnly.userID, baseOnly.orgID)
	require.NoError(t, err)
	again, err := baseOnly.eng.GetLightManifest(t.Context(), baseOnly.userID, baseOnly.orgID)
	require.NoError(t, err)
	assert.Equal(t, first.Checksum, again.Checksum)

	samePlan := newPlanCase(t, baseOnlyPlan(), dispatcher())
	other, err := samePlan.eng.GetLightManifest(t.Context(), samePlan.userID, samePlan.orgID)
	require.NoError(t, err)
	assert.Equal(t, first.Checksum, other.Checksum)

	withoutAccounting := newPlanCase(
		t,
		plantest.Without(platformcatalog.FeatureAccounting),
		dispatcher(),
	)
	narrower, err := withoutAccounting.eng.GetLightManifest(
		t.Context(),
		withoutAccounting.userID,
		withoutAccounting.orgID,
	)
	require.NoError(t, err)

	assert.Equal(t, first.Permissions, narrower.Permissions,
		"the same resources are withheld, so only the list of what is off differs")
	assert.NotEqual(t, first.Checksum, narrower.Checksum)
}

func TestPlan_ResourceDetailOfAnOffPlanResourceGrantsNothing(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	detail, err := c.eng.GetResourcePermissions(
		t.Context(),
		c.userID,
		c.orgID,
		permission.ResourceGeneralLedgerAccount.String(),
	)

	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, permission.ResourceGeneralLedgerAccount.String(), detail.Resource)
	assert.Empty(t, detail.Operations)
	assert.Empty(t, detail.DataScope)
	assert.Equal(t, permission.SensitivityInternal, detail.MaxSensitivity)
	require.NotNil(t, detail.PlanDenial)
	assert.Equal(t, accountingDenial, *detail.PlanDenial)
}

func TestPlan_ResourceDetailBatchAnswersEachResourceOnItsOwn(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())

	details, err := c.eng.GetResourcePermissionsBatch(t.Context(), c.userID, c.orgID, []string{
		permission.ResourceShipment.String(),
		permission.ResourceGeneralLedgerAccount.String(),
		"not_a_resource",
	})

	require.NoError(t, err)
	require.Len(t, details, 2)

	shipment := details[permission.ResourceShipment.String()]
	assert.ElementsMatch(t, []permission.Operation{permission.OpRead, permission.OpUpdate},
		shipment.Operations)
	assert.Nil(t, shipment.PlanDenial)

	ledger := details[permission.ResourceGeneralLedgerAccount.String()]
	assert.Empty(t, ledger.Operations)
	require.NotNil(t, ledger.PlanDenial)
	assert.Equal(t, accountingDenial, *ledger.PlanDenial)
}

func TestPlan_EffectivePermissionsMarkWhatIsOffThePlan(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, baseOnlyPlan(), dispatcher())
	roleID := pulid.MustNew("rol_")
	c.roles.On("GetUserRoleAssignments", mock.Anything, c.userID, c.orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: c.userID, OrganizationID: c.orgID},
		}, nil)
	c.roles.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID}).
		Return([]*rbac.Role{{
			ID:   roleID,
			Name: "Controller",
			Permissions: []*rbac.ResourcePermission{
				{
					Resource:   permission.ResourceShipment.String(),
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
				{
					Resource:   permission.ResourceGeneralLedgerAccount.String(),
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}}, nil)

	effective, err := c.eng.GetEffectivePermissions(t.Context(), c.userID, c.orgID)

	require.NoError(t, err)
	ledger := effective.Resources[permission.ResourceGeneralLedgerAccount.String()]
	require.NotNil(t, ledger.PlanDenial)
	assert.Equal(t, accountingDenial, *ledger.PlanDenial)
	assert.Equal(t, []permission.Operation{permission.OpRead}, ledger.Operations,
		"what the roles grant is still reported, for delegation checks")
	assert.Equal(t, []string{"Controller"}, ledger.GrantedBy)
	assert.Nil(t, effective.Resources[permission.ResourceShipment.String()].PlanDenial)
}

func TestPlan_AgentsAreUnusableWithoutTheAIAddOn(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	eng.withPlan(baseOnlyPlan())
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	usable, err := eng.AgentsUsable(t.Context(), actor, permission.OpCreate)

	require.NoError(t, err)
	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAIDesk)
	assert.Equal(t, &permissionports.UsableAgents{PlanDenial: &denial}, usable)
	cacheRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestPlan_RoleCoverageCountsAnOffPlanResourceAsMissing(t *testing.T) {
	t.Parallel()

	grant := func(resource permission.Resource) *rbac.ResourcePermission {
		return &rbac.ResourcePermission{
			Resource:   resource.String(),
			Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			DataScope:  permission.DataScopeOrganization,
		}
	}
	required := []permissionports.RequiredGrant{
		{Tool: "shipment_get", Resource: permission.ResourceShipment, Operation: permission.OpRead},
		{
			Tool:      "gl_account_get",
			Resource:  permission.ResourceGeneralLedgerAccount,
			Operation: permission.OpRead,
		},
	}

	tests := []struct {
		name     string
		gate     platformcatalogports.FeatureGate
		coverage agentreach.AudienceCoverage
		missing  []permission.Resource
	}{
		{
			name:     "every feature on covers it all",
			gate:     plantest.AllFeatures(),
			coverage: agentreach.CoverageFull,
			missing:  []permission.Resource{},
		},
		{
			name:     "without accounting the ledger is missing",
			gate:     plantest.Without(platformcatalog.FeatureAccounting),
			coverage: agentreach.CoveragePartial,
			missing:  []permission.Resource{permission.ResourceGeneralLedgerAccount},
		},
		{
			name:     "without the AI add-on nothing is covered",
			gate:     baseOnlyPlan(),
			coverage: agentreach.CoverageNone,
			missing:  []permission.Resource{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, roleRepo, _, _ := setupTestEngine(t)
			eng.withPlan(tt.gate)
			orgID := pulid.MustNew("org_")
			roleID := pulid.MustNew("rol_")
			roleRepo.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID}).
				Return([]*rbac.Role{{
					ID:             roleID,
					OrganizationID: orgID,
					Permissions: []*rbac.ResourcePermission{
						grant(permission.ResourceAssistant),
						grant(permission.ResourceShipment),
						grant(permission.ResourceGeneralLedgerAccount),
					},
				}}, nil)

			coverage, err := eng.RoleCoverage(t.Context(), &permissionports.RoleCoverageRequest{
				OrganizationID: orgID,
				RoleIDs:        []pulid.ID{roleID},
				Required:       required,
			})

			require.NoError(t, err)
			require.Len(t, coverage, 1)
			assert.Equal(t, tt.coverage, coverage[0].Coverage)
			assert.Equal(t, tt.missing, coverage[0].MissingResources)
		})
	}
}

func TestPlan_EveryFeatureOnAnswersAsBefore(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, plantest.AllFeatures(), dispatcher())

	result := c.check(t, permission.ResourceGeneralLedgerAccount, permission.OpRead)
	assert.True(t, result.Allowed)
	assert.Equal(t, "allowed", result.Reason)
	assert.Equal(t, permission.DataScopeOrganization, result.DataScope)
	assert.Nil(t, result.PlanDenial)

	manifest, err := c.eng.GetLightManifest(t.Context(), c.userID, c.orgID)
	require.NoError(t, err)
	assert.Empty(t, manifest.UnavailableFeatures)
	assert.Equal(t, c.eng.lightResourceBitmasks(dispatcher()), manifest.Permissions)

	detail, err := c.eng.GetResourcePermissions(
		t.Context(),
		c.userID,
		c.orgID,
		permission.ResourceGeneralLedgerAccount.String(),
	)
	require.NoError(t, err)
	assert.Equal(t, c.eng.resourceDetail(dispatcher(), detail.Resource), detail)
	assert.Nil(t, detail.PlanDenial)
}

func TestPlan_AGateErrorIsAnErrorNeverAnAllow(t *testing.T) {
	t.Parallel()

	c := newPlanCase(t, failingGate{}, administrator())

	result, err := c.eng.Check(
		t.Context(),
		c.request(permission.ResourceShipment, permission.OpRead),
	)
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, result)

	batch, err := c.eng.CheckBatch(t.Context(), &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    c.userID,
		UserID:         c.userID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: permission.ResourceShipment.String(), Operation: permission.OpRead},
		},
	})
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, batch)

	manifest, err := c.eng.GetLightManifest(t.Context(), c.userID, c.orgID)
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, manifest)

	detail, err := c.eng.GetResourcePermissions(
		t.Context(),
		c.userID,
		c.orgID,
		permission.ResourceShipment.String(),
	)
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, detail)

	details, err := c.eng.GetResourcePermissionsBatch(
		t.Context(),
		c.userID,
		c.orgID,
		[]string{permission.ResourceShipment.String()},
	)
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, details)

	usable, err := c.eng.AgentsUsable(t.Context(), &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    c.userID,
		UserID:         c.userID,
		OrganizationID: c.orgID,
		BusinessUnitID: c.buID,
	}, permission.OpCreate)
	require.ErrorIs(t, err, errPlanUnavailable)
	assert.Nil(t, usable)
}
