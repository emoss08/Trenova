package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentImportSuggestion struct {
	Label       string `json:"label"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type,omitempty"`        // "prompt" (default), "input", or "action"
	Placeholder string `json:"placeholder,omitempty"` // placeholder for input type
	Action      string `json:"action,omitempty"`      // action ID for action type (e.g. "shipment_create")
	SubmitLabel string `json:"submitLabel,omitempty"` // label for the submit button (default: "Search" for input type)
}

type ShipmentImportToolCallRecord struct {
	Name   string `json:"name"`
	CallID string `json:"callId,omitempty"`
	Status string `json:"status"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type ShipmentImportChatMessage struct {
	ID          string                         `json:"id"`
	Role        string                         `json:"role"`
	Text        string                         `json:"text"`
	ToolCalls   []ShipmentImportToolCallRecord `json:"toolCalls,omitempty"`
	Suggestions []ShipmentImportSuggestion     `json:"suggestions,omitempty"`
	CreatedAt   int64                          `json:"createdAt"`
}

type ShipmentImportChatHistoryResponse struct {
	DocumentID     string                      `json:"documentId"`
	ConversationID string                      `json:"conversationId,omitempty"`
	Status         string                      `json:"status,omitempty"`
	StatusReason   string                      `json:"statusReason,omitempty"`
	TurnCount      int                         `json:"turnCount"`
	LastMessageAt  *int64                      `json:"lastMessageAt,omitempty"`
	UpdatedAt      int64                       `json:"updatedAt"`
	Messages       []ShipmentImportChatMessage `json:"messages"`
}

type ShipmentImportAssistantService interface {
	GetHistory(
		ctx context.Context,
		documentID string,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentImportChatHistoryResponse, error)
	ArchiveHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
	CompleteHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
}
