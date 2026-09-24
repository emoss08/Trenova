package shipmentimportassistantservice

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// MaxToolRounds bounds the tool loop. Five rounds is enough for the assistant
// to search, confirm and set a field; past that it is looping rather than
// working, and a person is waiting on the reply.
const MaxToolRounds = 5

// SuggestQuickActionsTool is answered by the loop rather than by a handler: it
// carries the reply's follow-up chips, which are part of the turn's result, not
// a lookup the assistant is asking us to perform.
const SuggestQuickActionsTool = "suggest_quick_actions"

// The events a reader of a streamed turn receives, as the client names them.
const (
	EventTextDelta     = "text_delta"
	EventNewMessage    = "new_message"
	EventToolCallStart = "tool_call_start"
	EventToolCallDone  = "tool_call_done"
	EventSuggestions   = "suggestions"
	EventDone          = "done"
	EventError         = "error"
)

// Chat answers one message on a worker and returns the whole reply.
func (s *Service) Chat(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*serviceports.ShipmentImportChatResponse, error) {
	return s.turns.Chat(ctx, req)
}

// ChatStream answers one message on a worker and hands the reply to emit as
// it is written.
func (s *Service) ChatStream(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	emit func(serviceports.StreamEvent),
) error {
	return s.turns.ChatStream(ctx, req, emit)
}

// PreparedTurn is everything a turn's model calls need, read once so the
// workflow driving the turn never reads the database itself.
type PreparedTurn struct {
	ConversationID pulid.ID `json:"conversationId"`
	// RequestConversationID is the handle the turn answers, as the stored
	// conversation fills it in when the request left it out.
	RequestConversationID string                  `json:"requestConversationId,omitempty"`
	System                string                  `json:"system"`
	Messages              []serviceports.Message  `json:"messages"`
	Tools                 []serviceports.ToolSpec `json:"tools"`
}

// PrepareTurn opens the document's conversation and builds the turn's first
// request: the replayed exchange, the new message, and the system prompt that
// carries the import's current state.
func (s *Service) PrepareTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*PreparedTurn, error) {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	return &PreparedTurn{
		ConversationID:        conversation.ID,
		RequestConversationID: req.ConversationID,
		System:                s.systemMessage(ctx, req),
		Messages:              s.buildMessages(ctx, req, conversation),
		Tools:                 buildTools(),
	}, nil
}

// ToolOutcome is what one tool call returned: the text the model reads, and
// the actions the client applies to the draft.
type ToolOutcome struct {
	Output  string                              `json:"output"`
	Status  string                              `json:"status"`
	Actions []serviceports.ShipmentImportAction `json:"actions,omitempty"`
}

// RunTool runs one tool call. The call's arguments are already decoded: the
// protocols disagree about whether they arrive as a JSON string or an object,
// and the port settled that.
func (s *Service) RunTool(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	call *serviceports.ToolCall,
) ToolOutcome {
	output, actions := s.runToolCall(ctx, tenantInfo, call.Name, call.Arguments)

	return ToolOutcome{
		Output:  output,
		Status:  toolCallStatusFromResult(output),
		Actions: actions,
	}
}

// SuggestionsOutcome is what the model reads for a suggest_quick_actions call,
// whose chips the loop has taken.
func SuggestionsOutcome() ToolOutcome {
	return ToolOutcome{Output: `{"ok":true}`, Status: toolStatusCompleted}
}

// UnavailableToolOutcome is what the model reads for a call that could not be
// run at all, so it tells the person rather than claiming it happened.
func UnavailableToolOutcome() ToolOutcome {
	return ToolOutcome{
		Output: `{"error":"the tool could not be run just now; tell the person it did not happen"}`,
		Status: toolStatusError,
	}
}

// WritesTool reports whether a tool changes a record rather than reading or
// proposing one. Such a call is made at most once: retrying it would create
// the shipment or the location twice.
func WritesTool(name string) bool {
	switch name {
	case "create_shipment", "add_location":
		return true
	default:
		return false
	}
}

// ToolRecord is a call as the history panel shows it.
func ToolRecord(
	call *serviceports.ToolCall,
	outcome ToolOutcome,
) serviceports.ShipmentImportToolCallRecord {
	return serviceports.ShipmentImportToolCallRecord{
		Name:   call.Name,
		CallID: call.ID,
		Status: outcome.Status,
		Input:  encodeArguments(call.Arguments),
		Output: outcome.Output,
	}
}

// TurnRecord is what a turn produced across its tool rounds.
type TurnRecord struct {
	Message     string                                      `json:"message"`
	Suggestions []serviceports.ShipmentImportSuggestion     `json:"suggestions,omitempty"`
	ToolCalls   []serviceports.ShipmentImportToolCallRecord `json:"toolCalls,omitempty"`
	Actions     []serviceports.ShipmentImportAction         `json:"actions,omitempty"`
	Model       string                                      `json:"model,omitempty"`
}

// FinishTurn saves a turn that finished and returns it as the reply.
func (s *Service) FinishTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	record *TurnRecord,
) (*serviceports.ShipmentImportChatResponse, error) {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	suggestions := NormalizeSuggestions(record.Suggestions)
	handle := conversationHandle(conversation)
	if err = s.persistConversationTurn(
		ctx,
		req,
		conversation,
		handle,
		record.Message,
		suggestions,
		record.ToolCalls,
		record.Actions,
		record.Model,
		shipmentimportchat.TurnResultStatusCompleted,
		"",
	); err != nil {
		return nil, err
	}

	s.logAICall(ctx, req, record.Message)

	return &serviceports.ShipmentImportChatResponse{
		Message:        record.Message,
		ConversationID: handle,
		Actions:        record.Actions,
		Suggestions:    suggestions,
		ToolCalls:      record.ToolCalls,
	}, nil
}

// FailTurn saves a turn the model could not finish and returns what the person
// is told.
func (s *Service) FailTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	record *TurnRecord,
	cause error,
) string {
	message := friendlyCompletionError(cause)
	s.logger.Error("import assistant completion failed", zap.Error(cause))

	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		s.logger.Warn("a failed import assistant turn could not be recorded", zap.Error(err))

		return message
	}
	s.recordFailedTurn(
		ctx, req, conversation, conversationHandle(conversation),
		record.Message, record.ToolCalls, record.Actions, message,
	)

	return message
}

// ToolRound appends what a round of tool calls did, in the shape the next call
// needs: the assistant turn that asked, then one result per call.
func ToolRound(
	calls []serviceports.ToolCall,
	outcomes map[string]ToolOutcome,
) []serviceports.Message {
	messages := make([]serviceports.Message, 0, len(calls)+1)
	messages = append(messages, serviceports.Message{
		Role:      serviceports.RoleAssistant,
		ToolCalls: calls,
	})

	for _, call := range calls {
		outcome := outcomes[call.ID]
		messages = append(messages, serviceports.Message{
			Role:       serviceports.RoleTool,
			ToolCallID: call.ID,
			ToolName:   call.Name,
			Content:    outcome.Output,
			IsError:    outcome.Status == toolStatusError,
		})
	}

	return messages
}

// ReadSuggestions reads the chips a suggest_quick_actions call carries.
func ReadSuggestions(arguments map[string]any) []serviceports.ShipmentImportSuggestion {
	raw, err := sonic.Marshal(arguments)
	if err != nil {
		return nil
	}

	var decoded struct {
		Suggestions []serviceports.ShipmentImportSuggestion `json:"suggestions"`
	}
	if err = sonic.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	return decoded.Suggestions
}

// runToolCall dispatches a call to its handler.
func (s *Service) runToolCall(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	name string,
	arguments map[string]any,
) (string, []serviceports.ShipmentImportAction) {
	handler, ok := shipmentImportToolCallHandlers[name]
	if !ok {
		return `{"error":"unknown tool"}`, nil
	}

	if refusal, allowed := s.authorizeTool(ctx, tenantInfo, name); !allowed {
		return refusal, nil
	}

	return handler(s, ctx, tenantInfo, shipmentImportToolCallArgs{m: arguments})
}

// encodeArguments renders a call's arguments for the stored record, which the
// history panel shows verbatim. Arguments that will not marshal are recorded as
// an empty object rather than dropping the record.
func encodeArguments(arguments map[string]any) string {
	if len(arguments) == 0 {
		return "{}"
	}

	encoded, err := sonic.Marshal(arguments)
	if err != nil {
		return "{}"
	}

	return string(encoded)
}

// friendlyCompletionError turns a routing failure into something a dispatcher
// can act on. The router has already tried every provider that serves this
// task, so the distinctions that matter here are "nothing is configured",
// "the reply did not fit the schema", and "it did not work".
func friendlyCompletionError(err error) string {
	switch {
	case errors.Is(err, serviceports.ErrNoProviderConfigured):
		return "No AI provider is configured for this organization. " +
			"Add one under AI Control before using the import assistant."
	case errors.Is(err, serviceports.ErrModelSchemaValidation):
		return "The AI reply could not be read. Please try again."
	case errors.Is(err, context.DeadlineExceeded):
		return "The AI request timed out. Please try again."
	case errors.Is(err, context.Canceled):
		return "The request was canceled."
	default:
		return "AI assistant encountered an error. Please try again."
	}
}
