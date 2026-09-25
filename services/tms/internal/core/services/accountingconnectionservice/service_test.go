package accountingconnectionservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testRealm = "9341452431742015"

type harness struct {
	svc          *Service
	connections  *fakeConnections
	apps         *fakeApps
	states       *fakeStates
	integrations *fakeIntegrations
	connector    *fakeConnector
	audit        *fakeAudit
	watchtower   *fakeWatchtower
	refresher    *fakeRefresher
	events       *agenteventstest.Recorder
	encryption   *encryptionservice.Service
	tenant       pagination.TenantInfo
	userID       pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		connections:  newFakeConnections(),
		apps:         newFakeApps(),
		states:       newFakeStates(),
		integrations: newFakeIntegrations(),
		connector:    newFakeConnector(),
		audit:        &fakeAudit{},
		watchtower:   newFakeWatchtower(),
		refresher:    &fakeRefresher{},
		events:       &agenteventstest.Recorder{},
		encryption: encryptionservice.New(encryptionservice.Params{Config: &config.Config{
			Security: config.SecurityConfig{Encryption: config.EncryptionConfig{
				Key: "unit-test-encryption-key-with-at-least-32-bytes",
			}},
		}}),
		tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		userID: pulid.MustNew("usr_"),
	}
	h.svc = New(Params{
		Logger:       zap.NewNop(),
		DB:           dbtest.NopConnection{},
		Connections:  h.connections,
		Apps:         h.apps,
		States:       h.states,
		Integrations: h.integrations,
		Connectors:   fakeRegistry{connector: h.connector},
		Encryption:   h.encryption,
		AuditService: h.audit,
		Watchtower:   h.watchtower,
		Publisher:    h.events,
		Refresher:    h.refresher,
	})
	return h
}

func (h *harness) start(t *testing.T) string {
	t.Helper()
	started, err := h.svc.StartAuthorization(t.Context(), &services.StartAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	state := strings.TrimPrefix(started.AuthorizeURL, "https://appcenter.intuit.com/connect/oauth2?state=")
	require.NotEmpty(t, state)
	return state
}

func (h *harness) complete(t *testing.T, state string) (*accountingsync.AccountingConnection, error) {
	t.Helper()
	return h.svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		State:           state,
		Code:            "auth-code",
		RealmID:         testRealm,
	})
}

func (h *harness) connect(t *testing.T) *accountingsync.AccountingConnection {
	t.Helper()
	conn, err := h.complete(t, h.start(t))
	require.NoError(t, err)
	return conn
}

func (h *harness) decrypt(t *testing.T, conn *accountingsync.AccountingConnection, field, ciphertext string) string {
	t.Helper()
	value, err := h.encryption.DecryptStringWithAAD(ciphertext, h.svc.aad(conn, field))
	require.NoError(t, err)
	return value
}

func (h *harness) expireAccessToken(conn *accountingsync.AccountingConnection) {
	h.connections.mu.Lock()
	defer h.connections.mu.Unlock()
	tokens := h.connections.tokens[conn.ID]
	tokens.accessExpires = timeutils.NowUnix() + 30
	h.connections.tokens[conn.ID] = tokens
}

func (h *harness) markDue(conn *accountingsync.AccountingConnection) {
	h.connections.mu.Lock()
	defer h.connections.mu.Unlock()
	h.connections.rows[conn.ID].LastCheckedAt = nil
}

func TestStartAuthorizationRefusesWhenTheAppIsNotConfigured(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.available = false
	_, err := h.svc.StartAuthorization(t.Context(), &services.StartAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.states.states)
}

func TestStartAuthorizationStoresOnlyTheStateHash(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.start(t)

	_, rawStored := h.states.states[state]
	assert.False(t, rawStored, "the raw state must never be the storage key")
	stored, ok := h.states.states[tokenutils.Hash(state)]
	require.True(t, ok)
	assert.Equal(t, h.userID, stored.UserID)
	assert.Equal(t, h.tenant.OrgID, stored.OrganizationID)
	assert.Equal(t, 10*time.Minute, h.states.ttls[tokenutils.Hash(state)])
}

func TestStartAuthorizationRejectsNonAccountingTypes(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.StartAuthorization(t.Context(), &services.StartAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeSamsara,
	})
	require.Error(t, err)
}

func TestCompleteAuthorizationConnectsWithEncryptedTokens(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)

	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.Equal(t, "Acme Freight", conn.ExternalCompanyName)
	assert.Equal(t, testRealm, conn.ExternalRealmID)
	assert.Equal(t, h.userID, conn.ConnectedByID)

	stored := h.connections.tokens[conn.ID]
	assert.NotEqual(t, "access-1", stored.access)
	assert.NotEqual(t, "refresh-1", stored.refresh)
	assert.Equal(t, "access-1", h.decrypt(t, conn, accessTokenField, stored.access))
	assert.Equal(t, "refresh-1", h.decrypt(t, conn, refreshTokenField, stored.refresh))

	other := *conn
	other.ID = pulid.MustNew("acctc_")
	_, err := h.encryption.DecryptStringWithAAD(stored.access, h.svc.aad(&other, accessTokenField))
	require.Error(t, err, "a token must not decrypt under another connection's binding")

	assert.True(t, h.integrations.enabled(h.tenant))
	require.Len(t, h.audit.entries, 1)
	serialized := jsonutils.MustToJSONString(h.audit.entries[0].CurrentState)
	assert.NotContains(t, serialized, stored.access)
	assert.NotContains(t, serialized, "refresh-1")
	assert.True(t, h.audit.entries[0].Critical)
	assert.Empty(t, h.events.Published())
	assert.Equal(t, []pulid.ID{conn.ID}, h.refresher.requested, "connecting starts the reference data refresh")
}

func TestCompleteAuthorizationSurvivesARefreshThatCannotStart(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.refresher.err = errors.New("temporal unavailable")
	conn := h.connect(t)

	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.Equal(t, []pulid.ID{conn.ID}, h.refresher.requested)
}

func TestCompleteAuthorizationIsSingleUse(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.start(t)
	_, err := h.complete(t, state)
	require.NoError(t, err)

	_, err = h.complete(t, state)
	require.Error(t, err)
	assert.Len(t, h.connector.exchangedCodes, 1)
}

func TestCompleteAuthorizationRefusesAnotherUsersState(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.start(t)
	_, err := h.svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          pulid.MustNew("usr_"),
		IntegrationType: integration.TypeQuickBooksOnline,
		State:           state,
		Code:            "auth-code",
		RealmID:         testRealm,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, h.connector.exchangedCodes)
	assert.Empty(t, h.connections.rows)
}

func TestCompleteAuthorizationRefusesAnotherTenantsState(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.start(t)
	_, err := h.svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: h.tenant.BuID},
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		State:           state,
		Code:            "auth-code",
		RealmID:         testRealm,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, h.connector.exchangedCodes)
}

func TestCompleteAuthorizationValidatesInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		State:           "",
		Code:            "",
		RealmID:         "../etc",
	})
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Len(t, multiErr.Errors, 3)
}

func TestCompleteAuthorizationRefusesARealmHeldByAnotherTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connections.rows[pulid.MustNew("acctc_")] = &accountingsync.AccountingConnection{
		ID:              pulid.MustNew("acctc_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		IntegrationType: integration.TypeQuickBooksOnline,
		Status:          accountingsync.ConnectionStatusConnected,
		ExternalRealmID: testRealm,
	}

	_, err := h.complete(t, h.start(t))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, []string{"refresh-1"}, h.connector.revoked)
	assert.Len(t, h.connections.rows, 1)
	assert.False(t, h.integrations.enabled(h.tenant))
}

func TestCompleteAuthorizationRevokesWhenCompanyFactsFail(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.factsErrs = []error{errProvider}
	_, err := h.complete(t, h.start(t))
	require.Error(t, err)
	assert.Equal(t, []string{"refresh-1"}, h.connector.revoked)
	assert.Empty(t, h.connections.rows)
}

func TestReconnectAfterRevocationReusesTheRow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connections.rows[conn.ID].Status = accountingsync.ConnectionStatusRevoked
	h.connector.exchangeGrant = h.connector.refreshGrant

	again, err := h.complete(t, h.start(t))
	require.NoError(t, err)
	assert.Equal(t, conn.ID, again.ID)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, h.connections.rows[conn.ID].Status)
	assert.Equal(t, "access-2", h.decrypt(t, again, accessTokenField, h.connections.tokens[conn.ID].access))
	assert.Len(t, h.connections.rows, 1)
}

func TestConnectingADifferentCompanyRequiresDisconnectingFirst(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connect(t)

	_, err := h.svc.CompleteAuthorization(t.Context(), &services.CompleteAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		State:           h.start(t),
		Code:            "auth-code",
		RealmID:         "1111",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestDisconnectWipesTokensRevokesAndIsIdempotent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)

	disconnected, err := h.svc.Disconnect(t.Context(), &services.DisconnectAccountingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.ConnectionStatusDisconnected, disconnected.Status)
	assert.Empty(t, h.connections.tokens[conn.ID].access)
	assert.Empty(t, h.connections.tokens[conn.ID].refresh)
	assert.Equal(t, []string{"refresh-1"}, h.connector.revoked)
	assert.False(t, h.integrations.enabled(h.tenant))
	health, ok := h.watchtower.get(conn.ID.String())
	require.True(t, ok)
	assert.True(t, health.resolved)

	_, err = h.svc.Disconnect(t.Context(), &services.DisconnectAccountingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	assert.Len(t, h.connector.revoked, 1)
}

func TestCheckHealthUsesTheStoredTokenWhileItIsFresh(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connector.facts.CompanyName = "Acme Freight Renamed"

	checked, err := h.svc.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Empty(t, h.connector.refreshedWith)
	assert.Equal(t, []string{"access-1", "access-1"}, h.connector.factsCalls)
	assert.Equal(t, "Acme Freight Renamed", checked.ExternalCompanyName)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, checked.Status)
}

func TestCheckHealthRefreshesAnExpiringToken(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.expireAccessToken(conn)

	_, err := h.svc.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"refresh-1"}, h.connector.refreshedWith)
	stored := h.connections.tokens[conn.ID]
	assert.Equal(t, "access-2", h.decrypt(t, conn, accessTokenField, stored.access))
	assert.Equal(t, "refresh-2", h.decrypt(t, conn, refreshTokenField, stored.refresh))
	assert.Equal(t, "access-2", h.connector.factsCalls[len(h.connector.factsCalls)-1])
	assert.Greater(t, stored.accessExpires, timeutils.NowUnix()+3000)
}

func TestCheckHealthMarksARevokedAuthorization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.expireAccessToken(conn)
	h.connector.refreshErrs = []error{errProvider}
	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryRevoked

	checked, err := h.svc.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.ConnectionStatusRevoked, checked.Status)
	assert.Empty(t, h.connections.tokens[conn.ID].refresh)
	assert.False(t, h.integrations.enabled(h.tenant))

	events := h.events.Published()
	require.Len(t, events, 1)
	assert.Equal(t, agent.EventAccountingConnectionDegraded, events[0].Kind)
	assert.Equal(t, conn.ID, events[0].SubjectID)

	item, ok := h.watchtower.get(conn.ID.String())
	require.True(t, ok)
	assert.False(t, item.resolved)
	assert.Equal(t, watchtower.SeverityCritical, item.item.Severity)
	assert.Equal(t, "/admin/integrations?type=QuickBooksOnline", item.item.Path)
}

func TestTransientFailuresDegradeFailAndRecover(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryTransient
	h.connector.factsErrs = []error{errProvider, errProvider, errProvider}

	statuses := make([]accountingsync.ConnectionStatus, 0, 4)
	for range 4 {
		checked, err := h.svc.CheckHealth(t.Context(), h.tenant, conn.ID)
		require.NoError(t, err)
		statuses = append(statuses, checked.Status)
	}
	assert.Equal(t, []accountingsync.ConnectionStatus{
		accountingsync.ConnectionStatusDegraded,
		accountingsync.ConnectionStatusDegraded,
		accountingsync.ConnectionStatusFailing,
		accountingsync.ConnectionStatusConnected,
	}, statuses)
	assert.Len(t, h.events.Published(), 2, "one event entering Degraded, one entering Failing")
	item, ok := h.watchtower.get(conn.ID.String())
	require.True(t, ok)
	assert.True(t, item.resolved)
	assert.True(t, h.integrations.enabled(h.tenant))
}

func TestCheckDueChecksEveryDueConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.markDue(conn)

	sweep, err := h.svc.CheckDue(t.Context(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, sweep.Listed)
	assert.Equal(t, 1, sweep.Checked)
	assert.Zero(t, sweep.Failed)

	sweep, err = h.svc.CheckDue(t.Context(), 10)
	require.NoError(t, err)
	assert.Zero(t, sweep.Listed, "a connection just checked is not due again")
}

func TestReceiveWebhookVerifiesBeforeRecording(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connector.webhookRealms = []string{testRealm, "unknown"}
	h.connector.webhookErr = errProvider

	err := h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       "bad",
		Body:            []byte("[]"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Nil(t, h.connections.rows[conn.ID].LastWebhookAt)

	h.connector.webhookErr = nil
	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       "good",
		Body:            []byte("[]"),
	}))
	assert.NotNil(t, h.connections.rows[conn.ID].LastWebhookAt)
}

func TestStatusReportsAvailabilityAndConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	status, err := h.svc.Status(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	assert.True(t, status.Available)
	assert.Nil(t, status.Connection)
	assert.Equal(t, "QuickBooks Online", status.ProviderName)

	conn := h.connect(t)
	status, err = h.svc.Status(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	require.NotNil(t, status.Connection)
	assert.Equal(t, conn.ID, status.Connection.ID)
	assert.Empty(t, status.Connection.AccessTokenCiphertext)
}

func TestSessionHandsOutAFreshTokenForAConnectedCompany(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.expireAccessToken(conn)

	session, err := h.svc.Session(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "access-2", session.AccessToken, "an expiring token is refreshed first")
	assert.Equal(t, conn.ID, session.Connection.ID)
	assert.Equal(t, testRealm, session.Connection.ExternalRealmID)
	assert.NotNil(t, session.Connector)
}

func TestSessionRefusesADisconnectedCompany(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	_, err := h.svc.Disconnect(t.Context(), &services.DisconnectAccountingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)

	_, err = h.svc.Session(t.Context(), h.tenant, conn.ID)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestSessionRecordsARevokedAuthorizationAndRefuses(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.expireAccessToken(conn)
	h.connector.refreshErrs = []error{errProvider}
	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryRevoked

	_, err := h.svc.Session(t.Context(), h.tenant, conn.ID)
	require.Error(t, err)
	assert.Equal(t, accountingsync.ConnectionStatusRevoked, h.connections.rows[conn.ID].Status)
	require.Len(t, h.events.Published(), 1)
	assert.Equal(t, agent.EventAccountingConnectionDegraded, h.events.Published()[0].Kind)
}

func TestReportCallFailureRevokesOnlyForAuthorizationFailures(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)

	transient := errors.New("timeout")
	h.connector.errCategories[transient] = accountingsync.ErrorCategoryTransient
	assert.Equal(t, accountingsync.ErrorCategoryTransient, h.svc.ReportCallFailure(t.Context(), h.tenant, conn.ID, transient))
	assert.Equal(t, accountingsync.ConnectionStatusConnected, h.connections.rows[conn.ID].Status,
		"a transient failure during a pull is not a health failure")

	h.connector.errCategories[errProvider] = accountingsync.ErrorCategoryRevoked
	assert.Equal(t, accountingsync.ErrorCategoryRevoked, h.svc.ReportCallFailure(t.Context(), h.tenant, conn.ID, errProvider))
	assert.Equal(t, accountingsync.ConnectionStatusRevoked, h.connections.rows[conn.ID].Status)
	assert.Empty(t, h.connections.tokens[conn.ID].refresh)
	require.Len(t, h.events.Published(), 1)
}
