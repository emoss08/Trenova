package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetMailboxByTokenHashRequest carries the hash rather than the token.
//
// The token itself never reaches the repository: hashing is what makes the
// stored row useless to whoever reads it, and a repository that took the plain
// token would put it back in the query log.
type GetMailboxByTokenHashRequest struct {
	TokenHash string `json:"-"`
}

type ListMailboxesRequest struct {
	Filter *pagination.QueryOptions     `json:"filter"`
	Status inboundmessage.MailboxStatus `json:"status"`
}

type GetMailboxByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListInboundMessagesRequest struct {
	Filter         *pagination.QueryOptions      `json:"filter"`
	Cursor         pagination.CursorInfo         `json:"-"`
	Statuses       []inboundmessage.Status       `json:"statuses"`
	Classification inboundmessage.Classification `json:"classification"`
	MailboxID      pulid.ID                      `json:"mailboxId"`
	ShipmentID     pulid.ID                      `json:"shipmentId"`
	Since          int64                         `json:"since"`
}

type GetInboundMessageByIDRequest struct {
	ID                 pulid.ID              `json:"id"`
	TenantInfo         pagination.TenantInfo `json:"tenantInfo"`
	IncludeAttachments bool                  `json:"includeAttachments"`
}

// GetInboundMessageByProviderIDRequest is the idempotency lookup. A provider
// that redelivers a webhook — which every one of them does — must find the
// message it already created rather than make a second.
type GetInboundMessageByProviderIDRequest struct {
	MailboxID         pulid.ID `json:"mailboxId"`
	ProviderMessageID string   `json:"providerMessageId"`
}

type CountInboundMessagesRequest struct {
	TenantInfo pagination.TenantInfo   `json:"tenantInfo"`
	Statuses   []inboundmessage.Status `json:"statuses"`
	Since      int64                   `json:"since"`
}

// InboundMessageCount is one cell of the inbox's census: how many messages share
// a status, a reading and a mailbox. The service folds these into lanes, kinds
// and mailboxes, so one aggregate answers all three.
type InboundMessageCount struct {
	Status         inboundmessage.Status         `bun:"status"`
	Classification inboundmessage.Classification `bun:"classification"`
	MailboxID      pulid.ID                      `bun:"mailbox_id"`
	Count          int                           `bun:"count"`
}

type CountInboundAttachmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	MessageIDs []pulid.ID            `json:"messageIds"`
}

type ListRecentInboundMessagesForReviewRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
}

type DeleteInboundMessagesBeforeRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Before     int64                 `json:"before"`
	Limit      int                   `json:"limit"`
}

type InboundMailboxRepository interface {
	List(
		ctx context.Context,
		req *ListMailboxesRequest,
	) (*pagination.ListResult[*inboundmessage.Mailbox], error)
	GetByID(
		ctx context.Context,
		req GetMailboxByIDRequest,
	) (*inboundmessage.Mailbox, error)
	// GetByTokenHash is deliberately not tenant-scoped: the token is what
	// identifies the tenant, so there is nothing to scope by until it resolves.
	GetByTokenHash(
		ctx context.Context,
		req GetMailboxByTokenHashRequest,
	) (*inboundmessage.Mailbox, error)
	Create(
		ctx context.Context,
		entity *inboundmessage.Mailbox,
	) (*inboundmessage.Mailbox, error)
	Update(
		ctx context.Context,
		entity *inboundmessage.Mailbox,
	) (*inboundmessage.Mailbox, error)
}

type InboundMessageRepository interface {
	List(
		ctx context.Context,
		req *ListInboundMessagesRequest,
	) (*pagination.ListResult[*inboundmessage.InboundMessage], error)
	ListCursor(
		ctx context.Context,
		req *ListInboundMessagesRequest,
	) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error)
	GetByID(
		ctx context.Context,
		req GetInboundMessageByIDRequest,
	) (*inboundmessage.InboundMessage, error)
	GetByProviderID(
		ctx context.Context,
		req GetInboundMessageByProviderIDRequest,
	) (*inboundmessage.InboundMessage, error)
	Create(
		ctx context.Context,
		entity *inboundmessage.InboundMessage,
		attachments []*inboundmessage.InboundAttachment,
	) (*inboundmessage.InboundMessage, error)
	Update(
		ctx context.Context,
		entity *inboundmessage.InboundMessage,
	) (*inboundmessage.InboundMessage, error)
	UpdateAttachment(
		ctx context.Context,
		entity *inboundmessage.InboundAttachment,
	) (*inboundmessage.InboundAttachment, error)
	ListAttachments(
		ctx context.Context,
		messageID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) ([]*inboundmessage.InboundAttachment, error)
	CountBreakdown(
		ctx context.Context,
		req CountInboundMessagesRequest,
	) ([]InboundMessageCount, error)
	CountAttachmentsByMessageIDs(
		ctx context.Context,
		req CountInboundAttachmentsRequest,
	) (map[pulid.ID]int, error)
	ListRecentForReview(
		ctx context.Context,
		req ListRecentInboundMessagesForReviewRequest,
	) ([]*inboundmessage.InboundMessage, error)
	DeleteBefore(
		ctx context.Context,
		req DeleteInboundMessagesBeforeRequest,
	) (int64, error)
}
