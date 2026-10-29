package anthropicadapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/lrucache"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/google"
)

const (
	bedrockSigningService   = "bedrock-mantle"
	claudeAWSSigningService = "aws-external-anthropic"
	googleCloudScope        = "https://www.googleapis.com/auth/cloud-platform"
	entraIDScope            = "https://ai.azure.com/.default"
	entraIDTokenURL         = "https://login.microsoftonline.com/%s/oauth2/v2.0/token" //nolint:gosec // a public endpoint, not a credential
	vertexAPIVersion        = "vertex-2023-10-16"
	workspaceHeader         = "anthropic-workspace-id"
	platformCredentialSlots = 256
)

// serverCredentialsRefused is a provider that would sign in with the server's
// own cloud identity on a server that does not allow it. It is not retried:
// nothing about the next attempt would be different.
func serverCredentialsRefused() error {
	return errortypes.NewBusinessError(
		"This server does not let providers sign in with its own cloud credentials",
	)
}

/*
platformCredentials keeps what signing in to a platform costs to set up.

A Google or Entra ID token is minted by a call of its own and lasts about an
hour, and the server's AWS identity is found by reading files and asking the
instance metadata service. Done on every model call, that is a second network
round trip and a token endpoint rate limit per turn, so the token source or
credentials provider is kept, under a digest of everything that makes it what
it is, and refreshes itself when its token runs out. A changed key, region or
identity is a different digest, so a rotated credential is never served from
here.
*/
type platformCredentials struct {
	tokens lrucache.Cache[string, oauth2.TokenSource]
	aws    lrucache.Cache[string, aws.CredentialsProvider]
}

func NewPlatformCredentials() *platformCredentials {
	tokens, _ := lrucache.New[string, oauth2.TokenSource](platformCredentialSlots)
	awsCreds, _ := lrucache.New[string, aws.CredentialsProvider](platformCredentialSlots)

	return &platformCredentials{tokens: tokens, aws: awsCreds}
}

func credentialDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// platformOptions are what the provider's platform adds to a request: how it
// signs in, and for Vertex AI, the path and body its publisher API takes.
// Signing comes last so it covers every header set before it.
func (c *platformCredentials) PlatformOptions(
	ctx context.Context,
	call *modeladapter.Call,
) ([]option.RequestOption, error) {
	provider := call.Provider
	platform := provider.ResolvedPlatform()
	settings := provider.ResolvedSettings()
	if platform == aiplatform.PlatformDirect {
		return apiKeyOptions(call.APIKey), nil
	}
	if settings.Auth == aiplatform.PlatformAuthServer && !call.ServerCredentialsAllowed {
		return nil, serverCredentialsRefused()
	}

	switch platform {
	case aiplatform.PlatformBedrock:
		return c.awsOptions(ctx, call, bedrockSigningService, bearerKeyOption)
	case aiplatform.PlatformClaudeAWS:
		opts, err := c.awsOptions(ctx, call, claudeAWSSigningService, apiKeyOptions)
		if err != nil {
			return nil, err
		}
		workspace := option.WithHeader(workspaceHeader, settings.WorkspaceID)

		return append([]option.RequestOption{workspace}, opts...), nil
	case aiplatform.PlatformVertexAI:
		source, err := c.googleTokens(ctx, call)
		if err != nil {
			return nil, err
		}
		return []option.RequestOption{option.WithMiddleware(
			vertexMiddleware(settings.Region, settings.ProjectID),
			bearerMiddleware(source),
		)}, nil
	case aiplatform.PlatformClaudeGoogleCloud:
		source, err := c.googleTokens(ctx, call)
		if err != nil {
			return nil, err
		}
		return []option.RequestOption{option.WithMiddleware(bearerMiddleware(source))}, nil
	case aiplatform.PlatformFoundry:
		if settings.Auth == aiplatform.PlatformAuthEntraID {
			return []option.RequestOption{
				option.WithMiddleware(bearerMiddleware(c.entraTokens(call))),
			}, nil
		}
		return apiKeyOptions(call.APIKey), nil
	case aiplatform.PlatformDirect:
		return apiKeyOptions(call.APIKey), nil
	default:
		return nil, fmt.Errorf("no way to reach platform %q", platform)
	}
}

func apiKeyOptions(key string) []option.RequestOption {
	if key == "" {
		return nil
	}

	return []option.RequestOption{option.WithAPIKey(key)}
}

func bearerKeyOption(key string) []option.RequestOption {
	if key == "" {
		return nil
	}

	return []option.RequestOption{option.WithHeader("Authorization", "Bearer "+key)}
}

// awsOptions signs a request with an access key or the server's identity,
// or sends the platform's API key in the form it takes.
func (c *platformCredentials) awsOptions(
	ctx context.Context,
	call *modeladapter.Call,
	service string,
	keyOptions func(string) []option.RequestOption,
) ([]option.RequestOption, error) {
	settings := call.Provider.ResolvedSettings()
	switch settings.Auth {
	case aiplatform.PlatformAuthAPIKey:
		return keyOptions(call.APIKey), nil
	case aiplatform.PlatformAuthAccessKey:
		provider := credentials.NewStaticCredentialsProvider(settings.AccessKeyID, call.APIKey, "")
		return []option.RequestOption{
			option.WithMiddleware(sigV4Middleware(provider, service, settings.Region)),
		}, nil
	case aiplatform.PlatformAuthServer:
		provider, err := c.serverAWS(ctx, settings.Region)
		if err != nil {
			return nil, err
		}
		return []option.RequestOption{
			option.WithMiddleware(sigV4Middleware(provider, service, settings.Region)),
		}, nil
	case aiplatform.PlatformAuthServiceAccount, aiplatform.PlatformAuthEntraID:
		return nil, fmt.Errorf("an AWS platform does not sign in with %s", settings.Auth)
	default:
		return nil, fmt.Errorf("an AWS platform does not sign in with %s", settings.Auth)
	}
}

// serverAWS is the identity of the server Trenova runs on: its environment,
// shared configuration, or instance role, found the way every AWS tool finds
// it. The instance metadata service is a link-local address the providers'
// egress policy refuses, so it is reached with the AWS SDK's own client.
func (c *platformCredentials) serverAWS(
	ctx context.Context,
	region string,
) (aws.CredentialsProvider, error) {
	key := credentialDigest("aws-server", region)
	if cached, ok := c.aws.Get(key); ok {
		return cached, nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load the server's AWS configuration: %w", err)
	}
	if _, err = cfg.Credentials.Retrieve(ctx); err != nil {
		return nil, &modeladapter.TransportError{
			StatusCode: http.StatusUnauthorized,
			Message:    "the server has no AWS credentials to sign in with: " + err.Error(),
		}
	}
	c.aws.Add(key, cfg.Credentials)

	return cfg.Credentials, nil
}

// googleTokens mints Google access tokens from the provider's service
// account key, or from the server's application default credentials. The key
// is checked again here, as it was when it was saved: only a service account
// key that gets its tokens from Google is ever loaded.
func (c *platformCredentials) googleTokens(
	ctx context.Context,
	call *modeladapter.Call,
) (oauth2.TokenSource, error) {
	settings := call.Provider.ResolvedSettings()
	if settings.Auth == aiplatform.PlatformAuthServer {
		key := credentialDigest("google-server")
		if cached, ok := c.tokens.Get(key); ok {
			return cached, nil
		}
		creds, err := google.FindDefaultCredentials(ctx, googleCloudScope)
		if err != nil {
			return nil, &modeladapter.TransportError{
				StatusCode: http.StatusUnauthorized,
				Message:    "the server has no Google credentials to sign in with: " + err.Error(),
			}
		}
		c.tokens.Add(key, creds.TokenSource)

		return creds.TokenSource, nil
	}

	key := credentialDigest("google-service-account", call.APIKey)
	if cached, ok := c.tokens.Get(key); ok {
		return cached, nil
	}

	multiErr := errortypes.NewMultiError()
	call.Provider.ValidateCredential(call.APIKey, multiErr)
	if multiErr.HasErrors() || strings.TrimSpace(call.APIKey) == "" {
		return nil, &modeladapter.TransportError{
			StatusCode: http.StatusUnauthorized,
			Message:    "the stored credential is not a usable Google service account key",
		}
	}

	creds, err := google.CredentialsFromJSONWithTypeAndParams(
		tokenContext(call.Client),
		[]byte(call.APIKey),
		google.ServiceAccount,
		google.CredentialsParams{Scopes: []string{googleCloudScope}},
	)
	if err != nil {
		return nil, &modeladapter.TransportError{
			StatusCode: http.StatusUnauthorized,
			Message:    "the service account key could not be read: " + err.Error(),
		}
	}
	c.tokens.Add(key, creds.TokenSource)

	return creds.TokenSource, nil
}

// entraTokens takes tokens for an Entra ID app registration from the
// tenant's token endpoint.
func (c *platformCredentials) entraTokens(call *modeladapter.Call) oauth2.TokenSource {
	settings := call.Provider.ResolvedSettings()
	key := credentialDigest("entra-id", settings.TenantID, settings.ClientID, call.APIKey)
	if cached, ok := c.tokens.Get(key); ok {
		return cached
	}

	cfg := clientcredentials.Config{
		ClientID:     settings.ClientID,
		ClientSecret: call.APIKey,
		TokenURL:     fmt.Sprintf(entraIDTokenURL, url.PathEscape(settings.TenantID)),
		Scopes:       []string{entraIDScope},
		AuthStyle:    oauth2.AuthStyleInParams,
	}
	source := oauth2.ReuseTokenSource(nil, cfg.TokenSource(tokenContext(call.Client)))
	c.tokens.Add(key, source)

	return source
}

// tokenContext carries the provider's HTTP client to the token endpoint, so a
// token is fetched under the same egress policy as the call it is for. It is
// not the request's context: a token source outlives the call that made it and
// refreshes on a later one.
func tokenContext(client *http.Client) context.Context {
	ctx := context.Background()
	if client == nil {
		return ctx
	}

	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

// bearerMiddleware puts an access token on a request. A token endpoint that
// refuses the credential is the provider refusing it, which the router reads
// as a refused key rather than an outage.
func bearerMiddleware(source oauth2.TokenSource) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		token, err := source.Token()
		if err != nil {
			return nil, tokenError(err)
		}
		token.SetAuthHeader(req)

		return next(req)
	}
}

func tokenError(err error) error {
	var refused *oauth2.RetrieveError
	if errors.As(err, &refused) && refused.Response != nil {
		status := refused.Response.StatusCode
		if status >= http.StatusInternalServerError || status == http.StatusTooManyRequests {
			return &modeladapter.TransportError{
				StatusCode: status,
				Retryable:  true,
				Message:    "the platform's token endpoint failed: " + refused.Error(),
			}
		}

		return &modeladapter.TransportError{
			StatusCode: http.StatusUnauthorized,
			Message:    "the platform refused the credential: " + refused.Error(),
		}
	}

	return fmt.Errorf("get an access token: %w", err)
}

// sigV4Middleware signs a request for an AWS service. It runs last, so the
// signature covers every header the request carries.
func sigV4Middleware(
	provider aws.CredentialsProvider,
	service string,
	region string,
) option.Middleware {
	signer := v4.NewSigner()

	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		body, err := readRequestBody(req)
		if err != nil {
			return nil, err
		}

		creds, err := provider.Retrieve(req.Context())
		if err != nil {
			return nil, fmt.Errorf("retrieve AWS credentials: %w", err)
		}

		digest := sha256.Sum256(body)
		if err = signer.SignHTTP(
			req.Context(),
			creds,
			req,
			hex.EncodeToString(digest[:]),
			service,
			region,
			time.Now(),
		); err != nil {
			return nil, fmt.Errorf("sign request: %w", err)
		}

		return next(req)
	}
}

// vertexMiddleware turns a Messages API request into Vertex AI's publisher
// model call: the model moves from the body into the path, which names the
// project and region, and the body says which Vertex version of the API it
// speaks. A base URL entered for a private endpoint keeps its own path in
// front.
func vertexMiddleware(region, projectID string) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		body, err := readRequestBody(req)
		if err != nil {
			return nil, err
		}
		if req.Method != http.MethodPost || !gjson.ValidBytes(body) {
			return next(req)
		}

		prefix, suffix, ok := vertexEndpoint(req.URL.Path)
		if !ok {
			return next(req)
		}
		escapedPrefix := strings.TrimSuffix(req.URL.EscapedPath(), suffix)

		if !gjson.GetBytes(body, "anthropic_version").Exists() {
			if body, err = sjson.SetBytes(body, "anthropic_version", vertexAPIVersion); err != nil {
				return nil, fmt.Errorf("set the Vertex API version: %w", err)
			}
		}

		model, method := "count-tokens", "rawPredict"
		if suffix == vertexMessagesPath {
			model = gjson.GetBytes(body, "model").String()
			if model == "" {
				return nil, errors.New("a Vertex AI request names no model")
			}
			if gjson.GetBytes(body, "stream").Bool() {
				method = "streamRawPredict"
			}
			if body, err = sjson.DeleteBytes(body, "model"); err != nil {
				return nil, fmt.Errorf("move the model into the path: %w", err)
			}
		}

		setRequestBody(req, body)
		req.URL.Path = prefix + vertexModelPath(projectID, region, model, false) + ":" + method
		req.URL.RawPath = escapedPrefix +
			vertexModelPath(projectID, region, model, true) + ":" + method

		return next(req)
	}
}

const (
	vertexMessagesPath    = "/v1/messages"
	vertexCountTokensPath = "/v1/messages/count_tokens" //nolint:gosec // an API path, not a credential
)

func vertexEndpoint(path string) (prefix, suffix string, ok bool) {
	for _, candidate := range []string{vertexCountTokensPath, vertexMessagesPath} {
		if strings.HasSuffix(path, candidate) {
			return strings.TrimSuffix(path, candidate), candidate, true
		}
	}

	return "", "", false
}

func vertexModelPath(projectID, region, model string, escaped bool) string {
	segment := func(value string) string {
		if escaped {
			return url.PathEscape(value)
		}
		return value
	}

	return "/v1/projects/" + segment(projectID) +
		"/locations/" + segment(region) +
		"/publishers/anthropic/models/" + segment(model)
}

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()
	setRequestBody(req, body)

	return body, nil
}

func setRequestBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
}
