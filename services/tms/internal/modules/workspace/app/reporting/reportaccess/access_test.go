package reportaccess_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type groups struct {
	visible permission.FieldGroupSet
}

func (g groups) Visible(context.Context, *permissionports.RequestActor) permission.FieldGroupSet {
	return g.visible
}

// countingEngine counts the engine reads a request makes.
type countingEngine struct {
	*permtest.Engine
	batches int
	singles int
}

func (c *countingEngine) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	c.batches++
	return permtest.Batch(ctx, c.Engine, userID, orgID, resources)
}

func (c *countingEngine) GetResourcePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	c.singles++
	return c.Engine.GetResourcePermissions(ctx, userID, orgID, resource)
}

func without(engine *permtest.Engine, resources ...string) *permtest.Engine {
	for _, resource := range resources {
		engine.Overrides[resource] = permtest.Grant{
			DataScope:      permission.DataScopeOrganization,
			MaxSensitivity: permission.SensitivityInternal,
		}
	}
	return engine
}

func resolver(
	engine permissionports.PermissionEngine,
	visible permission.FieldGroupSet,
) *reportaccess.Resolver {
	registry := permission.NewRegistry()
	return reportaccess.New(reportaccess.Params{
		Engine:   engine,
		Registry: registry,
		Views:    refview.NewPort(refview.New(registry), engine),
		Fields:   groups{visible: visible},
	})
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func entity(t *testing.T, key string) *reportcatalog.Entity {
	t.Helper()
	found, ok := reportcatalog.Default.Entity(key)
	require.Truef(t, ok, "no catalog entity %q", key)
	return found
}

func fieldUse(t *testing.T, access *reportaccess.Access, entityKey, key string) reportaccess.Use {
	t.Helper()
	owner := entity(t, entityKey)
	field, ok := owner.Field(key)
	require.Truef(t, ok, "no field %s.%s", entityKey, key)
	use, err := access.Field(t.Context(), owner, field)
	require.NoError(t, err)
	return use
}

func edgeUse(t *testing.T, access *reportaccess.Access, entityKey, name string) reportaccess.Use {
	t.Helper()
	owner := entity(t, entityKey)
	edge, ok := owner.Edge(name)
	require.Truef(t, ok, "no edge %s.%s", entityKey, name)
	use, err := access.Edge(t.Context(), owner, edge)
	require.NoError(t, err)
	return use
}

func TestField_AKindTheCallerCannotReadOffersOnlyItsReferenceView(t *testing.T) {
	t.Parallel()

	access, err := resolver(without(permtest.AllowAll(), "customer"), permission.AllFieldGroups()).
		ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	name := fieldUse(t, access, "customer", "name")
	assert.Equal(t, reportaccess.Available, name.Decision)
	assert.True(t, name.Reference)

	dot := fieldUse(t, access, "customer", "dotNumber")
	assert.Equal(t, reportaccess.NeedsKindRead, dot.Decision)
	assert.Equal(t, permission.ResourceCustomer, dot.Kind)
	assert.False(t, dot.Reference)
}

func TestField_TheCeilingAppliesOnAReadableKindAndInsideAView(t *testing.T) {
	t.Parallel()

	engine := permtest.AllowAll()
	engine.DefaultGrant.MaxSensitivity = permission.SensitivityPublic
	engine.Overrides["customer"] = permtest.Grant{MaxSensitivity: permission.SensitivityPublic}
	access, err := resolver(engine, permission.AllFieldGroups()).ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	assert.Equal(t, reportaccess.WithheldByCeiling,
		fieldUse(t, access, "shipment", "status").Decision)
	assert.Equal(t, reportaccess.WithheldByCeiling,
		fieldUse(t, access, "customer", "name").Decision,
		"a reference field is still held to the reader's ceiling")
}

func TestField_AProtectedFieldNeedsItsGroup(t *testing.T) {
	t.Parallel()

	withheld, err := resolver(permtest.AllowAll(), 0).ForCatalog(t.Context(), tenant())
	require.NoError(t, err)
	granted, err := resolver(permtest.AllowAll(), permission.AllFieldGroups()).
		ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	use := fieldUse(t, withheld, "worker_profile", "licenseNumber")
	assert.Equal(t, reportaccess.NeedsFieldGroup, use.Decision)
	assert.Equal(t, permission.FieldGroupPersonal, use.Group)
	assert.False(t, use.HoldsGroup)

	use = fieldUse(t, granted, "worker_profile", "licenseNumber")
	assert.Equal(t, reportaccess.Available, use.Decision)
	assert.True(t, use.HoldsGroup)
}

func TestEdge_AnEdgeOutsideAnUnreadableKindsViewNeedsItsRead(t *testing.T) {
	t.Parallel()

	access, err := resolver(without(permtest.AllowAll(), "worker"), permission.AllFieldGroups()).
		ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	assert.Equal(t, reportaccess.Available, edgeUse(t, access, "worker", "fleetCode").Decision)
	profile := edgeUse(t, access, "worker", "profile")
	assert.Equal(t, reportaccess.NeedsKindRead, profile.Decision)
	assert.Equal(t, permission.ResourceWorker, profile.Kind)
	assert.Equal(t, reportaccess.Available, edgeUse(t, access, "shipment", "customer").Decision,
		"an edge out of a kind the caller reads is theirs to follow")
}

func TestEdge_APickEdgeNeedsReadOnTheRowsItPicksFrom(t *testing.T) {
	t.Parallel()

	access, err := resolver(
		without(permtest.AllowAll(), "shipment_stop"), permission.AllFieldGroups(),
	).ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	use := edgeUse(t, access, "shipment", "destinationStop")
	assert.Equal(t, reportaccess.NeedsKindRead, use.Decision,
		"the destination stop is picked from the shipment's stop rows")
	assert.Equal(t, permission.Resource("shipment_stop"), use.Kind)
}

func TestCatalog_ReadsThePermissionsInOneBatch(t *testing.T) {
	t.Parallel()

	engine := &countingEngine{Engine: without(permtest.AllowAll(), "customer", "worker")}
	access, err := resolver(engine, permission.AllFieldGroups()).ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	entities, err := access.Catalog(t.Context())
	require.NoError(t, err)

	assert.Len(t, entities, len(reportcatalog.Default.Entities))
	assert.Equal(t, 1, engine.batches)
	assert.Zero(t, engine.singles, "every kind the catalog decides was in the batch")
}

func TestCatalog_AnUnreadableKindIsUsableOnlyThroughAView(t *testing.T) {
	t.Parallel()

	access, err := resolver(without(permtest.AllowAll(), "customer", "worker"), 0).
		ForCatalog(t.Context(), tenant())
	require.NoError(t, err)

	customer, err := access.Entity(t.Context(), entity(t, "customer"))
	require.NoError(t, err)
	assert.False(t, customer.Readable)
	assert.False(t, customer.Exportable)
	assert.True(t, customer.Usable(), "a customer's name is in its view")

	profile, err := access.Entity(t.Context(), entity(t, "worker_profile"))
	require.NoError(t, err)
	assert.False(t, profile.Usable(), "a worker's profile is never in the worker view")
}

func TestReason_NamesTheMissingReadTierOrGroup(t *testing.T) {
	t.Parallel()

	r := resolver(permtest.AllowAll(), 0)
	customer := entity(t, "customer")

	assert.Equal(t,
		`you do not have read access to Customers, which "DOT Number" on Customer needs; `+
			"without it a report can use only its reference fields",
		r.Reason(customer, "DOT Number", reportaccess.Use{
			Decision: reportaccess.NeedsKindRead,
			Kind:     permission.ResourceCustomer,
		}))
	assert.Equal(t,
		`field "License Number" on Customer is in the Personal protected field group, `+
			"which your role does not grant",
		r.Reason(customer, "License Number", reportaccess.Use{
			Decision: reportaccess.NeedsFieldGroup,
			Group:    permission.FieldGroupPersonal,
		}))
	assert.Equal(t, `field "Name" on Customer requires internal access`,
		r.Reason(customer, "Name", reportaccess.Use{
			Decision:    reportaccess.WithheldByCeiling,
			Sensitivity: permission.SensitivityInternal,
		}))
	assert.Empty(t, r.Reason(customer, "Name", reportaccess.Use{}))
	assert.Equal(t, "customer read", r.Need(reportaccess.Use{
		Decision: reportaccess.NeedsKindRead,
		Kind:     permission.ResourceCustomer,
	}))
}
