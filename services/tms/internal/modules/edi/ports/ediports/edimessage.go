package ediports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIMessagesRequest struct {
	Filter         *pagination.QueryOptions        `json:"filter"`
	Cursor         pagination.CursorInfo           `json:"-"`
	TransactionSet edi.TransactionSet              `json:"transactionSet"`
	Direction      edi.DocumentDirection           `json:"direction"`
	PartnerID      pulid.ID                        `json:"partnerId"`
	Status         edi.MessageStatus               `json:"status"`
	Query          string                          `json:"query"`
	GeneratedFrom  int64                           `json:"generatedFrom"`
	GeneratedTo    int64                           `json:"generatedTo"`
	AckStatus      edi.MessageAcknowledgmentStatus `json:"ackStatus"`
	ShipmentID     pulid.ID                        `json:"shipmentId"`
	InvoiceID      pulid.ID                        `json:"invoiceId"`
	CreatedAfter   int64                           `json:"createdAfter"`
	CreatedBefore  int64                           `json:"createdBefore"`
}

type GetEDIMessageByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDIMessageByIdempotencyKeyRequest struct {
	TenantInfo     pagination.TenantInfo
	IdempotencyKey string
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

type ServiceFailure214StatusRow struct {
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

type LinkEDIMessageFlowRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	FlowInstanceID pulid.ID              `json:"flowInstanceId"`
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
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	Since          int64                 `json:"since"`
	PartnerID      pulid.ID              `json:"partnerId"`
	TransactionSet edi.TransactionSet    `json:"transactionSet,omitempty"`
}

type GetEDIPartnerScorecardsRequest struct {
	TenantInfo             pagination.TenantInfo
	Since                  int64
	PartnerID              pulid.ID
	OverdueAckPendingSince int64
	PendingOver4hBefore    int64
	PendingOver24hBefore   int64
}

type EDIPartnerScorecardRow struct {
	PartnerID           pulid.ID
	PartnerName         string
	PartnerCode         string
	OutboundTotal       int64
	SentCount           int64
	FailedCount         int64
	DeadLetteredCount   int64
	ReceivedCount       int64
	AvgAckSeconds       *float64
	P95AckSeconds       *float64
	OverdueAckCount     int64
	PendingOver4hCount  int64
	PendingOver24hCount int64
	OldestPendingAt     *int64
}

type GetEDIVolumeSeriesRequest struct {
	TenantInfo    pagination.TenantInfo
	Since         int64
	BucketSeconds int64
	PartnerID     pulid.ID
}

type GetEDIPartnerSetActivityRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerID  pulid.ID
	Since      int64
}

type EDIPartnerSetActivityRow struct {
	TransactionSet    edi.TransactionSet
	Direction         edi.DocumentDirection
	Count             int64
	FailedCount       int64
	DeadLetteredCount int64
	LastAt            int64
}

type EDIVolumePoint struct {
	BucketStart   int64
	OutboundCount int64
	SentCount     int64
	FailedCount   int64
	ReceivedCount int64
}

type GetEDIOverdueAckCountRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	PendingSince   int64                 `json:"pendingSince"`
	PartnerID      pulid.ID              `json:"partnerId"`
	TransactionSet edi.TransactionSet    `json:"transactionSet,omitempty"`
}

type ListOverdueEDIAcksRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	PendingSince   int64                 `json:"pendingSince"`
	PartnerID      pulid.ID              `json:"partnerId"`
	TransactionSet edi.TransactionSet    `json:"transactionSet,omitempty"`
	Limit          int                   `json:"limit"`
}

type OverdueEDIAckPartner struct {
	PartnerID         pulid.ID
	PartnerName       string
	PartnerCode       string
	Count             int
	OldestGeneratedAt int64
}

type EDIPartnerAckStatusCount struct {
	PartnerID   pulid.ID
	PartnerName string
	PartnerCode string
	AckStatus   edi.MessageAcknowledgmentStatus
	Count       int
}

type OverdueEDIAcks struct {
	Messages []*edi.EDIMessage
	Partners []*OverdueEDIAckPartner
}

type ListRecentEDIMessageFailuresRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	PartnerID      pulid.ID              `json:"partnerId"`
	TransactionSet edi.TransactionSet    `json:"transactionSet,omitempty"`
	Since          int64                 `json:"since"`
	Limit          int                   `json:"limit"`
}

type ListEDIReplayMessagesRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	PartnerID      pulid.ID              `json:"partnerId"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
	From           int64                 `json:"from"`
	To             int64                 `json:"to"`
	AfterID        pulid.ID              `json:"afterId"`
	Limit          int                   `json:"limit"`
}

type EDIMessageRepository interface {
	ListReplayMessages(
		ctx context.Context,
		req *ListEDIReplayMessagesRequest,
	) ([]*edi.EDIMessage, error)
	ListMessages(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.ListResult[*edi.EDIMessage], error)
	ListMessagesCursor(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.CursorListResult[*edi.EDIMessage], error)
	GetMessageByID(ctx context.Context, req GetEDIMessageByIDRequest) (*edi.EDIMessage, error)
	GetMessageByIdempotencyKey(
		ctx context.Context,
		req GetEDIMessageByIdempotencyKeyRequest,
	) (*edi.EDIMessage, error)
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
	) (*ServiceFailure214StatusRow, error)
	LinkMessageFlow(ctx context.Context, req *LinkEDIMessageFlowRequest) (*edi.EDIMessage, error)
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
	GetAckStatusCountsByPartner(
		ctx context.Context,
		req GetEDIMessageStatusCountsRequest,
	) ([]*EDIPartnerAckStatusCount, error)
	GetOverdueAckCount(ctx context.Context, req GetEDIOverdueAckCountRequest) (int, error)
	ListOverdueAcks(ctx context.Context, req *ListOverdueEDIAcksRequest) (*OverdueEDIAcks, error)
	ListRecentDeadLettered(
		ctx context.Context,
		req *ListRecentEDIMessageFailuresRequest,
	) ([]*edi.EDIMessage, error)
	CountDeadLetteredSince(ctx context.Context, since int64) (int64, error)
	GetPartnerScorecards(
		ctx context.Context,
		req *GetEDIPartnerScorecardsRequest,
	) ([]*EDIPartnerScorecardRow, error)
	GetVolumeSeries(ctx context.Context, req GetEDIVolumeSeriesRequest) ([]*EDIVolumePoint, error)
	GetPartnerSetActivity(
		ctx context.Context,
		req *GetEDIPartnerSetActivityRequest,
	) ([]*EDIPartnerSetActivityRow, error)
	PurgeRawX12Before(ctx context.Context, req PurgeEDIRawPayloadsRequest) (int64, error)
	GetOutboundMessageForAck(
		ctx context.Context,
		req GetEDIOutboundMessageForAckRequest,
	) (*edi.EDIMessage, error)
	UpdateMessageAcknowledgment(
		ctx context.Context,
		req *UpdateEDIMessageAcknowledgmentRequest,
	) (*edi.EDIMessage, error)
}
