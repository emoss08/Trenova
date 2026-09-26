package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
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
	) (*accountingsync.AccountingConnection, error)
	GetByID(
		ctx context.Context,
		req GetAccountingConnectionByIDRequest,
	) (*accountingsync.AccountingConnection, error)
	ListByTenant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*accountingsync.AccountingConnection, error)
	ListHoldingRealm(
		ctx context.Context,
		req ListAccountingConnectionsByRealmRequest,
	) ([]*accountingsync.AccountingConnection, error)
	ListDueForHealthCheck(
		ctx context.Context,
		req ListDueAccountingConnectionsRequest,
	) ([]*accountingsync.AccountingConnection, error)
	LockByTypeWithTokens(
		ctx context.Context,
		req GetAccountingConnectionRequest,
	) (*accountingsync.AccountingConnection, error)
	LockWithTokens(
		ctx context.Context,
		req GetAccountingConnectionByIDRequest,
	) (*accountingsync.AccountingConnection, error)
	Create(
		ctx context.Context,
		entity *accountingsync.AccountingConnection,
	) (*accountingsync.AccountingConnection, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingConnection,
	) (*accountingsync.AccountingConnection, error)
	StoreTokens(ctx context.Context, req StoreAccountingTokensRequest) error
	MarkWebhookReceived(ctx context.Context, req MarkAccountingWebhookRequest) (int64, error)
	MarkReferenceRefresh(ctx context.Context, req MarkAccountingReferenceRefreshRequest) error
	SaveChangeFeed(ctx context.Context, req *SaveAccountingChangeFeedRequest) (bool, error)
	SaveDriftCheck(ctx context.Context, req *SaveAccountingDriftCheckRequest) error
	ListActive(
		ctx context.Context,
		req ListActiveAccountingConnectionsRequest,
	) ([]*accountingsync.AccountingConnection, error)
}

type AccountingOAuthState struct {
	State           string                   `json:"state"`
	IntegrationType integration.Type         `json:"integrationType"`
	UserID          pulid.ID                 `json:"userId"`
	OrganizationID  pulid.ID                 `json:"organizationId"`
	BusinessUnitID  pulid.ID                 `json:"businessUnitId"`
	CreatedAt       int64                    `json:"createdAt"`
	AppSource       accountingsync.AppSource `json:"appSource"`
	AppFingerprint  string                   `json:"appFingerprint"`
}

type AccountingOAuthStateRepository interface {
	Save(ctx context.Context, state *AccountingOAuthState, ttl time.Duration) error
	Take(ctx context.Context, state string) (*AccountingOAuthState, error)
}

type GetAccountingAppCredentialRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
}

type AccountingAppCredentialRepository interface {
	GetByType(
		ctx context.Context,
		req GetAccountingAppCredentialRequest,
	) (*accountingsync.AccountingAppCredential, error)
	Create(
		ctx context.Context,
		entity *accountingsync.AccountingAppCredential,
	) (*accountingsync.AccountingAppCredential, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingAppCredential,
	) (*accountingsync.AccountingAppCredential, error)
	Delete(ctx context.Context, req GetAccountingAppCredentialRequest) error
}
