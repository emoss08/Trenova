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
