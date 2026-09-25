package qboconnector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNotConfigured = errors.New("quickbooks online is not configured on this instance")

const maxBoundApps = 512

var errNoRedirect = errors.New(
	"quickbooks online needs app.webBaseUrl or accounting.quickbooks.redirectUrl to build its callback",
)

type Params struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Limiter restx.Limiter `name:"accountingLimiter" optional:"true"`
}

type Provider struct {
	instance    *services.AccountingApp
	redirectURL string
	apiOpts     []quickbooks.Option
	oauthOpts   []quickbooks.Option
	l           *zap.Logger

	mu    sync.Mutex
	bound map[string]*Connector
}

var _ services.AccountingProvider = (*Provider)(nil)

type Connector struct {
	env      quickbooks.Environment
	oauth    *quickbooks.OAuthClient
	verifier string
	apiOpts  []quickbooks.Option
	l        *zap.Logger
}

var _ services.AccountingConnector = (*Connector)(nil)

func New(p Params) (*Provider, error) {
	qbo := p.Config.Accounting.QuickBooks
	environment, ok := accountingsync.ParseAppEnvironment(qbo.GetEnvironment())
	if !ok {
		return nil, fmt.Errorf("quickbooks: unknown environment %q", qbo.GetEnvironment())
	}

	provider := &Provider{
		redirectURL: qbo.GetRedirectURL(&p.Config.App),
		l:           p.Logger.Named("accounting.quickbooks"),
		bound:       make(map[string]*Connector),
	}
	if p.Limiter != nil {
		provider.apiOpts = append(provider.apiOpts, quickbooks.WithLimiter(p.Limiter, ""))
	}
	if qbo.IsConfigured(&p.Config.App) {
		provider.instance = &services.AccountingApp{
			Source:               accountingsync.AppSourceInstance,
			Environment:          environment,
			ClientID:             strings.TrimSpace(qbo.ClientID),
			ClientSecret:         strings.TrimSpace(qbo.ClientSecret),
			WebhookVerifierToken: strings.TrimSpace(qbo.WebhookVerifierToken),
		}
	}

	return provider, nil
}

func (p *Provider) IntegrationType() integration.Type {
	return integration.TypeQuickBooksOnline
}

func (p *Provider) InstanceApp() (*services.AccountingApp, bool) {
	if p.instance == nil {
		return nil, false
	}
	app := *p.instance
	return &app, true
}

func (p *Provider) RedirectURL() string {
	return p.redirectURL
}

func (p *Provider) Bind(app *services.AccountingApp) (services.AccountingConnector, error) {
	return p.bind(app)
}

func (p *Provider) bind(app *services.AccountingApp) (*Connector, error) {
	if app == nil {
		return nil, ErrNotConfigured
	}
	env, ok := apiEnvironment(app.Environment)
	if !ok {
		return nil, fmt.Errorf("quickbooks: unknown environment %q", app.Environment)
	}

	key := bindingKey(app)
	p.mu.Lock()
	defer p.mu.Unlock()
	if conn, found := p.bound[key]; found {
		return conn, nil
	}

	conn := &Connector{
		env:      env,
		verifier: strings.TrimSpace(app.WebhookVerifierToken),
		apiOpts:  p.apiOpts,
		l:        p.l,
	}
	if p.redirectURL != "" {
		oauth, err := quickbooks.NewOAuthClient(quickbooks.OAuthConfig{
			ClientID:     strings.TrimSpace(app.ClientID),
			ClientSecret: strings.TrimSpace(app.ClientSecret),
			RedirectURL:  p.redirectURL,
		}, p.oauthOpts...)
		if err != nil {
			return nil, err
		}
		conn.oauth = oauth
	}
	if len(p.bound) >= maxBoundApps {
		clear(p.bound)
	}
	p.bound[key] = conn
	return conn, nil
}

func (p *Provider) WebhookRealmIDs(body []byte) ([]string, error) {
	return quickbooks.ParseRealmIDs(body)
}

func (p *Provider) WebhookSignatureHeader() string {
	return quickbooks.SignatureHeader
}

func (p *Provider) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func apiEnvironment(environment accountingsync.AppEnvironment) (quickbooks.Environment, bool) {
	switch environment {
	case accountingsync.AppEnvironmentSandbox:
		return quickbooks.EnvironmentSandbox, true
	case accountingsync.AppEnvironmentProduction:
		return quickbooks.EnvironmentProduction, true
	default:
		return "", false
	}
}

func bindingKey(app *services.AccountingApp) string {
	return hashutils.SHA256Hex(strings.Join([]string{
		string(app.Environment),
		strings.TrimSpace(app.ClientID),
		strings.TrimSpace(app.ClientSecret),
		strings.TrimSpace(app.WebhookVerifierToken),
	}, "\x00"))
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeQuickBooksOnline
}

func (c *Connector) AuthorizeURL(state string) (string, error) {
	if c.oauth == nil {
		return "", errNoRedirect
	}
	return c.oauth.AuthorizeURL(state)
}

func (c *Connector) ExchangeCode(
	ctx context.Context,
	code string,
) (*services.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, errNoRedirect
	}
	token, err := c.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Refresh(
	ctx context.Context,
	refreshToken string,
) (*services.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, errNoRedirect
	}
	token, err := c.oauth.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Revoke(ctx context.Context, token string) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	return c.oauth.Revoke(ctx, token)
}

func (c *Connector) CompanyFacts(
	ctx context.Context,
	realmID, accessToken string,
) (*accountingsync.CompanyFacts, error) {
	client, err := quickbooks.New(c.env, realmID, accessToken, c.apiOpts...)
	if err != nil {
		return nil, err
	}

	info, err := client.CompanyInfo(ctx)
	if err != nil {
		return nil, err
	}
	prefs, err := client.Preferences(ctx)
	if err != nil {
		return nil, err
	}

	facts := &accountingsync.CompanyFacts{
		CompanyName:          info.CompanyName,
		LegalName:            info.LegalName,
		Country:              info.Country,
		HomeCurrency:         prefs.HomeCurrency,
		MultiCurrencyEnabled: prefs.MultiCurrencyEnabled,
	}
	if prefs.BooksClosedThrough != "" {
		closed, parseErr := time.Parse(time.DateOnly, prefs.BooksClosedThrough)
		if parseErr != nil {
			c.l.Warn("unreadable books-closed date from QuickBooks",
				zap.String("value", prefs.BooksClosedThrough))
		} else {
			endOfDay := closed.Add(24*time.Hour - time.Second).Unix()
			facts.BooksClosedThrough = &endOfDay
		}
	}

	return facts, nil
}

const credentialProbeToken = "trenova-credential-check"

func (c *Connector) VerifyApp(ctx context.Context) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	_, err := c.oauth.Refresh(ctx, credentialProbeToken)
	switch {
	case err == nil, quickbooks.IsInvalidGrant(err):
		return nil
	case quickbooks.IsInvalidClient(err), quickbooks.IsUnauthorized(err):
		return services.ErrAccountingAppRejected
	default:
		return err
	}
}

func (c *Connector) VerifyWebhook(signature string, body []byte) error {
	return quickbooks.VerifySignature(c.verifier, signature, body)
}

func (c *Connector) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func classifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotConfigured), errors.Is(err, errNoRedirect):
		return accountingsync.ErrorCategoryConfiguration
	case quickbooks.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case quickbooks.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case quickbooks.IsUnauthorized(err), quickbooks.IsForbidden(err):
		return accountingsync.ErrorCategoryUnauthorized
	case quickbooks.IsTransient(err), errors.Is(err, context.DeadlineExceeded):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}

func grantOf(token *quickbooks.Token) *services.AccountingTokenGrant {
	return &services.AccountingTokenGrant{
		AccessToken:     token.AccessToken,
		RefreshToken:    token.RefreshToken,
		AccessTokenTTL:  token.AccessTokenTTL,
		RefreshTokenTTL: token.RefreshTokenTTL,
	}
}
