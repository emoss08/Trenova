package nsconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNotConfigured = errors.New("netsuite is not configured on this instance")

const (
	maxBoundApps            = 512
	limiterPrefix           = "netsuite"
	maxConcurrentPerAccount = 2
	minClientIDLength       = 16
	maxClientIDLength       = 256
)

var (
	errNoRedirect = errors.New(
		"netsuite needs app.webBaseUrl or accounting.netSuite.redirectUrl to build its callback",
	)
	errNoWebhooks = errors.New(
		"netsuite sends no webhooks, so Trenova has no NetSuite notification route",
	)
	errNoAccount = errors.New("netsuite: the NetSuite account id is required")
)

type Params struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Limiter restx.Limiter `name:"accountingLimiter" optional:"true"`
}

type Provider struct {
	instance    *accountingconnectorports.AccountingApp
	redirectURL string
	apiOpts     []netsuite.Option
	oauthOpts   []netsuite.Option
	gate        *accountGate
	books       *currencyBooks
	now         func() time.Time
	l           *zap.Logger

	mu    sync.Mutex
	bound map[string]*Connector
}

var _ accountingconnectorports.AccountingProvider = (*Provider)(nil)

type Connector struct {
	oauth      *netsuite.OAuthClient
	apiOpts    []netsuite.Option
	httpClient *http.Client
	books      *currencyBooks
	now        func() time.Time
	l          *zap.Logger
}

var (
	_ accountingconnectorports.AccountingConnector                  = (*Connector)(nil)
	_ accountingconnectorports.AccountingClientCredentialsConnector = (*Connector)(nil)
)

func New(p Params) (*Provider, error) {
	cfg := p.Config.Accounting.NetSuite
	provider := &Provider{
		redirectURL: cfg.GetRedirectURL(&p.Config.App),
		gate:        newAccountGate(),
		books:       newCurrencyBooks(time.Now),
		now:         time.Now,
		l:           p.Logger.Named("accounting.netsuite"),
		bound:       make(map[string]*Connector),
	}
	if p.Limiter != nil {
		provider.apiOpts = append(provider.apiOpts, netsuite.WithLimiter(p.Limiter, limiterPrefix))
	}
	if cfg.IsConfigured(&p.Config.App) {
		provider.instance = &accountingconnectorports.AccountingApp{
			Source:       accountingconnection.AppSourceInstance,
			Environment:  accountingprovider.AppEnvironmentProduction,
			ClientID:     strings.TrimSpace(cfg.ClientID),
			ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		}
	}
	return provider, nil
}

func (p *Provider) IntegrationType() integration.Type {
	return integration.TypeNetSuite
}

func (p *Provider) InstanceApp() (*accountingconnectorports.AccountingApp, bool) {
	if p.instance == nil {
		return nil, false
	}
	app := *p.instance
	return &app, true
}

func (p *Provider) RedirectURL() string {
	return p.redirectURL
}

func (p *Provider) Bind(
	app *accountingconnectorports.AccountingApp,
) (accountingconnectorports.AccountingConnector, error) {
	return p.bind(app)
}

func (p *Provider) bind(app *accountingconnectorports.AccountingApp) (*Connector, error) {
	if app == nil {
		return nil, ErrNotConfigured
	}

	key := bindingKey(app)
	p.mu.Lock()
	defer p.mu.Unlock()
	if conn, found := p.bound[key]; found {
		return conn, nil
	}

	conn := &Connector{
		apiOpts:    p.apiOpts,
		httpClient: &http.Client{Transport: p.gate.transport(http.DefaultTransport)},
		books:      p.books,
		now:        p.now,
		l:          p.l,
	}
	if p.redirectURL != "" {
		oauth, err := netsuite.NewOAuthClient(netsuite.OAuthConfig{
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

func (p *Provider) WebhookRealmIDs(_ []byte) ([]string, error) {
	return nil, errNoWebhooks
}

func (p *Provider) WebhookSignatureHeader() string {
	return ""
}

func (p *Provider) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func bindingKey(app *accountingconnectorports.AccountingApp) string {
	return hashutils.SHA256Hex(
		strings.TrimSpace(app.ClientID) + "\x00" + strings.TrimSpace(app.ClientSecret),
	)
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeNetSuite
}

func (c *Connector) options() []netsuite.Option {
	opts := make([]netsuite.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, netsuite.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func accountOf(realmID string) (string, error) {
	if strings.TrimSpace(realmID) == "" {
		return "", errNoAccount
	}
	ref, err := netsuite.ParseCompanyRef(realmID)
	if err != nil {
		return "", err
	}
	return ref.AccountID, nil
}

func (c *Connector) AuthorizeURL(
	req accountingconnectorports.AccountingAuthorizeRequest,
) (string, error) {
	if c.oauth == nil {
		return "", errNoRedirect
	}
	return c.oauth.AuthorizeURL(netsuite.AuthorizeRequest{
		State:         req.State,
		CodeChallenge: req.CodeChallenge,
	})
}

func (c *Connector) ExchangeCode(
	ctx context.Context,
	req *accountingconnectorports.AccountingCodeExchange,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, errNoRedirect
	}
	account, err := accountOf(req.CallbackRealmID)
	if err != nil {
		return nil, err
	}
	token, err := c.oauth.ExchangeCode(ctx, netsuite.CodeExchange{
		AccountID:    account,
		Code:         req.Code,
		CodeVerifier: req.CodeVerifier,
	})
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Refresh(
	ctx context.Context,
	req *accountingconnectorports.AccountingRefreshRequest,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, errNoRedirect
	}
	account, err := accountOf(req.RealmID)
	if err != nil {
		return nil, err
	}
	token, err := c.oauth.Refresh(ctx, netsuite.RefreshRequest{
		AccountID:    account,
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Revoke(
	ctx context.Context,
	req *accountingconnectorports.AccountingRevokeRequest,
) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	account, err := accountOf(req.RealmID)
	if err != nil {
		return err
	}
	return c.oauth.Revoke(ctx, netsuite.RevokeRequest{AccountID: account, Token: req.Token})
}

func (c *Connector) ReleaseCompanies(
	_ context.Context,
	_ *accountingconnectorports.AccountingTokenGrant,
	_ []accountingconnectorports.AccountingCompany,
) error {
	return nil
}

func (c *Connector) VerifyApp(_ context.Context) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	if !validClientID(c.oauth.ClientID()) {
		return accountingconnectorports.ErrAccountingAppRejected
	}
	return nil
}

func validClientID(id string) bool {
	if len(id) < minClientIDLength || len(id) > maxClientIDLength {
		return false
	}
	for idx := range len(id) {
		ch := id[idx]
		alphanumeric := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9')
		if !alphanumeric && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

func (c *Connector) VerifyWebhook(_ string, _ []byte) error {
	return errNoWebhooks
}

func (c *Connector) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func grantOf(token *netsuite.Token) *accountingconnectorports.AccountingTokenGrant {
	grant := &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		AccessTokenTTL: token.ExpiresIn,
	}
	if token.RefreshToken != "" {
		grant.RefreshTokenTTL = accountingprovider.MustProfile(integration.TypeNetSuite).
			RefreshTokenLifetime
	}
	return grant
}
