package accountingconnectionflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const netSuiteRealm = "1234567_SB1"

func newPKCEHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.connector.typ = integration.TypeNetSuite
	require.True(t, accountingprovider.MustProfile(integration.TypeNetSuite).PKCE)
	return h
}

func (h *harness) startAs(t *testing.T, typ integration.Type) string {
	t.Helper()
	_, err := h.svc.auth.StartAuthorization(
		t.Context(),
		&accountingconnectionports.StartAccountingAuthorizationRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: typ,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, h.connector.authorizations)
	return h.connector.authorizations[len(h.connector.authorizations)-1].State
}

func (h *harness) completeAs(
	t *testing.T,
	typ integration.Type,
	state, realm string,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	t.Helper()
	return h.svc.auth.CompleteAuthorization(
		t.Context(),
		&accountingconnectionports.CompleteAccountingAuthorizationRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: typ,
			State:           state,
			Code:            "auth-code",
			RealmID:         realm,
		},
	)
}

func (h *harness) savedState(
	t *testing.T,
	state string,
) *accountingconnectionports.AccountingOAuthState {
	t.Helper()
	h.states.mu.Lock()
	defer h.states.mu.Unlock()
	saved, ok := h.states.states[tokenutils.Hash(state)]
	require.True(t, ok)
	return saved
}

func TestStartAuthorizationSendsNoChallengeWhenTheProviderDoesNotAskForPKCE(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.startAs(t, integration.TypeQuickBooksOnline)

	assert.Empty(t, h.connector.authorizations[0].CodeChallenge)
	assert.Empty(t, h.savedState(t, state).SealedVerifier)

	_, err := h.completeAs(t, integration.TypeQuickBooksOnline, state, testRealm)
	require.NoError(t, err)
	require.Len(t, h.connector.exchanges, 1)
	assert.Empty(t, h.connector.exchanges[0].CodeVerifier)
}

func TestPKCEVerifierIsSealedInTheStateAndMatchesTheChallenge(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	state := h.startAs(t, integration.TypeNetSuite)

	challenge := h.connector.authorizations[0].CodeChallenge
	require.Len(t, challenge, 43)
	saved := h.savedState(t, state)
	require.NotEmpty(t, saved.SealedVerifier)

	_, err := h.completeAs(t, integration.TypeNetSuite, state, netSuiteRealm)
	require.NoError(t, err)
	require.Len(t, h.connector.exchanges, 1)
	verifier := h.connector.exchanges[0].CodeVerifier
	assert.Len(t, verifier, 43)
	assert.Equal(t, challenge, tokenutils.PKCEChallenge(verifier))
	assert.NotContains(t, saved.SealedVerifier, verifier)
}

func TestPKCEVerifiersDifferPerAuthorization(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	h.startAs(t, integration.TypeNetSuite)
	h.startAs(t, integration.TypeNetSuite)

	require.Len(t, h.connector.authorizations, 2)
	assert.NotEqual(t,
		h.connector.authorizations[0].CodeChallenge,
		h.connector.authorizations[1].CodeChallenge,
	)
}

func TestPKCEVerifierIsSingleUse(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	state := h.startAs(t, integration.TypeNetSuite)
	_, err := h.completeAs(t, integration.TypeNetSuite, state, netSuiteRealm)
	require.NoError(t, err)

	_, err = h.completeAs(t, integration.TypeNetSuite, state, netSuiteRealm)
	require.Error(t, err)
	assert.Len(t, h.connector.exchanges, 1)
}

func TestPKCEStateWithoutAVerifierIsRefusedBeforeTheExchange(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	state := h.startAs(t, integration.TypeNetSuite)
	h.savedState(t, state).SealedVerifier = ""

	_, err := h.completeAs(t, integration.TypeNetSuite, state, netSuiteRealm)
	require.Error(t, err)
	assert.Empty(t, h.connector.exchanges)
	assert.Empty(t, h.connections.rows)
}

func TestPKCEVerifierSealedForAnotherStateDoesNotOpen(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	first := h.startAs(t, integration.TypeNetSuite)
	second := h.startAs(t, integration.TypeNetSuite)
	h.savedState(t, second).SealedVerifier = h.savedState(t, first).SealedVerifier

	_, err := h.completeAs(t, integration.TypeNetSuite, second, netSuiteRealm)
	require.Error(t, err)
	assert.Empty(t, h.connector.exchanges)
}

func TestCompleteAuthorizationHandsTheCallbackRealmToTheExchange(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	_, err := h.completeAs(
		t,
		integration.TypeNetSuite,
		h.startAs(t, integration.TypeNetSuite),
		netSuiteRealm,
	)
	require.NoError(t, err)

	require.Len(t, h.connector.exchanges, 1)
	assert.Equal(t, netSuiteRealm, h.connector.exchanges[0].CallbackRealmID)
	assert.Equal(t, "auth-code", h.connector.exchanges[0].Code)
}

func TestRefreshWithoutARefreshTokenKeepsTheStoredOneAndItsExpiry(t *testing.T) {
	t.Parallel()

	for _, typ := range []integration.Type{
		integration.TypeQuickBooksOnline,
		integration.TypeXero,
		integration.TypeBusinessCentral,
		integration.TypeNetSuite,
	} {
		t.Run(string(typ), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			h.connector.typ = typ
			completion, err := h.completeAs(t, typ, h.startAs(t, typ), netSuiteRealm)
			require.NoError(t, err)
			conn := completion.Connection
			before := h.connections.tokens[conn.ID]
			require.NotZero(t, before.refreshExpires)

			h.connector.refreshGrant = &accountingconnectorports.AccountingTokenGrant{
				AccessToken:    "access-2",
				AccessTokenTTL: h.connector.exchangeGrant.AccessTokenTTL,
			}
			h.expireAccessToken(conn)

			_, err = h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
			require.NoError(t, err)
			after := h.connections.tokens[conn.ID]
			assert.Equal(
				t,
				"access-2",
				h.decrypt(t, conn, accountingconnectionkit.AccessTokenField, after.access),
			)
			assert.Equal(
				t,
				"refresh-1",
				h.decrypt(t, conn, accountingconnectionkit.RefreshTokenField, after.refresh),
			)
			assert.Equal(t, before.refreshExpires, after.refreshExpires)
		})
	}
}

func TestRefreshAndRevokeNameTheConnectionsRealm(t *testing.T) {
	t.Parallel()

	h := newPKCEHarness(t)
	completion, err := h.completeAs(
		t,
		integration.TypeNetSuite,
		h.startAs(t, integration.TypeNetSuite),
		netSuiteRealm,
	)
	require.NoError(t, err)
	conn := completion.Connection
	h.expireAccessToken(conn)

	_, err = h.svc.health.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{netSuiteRealm}, h.connector.refreshRealms)

	_, err = h.svc.auth.Disconnect(
		t.Context(),
		&accountingconnectionports.DisconnectAccountingRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeNetSuite,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, h.connector.revokeRealms)
	assert.Equal(t, netSuiteRealm, h.connector.revokeRealms[len(h.connector.revokeRealms)-1])
}
