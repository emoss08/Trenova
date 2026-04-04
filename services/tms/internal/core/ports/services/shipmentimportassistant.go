package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentImportChatRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	UserMessage         string                `json:"message"`
	ConversationID      string                `json:"conversationId,omitempty"`
	DocumentID          string                `json:"documentId"`
	ReconciliationState map[string]any        `json:"reconciliationState"`
	RequiredFields      map[string]string     `json:"requiredFields"`
	Stops               []map[string]any      `json:"stops"`
	ShipmentData        map[string]any        `json:"shipmentData"`
}

type ShipmentImportAction struct {
	Type     string         `json:"type"`
	FieldKey string         `json:"fieldKey"`
	Value    string         `json:"value"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ShipmentImportSuggestion struct {
	Label       string `json:"label"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type,omitempty"`        // "prompt" (default), "input", or "action"
	Placeholder string `json:"placeholder,omitempty"` // placeholder for input type
	Action      string `json:"action,omitempty"`      // action ID for action type (e.g. "create_shipment")
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

type ShipmentImportChatResponse struct {
	Message        string                         `json:"message"`
	ConversationID string                         `json:"conversationId"`
	Actions        []ShipmentImportAction         `json:"actions"`
	Suggestions    []ShipmentImportSuggestion     `json:"suggestions"`
	ToolCalls      []ShipmentImportToolCallRecord `json:"toolCalls"`
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

type StreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type ShipmentImportAssistantService interface {
	Chat(ctx context.Context, req *ShipmentImportChatRequest) (*ShipmentImportChatResponse, error)
	ChatStream(ctx context.Context, req *ShipmentImportChatRequest, emit func(StreamEvent)) error
	GetHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) (*ShipmentImportChatHistoryResponse, error)
	ArchiveHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
	CompleteHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
}
