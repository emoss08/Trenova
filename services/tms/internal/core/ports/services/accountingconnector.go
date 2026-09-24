package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
)

type AccountingTokenGrant struct {
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type AccountingConnector interface {
	IntegrationType() integration.Type
	Available() bool
	AuthorizeURL(state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*AccountingTokenGrant, error)
	Refresh(ctx context.Context, refreshToken string) (*AccountingTokenGrant, error)
	Revoke(ctx context.Context, token string) error
	CompanyFacts(
		ctx context.Context,
		realmID, accessToken string,
	) (*accountingsync.CompanyFacts, error)
	VerifyWebhook(signature string, body []byte) error
	WebhookRealmIDs(body []byte) ([]string, error)
	WebhookSignatureHeader() string
	ClassifyError(err error) accountingsync.ErrorCategory
}

type AccountingConnectorRegistry interface {
	For(typ integration.Type) (AccountingConnector, bool)
}
