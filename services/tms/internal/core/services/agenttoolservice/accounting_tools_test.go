package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAccountingChecker struct {
	status  *serviceports.AccountingSyncStatus
	checked []pulid.ID
	tenant  pagination.TenantInfo
}

func (f *fakeAccountingChecker) Status(
	_ context.Context,
	tenant pagination.TenantInfo,
	_ integration.Type,
) (*serviceports.AccountingSyncStatus, error) {
	f.tenant = tenant
	return f.status, nil
}

func (f *fakeAccountingChecker) CheckHealth(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingConnection, error) {
	f.checked = append(f.checked, id)
	return f.status.Connection, nil
}

func connectedStatus(status accountingsync.ConnectionStatus) *serviceports.AccountingSyncStatus {
	return &serviceports.AccountingSyncStatus{
		IntegrationType: integration.TypeQuickBooksOnline,
		ProviderName:    "QuickBooks Online",
		Available:       true,
		Connection: &accountingsync.AccountingConnection{
			ID:                  pulid.MustNew("acctc_"),
			IntegrationType:     integration.TypeQuickBooksOnline,
			Status:              status,
			ExternalCompanyName: "Acme Freight",
		},
	}
}

func TestCheckAccountingConnection_ChecksTheTenantsConnection(t *testing.T) {
	t.Parallel()

	checker := &fakeAccountingChecker{status: connectedStatus(accountingsync.ConnectionStatusDegraded)}
	tool := newCheckAccountingConnectionTool(checker)
	params := executeParams(map[string]any{"system": "QuickBooksOnline"})

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{checker.status.Connection.ID}, checker.checked)
	assert.Equal(t, params.OrganizationID, checker.tenant.OrgID)
}

func TestCheckAccountingConnection_RefusesAConnectionThatNeedsAPerson(t *testing.T) {
	t.Parallel()

	for _, status := range []*serviceports.AccountingSyncStatus{
		connectedStatus(accountingsync.ConnectionStatusRevoked),
		connectedStatus(accountingsync.ConnectionStatusDisconnected),
		{IntegrationType: integration.TypeQuickBooksOnline, ProviderName: "QuickBooks Online"},
	} {
		checker := &fakeAccountingChecker{status: status}
		tool := newCheckAccountingConnectionTool(checker)
		params := executeParams(map[string]any{"system": "QuickBooksOnline"})

		err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
		require.Error(t, tool.Execute(t.Context(), params))
		assert.Empty(t, checker.checked)
	}
}

func TestCheckAccountingConnection_SimulatesWithoutCalling(t *testing.T) {
	t.Parallel()

	checker := &fakeAccountingChecker{status: connectedStatus(accountingsync.ConnectionStatusConnected)}
	tool := newCheckAccountingConnectionTool(checker)

	sim, err := tool.(serviceports.ToolSimulator).Simulate(
		t.Context(),
		executeParams(map[string]any{"system": "QuickBooksOnline"}),
	)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "Acme Freight")
	assert.True(t, sim.Previewed)
	require.Len(t, sim.Changes, 1)
	assert.Equal(t, "never", sim.Changes[0].From)
	assert.Empty(t, checker.checked)
}

func TestCheckAccountingConnection_PolicyStaysInside(t *testing.T) {
	t.Parallel()

	policy := newCheckAccountingConnectionTool(&fakeAccountingChecker{}).Policy()
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.Equal(t, agent.TierAutoExecute, policy.DefaultTier)
	assert.False(t, policy.Reversible)
	assert.Equal(t, agent.ExternalReadNever, policy.ReadsExternal)
}

func TestCheckAccountingConnection_RejectsUnknownSystems(t *testing.T) {
	t.Parallel()

	tool := newCheckAccountingConnectionTool(&fakeAccountingChecker{
		status: connectedStatus(accountingsync.ConnectionStatusConnected),
	})
	require.Error(t, tool.Execute(t.Context(), executeParams(map[string]any{"system": "Xero"})))
	require.Error(t, tool.Execute(t.Context(), executeParams(map[string]any{})))
}
