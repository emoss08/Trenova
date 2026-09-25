package watchtowersources

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func accountingConnection(status accountingsync.ConnectionStatus) *accountingsync.AccountingConnection {
	failed := int64(1_790_000_000)
	return &accountingsync.AccountingConnection{
		ID:                            pulid.MustNew("acctc_"),
		OrganizationID:                pulid.MustNew("org_"),
		BusinessUnitID:                pulid.MustNew("bu_"),
		IntegrationType:               integration.TypeQuickBooksOnline,
		Status:                        status,
		LastFailureAt:                 &failed,
		LastErrorMessage:              "The provider did not answer in time.",
		RefreshTokenAbsoluteExpiresAt: failed + 400*24*60*60,
	}
}

func TestDescribeAccountingConnectionHealth(t *testing.T) {
	t.Parallel()

	cases := map[accountingsync.ConnectionStatus]watchtower.Severity{
		accountingsync.ConnectionStatusDegraded: watchtower.SeverityWarning,
		accountingsync.ConnectionStatusFailing:  watchtower.SeverityCritical,
		accountingsync.ConnectionStatusRevoked:  watchtower.SeverityCritical,
	}
	for status, severity := range cases {
		conn := accountingConnection(status)
		item, open := DescribeAccountingConnectionHealth(conn)
		require.True(t, open, status)
		assert.Equal(t, severity, item.Severity, status)
		assert.Equal(t, watchtower.SourceAccountingSync, item.SourceKind)
		assert.Equal(t, conn.ID.String(), item.SourceID)
		assert.Equal(t, agent.SubjectAccountingConnection, item.SubjectType)
		assert.Equal(t, agent.EventAccountingConnectionDegraded, item.EventKind)
		assert.Equal(t, "/admin/integrations?type=QuickBooksOnline", item.Path)
		assert.Equal(t, *conn.LastFailureAt, item.OccurredAt)
		assert.Contains(t, item.Title, "QuickBooks Online")
	}

	for _, status := range []accountingsync.ConnectionStatus{
		accountingsync.ConnectionStatusConnected,
		accountingsync.ConnectionStatusDisconnected,
	} {
		_, open := DescribeAccountingConnectionHealth(accountingConnection(status))
		assert.False(t, open, status)
	}
}

func TestDescribeAccountingReconnect(t *testing.T) {
	t.Parallel()

	conn := accountingConnection(accountingsync.ConnectionStatusConnected)
	_, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-30*24*60*60)
	assert.False(t, open, "a month out is not yet worth raising")

	item, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-7*24*60*60)
	require.True(t, open)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, AccountingReconnectSourceID(conn), item.SourceID)
	assert.NotEqual(t, AccountingConnectionHealthSourceID(conn), item.SourceID)

	conn.Status = accountingsync.ConnectionStatusRevoked
	_, open = DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-7*24*60*60)
	assert.False(t, open, "a revoked connection is already a health item")
}
