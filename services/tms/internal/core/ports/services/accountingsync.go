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
	Connection      *accountingsync.AccountingConnection
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
	CheckDue(ctx context.Context, limit int) (int, error)
	ReceiveWebhook(ctx context.Context, req *ReceiveAccountingWebhookRequest) error
}
