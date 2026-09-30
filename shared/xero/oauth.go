package xero

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
	jwtSegments       = 3
	grantAuthCode     = "authorization_code"
	grantRefreshToken = "refresh_token"
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
	IDToken      string
	Scope        string
	ExpiresIn    time.Duration
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
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

type accessTokenClaims struct {
	AuthenticationEventID string `json:"authentication_event_id"`
}

func DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
		"offline_access",
		"accounting.contacts",
		"accounting.settings",
		"accounting.invoices",
		"accounting.payments",
		"accounting.reports.aged.read",
	}
}

func NewOAuthClient(cfg OAuthConfig, opts ...Option) (*OAuthClient, error) {
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, ErrClientIDRequired
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, ErrClientSecretRequired
	}
	if strings.TrimSpace(cfg.RedirectURL) == "" {
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
	credentials := base64.StdEncoding.EncodeToString(
		[]byte(cfg.ClientID + ":" + cfg.ClientSecret),
	)
	identity, err := restx.New(restx.Config{
		BaseURL:      settings.identityURL,
		Timeout:      settings.timeout,
		UserAgent:    settings.userAgent,
		HTTPClient:   settings.httpClient,
		Headers:      map[string]string{authorizationHeader: "Basic " + credentials},
		Retry:        restx.RetryConfig{},
		Observer:     settings.observer,
		ErrorDecoder: decodeOAuthError,
	})
	if err != nil {
		return nil, fmt.Errorf("xero: configure identity transport: %w", err)
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
		return "", fmt.Errorf("xero: parse authorize url: %w", err)
	}
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", c.cfg.ClientID)
	query.Set("redirect_uri", c.cfg.RedirectURL)
	query.Set("scope", c.scope)
	query.Set("state", state)
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
	})
}

func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrTokenRequired
	}

	return c.requestToken(ctx, url.Values{
		"grant_type":    {grantRefreshToken},
		"refresh_token": {refreshToken},
	})
}

func (c *OAuthClient) Revoke(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return ErrTokenRequired
	}

	_, err := c.identity.Do(ctx, &restx.Request{
		Endpoint: "oauth-revoke",
		Method:   http.MethodPost,
		Path:     revocationPath,
		Form:     url.Values{"token": {refreshToken}},
	})
	return err
}

func (c *OAuthClient) requestToken(ctx context.Context, form url.Values) (*Token, error) {
	var out tokenResponse
	if _, err := c.identity.Do(ctx, &restx.Request{
		Endpoint: "oauth-token",
		Method:   http.MethodPost,
		Path:     tokenPath,
		Form:     form,
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if out.AccessToken == "" || out.RefreshToken == "" || out.ExpiresIn <= 0 {
		return nil, ErrUnexpectedPayload
	}

	return &Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		IDToken:      out.IDToken,
		Scope:        out.Scope,
		ExpiresIn:    time.Duration(out.ExpiresIn) * time.Second,
	}, nil
}

func AuthEventID(accessToken string) (string, error) {
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
	eventID := strings.TrimSpace(claims.AuthenticationEventID)
	if eventID == "" {
		return "", ErrAuthEventMissing
	}
	return eventID, nil
}
