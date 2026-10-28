package watchtowersources

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func accountingConnection(
	status accountingconnection.ConnectionStatus,
) *accountingconnection.AccountingConnection {
	failed := int64(1_790_000_000)
	return &accountingconnection.AccountingConnection{
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

	cases := map[accountingconnection.ConnectionStatus]watchtower.Severity{
		accountingconnection.ConnectionStatusDegraded: watchtower.SeverityWarning,
		accountingconnection.ConnectionStatusFailing:  watchtower.SeverityCritical,
		accountingconnection.ConnectionStatusRevoked:  watchtower.SeverityCritical,
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

	for _, status := range []accountingconnection.ConnectionStatus{
		accountingconnection.ConnectionStatusConnected,
		accountingconnection.ConnectionStatusDisconnected,
	} {
		_, open := DescribeAccountingConnectionHealth(accountingConnection(status))
		assert.False(t, open, status)
	}
}

func TestDescribeAccountingReconnect(t *testing.T) {
	t.Parallel()

	conn := accountingConnection(accountingconnection.ConnectionStatusConnected)
	_, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-30*24*60*60)
	assert.False(t, open, "a month out is not yet worth raising")

	item, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-7*24*60*60)
	require.True(t, open)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, AccountingReconnectSourceID(conn), item.SourceID)
	assert.NotEqual(t, AccountingConnectionHealthSourceID(conn), item.SourceID)

	conn.Status = accountingconnection.ConnectionStatusRevoked
	_, open = DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-7*24*60*60)
	assert.False(t, open, "a revoked connection is already a health item")
}

func TestDescribeAccountingReconnectNamesTheExpiryDate(t *testing.T) {
	t.Parallel()

	conn := accountingConnection(accountingconnection.ConnectionStatusConnected)
	conn.RefreshTokenAbsoluteExpiresAt = time.Date(2026, time.October, 20, 15, 0, 0, 0, time.UTC).
		Unix()

	item, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-24*60*60)
	require.True(t, open)
	assert.Contains(t, item.Title, "Reconnect")
	assert.Contains(t, item.Summary, "October 20, 2026")
	assert.NotContains(t, item.Summary, "five years")
}

func TestDescribeAccountingReconnectAsksForANewCertificate(t *testing.T) {
	t.Parallel()

	conn := accountingConnection(accountingconnection.ConnectionStatusConnected)
	conn.IntegrationType = integration.TypeNetSuite
	conn.UseCertificate(accountingconnection.ClientCertificate{
		KeyCiphertext: "sealed",
		CertificateID: "cert-1",
		ExpiresAt:     time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC).Unix(),
	})

	_, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-20*24*60*60)
	assert.False(t, open, "a certificate warns 14 days ahead")

	item, open := DescribeAccountingReconnect(conn, conn.RefreshTokenAbsoluteExpiresAt-10*24*60*60)
	require.True(t, open)
	assert.Equal(t, "Renew the NetSuite certificate before it expires", item.Title)
	assert.Contains(t, item.Summary, "March 1, 2028")
	assert.Equal(t, conn.RefreshTokenAbsoluteExpiresAt-14*24*60*60, item.OccurredAt)
}

func TestDescribeRevokedCertificateConnectionAsksForTheMapping(t *testing.T) {
	t.Parallel()

	conn := accountingConnection(accountingconnection.ConnectionStatusRevoked)
	conn.IntegrationType = integration.TypeNetSuite
	conn.AuthMethod = accountingconnection.AuthMethodClientCredentials

	item, open := DescribeAccountingConnectionHealth(conn)
	require.True(t, open)
	assert.Contains(t, item.Summary, "certificate mapping")
}
