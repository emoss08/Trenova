package accountingconnectionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantClientID = "tenant-client"
	tenantSecret   = "tenant-secret"
	tenantVerifier = "tenant-verifier"
)

func (h *harness) saveApp(
	t *testing.T,
	req services.SaveAccountingAppRequest,
) (*services.AccountingSyncStatus, error) {
	t.Helper()
	req.TenantInfo = h.tenant
	req.UserID = h.userID
	req.IntegrationType = integration.TypeQuickBooksOnline
	return h.svc.SaveApp(t.Context(), &req)
}

func (h *harness) saveTenantApp(t *testing.T) *services.AccountingSyncStatus {
	t.Helper()
	status, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:          accountingsync.AppEnvironmentSandbox,
		ClientID:             tenantClientID,
		ClientSecret:         tenantSecret,
		WebhookVerifierToken: tenantVerifier,
	})
	require.NoError(t, err)
	return status
}

func (h *harness) removeApp(t *testing.T) (*services.AccountingSyncStatus, error) {
	t.Helper()
	return h.svc.RemoveApp(t.Context(), &services.RemoveAccountingAppRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
}

func TestStatusWithoutAnyAppIsUnavailable(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.available = false
	status, err := h.svc.Status(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	assert.False(t, status.Available)
	require.NotNil(t, status.App)
	assert.False(t, status.App.InstanceAppAvailable)
	assert.Empty(t, status.App.ActiveSource)
	assert.Nil(t, status.App.TenantApp)
	assert.Equal(t, h.connector.redirectURL, status.App.RedirectURL)
	assert.Equal(t, "/webhooks/accounting/quickbooks/", status.App.WebhookPath)
}

func TestSavingATenantAppMakesAnInstanceWithoutOneAvailable(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.available = false
	status := h.saveTenantApp(t)

	assert.True(t, status.Available)
	assert.Equal(t, accountingsync.AppSourceTenant, status.App.ActiveSource)
	require.NotNil(t, status.App.TenantApp)
	assert.Equal(t, tenantClientID, status.App.TenantApp.ClientID)
	assert.True(t, status.App.TenantApp.HasWebhookVerifier())
	assert.Len(t, h.audit.entries, 1)
}

func TestTenantAppSecretsAreSealed(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	row := h.apps.rows[integrationKey(h.tenant, integration.TypeQuickBooksOnline)]
	require.NotNil(t, row)
	assert.NotContains(t, row.ClientSecretCiphertext, tenantSecret)
	assert.NotContains(t, row.WebhookVerifierCiphertext, tenantVerifier)

	app, err := h.svc.openCredential(row)
	require.NoError(t, err)
	assert.Equal(t, tenantSecret, app.ClientSecret)
	assert.Equal(t, tenantVerifier, app.WebhookVerifierToken)
}

func TestTenantAppIsUsedAheadOfTheInstanceApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	conn := h.connect(t)

	assert.Equal(t, accountingsync.AppSourceTenant, conn.AppSource)
	assert.Equal(t, accountingsync.AppEnvironmentSandbox, conn.AppEnvironment)
	assert.Equal(
		t,
		accountingsync.AppFingerprint(accountingsync.AppEnvironmentSandbox, tenantClientID),
		conn.AppFingerprint,
	)
	bound := h.connector.lastBound()
	require.NotNil(t, bound)
	assert.Equal(t, tenantClientID, bound.ClientID)
	assert.Equal(t, tenantSecret, bound.ClientSecret)
}

func TestInstanceConnectionsRecordTheInstanceApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	assert.Equal(t, accountingsync.AppSourceInstance, conn.AppSource)
	assert.Equal(t, testInstanceApp.Identity().Fingerprint, conn.AppFingerprint)
}

func TestSaveAppRequiresASecretForANewApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment: accountingsync.AppEnvironmentSandbox,
		ClientID:    tenantClientID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, h.apps.rows)
}

func TestSaveAppValidatesItsInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  "Staging",
		ClientID:     "",
		ClientSecret: tenantSecret,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	_, err = h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     "bad client id!",
		ClientSecret: tenantSecret,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, h.apps.rows)
}

func TestSaveAppRefusesKeysTheProviderRejects(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.verifyAppErr = services.ErrAccountingAppRejected
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     tenantClientID,
		ClientSecret: "wrong",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, h.apps.rows)
}

func TestSaveAppReportsAnUnreachableProviderWithoutSaving(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.verifyAppErr = errProvider
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     tenantClientID,
		ClientSecret: tenantSecret,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.apps.rows)
}

func TestSaveAppRequiresARedirectAddress(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.redirectURL = ""
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     tenantClientID,
		ClientSecret: tenantSecret,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.apps.rows)
}

func TestSaveAppKeepsTheStoredSecretWhenLeftBlank(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment: accountingsync.AppEnvironmentSandbox,
		ClientID:    tenantClientID,
	})
	require.NoError(t, err)

	row := h.apps.rows[integrationKey(h.tenant, integration.TypeQuickBooksOnline)]
	app, err := h.svc.openCredential(row)
	require.NoError(t, err)
	assert.Equal(t, tenantSecret, app.ClientSecret)
	assert.Equal(t, tenantVerifier, app.WebhookVerifierToken)
}

func TestChangingTheClientIDNeedsItsOwnSecretAndDropsTheOldVerifier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment: accountingsync.AppEnvironmentSandbox,
		ClientID:    "another-client",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	status, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentProduction,
		ClientID:     "another-client",
		ClientSecret: "another-secret",
	})
	require.NoError(t, err)
	assert.Equal(t, "another-client", status.App.TenantApp.ClientID)
	assert.False(t, status.App.TenantApp.HasWebhookVerifier())
}

func TestSaveAppCanClearTheVerifier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	status, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:               accountingsync.AppEnvironmentSandbox,
		ClientID:                  tenantClientID,
		ClearWebhookVerifierToken: true,
	})
	require.NoError(t, err)
	assert.False(t, status.App.TenantApp.HasWebhookVerifier())
}

func TestSaveAppIsRefusedWhileConnectedThroughTheInstanceApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connect(t)
	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     tenantClientID,
		ClientSecret: tenantSecret,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.apps.rows)
}

func TestConnectedTenantAppCanRotateItsSecretButNotItsIdentity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	h.connect(t)

	_, err := h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentSandbox,
		ClientID:     tenantClientID,
		ClientSecret: "rotated-secret",
	})
	require.NoError(t, err)

	_, err = h.saveApp(t, services.SaveAccountingAppRequest{
		Environment:  accountingsync.AppEnvironmentProduction,
		ClientID:     tenantClientID,
		ClientSecret: "rotated-secret",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	row := h.apps.rows[integrationKey(h.tenant, integration.TypeQuickBooksOnline)]
	assert.Equal(t, accountingsync.AppEnvironmentSandbox, row.Environment)
}

func TestRemoveAppIsRefusedWhileConnectedThroughIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	h.connect(t)

	_, err := h.removeApp(t)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Len(t, h.apps.rows, 1)

	_, err = h.svc.Disconnect(t.Context(), &services.DisconnectAccountingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)

	status, err := h.removeApp(t)
	require.NoError(t, err)
	assert.Empty(t, h.apps.rows)
	assert.Equal(t, accountingsync.AppSourceInstance, status.App.ActiveSource)
}

func TestRemovingAnAppThatWasNeverSavedIsANoop(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	status, err := h.removeApp(t)
	require.NoError(t, err)
	assert.Nil(t, status.App.TenantApp)
	assert.Empty(t, h.audit.entries)
}

func TestCompleteAuthorizationRefusesWhenTheAppChangedMidFlight(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	state := h.start(t)
	h.saveTenantApp(t)

	_, err := h.complete(t, state)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, h.connector.exchangedCodes)
}

func TestConnectionFailsWhenItsAppDisappears(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connector.available = false

	checked, err := h.svc.CheckHealth(t.Context(), h.tenant, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, checked.LastErrorCategory)
	assert.Contains(t, checked.LastErrorMessage, "reconnect")

	_, err = h.svc.Session(t.Context(), h.tenant, conn.ID)
	require.Error(t, err)
}

func TestReceiveWebhookVerifiesWithTheConnectionsOwnApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.saveTenantApp(t)
	conn := h.connect(t)
	h.connector.webhookRealms = []string{testRealm}

	err := h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       "signed-by-someone-else",
		Body:            []byte("[]"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Nil(t, h.connections.rows[conn.ID].LastWebhookAt)

	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       tenantVerifier,
		Body:            []byte("[]"),
	}))
	assert.NotNil(t, h.connections.rows[conn.ID].LastWebhookAt)
}

func TestReceiveWebhookForUnknownCompaniesIsIgnored(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.webhookRealms = []string{"nobody"}
	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       "anything",
		Body:            []byte("[]"),
	}))
}
