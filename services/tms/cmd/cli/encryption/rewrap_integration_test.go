//go:build integration

package encryption

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/tenantsyncrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/enveloperepository"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/enveloperewrapservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oldKey = "the-previous-local-encryption-key-of-32-chars"
	newKey = "the-current-local-encryption-key-of-32-chars!"
)

func keyring(t *testing.T, active string, retired ...string) *encryptionservice.Service {
	t.Helper()
	manager, err := encryptionservice.NewLocalKeyring(active, retired)
	require.NoError(t, err)
	return encryptionservice.NewWithKeyManager(manager)
}

func TestRewrapMovesStoredSecretsToTheActiveKey(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	old := keyring(t, oldKey)

	providerID := pulid.MustNew("aip_")
	legacyAPIKey, err := old.EncryptString("sk-legacy")
	require.NoError(t, err)
	rotationExpiresAt := time.Now().Add(time.Hour).Unix()
	previousAPIKey, err := old.EncryptStringWithAAD(
		"sk-previous",
		encryptionservice.AIProviderCredentialAAD(tenant.OrgID, tenant.BuID, providerID),
	)
	require.NoError(t, err)
	err = aiproviderdb.Providers.Create(ctx, db, &aiprovider.Provider{
		ID:                   providerID,
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		Name:                 "Primary",
		Kind:                 aiprovider.KindOpenAIChat,
		BaseURL:              "https://api.example.com/v1",
		Model:                "gpt-test",
		Tasks:                []aiprovider.Task{aiprovider.TaskGeneral},
		StructuredOutputMode: aiprovider.KindOpenAIChat.DefaultStructuredOutputMode(),
		ReasoningEffort:      aiprovider.ReasoningOff,
		ThinkingStyle:        aiprovider.ThinkingStyleAuto,
		EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		TimeoutSeconds:       aiprovider.DefaultTimeoutSeconds,
		MaxTokens:            aiprovider.ClampMaxTokens(8192),
		Enabled:              true,
		APIKey:               legacyAPIKey,
		PreviousAPIKey:       previousAPIKey,
		RotationExpiresAt:    &rotationExpiresAt,
	})
	require.NoError(t, err)

	mfaAAD := encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeMFATOTPSecret,
		OrganizationID: tenant.OrgID,
		ResourceID:     data.User.ID.String(),
	}
	totp, err := old.EncryptStringWithAAD("JBSWY3DPEHPK3PXP", mfaAAD)
	require.NoError(t, err)
	authenticatorID := pulid.MustNew("mfa_")
	err = iamdb.MFAAuthenticators.Create(ctx, db, &iam.MFAAuthenticator{
		ID:             authenticatorID,
		UserID:         data.User.ID,
		OrganizationID: tenant.OrgID,
		Type:           iam.MFAAuthenticatorTypeTOTP,
		Name:           "Authenticator app",
		SecretCipher:   totp,
		Enabled:        true,
	})
	require.NoError(t, err)

	mapsKey, err := old.EncryptString("maps-key")
	require.NoError(t, err)
	carrierAAD := encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIntegrationSecret,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     "integration:" + integration.TypeCarrierOK.String() + ":apiKey",
	}
	carrierLegacyKey, err := old.EncryptString("carrier-key")
	require.NoError(t, err)
	integrations := []*integration.Integration{
		{
			ID:             pulid.MustNew("integ_"),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Type:           integration.TypeGoogleMaps,
			Name:           "Google Maps",
			Category:       integration.CategoryMappingRouting,
			Configuration:  map[string]any{"apiKey": mapsKey, "region": "us"},
		},
		{
			ID:             pulid.MustNew("integ_"),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Type:           integration.TypeCarrierOK,
			Name:           "CarrierOK",
			Category:       integration.CategoryFreightLogistics,
			Configuration:  map[string]any{"apiKey": carrierLegacyKey},
		},
	}
	err = integrationdb.Integrations.CreateMany(ctx, db, integrations)
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	service := enveloperewrapservice.New(enveloperewrapservice.Deps{
		Repository: enveloperepository.New(enveloperepository.Params{DB: conn}),
		Tenants:    tenantsyncrepository.NewTenantLister(tenantsyncrepository.Params{DB: conn}),
		Rewrapper:  encryptionservice.NewEnvelopeRewrapper(keyring(t, newKey, oldKey)),
	})
	opts := enveloperewrapservice.Options{OrganizationID: tenant.OrgID, BatchSize: 1}

	report, err := service.Run(ctx, envelopeSources(false), opts)
	require.NoError(t, err)
	assert.Zero(t, report.Failed())
	assert.Zero(t, report.Remaining())
	rewrapped := map[string]int{}
	for _, source := range report.Sources {
		rewrapped[source.Name] = source.Rewrapped
	}
	assert.Equal(t, map[string]int{
		"AI provider API key":          1,
		"AI provider previous API key": 1,
		"MFA authenticator secret":     1,
		"integration secrets":          2,
	}, nonZero(rewrapped))

	newOnly := keyring(t, newKey)
	providers := aiproviderdb.Providers
	stored, err := providers.Where(providers.ID.Eq(providerID)).One(ctx, db)
	require.NoError(t, err)
	bound := encryptionservice.AIProviderCredentialAAD(tenant.OrgID, tenant.BuID, providerID)
	apiKey, err := newOnly.DecryptStringWithAAD(stored.APIKey, bound)
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy", apiKey)
	previous, err := newOnly.DecryptStringWithAAD(stored.PreviousAPIKey, bound)
	require.NoError(t, err)
	assert.Equal(t, "sk-previous", previous)

	authenticators := iamdb.MFAAuthenticators
	authenticator, err := authenticators.Where(authenticators.ID.Eq(authenticatorID)).One(ctx, db)
	require.NoError(t, err)
	secret, err := newOnly.DecryptStringWithAAD(authenticator.SecretCipher, mfaAAD)
	require.NoError(t, err)
	assert.Equal(t, "JBSWY3DPEHPK3PXP", secret)

	byType := map[integration.Type]*integration.Integration{}
	loaded, err := integrationdb.Integrations.Query().All(ctx, db)
	require.NoError(t, err)
	for _, row := range loaded {
		byType[row.Type] = row
	}
	mapsSealed := byType[integration.TypeGoogleMaps].Configuration["apiKey"].(string)
	maps, err := newOnly.DecryptStringWithAAD(mapsSealed, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIntegrationSecret,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ResourceID:     "integration:" + integration.TypeGoogleMaps.String() + ":apiKey",
	})
	require.NoError(t, err, "the rewrap binds a secret sealed before binding")
	assert.Equal(t, "maps-key", maps)
	_, err = newOnly.DecryptString(mapsSealed)
	require.Error(t, err, "the rebound secret no longer opens without its binding")
	assert.Equal(t, "us", byType[integration.TypeGoogleMaps].Configuration["region"])
	carrier, err := newOnly.DecryptStringWithAAD(
		byType[integration.TypeCarrierOK].Configuration["apiKey"].(string),
		carrierAAD,
	)
	require.NoError(t, err)
	assert.Equal(t, "carrier-key", carrier)

	again, err := service.Run(ctx, envelopeSources(false), opts)
	require.NoError(t, err)
	for _, source := range again.Sources {
		assert.Zero(t, source.Rewrapped, source.Name)
	}
}

func nonZero(counts map[string]int) map[string]int {
	out := make(map[string]int, len(counts))
	for name, count := range counts {
		if count != 0 {
			out[name] = count
		}
	}
	return out
}
