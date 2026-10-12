package accountingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAccountingChecker struct {
	status  *accountingsyncports.AccountingSyncStatus
	checked []pulid.ID
	tenant  pagination.TenantInfo
	guard   agenttoolstest.WriteGuard
}

func (f *fakeAccountingChecker) Status(
	_ context.Context,
	tenant pagination.TenantInfo,
	_ integration.Type,
) (*accountingsyncports.AccountingSyncStatus, error) {
	f.tenant = tenant
	return f.status, nil
}

func (f *fakeAccountingChecker) CheckHealth(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingConnection, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.checked = append(f.checked, id)
	f.status.Connection.RecordSuccess(timeutils.NowUnix())
	return f.status.Connection, nil
}

func connectedStatus(status accountingsync.ConnectionStatus) *accountingsyncports.AccountingSyncStatus {
	return &accountingsyncports.AccountingSyncStatus{
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
	params := agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"})

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{checker.status.Connection.ID}, checker.checked)
	assert.Equal(t, params.OrganizationID, checker.tenant.OrgID)
}

func TestCheckAccountingConnection_RefusesAConnectionThatNeedsAPerson(t *testing.T) {
	t.Parallel()

	for _, status := range []*accountingsyncports.AccountingSyncStatus{
		connectedStatus(accountingsync.ConnectionStatusRevoked),
		connectedStatus(accountingsync.ConnectionStatusDisconnected),
		{IntegrationType: integration.TypeQuickBooksOnline, ProviderName: "QuickBooks Online"},
	} {
		checker := &fakeAccountingChecker{status: status}
		tool := newCheckAccountingConnectionTool(checker)
		params := agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"})

		err := tool.(agentports.ToolValidator).Validate(t.Context(), params)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
		require.Error(t, tool.Execute(t.Context(), params))
		assert.Empty(t, checker.checked)
	}
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
	require.Error(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{"system": "NetSuite"})))
	require.Error(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{})))
}
