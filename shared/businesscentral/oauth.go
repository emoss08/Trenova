package businesscentral

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	FinancialsScope      = "https://api.businesscentral.dynamics.com/Financials.ReadWrite.All"
	OfflineAccessScope   = "offline_access"
	RefreshTokenLifetime = 90 * 24 * time.Hour
	jwtSegments          = 3
	grantAuthCode        = "authorization_code"
	grantRefreshToken    = "refresh_token"
	credentialProbeToken = "trenova-credential-check"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresIn    time.Duration
	ExtExpiresIn time.Duration
}

type OAuthClient struct {
	cfg      OAuthConfig
	scope    string
	loginURL string
	identity *restx.Client
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	ExtExpiresIn int64  `json:"ext_expires_in"`
}

type accessTokenClaims struct {
	TenantID string `json:"tid"`
}

func DefaultScopes() []string {
	return []string{FinancialsScope, OfflineAccessScope}
}

func NewOAuthClient(cfg OAuthConfig, opts ...Option) (*OAuthClient, error) {
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	if cfg.ClientID == "" {
		return nil, ErrClientIDRequired
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, ErrClientSecretRequired
	}
	if cfg.RedirectURL == "" {
		return nil, ErrRedirectURLRequired
	}

	scopes := make([]string, 0, len(cfg.Scopes))
	for _, scope := range cfg.Scopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes()
	}
	cfg.Scopes = scopes

	settings := resolveOptions(opts)
	identity, err := restx.New(restx.Config{
		BaseURL:      settings.loginURL,
		Timeout:      settings.timeout,
		UserAgent:    settings.userAgent,
		HTTPClient:   settings.httpClient,
		Retry:        restx.RetryConfig{},
		Observer:     settings.observer,
		ErrorDecoder: decodeOAuthError,
	})
	if err != nil {
		return nil, fmt.Errorf("businesscentral: configure identity transport: %w", err)
	}

	return &OAuthClient{
		cfg:      cfg,
		scope:    strings.Join(scopes, " "),
		loginURL: settings.loginURL,
		identity: identity,
	}, nil
}

func (c *OAuthClient) AuthorizeURL(state string) (string, error) {
	if strings.TrimSpace(state) == "" {
		return "", ErrStateRequired
	}

	target, err := url.Parse(strings.TrimRight(c.loginURL, "/") + authorizePath)
	if err != nil {
		return "", fmt.Errorf("businesscentral: parse authorize url: %w", err)
	}
	query := url.Values{}
	query.Set("client_id", c.cfg.ClientID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", c.cfg.RedirectURL)
	query.Set("response_mode", "query")
	query.Set("scope", c.scope)
	query.Set("state", state)
	query.Set("prompt", "select_account")
	target.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")

	return target.String(), nil
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrCodeRequired
	}
	return c.requestToken(ctx, url.Values{
		"grant_type":   {grantAuthCode},
		"code":         {code},
		"redirect_uri": {c.cfg.RedirectURL},
	}, code)
}

func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrTokenRequired
	}
	return c.requestToken(ctx, url.Values{
		"grant_type":    {grantRefreshToken},
		"refresh_token": {refreshToken},
		"redirect_uri":  {c.cfg.RedirectURL},
	}, refreshToken)
}

func (c *OAuthClient) ProbeCredentials(ctx context.Context) error {
	_, err := c.Refresh(ctx, credentialProbeToken)
	return err
}

func (c *OAuthClient) requestToken(
	ctx context.Context,
	form url.Values,
	grant string,
) (*Token, error) {
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("scope", c.scope)

	var out tokenResponse
	if _, err := c.identity.Do(ctx, &restx.Request{
		Endpoint:       "oauth-token",
		Method:         http.MethodPost,
		Path:           tokenPath,
		Form:           form,
		Out:            &out,
		ExpectedStatus: []int{http.StatusOK},
	}); err != nil {
		redactSecrets(err, c.cfg.ClientSecret, grant)
		return nil, err
	}
	if out.AccessToken == "" || out.RefreshToken == "" || out.ExpiresIn <= 0 {
		return nil, ErrUnexpectedPayload
	}

	return &Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		TokenType:    out.TokenType,
		Scope:        out.Scope,
		ExpiresIn:    time.Duration(out.ExpiresIn) * time.Second,
		ExtExpiresIn: time.Duration(out.ExtExpiresIn) * time.Second,
	}, nil
}

func TenantIDFromAccessToken(accessToken string) (string, error) {
	segments := strings.Split(strings.TrimSpace(accessToken), ".")
	if len(segments) != jwtSegments || segments[1] == "" {
		return "", ErrMalformedToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(segments[1], "="))
	if err != nil {
		return "", ErrMalformedToken
	}
	var claims accessTokenClaims
	if err = sonic.Unmarshal(payload, &claims); err != nil {
		return "", ErrMalformedToken
	}
	if strings.TrimSpace(claims.TenantID) == "" {
		return "", ErrTenantClaim
	}
	tenant, err := guid(claims.TenantID)
	if err != nil {
		return "", ErrTenantClaim
	}
	return tenant, nil
}
