package services

import (
	"context"
	"errors"
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

var ErrAccountingAppRejected = errors.New("the accounting system did not accept the app's client id or secret")

type AccountingApp struct {
	Source               accountingsync.AppSource
	Environment          accountingsync.AppEnvironment
	ClientID             string
	ClientSecret         string
	WebhookVerifierToken string
}

func (a *AccountingApp) Identity() accountingsync.AppIdentity {
	return accountingsync.AppIdentity{
		Source:      a.Source,
		Environment: a.Environment,
		Fingerprint: accountingsync.AppFingerprint(a.Environment, a.ClientID),
	}
}

type AccountingConnector interface {
	IntegrationType() integration.Type
	AuthorizeURL(state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*AccountingTokenGrant, error)
	Refresh(ctx context.Context, refreshToken string) (*AccountingTokenGrant, error)
	Revoke(ctx context.Context, token string) error
	CompanyFacts(
		ctx context.Context,
		realmID, accessToken string,
	) (*accountingsync.CompanyFacts, error)
	VerifyApp(ctx context.Context) error
	VerifyWebhook(signature string, body []byte) error
	ClassifyError(err error) accountingsync.ErrorCategory
}

type AccountingProvider interface {
	IntegrationType() integration.Type
	InstanceApp() (*AccountingApp, bool)
	RedirectURL() string
	Bind(app *AccountingApp) (AccountingConnector, error)
	WebhookRealmIDs(body []byte) ([]string, error)
	WebhookSignatureHeader() string
	ClassifyError(err error) accountingsync.ErrorCategory
}

type AccountingConnectorRegistry interface {
	For(typ integration.Type) (AccountingProvider, bool)
}

type AccountingReferencePageRequest struct {
	RealmID       string
	AccessToken   string
	Kind          accountingsync.ReferenceKind
	StartPosition int
	PageSize      int
}

type AccountingReferencePage struct {
	Objects   []*accountingsync.AccountingReferenceObject
	NextStart int
}

type AccountingItemDraft struct {
	Name            string
	Description     string
	Sku             string
	IncomeAccountID string
}

type AccountingPartyDraft struct {
	DisplayName  string
	CompanyName  string
	Email        string
	AddressLine1 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Is1099       bool
}

type AccountingCreateReferenceRequest struct {
	RealmID     string
	AccessToken string
	RequestID   string
	Kind        accountingsync.ReferenceKind
	Item        *AccountingItemDraft
	Party       *AccountingPartyDraft
}

type AccountingReferenceReader interface {
	ListReference(
		ctx context.Context,
		req *AccountingReferencePageRequest,
	) (*AccountingReferencePage, error)
	MaxReferencePageSize() int
}

type AccountingReferenceCreator interface {
	CreateReference(
		ctx context.Context,
		req *AccountingCreateReferenceRequest,
	) (*accountingsync.AccountingReferenceObject, error)
	IsDuplicateName(err error) bool
	SanitizeName(kind accountingsync.ReferenceKind, name string) string
}
