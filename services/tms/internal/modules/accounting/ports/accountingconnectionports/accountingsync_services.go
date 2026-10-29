package accountingconnectionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingSyncStatus struct {
	IntegrationType      integration.Type
	ProviderName         string
	Profile              accountingprovider.ProviderProfile
	Available            bool
	App                  *AccountingAppSettings
	Connection           *accountingconnection.AccountingConnection
	WebhookSubscriptions *AccountingWebhookSubscriptionSummary
	PendingCertificate   *AccountingPreparedCertificate
}

type AccountingWebhookSubscriptionSummary struct {
	Configured   bool
	Active       int
	Pending      int
	Failed       int
	NextExpiryAt *int64
	LastError    string
}

type SyncAccountingWebhookSubscriptionsRequest struct {
	AfterID pulid.ID
	Limit   int
}

type AccountingSubscriptionSweep struct {
	Listed      int
	Created     int
	Renewed     int
	Removed     int
	Failed      int
	NextAfterID pulid.ID
}

type AccountingAppSettings struct {
	ActiveSource         accountingconnection.AppSource
	InstanceAppAvailable bool
	InstanceEnvironment  accountingprovider.AppEnvironment
	RedirectURL          string
	WebhookPath          string
	TenantApp            *accountingconnection.AccountingAppCredential
}

type SaveAccountingAppRequest struct {
	TenantInfo                pagination.TenantInfo
	UserID                    pulid.ID
	IntegrationType           integration.Type
	Environment               accountingprovider.AppEnvironment
	ClientID                  string
	ClientSecret              string
	WebhookVerifierToken      string
	ClearWebhookVerifierToken bool
}

type RemoveAccountingAppRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type StartAccountingAuthorizationRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type AccountingAuthorizationStart struct {
	AuthorizeURL string
	ExpiresAt    int64
}

type CompleteAccountingAuthorizationRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	State           string
	Code            string
	RealmID         string
}

type AccountingAuthorizationCompletion struct {
	Connection      *accountingconnection.AccountingConnection
	Companies       []accountingconnectorports.AccountingCompany
	ChoiceToken     string
	ChoiceExpiresAt int64
}

type PrepareAccountingCertificateRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type AccountingPreparedCertificate struct {
	IntegrationType integration.Type
	CertificatePEM  string
	ExpiresAt       int64
	PreparedAt      int64
}

type ConnectAccountingWithCertificateRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	AccountID       string
	CertificateID   string
}

type RenewAccountingCertificateRequest struct {
	TenantInfo    pagination.TenantInfo
	UserID        pulid.ID
	ConnectionID  pulid.ID
	CertificateID string
}

type ChooseAccountingCompanyRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	ChoiceToken     string
	CompanyID       string
}

type DisconnectAccountingRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type ReceiveAccountingWebhookRequest struct {
	IntegrationType integration.Type
	AppID           string
	Signature       string
	Body            []byte
}

type AccountingHealthSweep struct {
	Listed  int
	Checked int
	Failed  int
}

type AccountingConnectionStatusReader interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*AccountingSyncStatus, error)
}

type AccountingAuthorizationService interface {
	StartAuthorization(
		ctx context.Context,
		req *StartAccountingAuthorizationRequest,
	) (*AccountingAuthorizationStart, error)
	CompleteAuthorization(
		ctx context.Context,
		req *CompleteAccountingAuthorizationRequest,
	) (*AccountingAuthorizationCompletion, error)
	ChooseCompany(
		ctx context.Context,
		req *ChooseAccountingCompanyRequest,
	) (*accountingconnection.AccountingConnection, error)
	Disconnect(
		ctx context.Context,
		req *DisconnectAccountingRequest,
	) (*accountingconnection.AccountingConnection, error)
}

type AccountingCertificateService interface {
	PrepareCertificate(
		ctx context.Context,
		req *PrepareAccountingCertificateRequest,
	) (*AccountingPreparedCertificate, error)
	ConnectWithCertificate(
		ctx context.Context,
		req *ConnectAccountingWithCertificateRequest,
	) (*AccountingAuthorizationCompletion, error)
	RenewCertificate(
		ctx context.Context,
		req *RenewAccountingCertificateRequest,
	) (*accountingconnection.AccountingConnection, error)
}

type AccountingHealthChecker interface {
	CheckHealth(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) (*accountingconnection.AccountingConnection, error)
	CheckDue(ctx context.Context, limit int) (*AccountingHealthSweep, error)
}

type AccountingSessionProvider interface {
	Session(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) (*AccountingSession, error)
	ReportCallFailure(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
		cause error,
	) accountingsync.ErrorCategory
}

type AccountingWebhookSubscriptionSyncer interface {
	SyncWebhookSubscriptions(
		ctx context.Context,
		req *SyncAccountingWebhookSubscriptionsRequest,
	) (*AccountingSubscriptionSweep, error)
}

type AccountingWebhookReceiver interface {
	ReceiveWebhook(ctx context.Context, req *ReceiveAccountingWebhookRequest) error
}

type AccountingAppService interface {
	SaveApp(ctx context.Context, req *SaveAccountingAppRequest) (*AccountingSyncStatus, error)
	RemoveApp(ctx context.Context, req *RemoveAccountingAppRequest) (*AccountingSyncStatus, error)
}

type AccountingSession struct {
	Connection  *accountingconnection.AccountingConnection
	AccessToken string
	Connector   accountingconnectorports.AccountingConnector
}

type AccountingReferenceRefresher interface {
	RequestReferenceRefresh(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) error
}
