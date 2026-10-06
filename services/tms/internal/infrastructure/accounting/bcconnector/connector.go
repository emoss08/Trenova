package bcconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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

	Config  *config.Config
	Logger  *zap.Logger
	Limiter restx.Limiter `name:"accountingLimiter" optional:"true"`
}

type Provider struct {
	instance       *services.AccountingApp
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
	_ services.AccountingProvider             = (*Provider)(nil)
	_ services.AccountingSubscriptionProvider = (*Provider)(nil)
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

var _ services.AccountingConnector = (*Connector)(nil)

func New(p Params) (*Provider, error) {
	cfg := p.Config.Accounting.BusinessCentral
	provider := &Provider{
		redirectURL:    cfg.GetRedirectURL(&p.Config.App),
		webhookBaseURL: cfg.GetWebhookBaseURL(&p.Config.App),
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
	if cfg.IsConfigured(&p.Config.App) {
		provider.instance = &services.AccountingApp{
			Source:       accountingsync.AppSourceInstance,
			Environment:  accountingsync.AppEnvironmentProduction,
			ClientID:     strings.TrimSpace(cfg.ClientID),
			ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		}
	}
	return provider, nil
}

func (p *Provider) IntegrationType() integration.Type {
	return integration.TypeBusinessCentral
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
) ([]services.AccountingWebhookNotification, error) {
	parsed, err := businesscentral.ParseNotifications(body)
	if err != nil {
		return nil, err
	}
	out := make([]services.AccountingWebhookNotification, 0, len(parsed))
	for idx := range parsed {
		out = append(out, services.AccountingWebhookNotification{
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

func bindingKey(app *services.AccountingApp) string {
	return hashutils.SHA256Hex(strings.Join([]string{
		strings.TrimSpace(app.ClientID),
		strings.TrimSpace(app.ClientSecret),
	}, "\x00"))
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeBusinessCentral
}

func (c *Connector) options() []businesscentral.Option {
	opts := make([]businesscentral.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, businesscentral.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Connector) client(auth services.AccountingDocumentAuth) (*businesscentral.Client, error) {
	ref, err := businesscentral.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return nil, err
	}
	return businesscentral.New(ref, auth.AccessToken, c.options()...)
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

func (c *Connector) Revoke(_ context.Context, _ string) error {
	return nil
}

func (c *Connector) ReleaseCompanies(
	_ context.Context,
	_ *services.AccountingTokenGrant,
	_ []services.AccountingCompany,
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
		return services.ErrAccountingAppRejected
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

func grantOf(token *businesscentral.Token) *services.AccountingTokenGrant {
	return &services.AccountingTokenGrant{
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		AccessTokenTTL: token.ExpiresIn,
		RefreshTokenTTL: accountingsync.MustProfile(integration.TypeBusinessCentral).
			RefreshTokenLifetime,
	}
}
