package aiprovider_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiproviderchange"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serviceAccountJSON = `{
  "type": "service_account",
  "project_id": "freight-ops",
  "private_key_id": "0123456789abcdef0123456789abcdef01234567",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n",
  "client_email": "trenova@freight-ops.iam.gserviceaccount.com",
  "token_uri": "https://oauth2.googleapis.com/token"
}`

func platformProvider(
	platform aiplatform.Platform,
	settings aiplatform.PlatformSettings,
) *aiprovider.Provider {
	p := validProvider()
	p.Platform = platform
	p.PlatformSettings = &settings
	p.APIKey = "encrypted"

	return p
}

func credentialErrors(p *aiprovider.Provider, secret string) map[string]bool {
	multiErr := errortypes.NewMultiError()
	p.ValidateCredential(secret, multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestPlatform_EachPlatformIsReachedAtItsOwnAddress(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		platform aiplatform.Platform
		settings aiplatform.PlatformSettings
		baseURL  string
		want     string
	}{
		{
			name:     "bedrock",
			platform: aiplatform.PlatformBedrock,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: "AKIAIOSFODNN7EXAMPLE"},
			want:     "https://bedrock-mantle.us-east-1.api.aws/anthropic",
		},
		{
			name:     "bedrock private endpoint",
			platform: aiplatform.PlatformBedrock,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "eu-west-1"},
			baseURL:  "https://vpce-0abc.bedrock-mantle.eu-west-1.vpce.amazonaws.com/anthropic/",
			want:     "https://vpce-0abc.bedrock-mantle.eu-west-1.vpce.amazonaws.com/anthropic",
		},
		{
			name:     "claude platform on aws",
			platform: aiplatform.PlatformClaudeAWS,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "us-west-2", WorkspaceID: "wrkspc_01ABC"},
			want:     "https://aws-external-anthropic.us-west-2.api.aws",
		},
		{
			name:     "vertex regional",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops"},
			want:     "https://us-east5-aiplatform.googleapis.com",
		},
		{
			name:     "vertex global",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, Region: "global", ProjectID: "freight-ops"},
			want:     "https://aiplatform.googleapis.com",
		},
		{
			name:     "vertex multi-region",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, Region: "eu", ProjectID: "freight-ops"},
			want:     "https://aiplatform.eu.rep.googleapis.com",
		},
		{
			name:     "claude platform on google cloud",
			platform: aiplatform.PlatformClaudeGoogleCloud,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, ProjectID: "freight-ops", WorkspaceID: "wrkspc_01ABC"},
			want:     "https://claude.googleapis.com/v1alpha/projects/freight-ops/locations/global/workspaces/wrkspc_01ABC/invoke",
		},
		{
			name:     "foundry",
			platform: aiplatform.PlatformFoundry,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Resource: "trenova-ai"},
			want:     "https://trenova-ai.services.ai.azure.com/anthropic",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := platformProvider(tc.platform, tc.settings)
			p.BaseURL = tc.baseURL

			assert.Equal(t, tc.want, p.ResolvedBaseURL())
			assert.Empty(t, fieldErrors(t, p))
		})
	}
}

/*
Every setting ends up in a host name or a URL path. A region that is anything
but a region would send the call, and the credential with it, to an address of
the sender's choosing.
*/
func TestPlatform_RefusesSettingsThatWouldChangeTheAddress(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		platform aiplatform.Platform
		settings aiplatform.PlatformSettings
		field    string
	}{
		{
			name:     "region with a host",
			platform: aiplatform.PlatformBedrock,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1.evil.example"},
			field:    "platformSettings.region",
		},
		{
			name:     "region with a path",
			platform: aiplatform.PlatformClaudeAWS,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1/x", WorkspaceID: "w"},
			field:    "platformSettings.region",
		},
		{
			name:     "vertex region with an at sign",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, Region: "evil.example@us-east5", ProjectID: "freight-ops"},
			field:    "platformSettings.region",
		},
		{
			name:     "project with a slash",
			platform: aiplatform.PlatformClaudeGoogleCloud,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, ProjectID: "a/../../b", WorkspaceID: "w"},
			field:    "platformSettings.projectId",
		},
		{
			name:     "workspace with a query",
			platform: aiplatform.PlatformClaudeGoogleCloud,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, ProjectID: "freight-ops", WorkspaceID: "w?x=1"},
			field:    "platformSettings.workspaceId",
		},
		{
			name:     "foundry resource with a dot",
			platform: aiplatform.PlatformFoundry,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Resource: "evil.example"},
			field:    "platformSettings.resource",
		},
		{
			name:     "entra tenant with a path",
			platform: aiplatform.PlatformFoundry,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthEntraID, Resource: "r", TenantID: "evil.example/x", ClientID: "11111111-2222-3333-4444-555555555555"},
			field:    "platformSettings.tenantId",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := platformProvider(tc.platform, tc.settings)

			assert.True(t, fieldErrors(t, p)[tc.field], "expected an error on %s", tc.field)
			assert.NotContains(t, p.ResolvedBaseURL(), "evil")
		})
	}
}

func TestPlatform_EachPlatformNamesWhatItNeeds(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		platform aiplatform.Platform
		settings aiplatform.PlatformSettings
		fields   []string
	}{
		{
			name:     "bedrock without a region or access key id",
			platform: aiplatform.PlatformBedrock,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAccessKey},
			fields:   []string{"platformSettings.region", "platformSettings.accessKeyId"},
		},
		{
			name:     "claude on aws without a workspace",
			platform: aiplatform.PlatformClaudeAWS,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1"},
			fields:   []string{"platformSettings.workspaceId"},
		},
		{
			name:     "vertex without a project",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthServiceAccount, Region: "us-east5"},
			fields:   []string{"platformSettings.projectId"},
		},
		{
			name:     "foundry without a resource or address",
			platform: aiplatform.PlatformFoundry,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey},
			fields:   []string{"platformSettings.resource"},
		},
		{
			name:     "entra id without its identity",
			platform: aiplatform.PlatformFoundry,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthEntraID, Resource: "r"},
			fields:   []string{"platformSettings.tenantId", "platformSettings.clientId"},
		},
		{
			name:     "a sign-in the platform does not take",
			platform: aiplatform.PlatformVertexAI,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east5", ProjectID: "freight-ops"},
			fields:   []string{"platformSettings.auth"},
		},
		{
			name:     "a setting another platform takes",
			platform: aiplatform.PlatformBedrock,
			settings: aiplatform.PlatformSettings{Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1", ProjectID: "freight-ops", WorkspaceID: "w"},
			fields:   []string{"platformSettings.projectId", "platformSettings.workspaceId"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errs := fieldErrors(t, platformProvider(tc.platform, tc.settings))
			for _, field := range tc.fields {
				assert.True(t, errs[field], "expected an error on %s, got %v", field, errs)
			}
		})
	}
}

func TestPlatform_OnlyAnthropicMessagesProvidersTakeOne(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.BaseURL = "https://api.example.com/v1"
	p.Platform = aiplatform.PlatformBedrock

	assert.True(t, fieldErrors(t, p)["platform"])
	assert.Equal(
		t,
		aiplatform.PlatformDirect,
		p.ResolvedPlatform(),
		"another kind is always called directly",
	)

	p.Platform = aiplatform.PlatformDirect
	p.PlatformSettings = &aiplatform.PlatformSettings{Region: "us-east-1"}
	assert.True(t, fieldErrors(t, p)["platformSettings"])
}

func TestPlatform_ServerCredentialsStoreNoKey(t *testing.T) {
	t.Parallel()

	p := platformProvider(aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServer, Region: "us-east-1",
	})
	p.APIKey = ""

	assert.Empty(t, fieldErrors(t, p))
	assert.False(t, p.RequiresCredential())
	assert.True(t, p.UsesServerCredentials())
	assert.True(t, credentialErrors(p, "anything")["apiKey"])

	p.PlatformSettings.Auth = aiplatform.PlatformAuthAccessKey
	p.PlatformSettings.AccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	assert.True(t, fieldErrors(t, p)["apiKey"], "an access key sign-in needs its secret")
}

/*
Google's credential loader takes more than service account keys: an external
account configuration can read a file on the server or call any address it
names, and a service account key names its own token endpoint. Only a service
account key that mints its tokens at Google is taken.
*/
func TestPlatform_TakesOnlyAServiceAccountKeyThatTalksToGoogle(t *testing.T) {
	t.Parallel()

	p := platformProvider(aiplatform.PlatformVertexAI, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops",
	})

	assert.Empty(t, credentialErrors(p, serviceAccountJSON))

	for name, secret := range map[string]string{
		"not json":         "not json",
		"external account": `{"type":"external_account","credential_source":{"file":"/etc/passwd"},"token_url":"https://sts.googleapis.com/v1/token"}`,
		"impersonated":     `{"type":"impersonated_service_account","service_account_impersonation_url":"http://169.254.169.254/"}`,
		"token endpoint":   `{"type":"service_account","private_key_id":"k","private_key":"-----BEGIN PRIVATE KEY-----","client_email":"a@b","token_uri":"http://10.0.0.5/token"}`,
		"missing key":      `{"type":"service_account","client_email":"a@b"}`,
	} {
		assert.True(t, credentialErrors(p, secret)["apiKey"], name)
	}
}

func TestPlatform_ChecksTheShapeOfEachSecret(t *testing.T) {
	t.Parallel()

	accessKey := platformProvider(aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
	})
	assert.Empty(t, credentialErrors(accessKey, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"))
	assert.True(t, credentialErrors(accessKey, "has spaces in it and more")["apiKey"])

	apiKey := platformProvider(aiplatform.PlatformFoundry, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthAPIKey, Resource: "r",
	})
	assert.Empty(t, credentialErrors(apiKey, "abc123"))
	assert.True(t, credentialErrors(apiKey, "abc\r\nX-Injected: 1")["apiKey"])
}

/*
A credential is entered for one place and one identity. Moving the provider to
another platform, region, project, workspace or tenant while keeping the key
would send it somewhere it was never meant for.
*/
func TestPlatform_SameCredentialTargetFollowsWhereTheKeyGoes(t *testing.T) {
	t.Parallel()

	base := func() *aiprovider.Provider {
		return platformProvider(aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
		})
	}

	assert.True(t, aiproviderchange.SameCredentialTarget(base(), base()))

	moved := base()
	moved.PlatformSettings.Region = "eu-west-1"
	assert.False(t, aiproviderchange.SameCredentialTarget(base(), moved))

	otherKey := base()
	otherKey.PlatformSettings.AccessKeyID = "AKIAI44QH8DHBEXAMPLE"
	assert.False(t, aiproviderchange.SameCredentialTarget(base(), otherKey))

	direct := validProvider()
	assert.False(t, aiproviderchange.SameCredentialTarget(base(), direct))

	sameDirect := validProvider()
	sameDirect.BaseURL = "https://api.anthropic.com/"
	assert.True(t, aiproviderchange.SameCredentialTarget(validProvider(), sameDirect))
}

func TestPlatform_FingerprintsAServiceAccountKeyByItsKeyID(t *testing.T) {
	t.Parallel()

	p := platformProvider(aiplatform.PlatformVertexAI, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops",
	})
	assert.Equal(t,
		"0123456789abcdef0123456789abcdef01234567",
		p.CredentialFingerprintSubject(serviceAccountJSON),
	)

	assert.Equal(t, "sk-ant-x", validProvider().CredentialFingerprintSubject("sk-ant-x"))
}

func TestPlatform_NormalizesWhatIsStored(t *testing.T) {
	t.Parallel()

	p := platformProvider(aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Region: " US-EAST-1 ", AccessKeyID: " akiaiosfodnn7example ",
	})
	require.Empty(t, fieldErrors(t, p))
	assert.Equal(t, "us-east-1", p.PlatformSettings.Region)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", p.PlatformSettings.AccessKeyID)
	assert.Equal(
		t,
		aiplatform.PlatformAuthAccessKey,
		p.PlatformSettings.Auth,
		"the platform's default sign-in",
	)

	direct := validProvider()
	direct.PlatformSettings = &aiplatform.PlatformSettings{Region: "us-east-1"}
	assert.True(
		t,
		fieldErrors(t, direct)["platformSettings"],
		"a provider called directly takes no platform settings",
	)
}

func TestPlatform_VendorIsWhoServesTheCall(t *testing.T) {
	t.Parallel()

	for platform, vendor := range map[aiplatform.Platform]string{
		aiplatform.PlatformDirect:            "anthropic",
		aiplatform.PlatformBedrock:           "bedrock",
		aiplatform.PlatformVertexAI:          "vertex",
		aiplatform.PlatformClaudeAWS:         "anthropic",
		aiplatform.PlatformClaudeGoogleCloud: "anthropic",
		aiplatform.PlatformFoundry:           "foundry",
	} {
		p := validProvider()
		p.Platform = platform
		assert.Equal(t, vendor, p.Vendor(), platform)
	}
}
