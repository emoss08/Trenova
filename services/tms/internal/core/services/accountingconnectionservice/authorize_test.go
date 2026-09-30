package accountingconnectionservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var twoCompanies = []services.AccountingCompany{
	{ID: "1111", Name: "Acme East", ConnectionID: "conn-east"},
	{ID: "2222", Name: "Acme West", ConnectionID: "conn-west"},
}

func (h *harness) completeWithChoice(t *testing.T) *services.AccountingAuthorizationCompletion {
	t.Helper()
	h.connector.companies = twoCompanies
	completion, err := h.svc.CompleteAuthorization(
		t.Context(),
		&services.CompleteAccountingAuthorizationRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeQuickBooksOnline,
			State:           h.start(t),
			Code:            "auth-code",
			RealmID:         testRealm,
		},
	)
	require.NoError(t, err)
	return completion
}

func (h *harness) choose(
	t *testing.T,
	token, companyID string,
) (*accountingsync.AccountingConnection, error) {
	t.Helper()
	return h.svc.ChooseCompany(t.Context(), &services.ChooseAccountingCompanyRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		ChoiceToken:     token,
		CompanyID:       companyID,
	})
}

func TestSeveralCompaniesAreOfferedBeforeAnythingConnects(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	completion := h.completeWithChoice(t)

	assert.Nil(t, completion.Connection)
	assert.Equal(t, twoCompanies, completion.Companies)
	assert.NotEmpty(t, completion.ChoiceToken)
	assert.Positive(t, completion.ChoiceExpiresAt)
	assert.Empty(t, h.connections.rows)
	assert.Empty(t, h.connector.revoked)
}

func TestTheHeldGrantIsSealed(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.completeWithChoice(t)

	require.Len(t, h.states.states, 1)
	for _, state := range h.states.states {
		require.NotEmpty(t, state.SealedGrant)
		assert.NotContains(t, state.SealedGrant, "refresh-1")
		assert.NotContains(t, state.SealedGrant, "access-1")
		assert.Len(t, state.Companies, 2)
		assert.Equal(t, companyChoiceTTL, h.states.ttls[state.State])
	}
}

func TestChoosingACompanyConnectsItAndReleasesTheOthers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	completion := h.completeWithChoice(t)

	conn, err := h.choose(t, completion.ChoiceToken, "2222")
	require.NoError(t, err)
	assert.Equal(t, "2222", conn.ExternalRealmID)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.Equal(t, "refresh-1", h.decrypt(t, conn, refreshTokenField, h.connections.tokens[conn.ID].refresh))
	assert.Equal(t, []services.AccountingCompany{twoCompanies[0]}, h.connector.released)
	assert.Empty(t, h.connector.revoked)

	_, err = h.choose(t, completion.ChoiceToken, "2222")
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestChoosingACompanyThatWasNotOfferedRevokesTheGrant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	completion := h.completeWithChoice(t)

	_, err := h.choose(t, completion.ChoiceToken, "3333")
	require.Error(t, err)
	assert.Empty(t, h.connections.rows)
	assert.Contains(t, h.connector.revoked, "refresh-1")
	assert.ElementsMatch(t, twoCompanies, h.connector.released)
}

func TestOnlyThePersonWhoSignedInCanChoose(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	completion := h.completeWithChoice(t)

	_, err := h.svc.ChooseCompany(t.Context(), &services.ChooseAccountingCompanyRequest{
		TenantInfo:      h.tenant,
		UserID:          pulid.MustNew("usr_"),
		IntegrationType: integration.TypeQuickBooksOnline,
		ChoiceToken:     completion.ChoiceToken,
		CompanyID:       "1111",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, h.connections.rows)
}

func TestAChoiceTokenCannotFinishASignIn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	completion := h.completeWithChoice(t)

	_, err := h.complete(t, completion.ChoiceToken)
	require.Error(t, err)
	assert.Empty(t, h.connections.rows)
}

func TestAStateCannotBeUsedAsAChoiceToken(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.choose(t, h.start(t), "1111")
	require.Error(t, err)
	assert.Empty(t, h.connections.rows)
}

func TestASignInThatAuthorizedNoCompanyIsRevoked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.companies = []services.AccountingCompany{}

	_, err := h.complete(t, h.start(t))
	require.Error(t, err)
	assert.Contains(t, h.connector.revoked, "refresh-1")
	assert.Empty(t, h.connections.rows)
}

func TestCompaniesWithUnusableIDsAreDropped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.connector.companies = []services.AccountingCompany{
		{ID: "not valid/id", Name: "Broken"},
		{ID: "4444", Name: "Acme"},
	}

	conn, err := h.complete(t, h.start(t))
	require.NoError(t, err)
	assert.Equal(t, "4444", conn.ExternalRealmID)
}

func TestConnectingASecondAccountingSystemIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	other := &accountingsync.AccountingConnection{
		ID:              pulid.MustNew("acctc_"),
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		IntegrationType: integration.TypeXero,
		ExternalRealmID: "xero-tenant",
		Status:          accountingsync.ConnectionStatusConnected,
	}
	h.connections.rows[other.ID] = other

	_, err := h.svc.StartAuthorization(t.Context(), &services.StartAccountingAuthorizationRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Disconnect Xero"), err.Error())

	other.Status = accountingsync.ConnectionStatusDisconnected
	h.connect(t)
}

func TestAWebhookNamingATenantAppIsCheckedAgainstThatApp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	status := h.saveTenantApp(t)
	conn := h.connect(t)
	h.connector.webhookRealms = []string{testRealm}
	appID := status.App.TenantApp.ID.String()

	err := h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		AppID:           appID,
		Signature:       instanceVerifier,
		Body:            []byte("[]"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Nil(t, h.connections.rows[conn.ID].LastWebhookAt)

	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		AppID:           appID,
		Signature:       tenantVerifier,
		Body:            []byte("[]"),
	}))
	assert.NotNil(t, h.connections.rows[conn.ID].LastWebhookAt)
}

func TestAWebhookNamingAnUnknownAppIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for _, appID := range []string{"acctapp_01J00000000000000000000000", "not-an-id", "acctc_01J00000000000000000000000"} {
		err := h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
			IntegrationType: integration.TypeQuickBooksOnline,
			AppID:           appID,
			Signature:       tenantVerifier,
			Body:            []byte("[]"),
		})
		require.Error(t, err, appID)
		assert.True(t, errortypes.IsAuthenticationError(err), appID)
	}
}

func TestAVerifiedWebhookNamingNoCompanyIsAccepted(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	status := h.saveTenantApp(t)
	h.connector.webhookRealms = nil

	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		AppID:           status.App.TenantApp.ID.String(),
		Signature:       tenantVerifier,
		Body:            []byte("{}"),
	}))
	require.NoError(t, h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       instanceVerifier,
		Body:            []byte("{}"),
	}))

	err := h.svc.ReceiveWebhook(t.Context(), &services.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       "forged",
		Body:            []byte("{}"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
}

func TestATenantAppShowsItsOwnWebhookPath(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	status := h.saveTenantApp(t)
	assert.Equal(
		t,
		"/webhooks/accounting/quickbooks/"+status.App.TenantApp.ID.String()+"/",
		status.App.WebhookPath,
	)
}
