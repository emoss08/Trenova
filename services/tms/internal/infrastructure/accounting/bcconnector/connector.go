package bcconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcdocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcrefsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/infrastructure/config/accountingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralwebhook"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxBoundApps            = 512
	limiterPrefix           = "businesscentral"
	maxConcurrentPerCompany = 4
)

var (
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
	locks          *bcsession.BatchLocks
	setups         *bcsession.PostingSetups
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
	*bcsession.Session
	*bcdocsync.Documents
	*bcrefsync.References
	oauth  *businesscentral.OAuthClient
	setups *bcsession.PostingSetups
}

var (
	_ accountingconnectorports.AccountingConnector        = (*Connector)(nil)
	_ accountingconnectorports.AccountingDocumentWriter   = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceReader  = (*Connector)(nil)
	_ accountingconnectorports.AccountingReferenceCreator = (*Connector)(nil)
)

func New(p Params) (*Provider, error) {
	cfg := p.AccountingConfig.BusinessCentral
	provider := &Provider{
		redirectURL:    cfg.GetRedirectURL(p.AppConfig),
		webhookBaseURL: cfg.GetWebhookBaseURL(p.AppConfig),
		gate:           newCompanyGate(),
		locks:          bcsession.NewBatchLocks(),
		setups:         bcsession.NewPostingSetups(time.Now),
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
		return nil, bcsession.ErrNotConfigured
	}

	key := bindingKey(app)
	p.mu.Lock()
	defer p.mu.Unlock()
	if conn, found := p.bound[key]; found {
		return conn, nil
	}

	session := bcsession.NewSession(bcsession.SessionParams{
		APIOpts:    p.apiOpts,
		HTTPClient: &http.Client{Transport: p.gate.transport(http.DefaultTransport)},
		Locks:      p.locks,
		Setups:     p.setups,
		Now:        p.now,
	})
	conn := &Connector{
		Session:    session,
		Documents:  &bcdocsync.Documents{Session: session},
		References: &bcrefsync.References{Session: session},
		setups:     p.setups,
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
	return bcsession.ClassifyError(err)
}

func (p *Provider) NotificationBaseURL() string {
	return p.webhookBaseURL
}

func (p *Provider) ParseNotifications(
	body []byte,
) ([]accountingconnectorports.AccountingWebhookNotification, error) {
	parsed, err := businesscentralwebhook.ParseNotifications(body)
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
	return businesscentralwebhook.ValidationToken(raw)
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
		return "", bcsession.ErrNoRedirect
	}
	return c.oauth.AuthorizeURL(req.State)
}

func (c *Connector) ExchangeCode(
	ctx context.Context,
	req *accountingconnectorports.AccountingCodeExchange,
) (*accountingconnectorports.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, bcsession.ErrNoRedirect
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
		return nil, bcsession.ErrNoRedirect
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
		return bcsession.ErrNoRedirect
	}
	err := c.oauth.ProbeCredentials(ctx)
	switch {
	case err == nil, businesscentralerrors.IsInvalidGrant(err):
		return nil
	case businesscentralerrors.IsInvalidClient(err):
		return accountingconnectorports.ErrAccountingAppRejected
	default:
		return err
	}
}

func (c *Connector) VerifyWebhook(_ string, _ []byte) error {
	return errWebhookSignature
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
