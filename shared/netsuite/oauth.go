package netsuite

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	Scope                 = "rest_webservices"
	AccessTokenLifetime   = time.Hour
	RefreshTokenLifetime  = 7 * 24 * time.Hour
	CodeChallengeMethod   = "S256"
	minStateLength        = 22
	maxStateLength        = 1024
	minVerifierLength     = 43
	maxVerifierLength     = 128
	challengeLength       = 43
	grantAuthCode         = "authorization_code"
	grantRefreshToken     = "refresh_token"
	grantClientCredential = "client_credentials"
	formGrantType         = "grant_type"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    time.Duration
}

type AuthorizeRequest struct {
	State         string
	CodeChallenge string
}

type CodeExchange struct {
	AccountID    string
	Code         string
	CodeVerifier string
}

type RefreshRequest struct {
	AccountID    string
	RefreshToken string
}

type RevokeRequest struct {
	AccountID string
	Token     string
}

type OAuthClient struct {
	cfg      OAuthConfig
	settings options
	basic    string
}

type tokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    jsonflex.Int `json:"expires_in"`
}

type tokenCall struct {
	accountID      string
	endpoint       string
	path           string
	form           url.Values
	out            any
	expected       []int
	basic          bool
	secrets        []string
	requireRefresh bool
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
	credentials := cfg.ClientID + ":" + cfg.ClientSecret
	return &OAuthClient{
		cfg:      cfg,
		settings: resolveOptions(opts),
		basic:    "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials)),
	}, nil
}

func (c *OAuthClient) ClientID() string {
	return c.cfg.ClientID
}

func (c *OAuthClient) AuthorizeURL(req AuthorizeRequest) (string, error) {
	if !validState(req.State) {
		return "", ErrInvalidState
	}
	if !validChallenge(req.CodeChallenge) {
		return "", ErrInvalidChallenge
	}
	target, err := url.Parse(c.settings.loginURL + authorizePath)
	if err != nil {
		return "", fmt.Errorf("netsuite: parse authorize url: %w", err)
	}
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", c.cfg.ClientID)
	query.Set("redirect_uri", c.cfg.RedirectURL)
	query.Set("scope", Scope)
	query.Set("state", req.State)
	query.Set("code_challenge", req.CodeChallenge)
	query.Set("code_challenge_method", CodeChallengeMethod)
	target.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	return target.String(), nil
}

func (c *OAuthClient) TokenURL(accountID string) (string, error) {
	base, err := c.settings.apiBaseURL(accountID)
	if err != nil {
		return "", err
	}
	return base + tokenPath, nil
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, req CodeExchange) (*Token, error) {
	if strings.TrimSpace(req.Code) == "" {
		return nil, ErrCodeRequired
	}
	if !validVerifier(req.CodeVerifier) {
		return nil, ErrInvalidVerifier
	}
	return c.requestToken(ctx, &tokenCall{
		accountID: req.AccountID,
		endpoint:  "oauth-token",
		form: url.Values{
			formGrantType:   {grantAuthCode},
			"code":          {req.Code},
			"redirect_uri":  {c.cfg.RedirectURL},
			"code_verifier": {req.CodeVerifier},
		},
		basic:          true,
		secrets:        []string{req.Code, req.CodeVerifier},
		requireRefresh: true,
	})
}

func (c *OAuthClient) Refresh(ctx context.Context, req RefreshRequest) (*Token, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, ErrTokenRequired
	}
	return c.requestToken(ctx, &tokenCall{
		accountID: req.AccountID,
		endpoint:  "oauth-token",
		form: url.Values{
			formGrantType:   {grantRefreshToken},
			"refresh_token": {req.RefreshToken},
		},
		basic:   true,
		secrets: []string{req.RefreshToken},
	})
}

func (c *OAuthClient) Revoke(ctx context.Context, req RevokeRequest) error {
	if strings.TrimSpace(req.Token) == "" {
		return ErrTokenRequired
	}
	return c.send(ctx, &tokenCall{
		accountID: req.AccountID,
		endpoint:  "oauth-revoke",
		path:      revokePath,
		form:      url.Values{"token": {req.Token}},
		expected:  []int{http.StatusOK, http.StatusNoContent},
		basic:     true,
		secrets:   []string{req.Token},
	})
}

func (c *OAuthClient) requestToken(ctx context.Context, req *tokenCall) (*Token, error) {
	var out tokenResponse
	req.path = tokenPath
	req.out = &out
	req.expected = []int{http.StatusOK}
	if err := c.send(ctx, req); err != nil {
		return nil, err
	}
	expiresIn := out.ExpiresIn.Value()
	if out.AccessToken == "" || expiresIn <= 0 || (req.requireRefresh && out.RefreshToken == "") {
		return nil, ErrUnexpectedPayload
	}
	return &Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		TokenType:    out.TokenType,
		ExpiresIn:    time.Duration(expiresIn) * time.Second,
	}, nil
}

func (c *OAuthClient) send(ctx context.Context, req *tokenCall) error {
	baseURL, err := c.settings.apiBaseURL(req.accountID)
	if err != nil {
		return err
	}
	headers := map[string]string{}
	if req.basic {
		headers[authorizationHeader] = c.basic
	}
	identity, err := restx.New(restx.Config{
		BaseURL:      baseURL,
		Timeout:      c.settings.timeout,
		UserAgent:    c.settings.userAgent,
		HTTPClient:   c.settings.httpClient,
		Headers:      headers,
		Retry:        restx.RetryConfig{},
		Observer:     c.settings.observer,
		ErrorDecoder: decodeOAuthError,
	})
	if err != nil {
		return fmt.Errorf("netsuite: configure identity transport: %w", err)
	}
	if _, err = identity.Do(ctx, &restx.Request{
		Endpoint:       req.endpoint,
		Method:         http.MethodPost,
		Path:           req.path,
		Form:           req.form,
		Out:            req.out,
		ExpectedStatus: req.expected,
	}); err != nil {
		redactSecrets(err, append(req.secrets, c.cfg.ClientSecret)...)
		return err
	}
	return nil
}

func validState(state string) bool {
	if len(state) < minStateLength || len(state) > maxStateLength {
		return false
	}
	for idx := range len(state) {
		if state[idx] <= ' ' || state[idx] > '~' {
			return false
		}
	}
	return true
}

func validVerifier(verifier string) bool {
	if len(verifier) < minVerifierLength || len(verifier) > maxVerifierLength {
		return false
	}
	for idx := range len(verifier) {
		ch := verifier[idx]
		if !isUnreserved(ch) {
			return false
		}
	}
	return true
}

func validChallenge(challenge string) bool {
	if len(challenge) != challengeLength {
		return false
	}
	for idx := range len(challenge) {
		ch := challenge[idx]
		if !isAlphaNumeric(ch) && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

func isUnreserved(ch byte) bool {
	return isAlphaNumeric(ch) || ch == '-' || ch == '.' || ch == '_' || ch == '~'
}

func isAlphaNumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}
