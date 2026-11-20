package managedscope_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tenant() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
}

func contexts(t *testing.T) map[string]context.Context {
	t.Helper()
	return map[string]context.Context{
		"bare":   t.Context(),
		"tenant": dbscope.WithTenant(t.Context(), tenant()),
		"system": dbscope.WithSystem(t.Context(), "a sweep"),
	}
}

func TestADedicatedServerAnswersForTheInstanceWhateverTheContext(t *testing.T) {
	t.Parallel()

	resolver := managedscope.NewResolver(false)
	for name, ctx := range contexts(t) {
		scope, err := resolver.Scope(ctx)
		require.NoError(t, err, name)
		assert.True(t, scope.IsInstance(), name)
		assert.Equal(t, managedscope.InstanceKey, scope.Key(), name)
		_, isTenant := scope.Tenant()
		assert.False(t, isTenant, name)
	}

	scope, err := resolver.ForTenant(tenant())
	require.NoError(t, err)
	assert.True(t, scope.IsInstance(), "a tenant given by hand is the instance too")
}

func TestASharedServerAnswersForTheOrganizationInTheContext(t *testing.T) {
	t.Parallel()

	resolver := managedscope.NewResolver(true)
	of := tenant()
	scope, err := resolver.Scope(dbscope.WithTenant(t.Context(), of))
	require.NoError(t, err)
	assert.False(t, scope.IsInstance())
	assert.Equal(t, of.OrganizationID.String(), scope.Key())
	got, ok := scope.Tenant()
	require.True(t, ok)
	assert.Equal(t, of.OrganizationID, got.OrganizationID)
	assert.Equal(t, of.BusinessUnitID, got.BusinessUnitID)
	assert.True(t, got.UserID.IsNil(), "a scope is an organization, not a person")
}

func TestASharedServerRefusesAContextWithNoTenant(t *testing.T) {
	t.Parallel()

	resolver := managedscope.NewResolver(true)
	for name, ctx := range map[string]context.Context{
		"bare":   t.Context(),
		"system": dbscope.WithSystem(dbscope.WithTenant(t.Context(), tenant()), "a sweep"),
	} {
		_, err := resolver.Scope(ctx)
		require.ErrorIs(t, err, managedscope.ErrNoTenantScope, name)
	}

	_, err := resolver.ForTenant(dbscope.Tenant{})
	require.ErrorIs(t, err, managedscope.ErrNoTenantScope)
}

func TestTwoOrganizationsAreTwoScopes(t *testing.T) {
	t.Parallel()

	resolver := managedscope.NewResolver(true)
	a, err := resolver.ForTenant(tenant())
	require.NoError(t, err)
	b, err := resolver.ForTenant(tenant())
	require.NoError(t, err)
	assert.NotEqual(t, a.Key(), b.Key())
}

func TestAScopeComesBackFromItsStoredKey(t *testing.T) {
	t.Parallel()

	of := tenant()
	scope := managedscope.Organization(of)
	assert.Equal(t, scope, managedscope.FromKey(scope.Key(), of.BusinessUnitID))
	assert.Equal(
		t,
		managedscope.Instance(),
		managedscope.FromKey(managedscope.InstanceKey, pulid.Nil),
	)
}

func TestEachTenancyAcceptsOnlyItsOwnScopes(t *testing.T) {
	t.Parallel()

	dedicated, shared := managedscope.NewResolver(false), managedscope.NewResolver(true)
	org := managedscope.Organization(tenant())

	require.NoError(t, dedicated.Accepts(managedscope.Instance()))
	require.ErrorIs(t, dedicated.Accepts(org), managedscope.ErrWrongTenancy)
	require.NoError(t, shared.Accepts(org))
	require.ErrorIs(t, shared.Accepts(managedscope.Instance()), managedscope.ErrWrongTenancy)
	require.ErrorIs(t, shared.Accepts(managedscope.Scope{}), managedscope.ErrWrongTenancy)
	require.ErrorIs(t, dedicated.Accepts(managedscope.Scope{}), managedscope.ErrWrongTenancy)
}
