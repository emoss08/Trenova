package bcconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/accountingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNotConfigured = errors.New("business central is not configured on this instance")

const (
	maxBoundApps            = 512
	limiterPrefix           = "businesscentral"
	maxConcurrentPerCompany = 4
)

var (
	errNoRedirect = errors.New(
		"business central needs app.webBaseUrl or accounting.businessCentral.redirectUrl " +
			"to build its callback",
	)
	errWebhookSignature = errors.New(
		"business central notifications are verified by their subscription, not a signature",
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
	instance       *accountingconnectorports.AccountingApp
	redirectURL    string
	webhookBaseURL string
	apiOpts        []businesscentral.Option
	oauthOpts      []businesscentral.Option
	gate           *companyGate
	locks          *batchLocks
	setups         *postingSetups
	now            func() time.Time
	l              *zap.Logger

	mu    sync.Mutex
	bound map[string]*Connector
}

var (
	_ accountingconnectorports.AccountingProvider             = (*Provider)(nil)
	_ accountingconnectorports.AccountingSubscriptionProvider = (*Provider)(nil)
)

type Connector struct {
	oauth      *businesscentral.OAuthClient
	apiOpts    []businesscentral.Option
	httpClient *http.Client
	locks      *batchLocks
	setups     *postingSetups
	now        func() time.Time
	l          *zap.Logger
}

var _ accountingconnectorports.AccountingConnector = (*Connector)(nil)

func New(p Params) (*Provider, error) {
	cfg := p.AccountingConfig.BusinessCentral
	provider := &Provider{
		redirectURL:    cfg.GetRedirectURL(p.AppConfig),
		webhookBaseURL: cfg.GetWebhookBaseURL(p.AppConfig),
		gate:           newCompanyGate(),
		locks:          newBatchLocks(),
		setups:         newPostingSetups(time.Now),
		now:            time.Now,
		l:              p.Logger.Named("accounting.businesscentral"),
		bound:          make(map[string]*Connector),
	}
	if p.Limiter != nil {
		provider.apiOpts = append(
			provider.apiOpts,
			businesscentral.WithLimiter(p.Limiter, limiterPrefix),
		)
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
	return integration.TypeBusinessCentral
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
		locks:      p.locks,
		setups:     p.setups,
		now:        p.now,
		l:          p.l,
	}
	if p.redirectURL != "" {
		oauth, err := businesscentral.NewOAuthClient(businesscentral.OAuthConfig{
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
	return nil, nil
}

func (p *Provider) WebhookSignatureHeader() string {
	return ""
}

func (p *Provider) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func (p *Provider) NotificationBaseURL() string {
	return p.webhookBaseURL
}

func (p *Provider) ParseNotifications(
	body []byte,
) ([]accountingconnectorports.AccountingWebhookNotification, error) {
	parsed, err := businesscentral.ParseNotifications(body)
	if err != nil {
		return nil, err
	}
	out := make([]accountingconnectorports.AccountingWebhookNotification, 0, len(parsed))
	for idx := range parsed {
		out = append(out, accountingconnectorports.AccountingWebhookNotification{
			SubscriptionID: parsed[idx].SubscriptionID,
			ClientState:    parsed[idx].ClientState,
			Resource:       parsed[idx].Resource,
			ChangeType:     parsed[idx].ChangeType,
		})
	}
	return out, nil
}

func (p *Provider) ValidationToken(raw string) (string, bool) {
	return businesscentral.ValidationToken(raw)
}

func bindingKey(app *accountingconnectorports.AccountingApp) string {
	return hashutils.SHA256Hex(
		strings.TrimSpace(app.ClientID) + "\x00" + strings.TrimSpace(app.ClientSecret),
	)
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeBusinessCentral
}

func (c *Connector) options() []businesscentral.Option {
	opts := make([]businesscentral.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, businesscentral.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Connector) client(
	auth accountingconnectorports.AccountingDocumentAuth,
) (*businesscentral.Client, error) {
	ref, err := businesscentral.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return nil, err
	}
	return businesscentral.New(ref, auth.AccessToken, c.options()...)
}

func (c *Connector) AuthorizeURL(
	req accountingconnectorports.AccountingAuthorizeRequest,
) (string, error) {
	if c.oauth == nil {
		return "", errNoRedirect
	}
	return c.oauth.AuthorizeURL(req.State)
}

func (c *Connector) ExchangeCode(
	ctx context.Context,
	req *accountingconnectorports.AccountingCodeExchange,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, errNoRedirect
	}
	token, err := c.oauth.ExchangeCode(ctx, req.Code)
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
	token, err := c.oauth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Revoke(
	_ context.Context,
	_ *accountingconnectorports.AccountingRevokeRequest,
) error {
	return nil
}

func (c *Connector) ReleaseCompanies(
	_ context.Context,
	_ *accountingconnectorports.AccountingTokenGrant,
	_ []accountingconnectorports.AccountingCompany,
) error {
	return nil
}

func (c *Connector) VerifyApp(ctx context.Context) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	err := c.oauth.ProbeCredentials(ctx)
	switch {
	case err == nil, businesscentral.IsInvalidGrant(err):
		return nil
	case businesscentral.IsInvalidClient(err):
		return accountingconnectorports.ErrAccountingAppRejected
	default:
		return err
	}
}

func (c *Connector) VerifyWebhook(_ string, _ []byte) error {
	return errWebhookSignature
}

func (c *Connector) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func grantOf(token *businesscentral.Token) *accountingconnectorports.AccountingTokenGrant {
	return &accountingconnectorports.AccountingTokenGrant{
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		AccessTokenTTL: token.ExpiresIn,
		RefreshTokenTTL: accountingprovider.MustProfile(integration.TypeBusinessCentral).
			RefreshTokenLifetime,
	}
}
