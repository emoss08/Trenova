package aiproviderservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	serviceAccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	serviceSecretKey   = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	serviceAccountKey  = `{"type":"service_account","project_id":"freight-ops",` +
		`"private_key_id":"0123456789abcdef0123456789abcdef01234567",` +
		`"private_key":"-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n",` +
		`"client_email":"trenova@freight-ops.iam.gserviceaccount.com",` +
		`"token_uri":"https://oauth2.googleapis.com/token"}`
)

func bedrockRequest(provider *aiprovider.Provider, region string, key *string) *agentports.SaveAIProviderRequest {
	req := saveRequest(provider, "", key)
	req.Kind = aiprovider.KindAnthropicMessages
	req.Model = "anthropic.claude-opus-5-5"
	req.Platform = aiprovider.PlatformBedrock
	req.PlatformSettings = &aiprovider.PlatformSettings{
		Auth:        aiprovider.PlatformAuthAccessKey,
		Region:      region,
		AccessKeyID: serviceAccessKeyID,
	}

	return req
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}

	return fields
}

func TestApply_StoresThePlatformAndItsSecret(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)

	require.NoError(t, svc.apply(provider, bedrockRequest(provider, "US-EAST-1", strPtr(serviceSecretKey)), nil))

	assert.Equal(t, aiprovider.PlatformBedrock, provider.Platform)
	require.NotNil(t, provider.PlatformSettings)
	assert.Equal(t, "us-east-1", provider.PlatformSettings.Region)
	assert.Equal(t, "EKEY", provider.APIKeyLastFour)
	decrypted, err := svc.encryption.DecryptStringWithAAD(provider.APIKey, credentialAAD(provider))
	require.NoError(t, err)
	assert.Equal(t, serviceSecretKey, decrypted)
	_, err = svc.encryption.DecryptString(provider.APIKey)
	require.Error(t, err, "a provider key is never sealed without its binding")
}

// A service account key is told apart from another by its key id: every key
// file opens and closes the same way.
func TestApply_FingerprintsAServiceAccountKeyByItsKeyID(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)
	req := saveRequest(provider, "", strPtr(serviceAccountKey))
	req.Kind = aiprovider.KindAnthropicMessages
	req.Platform = aiprovider.PlatformVertexAI
	req.PlatformSettings = &aiprovider.PlatformSettings{
		Auth: aiprovider.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops",
	}

	require.NoError(t, svc.apply(provider, req, nil))
	assert.Equal(t, "4567", provider.APIKeyLastFour)
}

func TestApply_RefusesACredentialThePlatformCannotUse(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)
	req := saveRequest(provider, "", strPtr(`{"type":"external_account","credential_source":{"file":"/etc/shadow"}}`))
	req.Kind = aiprovider.KindAnthropicMessages
	req.Platform = aiprovider.PlatformVertexAI
	req.PlatformSettings = &aiprovider.PlatformSettings{
		Auth: aiprovider.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops",
	}

	err := svc.apply(provider, req, nil)

	assert.Contains(t, fieldsOf(t, err), "apiKey")
}

// A server that hosts many organizations leaves its own cloud identity off,
// so no organization's administrator can spend the operator's account.
func TestApply_RefusesServerCredentialsTheServerDisallows(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)
	req := bedrockRequest(provider, "us-east-1", nil)
	req.PlatformSettings = &aiprovider.PlatformSettings{Auth: aiprovider.PlatformAuthServer, Region: "us-east-1"}

	err := svc.apply(provider, req, nil)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "platformSettings.auth", validation.Field)

	svc.ai = &config.AIConfig{ServerCloudCredentials: true}
	require.NoError(t, svc.apply(provider, req, nil))
	assert.True(t, provider.UsesServerCredentials())
}

/*
A key entered for one region, account or identity is not carried to another.
Kept across the move, the next call would send it, decrypted, somewhere it was
never entered for.
*/
func TestUpdate_RefusesToKeepTheKeyForAnotherRegionOrIdentity(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	provider := testProvider(t, svc)
	require.NoError(t, svc.apply(provider, bedrockRequest(provider, "us-east-1", strPtr(serviceSecretKey)), nil))
	svc.repo = &fakeProviderRepo{provider: provider}

	_, err := svc.Update(t.Context(), bedrockRequest(provider, "eu-west-1", nil), nil)
	assert.Contains(t, fieldsOf(t, err), "apiKey")

	otherKey := bedrockRequest(provider, "us-east-1", nil)
	otherKey.PlatformSettings.AccessKeyID = "AKIAI44QH8DHBEXAMPLE"
	_, err = svc.Update(t.Context(), otherKey, nil)
	assert.Contains(t, fieldsOf(t, err), "apiKey")

	direct := saveRequest(provider, "https://api.anthropic.com", nil)
	direct.Kind = aiprovider.KindAnthropicMessages
	_, err = svc.Update(t.Context(), direct, nil)
	assert.Contains(t, fieldsOf(t, err), "apiKey", "leaving the platform is a move too")
}

/*
Testing a draft with no key typed reuses the provider's stored key only for
the place and identity it was entered for.
*/
func TestSeal_ReusesTheStoredKeyOnlyForItsOwnTarget(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeProviderRepo{}, &fakeProber{})
	stored := testProvider(t, svc)
	require.NoError(t, svc.apply(stored, bedrockRequest(stored, "us-east-1", strPtr(serviceSecretKey)), nil))
	svc.repo = &fakeProviderRepo{provider: stored}

	endpoint := func(region string) *agentports.AIProviderEndpoint {
		return &agentports.AIProviderEndpoint{
			ProviderID: stored.ID,
			Kind:       aiprovider.KindAnthropicMessages,
			Platform:   aiprovider.PlatformBedrock,
			PlatformSettings: &aiprovider.PlatformSettings{
				Auth: aiprovider.PlatformAuthAccessKey, Region: region, AccessKeyID: serviceAccessKeyID,
			},
		}
	}
	tenant := saveRequest(stored, "", nil).TenantInfo

	sealed, err := svc.seal(t.Context(), tenant, endpoint("us-east-1"))
	require.NoError(t, err)
	assert.NotEqual(t, stored.APIKey, sealed.SealedAPIKey)
	unsealed, err := svc.unseal(tenant, &sealed)
	require.NoError(t, err)
	assert.Equal(t, serviceSecretKey, unsealed)
	assert.Equal(t, aiprovider.PlatformBedrock, sealed.Platform)

	otherTenant := tenant
	otherTenant.OrgID = pulid.MustNew("org_")
	_, err = svc.unseal(otherTenant, &sealed)
	require.Error(t, err, "a sealed draft key does not open for another organization")

	_, err = svc.seal(t.Context(), tenant, endpoint("eu-west-1"))
	assert.Contains(t, fieldsOf(t, err), "apiKey", "another region gets no stored key")
}

func TestRunTestDraft_ProbesThePlatformItWasSealedFor(t *testing.T) {
	t.Parallel()

	prober := &fakeProber{result: &agentports.TestAIProviderResult{Success: true}}
	svc := newTestService(&fakeProviderRepo{}, prober)
	encrypted, err := svc.encryption.EncryptString(serviceSecretKey)
	require.NoError(t, err)

	_, err = svc.RunTestDraft(t.Context(), &agentports.ProbeAIProviderDraftRequest{
		Endpoint: agentports.SealedAIProviderEndpoint{
			Kind:     aiprovider.KindAnthropicMessages,
			Platform: aiprovider.PlatformBedrock,
			PlatformSettings: &aiprovider.PlatformSettings{
				Auth: aiprovider.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: serviceAccessKeyID,
			},
			SealedAPIKey: encrypted,
		},
		Model: "anthropic.claude-opus-5-5",
		Tasks: []aiprovider.Task{aiprovider.TaskGeneral},
	})
	require.NoError(t, err)

	require.NotNil(t, prober.probed)
	assert.Equal(t, aiprovider.PlatformBedrock, prober.probed.ResolvedPlatform())
	assert.Equal(t, "https://bedrock-mantle.us-east-1.api.aws/anthropic", prober.probed.ResolvedBaseURL())
	assert.Equal(t, serviceSecretKey, prober.seen)
}
