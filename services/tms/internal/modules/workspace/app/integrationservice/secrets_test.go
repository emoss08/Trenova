package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const webhookSecret = "the-samsara-webhook-secret"

func secretsService(t *testing.T) (*Service, *encryptionservice.Service) {
	t.Helper()

	enc := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("integration-secret-binding-test-key-32+"),
	)
	return &Service{l: zap.NewNop(), secrets: secretconfig.NewCodec(enc, zap.NewNop())}, enc
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestIntegrationSecretsAreBoundToTheirTenantTypeAndField(t *testing.T) {
	t.Parallel()

	svc, enc := secretsService(t)
	tenant := testTenant()
	merged, err := svc.secrets.Merge(
		integration.ConfigSpecs[integration.TypeSamsara].Fields,
		map[string]string{"token": "samsara-token", integration.WebhookSecretField: webhookSecret},
		nil,
		newSecretScope(tenant, integration.TypeSamsara),
	)
	require.NoError(t, err)
	sealed := configspec.ReadString(merged, integration.WebhookSecretField)

	opened, err := svc.OpenSecret(tenant, integration.TypeSamsara, integration.WebhookSecretField, sealed)
	require.NoError(t, err)
	assert.Equal(t, webhookSecret, opened)

	_, err = enc.DecryptString(sealed)
	require.Error(t, err, "the secret is not sealed with an empty binding")

	_, err = svc.OpenSecret(testTenant(), integration.TypeSamsara, integration.WebhookSecretField, sealed)
	require.Error(t, err, "another tenant's secret does not open")

	_, err = svc.OpenSecret(tenant, integration.TypeResend, integration.WebhookSecretField, sealed)
	require.Error(t, err, "a secret does not open under another integration")

	_, err = svc.OpenSecret(tenant, integration.TypeSamsara, "token", sealed)
	require.Error(t, err, "a secret does not open as another field")
}

func TestOpenSecretStillReadsASecretSealedBeforeBinding(t *testing.T) {
	t.Parallel()

	svc, enc := secretsService(t)
	legacy, err := enc.EncryptString(webhookSecret)
	require.NoError(t, err)

	opened, err := svc.OpenSecret(
		testTenant(),
		integration.TypeSamsara,
		integration.WebhookSecretField,
		legacy,
	)

	require.NoError(t, err)
	assert.Equal(t, webhookSecret, opened)
}

func TestSavingAnIntegrationRebindsASecretSealedBeforeBinding(t *testing.T) {
	t.Parallel()

	svc, enc := secretsService(t)
	tenant := testTenant()
	legacy, err := enc.EncryptString(webhookSecret)
	require.NoError(t, err)

	merged, err := svc.secrets.Merge(
		integration.ConfigSpecs[integration.TypeSamsara].Fields,
		map[string]string{"token": "samsara-token"},
		map[string]any{integration.WebhookSecretField: legacy},
		newSecretScope(tenant, integration.TypeSamsara),
	)
	require.NoError(t, err)
	rebound := configspec.ReadString(merged, integration.WebhookSecretField)

	assert.NotEqual(t, legacy, rebound)
	_, err = enc.DecryptString(rebound)
	require.Error(t, err, "the kept secret is now bound")
	opened, err := svc.OpenSecret(tenant, integration.TypeSamsara, integration.WebhookSecretField, rebound)
	require.NoError(t, err)
	assert.Equal(t, webhookSecret, opened)
}
