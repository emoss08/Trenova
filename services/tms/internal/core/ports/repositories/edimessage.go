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
	TransferID     pulid.ID                 `json:"transferId"`
	InboundFileID  pulid.ID                 `json:"inboundFileId"`
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

type GetEDIPartnerScorecardsRequest struct {
	TenantInfo             pagination.TenantInfo
	Since                  int64
	OverdueAckPendingSince int64
	PendingOver4hBefore    int64
	PendingOver24hBefore   int64
}

type EDIPartnerScorecardRow struct {
	PartnerID           pulid.ID `bun:"partner_id"`
	PartnerName         string   `bun:"partner_name"`
	PartnerCode         string   `bun:"partner_code"`
	OutboundTotal       int64    `bun:"outbound_total"`
	SentCount           int64    `bun:"sent_count"`
	FailedCount         int64    `bun:"failed_count"`
	DeadLetteredCount   int64    `bun:"dead_lettered_count"`
	ReceivedCount       int64    `bun:"received_count"`
	AvgAckSeconds       *float64 `bun:"avg_ack_seconds"`
	P95AckSeconds       *float64 `bun:"p95_ack_seconds"`
	OverdueAckCount     int64    `bun:"overdue_ack_count"`
	PendingOver4hCount  int64    `bun:"pending_over_4h_count"`
	PendingOver24hCount int64    `bun:"pending_over_24h_count"`
	OldestPendingAt     *int64   `bun:"oldest_pending_at"`
}

type GetEDIVolumeSeriesRequest struct {
	TenantInfo    pagination.TenantInfo
	Since         int64
	BucketSeconds int64
}

type EDIVolumePoint struct {
	BucketStart   int64 `bun:"bucket_start"`
	OutboundCount int64 `bun:"outbound_count"`
	SentCount     int64 `bun:"sent_count"`
	FailedCount   int64 `bun:"failed_count"`
	ReceivedCount int64 `bun:"received_count"`
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
	CountDeadLetteredSince(ctx context.Context, since int64) (int64, error)
	GetPartnerScorecards(
		ctx context.Context,
		req *GetEDIPartnerScorecardsRequest,
	) ([]*EDIPartnerScorecardRow, error)
	GetVolumeSeries(ctx context.Context, req GetEDIVolumeSeriesRequest) ([]*EDIVolumePoint, error)
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
