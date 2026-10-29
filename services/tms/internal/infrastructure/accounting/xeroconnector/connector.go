package xeroconnector

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
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNotConfigured = errors.New("xero is not configured on this instance")

const (
	maxBoundApps        = 512
	tenantTypeOrg       = "ORGANISATION"
	multicurrencyAction = "UseMulticurrency"
	limiterPrefix       = "xero"
	maxConcurrentPerOrg = 4
	endOfDayOffset      = 24*time.Hour - time.Second
	monthsInYear        = 12
)

//nolint:gosec // G101: a placeholder refresh token the endpoint rejects, not a credential
const credentialProbeToken = "trenova-credential-check"

var errNoRedirect = errors.New(
	"xero needs app.webBaseUrl or accounting.xero.redirectUrl to build its callback",
)

var errNoCompany = errors.New("xero did not authorise any organisation for this connection")

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
	apiOpts     []xero.Option
	oauthOpts   []xero.Option
	gate        *tenantGate
	codes       *accountCodes
	l           *zap.Logger

	mu    sync.Mutex
	bound map[string]*Connector
}

var _ accountingconnectorports.AccountingProvider = (*Provider)(nil)

type Connector struct {
	oauth      *xero.OAuthClient
	webhookKey string
	apiOpts    []xero.Option
	httpClient *http.Client
	codes      *accountCodes
	l          *zap.Logger
}

var _ accountingconnectorports.AccountingConnector = (*Connector)(nil)

func New(p Params) (*Provider, error) {
	cfg := p.AccountingConfig.Xero
	provider := &Provider{
		redirectURL: cfg.GetRedirectURL(p.AppConfig),
		gate:        newTenantGate(),
		codes:       newAccountCodes(time.Now),
		l:           p.Logger.Named("accounting.xero"),
		bound:       make(map[string]*Connector),
	}
	if p.Limiter != nil {
		provider.apiOpts = append(provider.apiOpts, xero.WithLimiter(p.Limiter, limiterPrefix))
	}
	if cfg.IsConfigured(p.AppConfig) {
		provider.instance = &accountingconnectorports.AccountingApp{
			Source:               accountingconnection.AppSourceInstance,
			Environment:          accountingprovider.AppEnvironmentProduction,
			ClientID:             strings.TrimSpace(cfg.ClientID),
			ClientSecret:         strings.TrimSpace(cfg.ClientSecret),
			WebhookVerifierToken: strings.TrimSpace(cfg.WebhookKey),
		}
	}

	return provider, nil
}

func (p *Provider) IntegrationType() integration.Type {
	return integration.TypeXero
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
		webhookKey: strings.TrimSpace(app.WebhookVerifierToken),
		apiOpts:    p.apiOpts,
		httpClient: &http.Client{Transport: p.gate.transport(http.DefaultTransport)},
		codes:      p.codes,
		l:          p.l,
	}
	if p.redirectURL != "" {
		oauth, err := xero.NewOAuthClient(xero.OAuthConfig{
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
	payload, err := xero.ParseWebhook(body)
	if err != nil {
		return nil, err
	}
	return payload.TenantIDs(), nil
}

func (p *Provider) WebhookSignatureHeader() string {
	return xero.SignatureHeader
}

func (p *Provider) ClassifyError(err error) accountingsync.ErrorCategory {
	return classifyError(err)
}

func bindingKey(app *accountingconnectorports.AccountingApp) string {
	return hashutils.SHA256Hex(strings.Join([]string{
		strings.TrimSpace(app.ClientID),
		strings.TrimSpace(app.ClientSecret),
		strings.TrimSpace(app.WebhookVerifierToken),
	}, "\x00"))
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeXero
}

func (c *Connector) options() []xero.Option {
	opts := make([]xero.Option, 0, len(c.apiOpts)+1)
	opts = append(opts, xero.WithHTTPClient(c.httpClient))
	return append(opts, c.apiOpts...)
}

func (c *Connector) client(
	auth accountingconnectorports.AccountingDocumentAuth,
) (*xero.Client, error) {
	return xero.New(auth.RealmID, auth.AccessToken, c.options()...)
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
	ctx context.Context,
	req *accountingconnectorports.AccountingRevokeRequest,
) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	return c.oauth.Revoke(ctx, req.Token)
}

func (c *Connector) connections(accessToken string) (*xero.ConnectionsClient, error) {
	return xero.NewConnectionsClient(accessToken, c.options()...)
}

func (c *Connector) Companies(
	ctx context.Context,
	grant *accountingconnectorports.AccountingTokenGrant,
	_ string,
) ([]accountingconnectorports.AccountingCompany, error) {
	if grant == nil {
		return nil, xeroerrors.ErrAccessTokenRequired
	}
	eventID, err := xero.AuthEventID(grant.AccessToken)
	if err != nil {
		return nil, err
	}
	client, err := c.connections(grant.AccessToken)
	if err != nil {
		return nil, err
	}
	listed, err := client.List(ctx, eventID)
	if err != nil {
		return nil, err
	}

	companies := make([]accountingconnectorports.AccountingCompany, 0, len(listed))
	for idx := range listed {
		conn := &listed[idx]
		if !strings.EqualFold(conn.TenantType, tenantTypeOrg) || conn.TenantID == "" {
			continue
		}
		companies = append(companies, accountingconnectorports.AccountingCompany{
			ID:           conn.TenantID,
			Name:         conn.TenantName,
			ConnectionID: conn.ID,
		})
	}
	if len(companies) == 0 {
		return nil, errNoCompany
	}
	return companies, nil
}

func (c *Connector) ReleaseCompanies(
	ctx context.Context,
	grant *accountingconnectorports.AccountingTokenGrant,
	companies []accountingconnectorports.AccountingCompany,
) error {
	if len(companies) == 0 {
		return nil
	}
	if grant == nil {
		return xeroerrors.ErrAccessTokenRequired
	}
	client, err := c.connections(grant.AccessToken)
	if err != nil {
		return err
	}
	var errs []error
	for idx := range companies {
		id := strings.TrimSpace(companies[idx].ConnectionID)
		if id == "" {
			continue
		}
		if delErr := client.Delete(ctx, id); delErr != nil && !xeroerrors.IsNotFound(delErr) {
			errs = append(errs, delErr)
		}
	}
	return errors.Join(errs...)
}

func (c *Connector) CompanyFacts(
	ctx context.Context,
	realmID, accessToken string,
) (*accountingconnection.CompanyFacts, error) {
	client, err := c.client(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     realmID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}

	org, err := client.Organisation(ctx)
	if err != nil {
		return nil, err
	}
	actions, err := client.OrganisationActions(ctx)
	if err != nil {
		return nil, err
	}

	facts := &accountingconnection.CompanyFacts{
		CompanyName:          org.Name,
		LegalName:            org.LegalName,
		Country:              org.CountryCode,
		HomeCurrency:         org.BaseCurrency,
		FiscalYearStartMonth: fiscalYearStart(org.FinancialYearEndMonth),
		ShortCode:            org.ShortCode,
	}
	for idx := range actions {
		if strings.EqualFold(actions[idx].Name, multicurrencyAction) {
			facts.MultiCurrencyEnabled = actions[idx].Allowed()
		}
	}
	if locked := laterOf(org.PeriodLockDate, org.EndOfYearLockDate); locked != nil {
		day := time.Date(locked.Year(), locked.Month(), locked.Day(), 0, 0, 0, 0, time.UTC)
		closed := day.Add(endOfDayOffset).Unix()
		facts.BooksClosedThrough = &closed
	}
	return facts, nil
}

func fiscalYearStart(yearEnd time.Month) time.Month {
	if yearEnd < time.January || yearEnd > time.December {
		return 0
	}
	return time.Month(int(yearEnd)%monthsInYear + 1)
}

func laterOf(first, second *time.Time) *time.Time {
	switch {
	case first == nil:
		return second
	case second == nil:
		return first
	case second.After(*first):
		return second
	default:
		return first
	}
}

func (c *Connector) VerifyApp(ctx context.Context) error {
	if c.oauth == nil {
		return errNoRedirect
	}
	_, err := c.oauth.Refresh(ctx, credentialProbeToken)
	switch {
	case err == nil, xeroerrors.IsInvalidGrant(err):
		return nil
	case xeroerrors.IsInvalidClient(err), xeroerrors.IsAuth(err):
		return accountingconnectorports.ErrAccountingAppRejected
	default:
		return err
	}
}

func (c *Connector) VerifyWebhook(signature string, body []byte) error {
	return xero.VerifySignature(c.webhookKey, body, signature)
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
	case xeroerrors.IsInvalidGrant(err):
		return accountingsync.ErrorCategoryRevoked
	case xeroerrors.IsRateLimited(err):
		return accountingsync.ErrorCategoryRateLimited
	case xeroerrors.IsInvalidClient(err), xeroerrors.IsAuth(err), xeroerrors.IsForbidden(err):
		return accountingsync.ErrorCategoryUnauthorized
	case xeroerrors.IsTransient(err), errors.Is(err, context.DeadlineExceeded):
		return accountingsync.ErrorCategoryTransient
	default:
		return accountingsync.ErrorCategoryUnknown
	}
}

func grantOf(token *xero.Token) *accountingconnectorports.AccountingTokenGrant {
	return &accountingconnectorports.AccountingTokenGrant{
		AccessToken:     token.AccessToken,
		RefreshToken:    token.RefreshToken,
		AccessTokenTTL:  token.ExpiresIn,
		RefreshTokenTTL: accountingprovider.MustProfile(integration.TypeXero).RefreshTokenLifetime,
	}
}
