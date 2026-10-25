package shipmentimportchat

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type rawJSON []byte

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
	keel.Table `keel:"shipment_import_chat_conversations"`

	ID                     pulid.ID                 `json:"id"                     keel:"id,pk,autoid:sic_"`
	OrganizationID         pulid.ID                 `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID                 `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	DocumentID             pulid.ID                 `json:"documentId"             keel:"document_id"`
	UserID                 pulid.ID                 `json:"userId"                 keel:"user_id"`
	ExternalConversationID string                   `json:"externalConversationId" keel:"external_conversation_id,nullzero"`
	Status                 ConversationStatus       `json:"status"                 keel:"status"`
	StatusReason           ConversationStatusReason `json:"statusReason"           keel:"status_reason,nullzero"`
	TurnCount              int                      `json:"turnCount"              keel:"turn_count"`
	LastMessageAt          *int64                   `json:"lastMessageAt"          keel:"last_message_at"`
	Version                int64                    `json:"version"                keel:"version,version"`
	CreatedAt              int64                    `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64                    `json:"updatedAt"              keel:"updated_at,autoupdate"`
}

func (c *Conversation) BeforeUpdate(_ context.Context) error {
	if c.Status == ConversationStatusActive {
		c.StatusReason = ConversationStatusReasonNone
	}

	return nil
}

func (c *Conversation) BeforeCreate(_ context.Context) error {
	if c.Status == "" {
		c.Status = ConversationStatusActive
	}
	if c.Status == ConversationStatusActive {
		c.StatusReason = ConversationStatusReasonNone
	}

	return nil
}

type Turn struct {
	keel.Table `keel:"shipment_import_chat_turns"`

	ID                     pulid.ID         `json:"id"                     keel:"id,pk,autoid:sit_"`
	ConversationID         pulid.ID         `json:"conversationId"         keel:"conversation_id"`
	OrganizationID         pulid.ID         `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID         `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	DocumentID             pulid.ID         `json:"documentId"             keel:"document_id"`
	UserID                 pulid.ID         `json:"userId"                 keel:"user_id"`
	TurnIndex              int              `json:"turnIndex"              keel:"turn_index"`
	UserMessage            string           `json:"userMessage"            keel:"user_message"`
	AssistantMessage       string           `json:"assistantMessage"       keel:"assistant_message"`
	RequestConversationID  string           `json:"requestConversationId"  keel:"request_conversation_id,nullzero"`
	ResponseConversationID string           `json:"responseConversationId" keel:"response_conversation_id,nullzero"`
	Model                  string           `json:"model"                  keel:"model,nullzero"`
	ResultStatus           TurnResultStatus `json:"resultStatus"           keel:"result_status"`
	ErrorMessage           string           `json:"errorMessage"           keel:"error_message,nullzero"`
	ContextJSON            rawJSON          `json:"contextJson"            keel:"context_json,jsonb"`
	SuggestionsJSON        rawJSON          `json:"suggestionsJson"        keel:"suggestions_json,jsonb"`
	ToolCallsJSON          rawJSON          `json:"toolCallsJson"          keel:"tool_calls_json,jsonb"`
	ActionsJSON            rawJSON          `json:"actionsJson"            keel:"actions_json,jsonb"`
	CreatedAt              int64            `json:"createdAt"              keel:"created_at,autocreate"`
}

func (t *Turn) BeforeCreate(_ context.Context) error {
	if t.ResultStatus == "" {
		t.ResultStatus = TurnResultStatusCompleted
	}

	return nil
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

func (c *Conversation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&c.UserID, validation.Required.Error("User is required")),
		validation.Field(&c.ExternalConversationID,
			validation.Length(0, maxExternalConversationIDLength).
				Error("External conversation id cannot be longer than 255 characters"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				ConversationStatusActive,
				ConversationStatusCompleted,
				ConversationStatusSuperseded,
			).Error("Status is invalid"),
		),
		validation.Field(&c.StatusReason,
			validation.In(
				ConversationStatusReasonNone,
				ConversationStatusReasonReextract,
				ConversationStatusReasonShipmentCreated,
				ConversationStatusReasonManualRestart,
			).Error("Status reason is invalid"),
		),
		validation.Field(&c.TurnCount,
			validation.Min(0).Error("Turn count cannot be negative"),
		),
	))
}

func (t *Turn) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.ConversationID,
			validation.Required.Error("Conversation is required"),
		),
		validation.Field(&t.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&t.UserID, validation.Required.Error("User is required")),
		// Turns are replayed in order to rebuild the transcript, so the index
		// has to be a real position in it.
		validation.Field(&t.TurnIndex,
			validation.Min(0).Error("Turn index cannot be negative"),
		),
		validation.Field(&t.UserMessage, validation.Required.Error("User message is required")),
		validation.Field(&t.ResultStatus,
			validation.Required.Error("Result status is required"),
			validation.In(TurnResultStatusCompleted, TurnResultStatusFailed).
				Error("Result status is invalid"),
		),
		validation.Field(&t.ErrorMessage,
			validation.When(
				t.ResultStatus == TurnResultStatusFailed,
				validation.Required.Error("A failed turn must record why it failed"),
			),
		),
		validation.Field(&t.Model,
			validation.Length(0, maxChatModelLength).
				Error("Model cannot be longer than 100 characters"),
		),
		validation.Field(&t.RequestConversationID,
			validation.Length(0, maxExternalConversationIDLength).
				Error("Request conversation id cannot be longer than 255 characters"),
		),
		validation.Field(&t.ResponseConversationID,
			validation.Length(0, maxExternalConversationIDLength).
				Error("Response conversation id cannot be longer than 255 characters"),
		),
	))
}

const (
	maxExternalConversationIDLength = 255
	maxChatModelLength              = 100
)
