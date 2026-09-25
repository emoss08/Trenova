package qboconnector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrNotConfigured = errors.New("quickbooks online is not configured on this instance")

type Params struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Limiter restx.Limiter `name:"accountingLimiter" optional:"true"`
}

type Connector struct {
	env      quickbooks.Environment
	oauth    *quickbooks.OAuthClient
	verifier string
	apiOpts  []quickbooks.Option
	l        *zap.Logger
}

func New(p Params) (*Connector, error) {
	qbo := p.Config.Accounting.QuickBooks
	env, ok := quickbooks.ParseEnvironment(qbo.GetEnvironment())
	if !ok {
		return nil, fmt.Errorf("quickbooks: unknown environment %q", qbo.GetEnvironment())
	}

	conn := &Connector{
		env:      env,
		verifier: strings.TrimSpace(qbo.WebhookVerifierToken),
		l:        p.Logger.Named("accounting.quickbooks"),
	}
	if p.Limiter != nil {
		conn.apiOpts = append(conn.apiOpts, quickbooks.WithLimiter(p.Limiter, ""))
	}
	if !qbo.IsConfigured(&p.Config.App) {
		return conn, nil
	}

	oauth, err := quickbooks.NewOAuthClient(quickbooks.OAuthConfig{
		ClientID:     qbo.ClientID,
		ClientSecret: qbo.ClientSecret,
		RedirectURL:  qbo.GetRedirectURL(&p.Config.App),
	})
	if err != nil {
		return nil, err
	}
	conn.oauth = oauth

	return conn, nil
}

func (c *Connector) IntegrationType() integration.Type {
	return integration.TypeQuickBooksOnline
}

func (c *Connector) Available() bool {
	return c.oauth != nil
}

func (c *Connector) AuthorizeURL(state string) (string, error) {
	if c.oauth == nil {
		return "", ErrNotConfigured
	}
	return c.oauth.AuthorizeURL(state)
}

func (c *Connector) ExchangeCode(
	ctx context.Context,
	code string,
) (*services.AccountingTokenGrant, error) {
	if c.oauth == nil {
		return nil, ErrNotConfigured
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
		return nil, ErrNotConfigured
	}
	token, err := c.oauth.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return grantOf(token), nil
}

func (c *Connector) Revoke(ctx context.Context, token string) error {
	if c.oauth == nil {
		return ErrNotConfigured
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

func (c *Connector) VerifyWebhook(signature string, body []byte) error {
	return quickbooks.VerifySignature(c.verifier, signature, body)
}

func (c *Connector) WebhookRealmIDs(body []byte) ([]string, error) {
	return quickbooks.ParseRealmIDs(body)
}

func (c *Connector) WebhookSignatureHeader() string {
	return quickbooks.SignatureHeader
}

func (c *Connector) ClassifyError(err error) accountingsync.ErrorCategory {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotConfigured):
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
