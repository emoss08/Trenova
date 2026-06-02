package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIMessagesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
	PartnerID      pulid.ID                 `json:"partnerId"`
	Status         edi.MessageStatus        `json:"status"`
	Query          string                   `json:"query"`
	GeneratedFrom  int64                    `json:"generatedFrom"`
	GeneratedTo    int64                    `json:"generatedTo"`
}

type GetEDIMessageByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDIMessageWithDiagnosticsRequest struct {
	Message     *edi.EDIMessage                  `json:"message"`
	Diagnostics []*edi.EDIMessageValidationError `json:"diagnostics"`
}

type GetServiceFailure214LifecycleMessageRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	ServiceFailureID pulid.ID              `json:"serviceFailureId"`
	Trigger          string                `json:"trigger"`
}

type UpdateEDIMessageDeliveryRequest struct {
	ID                    pulid.ID                  `json:"id"`
	TenantInfo            pagination.TenantInfo     `json:"tenantInfo"`
	DeliveryStatus        edi.MessageDeliveryStatus `json:"deliveryStatus"`
	DeliveryRemotePath    string                    `json:"deliveryRemotePath"`
	IncrementAttempts     bool                      `json:"incrementAttempts"`
	DeliveryLastAttemptAt *int64                    `json:"deliveryLastAttemptAt"`
	DeliverySentAt        *int64                    `json:"deliverySentAt"`
	DeliveryLastError     string                    `json:"deliveryLastError"`
}

type EDIMessageRepository interface {
	ListMessages(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.ListResult[*edi.EDIMessage], error)
	GetMessageByID(ctx context.Context, req GetEDIMessageByIDRequest) (*edi.EDIMessage, error)
	CreateMessageWithDiagnostics(
		ctx context.Context,
		req CreateEDIMessageWithDiagnosticsRequest,
	) (*edi.EDIMessage, error)
	GetServiceFailure214LifecycleMessage(
		ctx context.Context,
		req GetServiceFailure214LifecycleMessageRequest,
	) (*edi.EDIMessage, error)
	UpdateMessageDelivery(
		ctx context.Context,
		req *UpdateEDIMessageDeliveryRequest,
	) (*edi.EDIMessage, error)
}
