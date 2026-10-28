package reportaccess_test

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sharedCatalog = sync.OnceValues(reportengine.NewCatalog)

type groupAccess struct {
	visible permission.FieldGroupSet
}

func (g groupAccess) Visible(
	context.Context,
	*permissionports.RequestActor,
) permission.FieldGroupSet {
	return g.visible
}

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
	return c.Engine.GetResourcePermissionsBatch(ctx, userID, orgID, resources)
}

func (c *countingEngine) GetResourcePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	c.singles++
	return c.Engine.GetResourcePermissions(ctx, userID, orgID, resource)
}

type fixture struct {
	catalog  *analytics.Catalog
	engine   *countingEngine
	resolver *reportaccess.Resolver
}

func newFixture(t *testing.T, groups permission.FieldGroupSet) *fixture {
	t.Helper()
	catalog, err := sharedCatalog()
	require.NoError(t, err)
	engine := &countingEngine{Engine: permtest.AllowAll()}
	registry := permissioncatalog.NewRegistry()
	resolver, err := reportaccess.New(reportaccess.Params{
		Catalog:  catalog,
		Engine:   engine,
		Registry: registry,
		Views:    refview.NewPort(refview.New(registry), engine),
		Fields:   groupAccess{visible: groups},
	})
	require.NoError(t, err)
	return &fixture{catalog: catalog, engine: engine, resolver: resolver}
}

func (f *fixture) access(t *testing.T, mode reportaccess.Mode) *reportaccess.Access {
	t.Helper()
	access, err := f.resolver.For(t.Context(), pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}, mode)
	require.NoError(t, err)
	return access
}

func (f *fixture) field(t *testing.T, entity, path string) *analytics.Field {
	t.Helper()
	_, field, ok := f.catalog.Resolve(entity, path)
	require.Truef(t, ok, "%s has no field %s", entity, path)
	return field
}

func (f *fixture) edge(t *testing.T, entity, name string) *analytics.Edge {
	t.Helper()
	e, ok := f.catalog.Entity(entity)
	require.True(t, ok)
	edge, ok := e.Edge(name)
	require.Truef(t, ok, "%s has no edge %s", entity, name)
	return edge
}

func TestNew_RefusesACatalogEntityWithoutAKind(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	for _, entity := range f.catalog.Entities() {
		assert.NotEmptyf(t, f.resolver.Resource(entity), "%s has no resource", entity.Key)
	}
	_, err := reportaccess.New(reportaccess.Params{})
	require.Error(t, err)
}

func TestFor_ReadsThePermissionsInOneBatch(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	access := f.access(t, reportaccess.ModeRead)

	for _, entity := range f.catalog.Entities() {
		for _, field := range entity.Fields {
			access.FieldUse(t.Context(), field)
		}
	}
	assert.Equal(t, 1, f.engine.batches)
	assert.Zero(t, f.engine.singles, "every kind's grant came from the one batch")
}

func TestField_AKindTheCallerCannotReadOffersOnlyItsReferenceView(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	f.engine.Overrides[permission.ResourceCustomer.String()] = permtest.Grant{}
	access := f.access(t, reportaccess.ModeRead)

	name := access.FieldUse(t.Context(), f.field(t, "customer", "name"))
	assert.True(t, name.Reference)
	assert.Equal(
		t,
		reportaccess.Available,
		name.Decision,
		"a customer's name is in its reference view",
	)

	for _, field := range f.field(t, "customer", "name").Entity.Fields {
		use := access.FieldUse(t.Context(), field)
		if !use.Reference {
			assert.Equalf(t, reportaccess.NeedsKindRead, use.Decision, "customer.%s", field.Key)
			assert.Equal(t, permission.ResourceCustomer, use.Kind)
		}
	}

	entity, _ := f.catalog.Entity("customer")
	assert.Equal(t, reportaccess.NeedsKindRead, access.EntityUse(entity).Decision,
		"a report never starts from a kind the caller cannot read")
}

func TestField_TheCeilingAppliesOnAReadableKind(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	f.engine.Overrides[permission.ResourceShipment.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead},
		MaxSensitivity: permission.SensitivityPublic,
	}
	access := f.access(t, reportaccess.ModeRead)

	withheld := 0
	entity, _ := f.catalog.Entity("shipment")
	for _, field := range entity.Fields {
		use := access.FieldUse(t.Context(), field)
		if use.Sensitivity != permission.SensitivityPublic {
			assert.Equalf(t, reportaccess.WithheldByCeiling, use.Decision, "shipment.%s", field.Key)
			withheld++
		}
	}
	assert.Positive(t, withheld, "a public ceiling withholds the shipment's internal fields")
}

func TestField_AProtectedFieldNeedsItsGroup(t *testing.T) {
	t.Parallel()

	without := newFixture(t, 0)
	with := newFixture(t, permission.AllFieldGroups())
	denied := without.access(t, reportaccess.ModeRead)
	granted := with.access(t, reportaccess.ModeRead)

	checked := 0
	for _, entity := range without.catalog.Entities() {
		for _, field := range entity.Fields {
			group, protected := without.resolver.Protection(field)
			if !protected {
				continue
			}
			use := denied.FieldUse(t.Context(), field)
			assert.Equalf(
				t,
				reportaccess.NeedsFieldGroup,
				use.Decision,
				"%s.%s",
				entity.Key,
				field.Key,
			)
			assert.Equal(t, group, use.Group)
			assert.False(t, use.HoldsGroup)

			held := granted.FieldUse(t.Context(), field)
			assert.Equalf(t, reportaccess.Available, held.Decision, "%s.%s", entity.Key, field.Key)
			checked++
		}
	}
	assert.Positive(t, checked, "the catalog has no protected field")
}

func TestExport_NeedsTheKindsExport(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	f.engine.Overrides[permission.ResourceShipment.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead},
		MaxSensitivity: permission.SensitivityConfidential,
	}
	shipment, _ := f.catalog.Entity("shipment")
	field := f.field(t, "shipment", "weight")
	reference := f.field(t, "shipment", "proNumber")

	read := f.access(t, reportaccess.ModeRead)
	assert.Equal(t, reportaccess.Available, read.EntityUse(shipment).Decision)
	assert.Equal(t, reportaccess.Available, read.FieldUse(t.Context(), field).Decision)

	export := f.access(t, reportaccess.ModeExport)
	assert.Equal(t, reportaccess.ModeExport, export.Mode())
	assert.Equal(t, reportaccess.NeedsExport, export.EntityUse(shipment).Decision)
	assert.Equal(t, reportaccess.NeedsExport, export.FieldUse(t.Context(), field).Decision)

	use := export.FieldUse(t.Context(), reference)
	require.True(t, use.Reference)
	assert.Equal(t, reportaccess.Available, use.Decision,
		"a reference field is what any linked record shows, so it needs no export of its own")
}

func TestEdge_APickEdgeNeedsReadOnTheRowsItPicksFrom(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	pick := f.edge(t, "shipment", "originStop")
	require.True(t, pick.Pick)
	via := f.resolver.PickVia(pick)
	require.NotEmpty(t, via)

	assert.Equal(
		t,
		reportaccess.Available,
		f.access(t, reportaccess.ModeRead).EdgeUse(t.Context(), pick).Decision,
	)

	blocked := f.resolver.Resource(via[0])
	f.engine.Overrides[blocked.String()] = permtest.Grant{}
	use := f.access(t, reportaccess.ModeRead).EdgeUse(t.Context(), pick)
	assert.Equal(t, reportaccess.NeedsKindRead, use.Decision)
	assert.Equal(t, blocked, use.Kind)
}

func TestRows_AnOwnScopeGrantNarrowsToTheCallersRecords(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	f.engine.Overrides[permission.ResourceShipment.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead},
		DataScope:      permission.DataScopeOwn,
		MaxSensitivity: permission.SensitivityConfidential,
	}
	f.engine.Overrides[permission.ResourceCustomer.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead},
		DataScope:      permission.DataScopeOwn,
		MaxSensitivity: permission.SensitivityConfidential,
	}
	access := f.access(t, reportaccess.ModeRead)

	shipment, _ := f.catalog.Entity("shipment")
	assert.True(t, f.resolver.OwnScopeSupported(shipment))
	assert.Len(t, access.Rows(t.Context(), shipment), 1)
	assert.Equal(t, reportaccess.Available, access.EntityUse(shipment).Decision)

	customer, _ := f.catalog.Entity("customer")
	assert.False(t, f.resolver.OwnScopeSupported(customer))
	assert.Empty(t, access.Rows(t.Context(), customer))
	assert.Equal(t, reportaccess.OwnScopeUnsupported, access.EntityUse(customer).Decision,
		"a kind with no owner column cannot honour an own-scope grant, so it is refused")
}

func TestRefusal_NamesWhatTheCallerLacks(t *testing.T) {
	t.Parallel()

	f := newFixture(t, permission.AllFieldGroups())
	f.engine.Overrides[permission.ResourceCustomer.String()] = permtest.Grant{}
	access := f.access(t, reportaccess.ModeRead)

	entity, _ := f.catalog.Entity("customer")
	access.Entity(t.Context(), entity)
	require.NotEmpty(t, access.Denials())

	err := access.Refusal(
		t.Context(),
		&analytics.Error{Code: analytics.CodeDenied, Message: "denied"},
	)
	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Contains(t, err.Error(), "Customers")
}

func TestDecisionsRoundTripThroughTheirReasonCodes(t *testing.T) {
	t.Parallel()

	for _, decision := range []reportaccess.Decision{
		reportaccess.Available,
		reportaccess.NeedsKindRead,
		reportaccess.WithheldByCeiling,
		reportaccess.NeedsFieldGroup,
		reportaccess.NeedsExport,
		reportaccess.OwnScopeUnsupported,
	} {
		assert.Equal(t, decision, reportaccess.DecisionOf(decision.String()))
	}
}
