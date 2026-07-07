package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIMessagesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	Cursor         pagination.CursorInfo    `json:"-"`
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

type GetServiceFailure214StatusRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	ServiceFailureID pulid.ID              `json:"serviceFailureId"`
}

type ServiceFailure214Status struct {
	ServiceFailureID  pulid.ID                        `json:"serviceFailureId"`
	ReviewedMessageID pulid.ID                        `json:"reviewedMessageId,omitempty"`
	ResolvedMessageID pulid.ID                        `json:"resolvedMessageId,omitempty"`
	LastMessageID     pulid.ID                        `json:"lastMessageId,omitempty"`
	GeneratedStatus   edi.MessageStatus               `json:"generatedStatus,omitempty"`
	DeliveryStatus    edi.MessageDeliveryStatus       `json:"deliveryStatus,omitempty"`
	AckStatus         edi.MessageAcknowledgmentStatus `json:"ackStatus,omitempty"`
	LastDiagnostic    string                          `json:"lastDiagnostic,omitempty"`
	LastGeneratedAt   int64                           `json:"lastGeneratedAt,omitempty"`
}

type GetEDIOutboundMessageForAckRequest struct {
	TenantInfo               pagination.TenantInfo `json:"tenantInfo"`
	PartnerID                pulid.ID              `json:"partnerId"`
	TransactionSet           edi.TransactionSet    `json:"transactionSet"`
	GroupControlNumber       string                `json:"groupControlNumber"`
	TransactionControlNumber string                `json:"transactionControlNumber"`
}

type UpdateEDIMessageAcknowledgmentRequest struct {
	ID            pulid.ID                        `json:"id"`
	TenantInfo    pagination.TenantInfo           `json:"tenantInfo"`
	AckStatus     edi.MessageAcknowledgmentStatus `json:"ackStatus"`
	AckMessageID  pulid.ID                        `json:"ackMessageId"`
	AckReceivedAt *int64                          `json:"ackReceivedAt"`
	AckLastError  string                          `json:"ackLastError"`
}

type UpdateEDIMessageDeliveryRequest struct {
	ID                    pulid.ID                  `json:"id"`
	TenantInfo            pagination.TenantInfo     `json:"tenantInfo"`
	DeliveryStatus        edi.MessageDeliveryStatus `json:"deliveryStatus"`
	DeliveryRemotePath    string                    `json:"deliveryRemotePath"`
	AS2MessageID          string                    `json:"as2MessageId"`
	AS2MIC                string                    `json:"as2Mic"`
	IncrementAttempts     bool                      `json:"incrementAttempts"`
	DeliveryLastAttemptAt *int64                    `json:"deliveryLastAttemptAt"`
	DeliverySentAt        *int64                    `json:"deliverySentAt"`
	DeliveryLastError     string                    `json:"deliveryLastError"`
}

type GetEDIMessageStatusCountsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Since      int64                 `json:"since"`
}

type GetEDIOverdueAckCountRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	PendingSince int64                 `json:"pendingSince"`
}

type ListRecentEDIMessageFailuresRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
}

type EDIMessageRepository interface {
	ListMessages(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.ListResult[*edi.EDIMessage], error)
	ListMessagesCursor(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.CursorListResult[*edi.EDIMessage], error)
	GetMessageByID(ctx context.Context, req GetEDIMessageByIDRequest) (*edi.EDIMessage, error)
	CreateMessageWithDiagnostics(
		ctx context.Context,
		req CreateEDIMessageWithDiagnosticsRequest,
	) (*edi.EDIMessage, error)
	GetServiceFailure214LifecycleMessage(
		ctx context.Context,
		req GetServiceFailure214LifecycleMessageRequest,
	) (*edi.EDIMessage, error)
	GetServiceFailure214Status(
		ctx context.Context,
		req GetServiceFailure214StatusRequest,
	) (*ServiceFailure214Status, error)
	UpdateMessageDelivery(
		ctx context.Context,
		req *UpdateEDIMessageDeliveryRequest,
	) (*edi.EDIMessage, error)
	GetOutboundMessageByAS2MessageID(
		ctx context.Context,
		as2MessageID string,
	) (*edi.EDIMessage, error)
	GetDeliveryStatusCounts(
		ctx context.Context,
		req GetEDIMessageStatusCountsRequest,
	) (map[edi.MessageDeliveryStatus]int, error)
	GetAckStatusCounts(
		ctx context.Context,
		req GetEDIMessageStatusCountsRequest,
	) (map[edi.MessageAcknowledgmentStatus]int, error)
	GetOverdueAckCount(ctx context.Context, req GetEDIOverdueAckCountRequest) (int, error)
	ListRecentDeadLettered(
		ctx context.Context,
		req *ListRecentEDIMessageFailuresRequest,
	) ([]*edi.EDIMessage, error)
	GetOutboundMessageForAck(
		ctx context.Context,
		req GetEDIOutboundMessageForAckRequest,
	) (*edi.EDIMessage, error)
	UpdateMessageAcknowledgment(
		ctx context.Context,
		req *UpdateEDIMessageAcknowledgmentRequest,
	) (*edi.EDIMessage, error)
}
