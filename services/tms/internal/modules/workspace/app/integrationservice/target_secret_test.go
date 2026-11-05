package integrationservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	savedFuelPassword   = "sealed-password"
	savedFuelPrivateKey = "sealed-private-key"
)

func fuelFeedRequest() map[string]string {
	return map[string]string{
		integration.ConfigKeyFuelCardProvider:  "WEX",
		integration.ConfigKeyFuelAccountNumber: "12345",
		integration.ConfigKeyFuelHost:          "sftp.wex.test",
		integration.ConfigKeyFuelPort:          "22",
		integration.ConfigKeyFuelUsername:      "carrier",
		integration.ConfigKeyFuelAuthMode:      integration.FuelAuthModePassword,
		integration.ConfigKeyFuelKnownHostKey:  "ssh-ed25519 AAAA",
		integration.ConfigKeyFuelRemoteDir:     "/outbound",
		integration.ConfigKeyFuelFileFormat:    integration.FuelFileFormatDelimited,
	}
}

func savedFuelFeed() *integration.Integration {
	configuration := make(map[string]any)
	for key, value := range fuelFeedRequest() {
		configuration[key] = value
	}
	configuration[integration.ConfigKeyFuelPassword] = savedFuelPassword
	return &integration.Integration{
		Type:          integration.TypeWEXFuel,
		Configuration: configuration,
	}
}

func saveFuelFeed(
	t *testing.T,
	saved *integration.Integration,
	configuration map[string]string,
) error {
	t.Helper()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         &stubIntegrationRepo{getByTypeResult: saved},
		Encryption:   testEncryptionService(),
		AuditService: &stubAuditService{},
	})
	_, err := svc.UpdateConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeWEXFuel,
		&integrationports.UpdateConfigRequest{Configuration: configuration},
		"",
	)
	return err
}

func TestFuelFeedRequiresThePasswordAgainWhenTheSFTPTargetChanges(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "host", key: integration.ConfigKeyFuelHost, value: "sftp.attacker.test"},
		{name: "port", key: integration.ConfigKeyFuelPort, value: "2222"},
		{
			name:  "known host key",
			key:   integration.ConfigKeyFuelKnownHostKey,
			value: "ssh-ed25519 BBBB",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			configuration := fuelFeedRequest()
			configuration[tc.key] = tc.value

			err := saveFuelFeed(t, savedFuelFeed(), configuration)

			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(
				t,
				"configuration."+integration.ConfigKeyFuelPassword,
				multiErr.Errors[0].Field,
			)
		})
	}
}

func TestFuelFeedKeepsThePasswordWhenTheSFTPTargetIsUnchanged(t *testing.T) {
	t.Parallel()

	configuration := fuelFeedRequest()
	configuration[integration.ConfigKeyFuelRemoteDir] = "/elsewhere"

	require.NoError(t, saveFuelFeed(t, savedFuelFeed(), configuration))
}

func TestFuelFeedAcceptsANewTargetWithThePasswordReentered(t *testing.T) {
	t.Parallel()

	configuration := fuelFeedRequest()
	configuration[integration.ConfigKeyFuelHost] = "sftp.moved.test"
	configuration[integration.ConfigKeyFuelPassword] = "new-password"

	require.NoError(t, saveFuelFeed(t, savedFuelFeed(), configuration))
}

func TestFuelFeedDropsTheUnusedCredentialWhenTheSFTPTargetChanges(t *testing.T) {
	t.Parallel()

	saved := savedFuelFeed()
	saved.Configuration[integration.ConfigKeyFuelPrivateKey] = savedFuelPrivateKey
	configuration := fuelFeedRequest()
	configuration[integration.ConfigKeyFuelHost] = "sftp.moved.test"
	configuration[integration.ConfigKeyFuelPassword] = "new-password"
	final := map[string]any{
		integration.ConfigKeyFuelHost:       "sftp.moved.test",
		integration.ConfigKeyFuelAuthMode:   integration.FuelAuthModePassword,
		integration.ConfigKeyFuelPrivateKey: savedFuelPrivateKey,
		integration.ConfigKeyFuelPassword:   "sealed-new-password",
	}

	err := requireSecretsForNewTarget(
		integration.ConfigSpecs[integration.TypeWEXFuel],
		configuration,
		saved.Configuration,
		final,
	)

	require.NoError(t, err)
	assert.Empty(t, final[integration.ConfigKeyFuelPrivateKey])
	assert.Equal(t, "sealed-new-password", final[integration.ConfigKeyFuelPassword])
}

func updateWebhookConfig(
	t *testing.T,
	repo *stubIntegrationRepo,
	token string,
) *integrationports.ConfigResponse {
	t.Helper()

	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Encryption:   testEncryptionService(),
		AuditService: &stubAuditService{},
	})
	resp, err := svc.UpdateConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeResend,
		&integrationports.UpdateConfigRequest{
			Configuration: map[string]string{
				"apiKey":                      "re_test_key",
				"baseUrl":                     "https://api.resend.com",
				integration.WebhookTokenField: token,
			},
		},
		"",
	)
	require.NoError(t, err)
	return resp
}

func TestUpdateConfigIgnoresAWebhookTokenTheClientChose(t *testing.T) {
	t.Parallel()

	resp := updateWebhookConfig(
		t,
		&stubIntegrationRepo{getByTypeErr: errortypes.NewNotFoundError("integration not found")},
		"another-tenants-token",
	)

	token := findConfigField(t, resp.Fields, integration.WebhookTokenField)
	assert.NotEqual(t, "another-tenants-token", token.Value)
	assert.True(t, strings.HasPrefix(token.Value, "resend_"))
}

func TestUpdateConfigKeepsTheStoredWebhookToken(t *testing.T) {
	t.Parallel()

	stored := &integration.Integration{
		Type: integration.TypeResend,
		Configuration: map[string]any{
			"baseUrl":                     "https://api.resend.com",
			integration.WebhookTokenField: "resend_stored",
		},
	}

	for name, token := range map[string]string{"overwritten": "chosen", "omitted": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := updateWebhookConfig(t, &stubIntegrationRepo{getByTypeResult: stored}, token)

			assert.Equal(
				t,
				"resend_stored",
				findConfigField(t, resp.Fields, integration.WebhookTokenField).Value,
			)
		})
	}
}
