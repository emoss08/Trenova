package dbscope

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func testTenant() Tenant {
	return Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
}

func TestFrom_EmptyContextHasNoScope(t *testing.T) {
	t.Parallel()

	scope := From(t.Context())
	assert.Equal(t, KindNone, scope.Kind())
	_, ok := scope.Tenant()
	assert.False(t, ok)
	assert.False(t, IsSystem(t.Context()))
}

func TestWithTenant_RoundTrips(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	ctx := WithTenant(t.Context(), tenant)

	got, ok := TenantFrom(ctx)
	assert.True(t, ok)
	assert.Equal(t, tenant, got)
	assert.Equal(t, KindTenant, From(ctx).Kind())
	assert.False(t, IsSystem(ctx))
}

func TestWithSystem_ReplacesTenantScope(t *testing.T) {
	t.Parallel()

	ctx := WithSystem(WithTenant(t.Context(), testTenant()), "  nightly sweep  ")

	assert.True(t, IsSystem(ctx))
	assert.Equal(t, "nightly sweep", From(ctx).Reason())
	_, ok := TenantFrom(ctx)
	assert.False(t, ok)
}

func TestWithTenant_ReplacesSystemScope(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	ctx := WithTenant(WithSystem(t.Context(), "fan out"), tenant)

	got, ok := TenantFrom(ctx)
	assert.True(t, ok)
	assert.Equal(t, tenant, got)
	assert.False(t, IsSystem(ctx))
}

func TestTenant_Valid(t *testing.T) {
	t.Parallel()

	assert.True(t, testTenant().Valid())
	assert.True(t, Tenant{OrganizationID: pulid.MustNew("org_"), BusinessUnitID: pulid.MustNew("bu_")}.Valid())
	assert.False(t, Tenant{OrganizationID: pulid.MustNew("org_")}.Valid())
	assert.False(t, Tenant{BusinessUnitID: pulid.MustNew("bu_")}.Valid())
}

func TestScope_Matches(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	assert.True(t, From(WithTenant(t.Context(), tenant)).Matches(From(WithTenant(t.Context(), tenant))))
	assert.False(t, From(WithTenant(t.Context(), tenant)).Matches(From(WithTenant(t.Context(), testTenant()))))
	assert.False(t, From(WithTenant(t.Context(), tenant)).Matches(From(WithSystem(t.Context(), "x"))))
	assert.True(t, From(WithSystem(t.Context(), "x")).Matches(From(WithSystem(t.Context(), "y"))))
}

func TestWithValidTenant_BindsOnlyACompleteTenant(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	got, ok := TenantFrom(WithValidTenant(t.Context(), tenant))
	assert.True(t, ok)
	assert.Equal(t, tenant, got)

	system := WithSystem(t.Context(), "test")
	kept := WithValidTenant(system, Tenant{OrganizationID: tenant.OrganizationID})
	assert.True(t, IsSystem(kept), "an incomplete tenant leaves the caller's scope alone")
}

func TestEnsureTenant(t *testing.T) {
	t.Parallel()

	tenant := testTenant()

	t.Run("keeps a scope that already covers the tenant", func(t *testing.T) {
		t.Parallel()

		ctx := WithTenant(t.Context(), tenant)
		ensured := EnsureTenant(ctx, Tenant{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		})
		got, ok := TenantFrom(ensured)
		assert.True(t, ok)
		assert.Equal(t, tenant, got, "the user on the covering scope must survive")
	})

	t.Run("binds the tenant when the scope names another", func(t *testing.T) {
		t.Parallel()

		other := testTenant()
		ctx := WithTenant(t.Context(), other)
		ensured := EnsureTenant(ctx, tenant)
		got, ok := TenantFrom(ensured)
		assert.True(t, ok)
		assert.Equal(t, tenant, got)
	})

	t.Run("binds the tenant on an unscoped context", func(t *testing.T) {
		t.Parallel()

		got, ok := TenantFrom(EnsureTenant(t.Context(), tenant))
		assert.True(t, ok)
		assert.Equal(t, tenant, got)
	})

	t.Run("keeps a system scope", func(t *testing.T) {
		t.Parallel()

		ctx := WithSystem(t.Context(), "test")
		assert.True(t, IsSystem(EnsureTenant(ctx, tenant)))
	})

	t.Run("ignores an invalid tenant", func(t *testing.T) {
		t.Parallel()

		ensured := EnsureTenant(t.Context(), Tenant{})
		assert.Equal(t, KindNone, From(ensured).Kind())
	})
}
