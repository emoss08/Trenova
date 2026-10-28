package accountingconnection

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppFingerprintSeparatesEnvironmentsAndIgnoresPadding(t *testing.T) {
	t.Parallel()

	sandbox := AppFingerprint(accountingprovider.AppEnvironmentSandbox, "client")
	assert.Len(t, sandbox, appFingerprintLength)
	assert.Equal(t, sandbox, AppFingerprint(accountingprovider.AppEnvironmentSandbox, "  client "))
	assert.NotEqual(
		t,
		sandbox,
		AppFingerprint(accountingprovider.AppEnvironmentProduction, "client"),
	)
	assert.NotEqual(t, sandbox, AppFingerprint(accountingprovider.AppEnvironmentSandbox, "other"))
	assert.NotContains(t, sandbox, "client")
}

func TestParseAppEnvironment(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]accountingprovider.AppEnvironment{
		"sandbox":      accountingprovider.AppEnvironmentSandbox,
		" Production ": accountingprovider.AppEnvironmentProduction,
	} {
		got, ok := accountingprovider.ParseAppEnvironment(input)
		require.True(t, ok, input)
		assert.Equal(t, want, got)
	}
	_, ok := accountingprovider.ParseAppEnvironment("staging")
	assert.False(t, ok)
}

func TestConnectedThroughMatchesTheRecordedApp(t *testing.T) {
	t.Parallel()

	tenantApp := AppIdentity{
		Source:      AppSourceTenant,
		Environment: accountingprovider.AppEnvironmentSandbox,
		Fingerprint: AppFingerprint(accountingprovider.AppEnvironmentSandbox, "tenant"),
	}
	conn := &AccountingConnection{}
	conn.BindApp(tenantApp)

	assert.True(t, conn.ConnectedThrough(tenantApp))
	assert.False(t, conn.ConnectedThrough(AppIdentity{
		Source:      AppSourceTenant,
		Fingerprint: AppFingerprint(accountingprovider.AppEnvironmentSandbox, "rotated"),
	}))
	assert.False(t, conn.ConnectedThrough(AppIdentity{
		Source:      AppSourceInstance,
		Fingerprint: tenantApp.Fingerprint,
	}))
}

func TestConnectionsFromBeforeAppsWereRecordedBelongToTheInstance(t *testing.T) {
	t.Parallel()

	legacy := &AccountingConnection{AppSource: AppSourceInstance}
	assert.True(t, legacy.ConnectedThrough(AppIdentity{
		Source:      AppSourceInstance,
		Fingerprint: AppFingerprint(accountingprovider.AppEnvironmentProduction, "instance"),
	}))
	assert.False(t, legacy.ConnectedThrough(AppIdentity{
		Source:      AppSourceTenant,
		Fingerprint: AppFingerprint(accountingprovider.AppEnvironmentProduction, "tenant"),
	}))
}

func validCredential() *AccountingAppCredential {
	cred := &AccountingAppCredential{
		IntegrationType:        integration.TypeQuickBooksOnline,
		Environment:            accountingprovider.AppEnvironmentSandbox,
		ClientID:               " ABcd-123_x.y ",
		ClientSecretCiphertext: "sealed",
	}
	cred.Stamp()
	return cred
}

func TestAppCredentialStampTrimsAndFingerprints(t *testing.T) {
	t.Parallel()

	cred := validCredential()
	assert.Equal(t, "ABcd-123_x.y", cred.ClientID)
	assert.Equal(
		t,
		AppFingerprint(accountingprovider.AppEnvironmentSandbox, "ABcd-123_x.y"),
		cred.Fingerprint,
	)

	multiErr := errortypes.NewMultiError()
	cred.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestAppCredentialValidation(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*AccountingAppCredential){
		"not an accounting system": func(c *AccountingAppCredential) {
			c.IntegrationType = integration.TypeSamsara
		},
		"unknown environment": func(c *AccountingAppCredential) { c.Environment = "Staging" },
		"missing client id":   func(c *AccountingAppCredential) { c.ClientID = "" },
		"client id with spaces": func(c *AccountingAppCredential) {
			c.ClientID = "has space"
		},
		"missing secret": func(c *AccountingAppCredential) { c.ClientSecretCiphertext = "" },
	}
	for name, mutate := range cases {
		cred := validCredential()
		mutate(cred)
		multiErr := errortypes.NewMultiError()
		cred.Validate(multiErr)
		assert.True(t, multiErr.HasErrors(), name)
	}
}

func TestWebhookPathIsOnlyForAccountingSystems(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"/webhooks/accounting/quickbooks/",
		accountingprovider.WebhookPath(integration.TypeQuickBooksOnline),
	)
	assert.Empty(t, accountingprovider.WebhookPath(integration.TypeSamsara))
}
