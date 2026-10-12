package inboundmessageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const InboundReplyIdempotencyPrefix = "inbound-reply:"

type GetInboundReplyRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetInboundRepliesRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetInboundReplyByKeyRequest struct {
	TenantInfo     pagination.TenantInfo
	IdempotencyKey string
}

type InboundReplyMessageRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
}

type InboundRepliesByMessagesRequest struct {
	TenantInfo pagination.TenantInfo
	MessageIDs []pulid.ID
}

type InboundReplyRepository interface {
	InboundReplyReader
	Create(ctx context.Context, entity *triage.Reply) (*triage.Reply, error)
	Update(ctx context.Context, entity *triage.Reply) (*triage.Reply, error)
	GetByID(ctx context.Context, req GetInboundReplyRequest) (*triage.Reply, error)
	GetByIDs(ctx context.Context, req GetInboundRepliesRequest) ([]*triage.Reply, error)
	GetByIdempotencyKey(
		ctx context.Context,
		req GetInboundReplyByKeyRequest,
	) (*triage.Reply, error)
	UpsertDraft(ctx context.Context, entity *triage.Reply) (*triage.Reply, error)
	DeleteDraft(ctx context.Context, req InboundReplyMessageRequest) (int, error)
	ListByMessageIDs(
		ctx context.Context,
		req InboundRepliesByMessagesRequest,
	) (map[pulid.ID][]*triage.Reply, error)
	DraftsByMessageIDs(
		ctx context.Context,
		req InboundRepliesByMessagesRequest,
	) (map[pulid.ID]*triage.Reply, error)
}

type GetTriageReplyTemplateRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type TriageReplyTemplateRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*triage.ReplyTemplate, error)
	Count(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
	GetByID(
		ctx context.Context,
		req GetTriageReplyTemplateRequest,
	) (*triage.ReplyTemplate, error)
	NameTaken(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		name string,
		exceptID pulid.ID,
	) (bool, error)
	Create(ctx context.Context, entity *triage.ReplyTemplate) (*triage.ReplyTemplate, error)
	Update(ctx context.Context, entity *triage.ReplyTemplate) (*triage.ReplyTemplate, error)
	Delete(ctx context.Context, req GetTriageReplyTemplateRequest) error
}

type ReplyContext struct {
	SenderFirstName string
	SenderAddress   string
	ProNumber       string
	PONumber        string
	CustomerName    string
	Status          string
	ETA             string
	Lane            string
}
