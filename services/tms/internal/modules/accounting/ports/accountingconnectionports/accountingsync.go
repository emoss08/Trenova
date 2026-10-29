package accountingconnectionports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingConnectionRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
}

type GetAccountingConnectionByIDRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListAccountingConnectionsByRealmRequest struct {
	IntegrationType integration.Type
	RealmIDs        []string
}

type ListDueAccountingConnectionsRequest struct {
	CheckedBefore int64
	Limit         int
}

type StoreAccountingTokensRequest struct {
	TenantInfo             pagination.TenantInfo
	ID                     pulid.ID
	AccessTokenCiphertext  string
	AccessTokenExpiresAt   int64
	RefreshTokenCiphertext string
	RefreshTokenExpiresAt  int64
	ClientKeyCiphertext    string
	RefreshedAt            *int64
	At                     int64
}

type MarkAccountingWebhookRequest struct {
	IntegrationType integration.Type
	RealmIDs        []string
	ReceivedAt      int64
}

type MarkAccountingReferenceRefreshRequest struct {
	TenantInfo  pagination.TenantInfo
	ID          pulid.ID
	StartedAt   *int64
	RefreshedAt *int64
	Error       string
}

type SaveAccountingChangeFeedRequest struct {
	TenantInfo    pagination.TenantInfo
	ID            pulid.ID
	Cursor        string
	OnlyIfEmpty   bool
	ReadAt        *int64
	ErrorCategory accountingsync.SyncErrorCategory
	ErrorMessage  string
}

type SaveAccountingDriftCheckRequest struct {
	TenantInfo    pagination.TenantInfo
	ID            pulid.ID
	CheckedAt     *int64
	ErrorCategory accountingsync.SyncErrorCategory
	ErrorMessage  string
}

type ListActiveAccountingConnectionsRequest struct {
	AfterID pulid.ID
	Limit   int
}

type AccountingConnectionRepository interface {
	GetByType(
		ctx context.Context,
		req GetAccountingConnectionRequest,
	) (*accountingconnection.AccountingConnection, error)
	GetByID(
		ctx context.Context,
		req GetAccountingConnectionByIDRequest,
	) (*accountingconnection.AccountingConnection, error)
	ListByTenant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*accountingconnection.AccountingConnection, error)
	ListHoldingRealm(
		ctx context.Context,
		req ListAccountingConnectionsByRealmRequest,
	) ([]*accountingconnection.AccountingConnection, error)
	ListDueForHealthCheck(
		ctx context.Context,
		req ListDueAccountingConnectionsRequest,
	) ([]*accountingconnection.AccountingConnection, error)
	LockByTypeWithTokens(
		ctx context.Context,
		req GetAccountingConnectionRequest,
	) (*accountingconnection.AccountingConnection, error)
	LockWithTokens(
		ctx context.Context,
		req GetAccountingConnectionByIDRequest,
	) (*accountingconnection.AccountingConnection, error)
	Create(
		ctx context.Context,
		entity *accountingconnection.AccountingConnection,
	) (*accountingconnection.AccountingConnection, error)
	Update(
		ctx context.Context,
		entity *accountingconnection.AccountingConnection,
	) (*accountingconnection.AccountingConnection, error)
	StoreTokens(ctx context.Context, req StoreAccountingTokensRequest) error
	MarkWebhookReceived(ctx context.Context, req MarkAccountingWebhookRequest) (int64, error)
	MarkReferenceRefresh(ctx context.Context, req MarkAccountingReferenceRefreshRequest) error
	SaveChangeFeed(ctx context.Context, req *SaveAccountingChangeFeedRequest) (bool, error)
	SaveDriftCheck(ctx context.Context, req *SaveAccountingDriftCheckRequest) error
	ListActive(
		ctx context.Context,
		req ListActiveAccountingConnectionsRequest,
	) ([]*accountingconnection.AccountingConnection, error)
}

type AccountingOAuthState struct {
	State           string                         `json:"state"`
	IntegrationType integration.Type               `json:"integrationType"`
	UserID          pulid.ID                       `json:"userId"`
	OrganizationID  pulid.ID                       `json:"organizationId"`
	BusinessUnitID  pulid.ID                       `json:"businessUnitId"`
	CreatedAt       int64                          `json:"createdAt"`
	AppSource       accountingconnection.AppSource `json:"appSource"`
	AppFingerprint  string                         `json:"appFingerprint"`
	SealedGrant     string                         `json:"sealedGrant,omitempty"`
	SealedVerifier  string                         `json:"sealedVerifier,omitempty"`
	GrantRealmID    string                         `json:"grantRealmId,omitempty"`
	Companies       []AccountingOAuthCompany       `json:"companies,omitempty"`
}

type AccountingOAuthCompany struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ConnectionID string `json:"connectionId"`
}

type AccountingOAuthStateRepository interface {
	Save(ctx context.Context, state *AccountingOAuthState, ttl time.Duration) error
	Take(ctx context.Context, state string) (*AccountingOAuthState, error)
}

type GetAccountingAppCredentialRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
}

type AccountingPendingCertificateRepository interface {
	GetByType(
		ctx context.Context,
		req GetAccountingConnectionRequest,
	) (*accountingconnection.AccountingPendingCertificate, error)
	Replace(
		ctx context.Context,
		entity *accountingconnection.AccountingPendingCertificate,
	) (*accountingconnection.AccountingPendingCertificate, error)
	Delete(ctx context.Context, req GetAccountingConnectionRequest) error
}

type AccountingAppCredentialRepository interface {
	GetByType(
		ctx context.Context,
		req GetAccountingAppCredentialRequest,
	) (*accountingconnection.AccountingAppCredential, error)
	Create(
		ctx context.Context,
		entity *accountingconnection.AccountingAppCredential,
	) (*accountingconnection.AccountingAppCredential, error)
	Update(
		ctx context.Context,
		entity *accountingconnection.AccountingAppCredential,
	) (*accountingconnection.AccountingAppCredential, error)
	Delete(ctx context.Context, req GetAccountingAppCredentialRequest) error
	GetForWebhook(
		ctx context.Context,
		id pulid.ID,
		integrationType integration.Type,
	) (*accountingconnection.AccountingAppCredential, error)
}

type ListAccountingWebhookSubscriptionsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
}

type DeleteAccountingWebhookSubscriptionRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListWebhookSubscriptionConnectionsRequest struct {
	IntegrationTypes []integration.Type
	AfterID          pulid.ID
	Limit            int
}

type AccountingWebhookSubscriptionRepository interface {
	ListByConnection(
		ctx context.Context,
		req ListAccountingWebhookSubscriptionsRequest,
	) ([]*accountingconnection.AccountingWebhookSubscription, error)
	ListByExternalIDs(
		ctx context.Context,
		integrationType integration.Type,
		externalIDs []string,
	) ([]*accountingconnection.AccountingWebhookSubscription, error)
	ListConnections(
		ctx context.Context,
		req ListWebhookSubscriptionConnectionsRequest,
	) ([]*accountingconnection.AccountingConnection, error)
	Create(
		ctx context.Context,
		entity *accountingconnection.AccountingWebhookSubscription,
	) (*accountingconnection.AccountingWebhookSubscription, error)
	Update(
		ctx context.Context,
		entity *accountingconnection.AccountingWebhookSubscription,
	) (*accountingconnection.AccountingWebhookSubscription, error)
	Delete(ctx context.Context, req DeleteAccountingWebhookSubscriptionRequest) error
}
