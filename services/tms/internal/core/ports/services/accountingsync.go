package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingSyncStatus struct {
	IntegrationType integration.Type
	ProviderName    string
	Available       bool
	App             *AccountingAppSettings
	Connection      *accountingsync.AccountingConnection
}

type AccountingAppSettings struct {
	ActiveSource         accountingsync.AppSource
	InstanceAppAvailable bool
	InstanceEnvironment  accountingsync.AppEnvironment
	RedirectURL          string
	WebhookPath          string
	TenantApp            *accountingsync.AccountingAppCredential
}

type SaveAccountingAppRequest struct {
	TenantInfo                pagination.TenantInfo
	UserID                    pulid.ID
	IntegrationType           integration.Type
	Environment               accountingsync.AppEnvironment
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

type DisconnectAccountingRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type ReceiveAccountingWebhookRequest struct {
	IntegrationType integration.Type
	Signature       string
	Body            []byte
}

type AccountingHealthSweep struct {
	Listed  int
	Checked int
	Failed  int
}

type AccountingConnectionService interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*AccountingSyncStatus, error)
	StartAuthorization(
		ctx context.Context,
		req *StartAccountingAuthorizationRequest,
	) (*AccountingAuthorizationStart, error)
	CompleteAuthorization(
		ctx context.Context,
		req *CompleteAccountingAuthorizationRequest,
	) (*accountingsync.AccountingConnection, error)
	Disconnect(
		ctx context.Context,
		req *DisconnectAccountingRequest,
	) (*accountingsync.AccountingConnection, error)
	CheckHealth(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) (*accountingsync.AccountingConnection, error)
	CheckDue(ctx context.Context, limit int) (*AccountingHealthSweep, error)
	ReceiveWebhook(ctx context.Context, req *ReceiveAccountingWebhookRequest) error
	SaveApp(ctx context.Context, req *SaveAccountingAppRequest) (*AccountingSyncStatus, error)
	RemoveApp(ctx context.Context, req *RemoveAccountingAppRequest) (*AccountingSyncStatus, error)
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

type AccountingSession struct {
	Connection  *accountingsync.AccountingConnection
	AccessToken string
	Connector   AccountingConnector
}

type AccountingReferenceRefresher interface {
	RequestReferenceRefresh(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) error
}
