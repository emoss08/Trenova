package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedEvents struct {
	events []scopeEvent
}

func (r *recordedEvents) Report(_ context.Context, event *scopeEvent) {
	r.events = append(r.events, *event)
}

func testGuard(t *testing.T, pool scopePool, enforce, systemFallback bool) (*scopeGuard, *recordedEvents) {
	t.Helper()

	signer, err := newScopeSigner("ktest", make([]byte, minScopeKeyBytes), time.Minute)
	require.NoError(t, err)

	events := &recordedEvents{}

	return newScopeGuard(scopeGuardConfig{
		Pool:           pool,
		Signer:         signer,
		Enforce:        enforce,
		SystemFallback: systemFallback,
		Reporter:       events,
	}), events
}

func testTenant() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
}

func TestScopeGuard_OnBeginSetsTheSignedTenantScope(t *testing.T) {
	t.Parallel()

	guard, _ := testGuard(t, scopePoolTenant, true, false)
	tenant := testTenant()
	ctx := dbscope.WithTenant(t.Context(), tenant)

	statements, err := guard.OnBegin(ctx, keel.TxInfo{})
	require.NoError(t, err)
	require.Len(t, statements, 1)
	assert.Equal(t, setScopeSQL, statements[0].SQL)
	require.Len(t, statements[0].Args, 1)

	token, ok := statements[0].Args[0].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(token, "v1.ktest."+tenant.OrganizationID.String()+"."))
}

func TestScopeGuard_OnBeginCarriesTheReadScope(t *testing.T) {
	t.Parallel()

	guard, _ := testGuard(t, scopePoolTenant, true, false)
	manager := pulid.MustNew("usr_")
	ctx := readscope.With(
		dbscope.WithTenant(t.Context(), testTenant()),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{
				"worker": {Scope: readscope.ScopeTeam, Team: []pulid.ID{manager}},
			}, nil
		},
	)

	statements, err := guard.OnBegin(ctx, keel.TxInfo{ReadOnly: true})
	require.NoError(t, err)
	require.Len(t, statements, 1)
	assert.Equal(t, setScopeAndReadScope, statements[0].SQL)
	assert.Equal(t, "worker:team:"+manager.String(), statements[0].Args[1])
}

func TestScopeGuard_EnforceRefusesAnUnscopedTransaction(t *testing.T) {
	t.Parallel()

	guard, events := testGuard(t, scopePoolTenant, true, false)

	statements, err := guard.OnBegin(t.Context(), keel.TxInfo{Implicit: true})
	require.ErrorIs(t, err, ErrNoTenantScope)
	assert.Empty(t, statements)
	require.Len(t, events.events, 1)
	assert.Equal(t, scopeEventUnscopedTx, events.events[0].Event)
	assert.Equal(t, scopeOutcomeRefused, events.events[0].Outcome)
}

func TestScopeGuard_ObserveAllowsAnUnscopedTransaction(t *testing.T) {
	t.Parallel()

	guard, events := testGuard(t, scopePoolTenant, false, false)

	statements, err := guard.OnBegin(t.Context(), keel.TxInfo{})
	require.NoError(t, err)
	assert.Empty(t, statements)
	require.Len(t, events.events, 1)
	assert.Equal(t, scopeOutcomeObserved, events.events[0].Outcome)
}

func TestScopeGuard_SystemPoolServesOnlySystemScopes(t *testing.T) {
	t.Parallel()

	guard, _ := testGuard(t, scopePoolSystem, true, false)

	statements, err := guard.OnBegin(dbscope.WithSystem(t.Context(), "sweep"), keel.TxInfo{})
	require.NoError(t, err)
	assert.Empty(t, statements, "the system pool's role bypasses row-level security, so no scope is set")

	_, err = guard.OnBegin(dbscope.WithTenant(t.Context(), testTenant()), keel.TxInfo{})
	require.ErrorIs(t, err, ErrSystemScopeRequired)

	_, err = guard.OnBegin(dbscope.WithSystem(t.Context(), " "), keel.TxInfo{})
	require.ErrorIs(t, err, ErrSystemScopeReason)
}

func TestScopeGuard_TenantPoolServesSystemScopesOnlyAsAFallback(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "sweep")

	fallback, _ := testGuard(t, scopePoolTenant, false, true)
	_, err := fallback.OnBegin(ctx, keel.TxInfo{})
	require.NoError(t, err)

	strict, events := testGuard(t, scopePoolTenant, true, true)
	_, err = strict.OnBegin(ctx, keel.TxInfo{})
	require.ErrorIs(t, err, ErrNoTenantScope)
	assert.Equal(t, scopeEventWrongPool, events.events[0].Event)
}
