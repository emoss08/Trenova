package accountingconnectionflows

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCertificateID = "Zm9vYmFyLWNlcnRpZmljYXRl"

func newCertificateHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.connector.typ = integration.TypeNetSuite
	require.True(t, accountingprovider.MustProfile(integration.TypeNetSuite).ClientCredentials)
	return h
}

func (h *harness) prepareCertificate(
	t *testing.T,
) *accountingconnectionports.AccountingPreparedCertificate {
	t.Helper()
	prepared, err := h.svc.certs.PrepareCertificate(
		t.Context(),
		&accountingconnectionports.PrepareAccountingCertificateRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	return prepared
}

func (h *harness) connectWithCertificate(
	t *testing.T,
	accountID, certificateID string,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	t.Helper()
	return h.svc.certs.ConnectWithCertificate(
		t.Context(),
		&accountingconnectionports.ConnectAccountingWithCertificateRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeNetSuite,
			AccountID:       accountID,
			CertificateID:   certificateID,
		},
	)
}

func (h *harness) certificateConnection(t *testing.T) *accountingconnection.AccountingConnection {
	t.Helper()
	h.prepareCertificate(t)
	completion, err := h.connectWithCertificate(t, netSuiteRealm, testCertificateID)
	require.NoError(t, err)
	require.NotNil(t, completion.Connection)
	return completion.Connection
}

func (h *harness) renewCertificate(
	t *testing.T,
	conn *accountingconnection.AccountingConnection,
	certificateID string,
) (*accountingconnection.AccountingConnection, error) {
	t.Helper()
	return h.svc.certs.RenewCertificate(
		t.Context(),
		&accountingconnectionports.RenewAccountingCertificateRequest{
			TenantInfo:    h.tenant,
			UserID:        h.userID,
			ConnectionID:  conn.ID,
			CertificateID: certificateID,
		},
	)
}

func TestPrepareCertificateIsRefusedForAProviderWithoutCertificates(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.certs.PrepareCertificate(
		t.Context(),
		&accountingconnectionports.PrepareAccountingCertificateRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeQuickBooksOnline,
		},
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Zero(t, h.certificates.count())
}

func TestPrepareCertificateSealsTheKeyAndReturnsTheCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	prepared := h.prepareCertificate(t)

	assert.Equal(t, testCertificatePEM, prepared.CertificatePEM)
	assert.Equal(t, h.connector.clientCredentials().expiresAt.Unix(), prepared.ExpiresAt)
	assert.Equal(t, []string{"Trenova"}, h.connector.clientCredentials().subjects)

	pending, err := h.certificates.GetByType(
		t.Context(),
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      h.tenant,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	assert.NotContains(t, pending.PrivateKeyCiphertext, "PRIVATE KEY")
	key, err := h.encryption.DecryptStringWithAAD(
		pending.PrivateKeyCiphertext,
		accountingconnectionkit.PendingKeyAAD(pending),
	)
	require.NoError(t, err)
	assert.Equal(t, h.connector.clientCredentials().preparedKeys[0], key)
	assert.Equal(t, h.userID, pending.PreparedByID)
}

func TestPreparingAgainReplacesThePendingCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	h.prepareCertificate(t)
	first, err := h.certificates.GetByType(
		t.Context(),
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      h.tenant,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	h.prepareCertificate(t)

	assert.Equal(t, 1, h.certificates.count())
	second, err := h.certificates.GetByType(
		t.Context(),
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      h.tenant,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestConnectWithCertificateValidatesTheAccountAndCertificateIDs(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	h.prepareCertificate(t)

	_, err := h.connectWithCertificate(t, "not an account", "")
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"accountId", "certificateId"}, fields)
	assert.Empty(t, h.connector.clientCredentials().assertions)
}

func TestConnectWithCertificateNeedsAPreparedCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	_, err := h.connectWithCertificate(t, netSuiteRealm, testCertificateID)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.connector.clientCredentials().assertions)
	assert.Empty(t, h.connections.rows)
}

func TestConnectWithCertificateStoresTheKeyAndNoRefreshToken(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	h.prepareCertificate(t)
	completion, err := h.connectWithCertificate(t, "1234567_sb1", testCertificateID)
	require.NoError(t, err)
	conn := completion.Connection
	require.NotNil(t, conn)

	cc := h.connector.clientCredentials()
	require.Len(t, cc.assertions, 1)
	assert.Equal(t, netSuiteRealm, cc.assertions[0].realmID)
	assert.Equal(t, testCertificateID, cc.assertions[0].certificateID)
	assert.Equal(t, cc.preparedKeys[0], cc.assertions[0].privateKey)

	assert.True(t, conn.UsesCertificate())
	assert.Equal(t, netSuiteRealm, conn.ExternalRealmID)
	assert.Equal(t, testCertificateID, conn.ClientCertificateID)
	require.NotNil(t, conn.ClientCertificateExpiresAt)
	assert.Equal(t, cc.expiresAt.Unix(), *conn.ClientCertificateExpiresAt)
	assert.Equal(t, cc.expiresAt.Unix(), conn.RefreshTokenAbsoluteExpiresAt)

	stored := h.connections.tokens[conn.ID]
	assert.Empty(t, stored.refresh)
	assert.Equal(
		t,
		"cert-access",
		h.decrypt(t, conn, accountingconnectionkit.AccessTokenField, stored.access),
	)
	assert.Equal(
		t,
		cc.preparedKeys[0],
		h.decrypt(t, conn, accountingconnectionkit.ClientKeyField, stored.clientKey),
	)
	assert.Zero(t, h.certificates.count(), "the pending certificate is consumed by the connection")
	assert.Empty(
		t,
		h.connector.exchanges,
		"a certificate connection never uses the sign-in exchange",
	)
}

func TestConnectWithCertificateRefusedByTheProviderKeepsThePendingCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	h.prepareCertificate(t)
	h.connector.clientCredentials().assertionErrs = []error{errProvider}

	_, err := h.connectWithCertificate(t, netSuiteRealm, testCertificateID)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, 1, h.certificates.count())
	assert.Empty(t, h.connections.rows)
}

func TestConnectWithCertificateOffersTheSubsidiariesAndKeepsTheKeyOutOfTheChoice(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	h.connector.companies = twoCompanies
	h.prepareCertificate(t)

	completion, err := h.connectWithCertificate(t, netSuiteRealm, testCertificateID)
	require.NoError(t, err)
	require.Nil(t, completion.Connection)
	require.Len(t, completion.Companies, 2)

	h.states.mu.Lock()
	for _, state := range h.states.states {
		assert.NotContains(t, state.SealedGrant, "PRIVATE KEY")
	}
	h.states.mu.Unlock()

	conn, err := h.svc.auth.ChooseCompany(
		t.Context(),
		&accountingconnectionports.ChooseAccountingCompanyRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeNetSuite,
			ChoiceToken:     completion.ChoiceToken,
			CompanyID:       "2222",
		},
	)
	require.NoError(t, err)
	assert.True(t, conn.UsesCertificate())
	assert.Equal(t, "2222", conn.ExternalRealmID)
	assert.Equal(t, testCertificateID, conn.ClientCertificateID)
	stored := h.connections.tokens[conn.ID]
	assert.Equal(
		t,
		h.connector.clientCredentials().preparedKeys[0],
		h.decrypt(t, conn, accountingconnectionkit.ClientKeyField, stored.clientKey),
	)
	assert.Zero(t, h.certificates.count())
}

func TestCertificateConnectionTakesATokenPerAssertionAndNeverRefreshes(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	h.connector.clientCredentials().assertionGrant = &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    "cert-access-2",
		AccessTokenTTL: time.Hour,
	}
	h.expireAccessToken(conn)

	checked, err := h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, accountingconnection.ConnectionStatusConnected, checked.Status)
	assert.Empty(t, h.connector.refreshedWith)
	cc := h.connector.clientCredentials()
	require.Len(t, cc.assertions, 2)
	assert.Equal(t, netSuiteRealm, cc.assertions[1].realmID)
	assert.Equal(t, cc.preparedKeys[0], cc.assertions[1].privateKey)
	stored := h.connections.tokens[conn.ID]
	assert.Equal(
		t,
		"cert-access-2",
		h.decrypt(t, conn, accountingconnectionkit.AccessTokenField, stored.access),
	)
	assert.Empty(t, stored.refresh)
	assert.Equal(t, "cert-access-2", h.connector.factsCalls[len(h.connector.factsCalls)-1])
}

func TestCertificateConnectionReusesAFreshAccessToken(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)

	_, err := h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Len(t, h.connector.clientCredentials().assertions, 1)
}

func TestCertificateConnectionGoesRevokedWhenTheCertificateIsRefused(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	h.expireAccessToken(conn)
	h.connector.clientCredentials().assertionErrs = []error{errProvider}
	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryRevoked

	checked, err := h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, accountingconnection.ConnectionStatusRevoked, checked.Status)
	stored := h.connections.tokens[conn.ID]
	assert.Empty(t, stored.access)
	assert.Empty(t, stored.clientKey, "a refused key is not kept")
	assert.Empty(t, h.connector.refreshedWith)
}

func TestRenewCertificateReplacesTheKeyAndCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	cc := h.connector.clientCredentials()
	cc.expiresAt = cc.expiresAt.Add(30 * 24 * time.Hour)
	h.prepareCertificate(t)

	renewed, err := h.renewCertificate(t, conn, "bmV3LWNlcnRpZmljYXRl")
	require.NoError(t, err)
	assert.Equal(t, "bmV3LWNlcnRpZmljYXRl", renewed.ClientCertificateID)
	assert.Equal(t, cc.expiresAt.Unix(), renewed.RefreshTokenAbsoluteExpiresAt)
	require.Len(t, cc.assertions, 2)
	assert.Equal(t, "bmV3LWNlcnRpZmljYXRl", cc.assertions[1].certificateID)
	assert.Equal(t, cc.preparedKeys[1], cc.assertions[1].privateKey)

	stored := h.connections.tokens[conn.ID]
	assert.Equal(
		t,
		cc.preparedKeys[1],
		h.decrypt(t, conn, accountingconnectionkit.ClientKeyField, stored.clientKey),
	)
	assert.Zero(t, h.certificates.count())
	assert.NotEmpty(t, h.audit.entries)
	assert.True(t, h.audit.entries[len(h.audit.entries)-1].Critical)
}

func TestRenewCertificateRestoresARevokedConnection(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	h.expireAccessToken(conn)
	h.connector.clientCredentials().assertionErrs = []error{errProvider}
	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryRevoked
	checked, err := h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	require.Equal(t, accountingconnection.ConnectionStatusRevoked, checked.Status)

	h.prepareCertificate(t)
	renewed, err := h.renewCertificate(t, conn, testCertificateID)
	require.NoError(t, err)
	assert.Equal(t, accountingconnection.ConnectionStatusConnected, renewed.Status)
	stored := h.connections.tokens[conn.ID]
	assert.NotEmpty(t, stored.clientKey)
	assert.NotEmpty(t, stored.access)
}

func TestRenewCertificateRefusedByTheProviderLeavesTheConnectionAlone(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	before := h.connections.tokens[conn.ID]
	h.prepareCertificate(t)
	h.connector.clientCredentials().assertionErrs = []error{errProvider}

	_, err := h.renewCertificate(t, conn, "bmV3LWNlcnRpZmljYXRl")
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, before, h.connections.tokens[conn.ID])
	assert.Equal(t, testCertificateID, h.connections.rows[conn.ID].ClientCertificateID)
	assert.Equal(t, 1, h.certificates.count())
}

func TestRenewCertificateIsRefusedForASignInConnection(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	completion, err := h.completeAs(
		t,
		integration.TypeNetSuite,
		h.startAs(t, integration.TypeNetSuite),
		netSuiteRealm,
	)
	require.NoError(t, err)
	h.prepareCertificate(t)

	_, err = h.renewCertificate(t, completion.Connection, testCertificateID)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.connector.clientCredentials().assertions)
}

func TestSigningInAfterACertificateConnectionDropsTheCertificate(t *testing.T) {
	t.Parallel()

	h := newCertificateHarness(t)
	conn := h.certificateConnection(t)
	_, err := h.svc.auth.Disconnect(
		t.Context(),
		&accountingconnectionports.DisconnectAccountingRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, h.connections.tokens[conn.ID].clientKey)
	assert.Empty(
		t,
		h.connector.revoked,
		"a certificate connection holds no refresh token to revoke",
	)

	completion, err := h.completeAs(
		t,
		integration.TypeNetSuite,
		h.startAs(t, integration.TypeNetSuite),
		netSuiteRealm,
	)
	require.NoError(t, err)
	reconnected := completion.Connection
	assert.Equal(t, conn.ID, reconnected.ID)
	assert.False(t, reconnected.UsesCertificate())
	assert.Empty(t, reconnected.ClientCertificateID)
	assert.Nil(t, reconnected.ClientCertificateExpiresAt)
	assert.Equal(
		t,
		"refresh-1",
		h.decrypt(
			t,
			reconnected,
			accountingconnectionkit.RefreshTokenField,
			h.connections.tokens[conn.ID].refresh,
		),
	)
}
