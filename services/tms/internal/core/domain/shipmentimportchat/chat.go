package shipmentimportchat

import (
	"context"
	"encoding/json"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type ConversationStatus string

const (
	ConversationStatusActive     ConversationStatus = "Active"
	ConversationStatusCompleted  ConversationStatus = "Completed"
	ConversationStatusSuperseded ConversationStatus = "Superseded"
)

type ConversationStatusReason string

const (
	ConversationStatusReasonNone            ConversationStatusReason = ""
	ConversationStatusReasonReextract       ConversationStatusReason = "reextract"
	ConversationStatusReasonShipmentCreated ConversationStatusReason = "shipment_created"
	ConversationStatusReasonManualRestart   ConversationStatusReason = "manual_restart"
)

type TurnResultStatus string

const (
	TurnResultStatusCompleted TurnResultStatus = "Completed"
	TurnResultStatusFailed    TurnResultStatus = "Failed"
)

type Conversation struct {
	bun.BaseModel `bun:"table:shipment_import_chat_conversations,alias:sicc" json:"-"`

	ID                     pulid.ID                 `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID                 `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID         pulid.ID                 `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	DocumentID             pulid.ID                 `json:"documentId"           bun:"document_id,type:VARCHAR(100),notnull"`
	UserID                 pulid.ID                 `json:"userId"               bun:"user_id,type:VARCHAR(100),notnull"`
	ExternalConversationID string                   `json:"externalConversationId" bun:"external_conversation_id,type:VARCHAR(255),nullzero"`
	Status                 ConversationStatus       `json:"status"               bun:"status,type:VARCHAR(32),notnull,default:'Active'"`
	StatusReason           ConversationStatusReason `json:"statusReason"         bun:"status_reason,type:VARCHAR(64),nullzero"`
	TurnCount              int                      `json:"turnCount"            bun:"turn_count,type:INTEGER,notnull,default:0"`
	LastMessageAt          *int64                   `json:"lastMessageAt"        bun:"last_message_at,type:BIGINT,nullzero"`
	Version                int64                    `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64                    `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                    `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type Turn struct {
	bun.BaseModel `bun:"table:shipment_import_chat_turns,alias:sict" json:"-"`

	ID                     pulid.ID         `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	ConversationID         pulid.ID         `json:"conversationId"       bun:"conversation_id,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID         `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID         pulid.ID         `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	DocumentID             pulid.ID         `json:"documentId"           bun:"document_id,type:VARCHAR(100),notnull"`
	UserID                 pulid.ID         `json:"userId"               bun:"user_id,type:VARCHAR(100),notnull"`
	TurnIndex              int              `json:"turnIndex"            bun:"turn_index,type:INTEGER,notnull"`
	UserMessage            string           `json:"userMessage"          bun:"user_message,type:TEXT,notnull"`
	AssistantMessage       string           `json:"assistantMessage"     bun:"assistant_message,type:TEXT,notnull"`
	RequestConversationID  string           `json:"requestConversationId" bun:"request_conversation_id,type:VARCHAR(255),nullzero"`
	ResponseConversationID string           `json:"responseConversationId" bun:"response_conversation_id,type:VARCHAR(255),nullzero"`
	Model                  string           `json:"model"                bun:"model,type:VARCHAR(100),nullzero"`
	ResultStatus           TurnResultStatus `json:"resultStatus"         bun:"result_status,type:VARCHAR(32),notnull,default:'Completed'"`
	ErrorMessage           string           `json:"errorMessage"         bun:"error_message,type:TEXT,nullzero"`
	ContextJSON            json.RawMessage  `json:"contextJson"          bun:"context_json,type:JSONB,notnull,default:'{}'::jsonb"`
	SuggestionsJSON        json.RawMessage  `json:"suggestionsJson"      bun:"suggestions_json,type:JSONB,notnull,default:'[]'::jsonb"`
	ToolCallsJSON          json.RawMessage  `json:"toolCallsJson"        bun:"tool_calls_json,type:JSONB,notnull,default:'[]'::jsonb"`
	ActionsJSON            json.RawMessage  `json:"actionsJson"          bun:"actions_json,type:JSONB,notnull,default:'[]'::jsonb"`
	CreatedAt              int64            `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type HistorySnapshot struct {
	DocumentID     string                   `json:"documentId"`
	ConversationID string                   `json:"conversationId"`
	Status         ConversationStatus       `json:"status"`
	StatusReason   ConversationStatusReason `json:"statusReason,omitempty"`
	TurnCount      int                      `json:"turnCount"`
	LastMessageAt  *int64                   `json:"lastMessageAt,omitempty"`
	Messages       []HistoryMessage         `json:"messages"`
	UpdatedAt      int64                    `json:"updatedAt"`
}

type HistoryToolCall struct {
	Name   string `json:"name"`
	CallID string `json:"callId,omitempty"`
	Status string `json:"status"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type HistorySuggestion struct {
	Label       string `json:"label"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Action      string `json:"action,omitempty"`
	SubmitLabel string `json:"submitLabel,omitempty"`
}

type HistoryMessage struct {
	ID          string              `json:"id"`
	Role        string              `json:"role"`
	Text        string              `json:"text"`
	ToolCalls   []HistoryToolCall   `json:"toolCalls,omitempty"`
	Suggestions []HistorySuggestion `json:"suggestions,omitempty"`
	CreatedAt   int64               `json:"createdAt"`
}

func (c *Conversation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("sic_")
		}
		if c.Status == "" {
			c.Status = ConversationStatusActive
		}
		if c.Status == ConversationStatusActive {
			c.StatusReason = ConversationStatusReasonNone
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		if c.Status == ConversationStatusActive {
			c.StatusReason = ConversationStatusReasonNone
		}
		c.UpdatedAt = now
	}

	return nil
}

func (t *Turn) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("sit_")
		}
		if t.ResultStatus == "" {
			t.ResultStatus = TurnResultStatusCompleted
		}
		if t.CreatedAt == 0 {
			t.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}
