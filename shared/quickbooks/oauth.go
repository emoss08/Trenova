package quickbooks

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Token struct {
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type OAuthClient struct {
	cfg    OAuthConfig
	token  *restx.Client
	revoke *restx.Client
}

type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"x_refresh_token_expires_in"`
}

type revokeBody struct {
	Token string `json:"token"`
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

	settings := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(cfg.ClientID + ":" + cfg.ClientSecret))
	build := func(baseURL string) (*restx.Client, error) {
		if settings.baseURL != "" {
			baseURL = settings.baseURL
		}
		return restx.New(restx.Config{
			BaseURL:      baseURL,
			Timeout:      settings.timeout,
			UserAgent:    settings.userAgent,
			HTTPClient:   settings.httpClient,
			Headers:      map[string]string{"Authorization": "Basic " + credentials},
			Retry:        restx.RetryConfig{},
			Observer:     settings.observer,
			ErrorDecoder: decodeOAuthError,
		})
	}

	tokenClient, err := build(oauthBaseURL)
	if err != nil {
		return nil, fmt.Errorf("quickbooks: configure token transport: %w", err)
	}
	revokeClient, err := build(revokeBaseURL)
	if err != nil {
		return nil, fmt.Errorf("quickbooks: configure revoke transport: %w", err)
	}

	return &OAuthClient{cfg: cfg, token: tokenClient, revoke: revokeClient}, nil
}

func (c *OAuthClient) AuthorizeURL(state string) (string, error) {
	if strings.TrimSpace(state) == "" {
		return "", ErrStateRequired
	}

	target, err := url.Parse(authorizeURL)
	if err != nil {
		return "", fmt.Errorf("quickbooks: parse authorize url: %w", err)
	}
	query := url.Values{}
	query.Set("client_id", c.cfg.ClientID)
	query.Set("response_type", "code")
	query.Set("scope", accountingScope)
	query.Set("redirect_uri", c.cfg.RedirectURL)
	query.Set("state", state)
	target.RawQuery = query.Encode()

	return target.String(), nil
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrCodeRequired
	}

	return c.requestToken(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.cfg.RedirectURL},
	})
}

func (c *OAuthClient) Refresh(ctx context.Context, refreshToken string) (*Token, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrTokenRequired
	}

	return c.requestToken(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (c *OAuthClient) Revoke(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrTokenRequired
	}

	_, err := c.revoke.Do(ctx, &restx.Request{
		Endpoint: "oauth-revoke",
		Method:   http.MethodPost,
		Path:     revokePath,
		Body:     revokeBody{Token: token},
	})
	return err
}

func (c *OAuthClient) requestToken(ctx context.Context, form url.Values) (*Token, error) {
	var out tokenResponse
	if _, err := c.token.Do(ctx, &restx.Request{
		Endpoint: "oauth-token",
		Method:   http.MethodPost,
		Path:     oauthExchangePath,
		Form:     form,
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if out.AccessToken == "" || out.RefreshToken == "" || out.ExpiresIn <= 0 {
		return nil, ErrUnexpectedPayload
	}

	return &Token{
		AccessToken:     out.AccessToken,
		RefreshToken:    out.RefreshToken,
		AccessTokenTTL:  time.Duration(out.ExpiresIn) * time.Second,
		RefreshTokenTTL: time.Duration(out.RefreshTokenExpiresIn) * time.Second,
	}, nil
}
