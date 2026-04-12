package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetShipmentImportConversationRequest struct {
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	Status     shipmentimportchat.ConversationStatus
}

type ListShipmentImportTurnsRequest struct {
	ConversationID pulid.ID
	TenantInfo     pagination.TenantInfo
}

type ShipmentImportChatRepository interface {
	GetConversationByDocument(
		ctx context.Context,
		req GetShipmentImportConversationRequest,
	) (*shipmentimportchat.Conversation, error)
	CreateConversation(
		ctx context.Context,
		entity *shipmentimportchat.Conversation,
	) (*shipmentimportchat.Conversation, error)
	UpdateConversation(
		ctx context.Context,
		entity *shipmentimportchat.Conversation,
	) (*shipmentimportchat.Conversation, error)
	AppendTurn(
		ctx context.Context,
		entity *shipmentimportchat.Turn,
	) (*shipmentimportchat.Turn, error)
	ListTurns(
		ctx context.Context,
		req ListShipmentImportTurnsRequest,
	) ([]*shipmentimportchat.Turn, error)
	UpdateActiveConversationStatusByDocument(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		status shipmentimportchat.ConversationStatus,
		reason shipmentimportchat.ConversationStatusReason,
	) error
}

type ShipmentImportChatCacheRepository interface {
	GetHistory(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentimportchat.HistorySnapshot, error)
	SetHistory(
		ctx context.Context,
		snapshot *shipmentimportchat.HistorySnapshot,
		tenantInfo pagination.TenantInfo,
	) error
	DeleteHistory(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) error
}
