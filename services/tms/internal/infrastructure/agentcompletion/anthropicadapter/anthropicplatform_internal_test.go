package anthropicadapter

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	testSecretKey   = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

type platformRequest struct {
	path    string
	rawPath string
	query   string
	headers http.Header
	body    map[string]any
	raw     string
}

// platformServer answers every request with a finished reply and keeps what
// it was sent.
func platformServer(t *testing.T, stream bool) (*httptest.Server, func() []platformRequest) {
	t.Helper()

	var (
		mu   sync.Mutex
		seen []platformRequest
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		_ = sonic.Unmarshal(raw, &body)
		mu.Lock()
		seen = append(seen, platformRequest{
			path:    r.URL.Path,
			rawPath: r.URL.EscapedPath(),
			query:   r.URL.RawQuery,
			headers: r.Header.Clone(),
			body:    body,
			raw:     string(raw),
		})
		mu.Unlock()
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(wireStreamReply))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wireBlockingReply))
	}))
	t.Cleanup(server.Close)

	return server, func() []platformRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]platformRequest(nil), seen...)
	}
}

func newTokenServer(t *testing.T, status int, reply string) *modeladaptertest.TokenServer {
	t.Helper()

	ts := &modeladaptertest.TokenServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.Requests.Add(1)
		_ = r.ParseForm()
		ts.Mu.Lock()
		ts.Forms = append(ts.Forms, r.PostForm)
		ts.Paths = append(ts.Paths, r.URL.Path)
		ts.Mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(ts.Server.Close)

	return ts
}

func routedClient(routes map[string]*url.URL) *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: modeladaptertest.HostRouter{Routes: routes},
	}
}

func testServiceAccountKey(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	encoded, err := sonic.MarshalString(map[string]string{
		"type":           "service_account",
		"project_id":     "freight-ops",
		"private_key_id": "0123456789abcdef0123456789abcdef01234567",
		"private_key":    pemKey,
		"client_email":   "trenova@freight-ops.iam.gserviceaccount.com",
		"token_uri":      "https://oauth2.googleapis.com/token",
	})
	require.NoError(t, err)

	return encoded
}

func platformCall(
	baseURL string,
	platform aiplatform.Platform,
	settings aiplatform.PlatformSettings,
	secret string,
) *modeladapter.Call {
	call := modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		baseURL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	)
	call.Provider.Model = "claude-opus-4-8"
	call.Provider.Platform = platform
	call.Provider.PlatformSettings = &settings
	call.APIKey = secret

	return call
}

func sigV4Credential(t *testing.T, header http.Header) (credential, signedHeaders string) {
	t.Helper()

	auth := header.Get("Authorization")
	require.True(t, strings.HasPrefix(auth, "AWS4-HMAC-SHA256 "), "signed with SigV4, got %q", auth)
	for _, part := range strings.Split(strings.TrimPrefix(auth, "AWS4-HMAC-SHA256 "), ", ") {
		name, value, _ := strings.Cut(part, "=")
		switch name {
		case "Credential":
			credential = value
		case "SignedHeaders":
			signedHeaders = value
		}
	}

	return credential, signedHeaders
}

/*
On Bedrock an IAM access key signs each request; the secret access key never
travels. It is not sent as an API key, not put in a header and not put in the
body, and the signature names the access key, the region and Bedrock's service.
*/
func TestPlatform_BedrockSignsWithTheAccessKeyAndNeverSendsTheSecret(t *testing.T) {
	t.Parallel()

	server, seen := platformServer(t, false)
	call := platformCall(
		server.URL+"/anthropic",
		aiplatform.PlatformBedrock,
		aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: testAccessKeyID,
		},
		testSecretKey,
	)

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 1)
	req := requests[0]
	assert.Equal(t, "/anthropic/v1/messages", req.path)
	credential, signed := sigV4Credential(t, req.headers)
	assert.True(t, strings.HasPrefix(credential, testAccessKeyID+"/"))
	assert.True(t, strings.HasSuffix(credential, "/us-east-1/bedrock-mantle/aws4_request"))
	assert.Contains(t, signed, "host")
	assert.Contains(t, signed, "x-amz-date")
	assert.Contains(t, signed, "anthropic-version")
	assert.Empty(t, req.headers.Get("X-Api-Key"))
	for name, values := range req.headers {
		for _, value := range values {
			assert.NotContains(t, value, testSecretKey, "header %s", name)
		}
	}
	assert.NotContains(t, req.raw, testSecretKey)
	assertValidSignature(t, server.URL, req, signed, "bedrock-mantle", "us-east-1")
}

// assertValidSignature signs the request again as AWS would on receipt, from
// the headers it says it signed and the body that arrived, and requires the
// same signature: a header changed after signing would fail here as it would
// at AWS.
func assertValidSignature(
	t *testing.T,
	serverURL string,
	req platformRequest,
	signedHeaders string,
	service string,
	region string,
) {
	t.Helper()

	signedAt, err := time.Parse("20060102T150405Z", req.headers.Get("X-Amz-Date"))
	require.NoError(t, err)

	target := serverURL + req.rawPath
	if req.query != "" {
		target += "?" + req.query
	}
	resign, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		target,
		strings.NewReader(req.raw),
	)
	require.NoError(t, err)
	for _, name := range strings.Split(signedHeaders, ";") {
		if name == "host" {
			continue
		}
		resign.Header.Set(name, req.headers.Get(name))
	}
	resign.ContentLength = int64(len(req.raw))

	digest := sha256.Sum256([]byte(req.raw))
	require.NoError(t, v4.NewSigner().SignHTTP(
		t.Context(),
		aws.Credentials{AccessKeyID: testAccessKeyID, SecretAccessKey: testSecretKey},
		resign,
		hex.EncodeToString(digest[:]),
		service,
		region,
		signedAt,
	))

	assert.Equal(t, resign.Header.Get("Authorization"), req.headers.Get("Authorization"))
}

func TestPlatform_BedrockSendsItsAPIKeyAsABearerToken(t *testing.T) {
	t.Parallel()

	server, seen := platformServer(t, true)
	call := platformCall(server.URL, aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1",
	}, "bedrock-api-key-123")

	_, deltas := modeladaptertest.StreamWith(t, NewAnthropicAdapter(), call)
	assert.Empty(t, deltas)

	requests := seen()
	require.Len(t, requests, 1)
	assert.Equal(t, "Bearer bedrock-api-key-123", requests[0].headers.Get("Authorization"))
	assert.Empty(t, requests[0].headers.Get("X-Api-Key"))
	assert.Equal(t, true, requests[0].body["stream"])
}

// Claude Platform on AWS names the workspace on every request, and an access
// key's signature covers it, so it cannot be changed on the way.
func TestPlatform_ClaudeOnAWSNamesTheWorkspace(t *testing.T) {
	t.Parallel()

	server, seen := platformServer(t, false)

	keyed := platformCall(server.URL, aiplatform.PlatformClaudeAWS, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthAPIKey, Region: "us-west-2", WorkspaceID: "wrkspc_01ABC",
	}, "sk-ant-aws")
	_, err := NewAnthropicAdapter().Complete(t.Context(), keyed)
	require.NoError(t, err)

	signedCall := platformCall(
		server.URL,
		aiplatform.PlatformClaudeAWS,
		aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthAccessKey, Region: "us-west-2", WorkspaceID: "wrkspc_01ABC",
			AccessKeyID: testAccessKeyID,
		},
		testSecretKey,
	)
	_, err = NewAnthropicAdapter().Complete(t.Context(), signedCall)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 2)
	assert.Equal(t, "sk-ant-aws", requests[0].headers.Get("X-Api-Key"))
	assert.Equal(t, "wrkspc_01ABC", requests[0].headers.Get("Anthropic-Workspace-Id"))

	credential, signed := sigV4Credential(t, requests[1].headers)
	assert.True(t, strings.HasSuffix(credential, "/us-west-2/aws-external-anthropic/aws4_request"))
	assert.Contains(t, signed, "anthropic-workspace-id")
	assert.Equal(t, "wrkspc_01ABC", requests[1].headers.Get("Anthropic-Workspace-Id"))
	assertValidSignature(t, server.URL, requests[1], signed, "aws-external-anthropic", "us-west-2")
	assert.Empty(t, requests[1].headers.Get("X-Api-Key"))
}

/*
Vertex AI serves Claude as a publisher model: the model moves from the body
into the path, which names the project and region, the body names the Vertex
version of the API, and a streamed call goes to streamRawPredict. The token is
minted from the service account key at Google's token endpoint, once.
*/
func TestPlatform_VertexCallsThePublisherModelWithAGoogleToken(t *testing.T) {
	t.Parallel()

	tokens := newTokenServer(t, http.StatusOK,
		`{"access_token":"ya29.vertex","token_type":"Bearer","expires_in":3600}`)
	server, seen := platformServer(t, false)
	streamingServer, streamSeen := platformServer(t, true)

	adapter := NewAnthropicAdapter()
	key := testServiceAccountKey(t)
	settings := aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServiceAccount, Region: "us-east5", ProjectID: "freight-ops",
	}
	routes := map[string]*url.URL{"oauth2.googleapis.com": tokens.URL(t)}

	call := platformCall(server.URL, aiplatform.PlatformVertexAI, settings, key)
	call.Client = routedClient(routes)
	_, err := adapter.Complete(t.Context(), call)
	require.NoError(t, err)

	streamed := platformCall(streamingServer.URL, aiplatform.PlatformVertexAI, settings, key)
	streamed.Client = routedClient(routes)
	streamed.Provider.Model = "claude-sonnet-4-6@20260101"
	_, _ = modeladaptertest.StreamWith(t, adapter, streamed)

	requests := seen()
	require.Len(t, requests, 1)
	req := requests[0]
	assert.Equal(
		t,
		"/v1/projects/freight-ops/locations/us-east5/publishers/anthropic/models/claude-opus-4-8:rawPredict",
		req.path,
	)
	assert.Equal(t, "beta=true", req.query)
	assert.Equal(t, "vertex-2023-10-16", req.body["anthropic_version"])
	assert.NotContains(t, req.body, "model")
	assert.Equal(t, "Bearer ya29.vertex", req.headers.Get("Authorization"))
	assert.Empty(t, req.headers.Get("X-Api-Key"))

	streamRequests := streamSeen()
	require.Len(t, streamRequests, 1)
	assert.Equal(
		t,
		"/v1/projects/freight-ops/locations/us-east5/publishers/anthropic/models/claude-sonnet-4-6@20260101:streamRawPredict",
		streamRequests[0].path,
	)

	assert.Equal(t, int32(1), tokens.Requests.Load(), "the token is minted once and kept")
	tokens.Mu.Lock()
	defer tokens.Mu.Unlock()
	assert.Equal(
		t,
		"urn:ietf:params:oauth:grant-type:jwt-bearer",
		tokens.Forms[0].Get("grant_type"),
	)
}

// A model name is a path segment, never a path: whatever it holds stays in
// its one segment on Google's host.
func TestPlatform_VertexKeepsTheModelInOneSegment(t *testing.T) {
	t.Parallel()

	tokens := newTokenServer(t, http.StatusOK,
		`{"access_token":"ya29.vertex","token_type":"Bearer","expires_in":3600}`)
	server, seen := platformServer(t, false)

	call := platformCall(server.URL, aiplatform.PlatformVertexAI, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServiceAccount, Region: "global", ProjectID: "freight-ops",
	}, testServiceAccountKey(t))
	call.Client = routedClient(map[string]*url.URL{"oauth2.googleapis.com": tokens.URL(t)})
	call.Provider.Model = "../../evil?x=1"

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 1)
	assert.Equal(
		t,
		"/v1/projects/freight-ops/locations/global/publishers/anthropic/models/..%2F..%2Fevil%3Fx=1:rawPredict",
		requests[0].rawPath,
	)
	assert.Equal(t, "beta=true", requests[0].query)
}

func TestPlatform_ClaudeOnGoogleCloudCallsTheGatewayWithAGoogleToken(t *testing.T) {
	t.Parallel()

	tokens := newTokenServer(t, http.StatusOK,
		`{"access_token":"ya29.gateway","token_type":"Bearer","expires_in":3600}`)
	server, seen := platformServer(t, false)

	call := platformCall(
		server.URL+"/v1alpha/projects/freight-ops/locations/global/workspaces/wrkspc_01ABC/invoke",
		aiplatform.PlatformClaudeGoogleCloud,
		aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthServiceAccount, ProjectID: "freight-ops", WorkspaceID: "wrkspc_01ABC",
		},
		testServiceAccountKey(t),
	)
	call.Client = routedClient(map[string]*url.URL{"oauth2.googleapis.com": tokens.URL(t)})

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 1)
	assert.Equal(t,
		"/v1alpha/projects/freight-ops/locations/global/workspaces/wrkspc_01ABC/invoke/v1/messages",
		requests[0].path)
	assert.Equal(t, "claude-opus-4-8", requests[0].body["model"])
	assert.Equal(t, "Bearer ya29.gateway", requests[0].headers.Get("Authorization"))
}

func TestPlatform_FoundryTakesAKeyOrAnEntraIDToken(t *testing.T) {
	t.Parallel()

	server, seen := platformServer(t, false)
	keyed := platformCall(
		server.URL+"/anthropic",
		aiplatform.PlatformFoundry,
		aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthAPIKey, Resource: "trenova-ai",
		},
		"foundry-key",
	)
	_, err := NewAnthropicAdapter().Complete(t.Context(), keyed)
	require.NoError(t, err)

	tokens := newTokenServer(t, http.StatusOK,
		`{"access_token":"entra-token","token_type":"Bearer","expires_in":3600}`)
	entra := platformCall(
		server.URL+"/anthropic",
		aiplatform.PlatformFoundry,
		aiplatform.PlatformSettings{
			Auth:     aiplatform.PlatformAuthEntraID,
			Resource: "trenova-ai",
			TenantID: "contoso.onmicrosoft.com",
			ClientID: "11111111-2222-3333-4444-555555555555",
		},
		"client-secret",
	)
	entra.Client = routedClient(map[string]*url.URL{"login.microsoftonline.com": tokens.URL(t)})
	_, err = NewAnthropicAdapter().Complete(t.Context(), entra)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 2)
	assert.Equal(t, "/anthropic/v1/messages", requests[0].path)
	assert.Equal(t, "foundry-key", requests[0].headers.Get("X-Api-Key"))
	assert.Equal(t, "Bearer entra-token", requests[1].headers.Get("Authorization"))
	assert.Empty(t, requests[1].headers.Get("X-Api-Key"))

	tokens.Mu.Lock()
	defer tokens.Mu.Unlock()
	require.Len(t, tokens.Forms, 1)
	assert.Equal(t, "/contoso.onmicrosoft.com/oauth2/v2.0/token", tokens.Paths[0])
	assert.Equal(t, "client_credentials", tokens.Forms[0].Get("grant_type"))
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", tokens.Forms[0].Get("client_id"))
	assert.Equal(t, "client-secret", tokens.Forms[0].Get("client_secret"))
	assert.Equal(t, "https://ai.azure.com/.default", tokens.Forms[0].Get("scope"))
}

/*
A token endpoint that refuses the credential is the platform refusing the key.
It reads as a refused key, which is not retried and lets the router try the
key a rotation replaced, rather than as an outage worth hammering.
*/
func TestPlatform_ARefusedCredentialIsARefusedKey(t *testing.T) {
	t.Parallel()

	tokens := newTokenServer(t, http.StatusBadRequest,
		`{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret"}`)
	server, seen := platformServer(t, false)
	call := platformCall(server.URL, aiplatform.PlatformFoundry, aiplatform.PlatformSettings{
		Auth:     aiplatform.PlatformAuthEntraID,
		Resource: "trenova-ai",
		TenantID: "11111111-2222-3333-4444-555555555555",
		ClientID: "11111111-2222-3333-4444-555555555555",
	}, "wrong-secret")
	call.Client = routedClient(map[string]*url.URL{"login.microsoftonline.com": tokens.URL(t)})

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)

	var transport *modeladapter.TransportError
	require.ErrorAs(t, err, &transport)
	assert.Equal(t, http.StatusUnauthorized, transport.StatusCode)
	assert.False(t, transport.Retryable)
	assert.Empty(t, seen(), "nothing reaches the model without a token")
}

// A server that has not allowed its own cloud identity refuses a provider
// that would use it, before any request is made.
func TestPlatform_ServerCredentialsNeedTheServersConsent(t *testing.T) {
	t.Parallel()

	server, seen := platformServer(t, false)
	for _, platform := range []aiplatform.Platform{
		aiplatform.PlatformBedrock, aiplatform.PlatformClaudeAWS,
		aiplatform.PlatformVertexAI, aiplatform.PlatformClaudeGoogleCloud,
	} {
		call := platformCall(server.URL, platform, aiplatform.PlatformSettings{
			Auth: aiplatform.PlatformAuthServer, Region: "us-east-1", ProjectID: "freight-ops",
			WorkspaceID: "w",
		}, "")

		_, err := NewAnthropicAdapter().Complete(t.Context(), call)

		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business, platform)
		assert.False(t, modeladapter.IsRetryable(err), platform)
	}
	assert.Empty(t, seen())
}

// The server's own AWS identity, once allowed, is found the way every AWS tool
// finds it, and signs like an access key.
func TestPlatform_ServerAWSCredentialsSignWhenAllowed(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIASERVEREXAMPLE123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", testSecretKey)
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	server, seen := platformServer(t, false)
	call := platformCall(server.URL, aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthServer, Region: "eu-west-1",
	}, "")
	call.ServerCredentialsAllowed = true

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	requests := seen()
	require.Len(t, requests, 1)
	credential, _ := sigV4Credential(t, requests[0].headers)
	assert.True(t, strings.HasPrefix(credential, "AKIASERVEREXAMPLE123/"))
	assert.True(t, strings.HasSuffix(credential, "/eu-west-1/bedrock-mantle/aws4_request"))
}

/*
A provider's own access key is used even on a server whose environment holds
AWS credentials of its own: those belong to the operator, and only a provider
set up to use the server's identity, on a server that allows it, may.
*/
func TestPlatform_AnAccessKeyIgnoresTheServersAWSEnvironment(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIASERVEREXAMPLE123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "server-secret-server-secret-1234")
	t.Setenv("AWS_REGION", "ap-south-1")

	server, seen := platformServer(t, false)
	call := platformCall(server.URL, aiplatform.PlatformBedrock, aiplatform.PlatformSettings{
		Auth: aiplatform.PlatformAuthAccessKey, Region: "us-east-1", AccessKeyID: testAccessKeyID,
	}, testSecretKey)

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	credential, _ := sigV4Credential(t, seen()[0].headers)
	assert.True(t, strings.HasPrefix(credential, testAccessKeyID+"/"))
	assert.Contains(t, credential, "/us-east-1/")
}
