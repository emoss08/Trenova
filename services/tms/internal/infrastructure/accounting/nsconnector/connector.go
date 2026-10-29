package nsconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsdocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/infrastructure/config/accountingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
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

const (
	maxBoundApps            = 512
	limiterPrefix           = "netsuite"
	maxConcurrentPerAccount = 2
	minClientIDLength       = 16
	maxClientIDLength       = 256
)

var (
	errNoWebhooks = errors.New(
		"netsuite sends no webhooks, so Trenova has no NetSuite notification route",
	)
)

type Params struct {
	fx.In

	AccountingConfig *accountingconfig.Config
	AppConfig        *appconfig.Config
	Logger           *zap.Logger
	Limiter          restx.Limiter `name:"accountingLimiter" optional:"true"`
}

type Provider struct {
	instance    *accountingconnectorports.AccountingApp
	redirectURL string
	apiOpts     []netsuite.Option
	oauthOpts   []netsuite.Option
	gate        *accountGate
	books       *nsbinding.CurrencyBooks
	now         func() time.Time
	l           *zap.Logger

	mu    sync.Mutex
	bound map[string]*Connector
}

var _ accountingconnectorports.AccountingProvider = (*Provider)(nil)

type Connector struct {
	*nsbinding.Binding
	*nsdocsync.Documents
	*nsrefsync.References
	oauth *netsuite.OAuthClient
}

var (
	_ accountingconnectorports.AccountingConnector                  = (*Connector)(nil)
	_ accountingconnectorports.AccountingClientCredentialsConnector = (*Connector)(nil)
	_ accountingconnectorports.AccountingDocumentWriter             = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceReader            = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceCreator           = (*Connector)(nil)
)

func New(p Params) (*Provider, error) {
	cfg := p.AccountingConfig.NetSuite
	provider := &Provider{
		redirectURL: cfg.GetRedirectURL(p.AppConfig),
		gate:        newAccountGate(),
		books:       nsbinding.NewCurrencyBooks(time.Now),
		now:         time.Now,
		l:           p.Logger.Named("accounting.netsuite"),
		bound:       make(map[string]*Connector),
	}
	if p.Limiter != nil {
		provider.apiOpts = append(provider.apiOpts, netsuite.WithLimiter(p.Limiter, limiterPrefix))
	}
	if cfg.IsConfigured(p.AppConfig) {
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
		return nil, nsbinding.ErrNotConfigured
	}

	key := bindingKey(app)
	p.mu.Lock()
	defer p.mu.Unlock()
	if conn, found := p.bound[key]; found {
		return conn, nil
	}

	binding := nsbinding.NewBinding(nsbinding.BindingParams{
		APIOpts:    p.apiOpts,
		HTTPClient: &http.Client{Transport: p.gate.transport(http.DefaultTransport)},
		Books:      p.books,
		Now:        p.now,
	})
	conn := &Connector{
		Binding:    binding,
		Documents:  &nsdocsync.Documents{Binding: binding},
		References: &nsrefsync.References{Binding: binding},
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
	return nsbinding.ClassifyError(err)
}

func bindingKey(app *accountingconnectorports.AccountingApp) string {
	return hashutils.SHA256Hex(
		strings.TrimSpace(app.ClientID) + "\x00" + strings.TrimSpace(app.ClientSecret),
	)
}

func (c *Connector) AuthorizeURL(
	req accountingconnectorports.AccountingAuthorizeRequest,
) (string, error) {
	if c.oauth == nil {
		return "", nsbinding.ErrNoRedirect
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
		return nil, nsbinding.ErrNoRedirect
	}
	account, err := nsbinding.AccountOf(req.CallbackRealmID)
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
		return nil, nsbinding.ErrNoRedirect
	}
	account, err := nsbinding.AccountOf(req.RealmID)
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
		return nsbinding.ErrNoRedirect
	}
	account, err := nsbinding.AccountOf(req.RealmID)
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
		return nsbinding.ErrNoRedirect
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
