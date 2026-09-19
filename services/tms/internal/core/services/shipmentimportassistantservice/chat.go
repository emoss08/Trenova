package shipmentimportassistantservice

import (
	"context"
	"errors"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// maxToolRounds bounds the tool loop. Five rounds is enough for the assistant
// to search, confirm and set a field; past that it is looping rather than
// working, and a person is waiting on the reply.
const maxToolRounds = 5

// suggestQuickActionsTool is answered by the loop rather than by a handler: it
// carries the reply's follow-up chips, which are part of the turn's result, not
// a lookup the assistant is asking us to perform.
const suggestQuickActionsTool = "suggest_quick_actions"

// turnState accumulates what a turn produced across its tool rounds.
type turnState struct {
	text        strings.Builder
	actions     []serviceports.ShipmentImportAction
	suggestions []serviceports.ShipmentImportSuggestion
	toolCalls   []serviceports.ShipmentImportToolCallRecord
	model       string
}

func (t *turnState) message() string { return t.text.String() }

func (s *Service) Chat(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*serviceports.ShipmentImportChatResponse, error) {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	state := &turnState{}
	messages := s.buildMessages(ctx, req, conversation)
	system := s.systemMessage(ctx, req)

	for range maxToolRounds {
		result, callErr := s.completion.CompleteChat(ctx, &serviceports.ChatCompletionRequest{
			TenantInfo: req.TenantInfo,
			System:     system,
			Messages:   messages,
			Tools:      buildTools(),
		})
		if callErr != nil {
			return nil, s.failTurn(ctx, req, conversation, state, callErr)
		}

		state.model = result.ModelIdentifier
		if strings.TrimSpace(result.Text) != "" {
			state.text.WriteString(result.Text)
		}

		if len(result.ToolCalls) == 0 {
			break
		}

		messages = append(messages, toolRound(
			result.ToolCalls,
			s.runToolCalls(ctx, req.TenantInfo, result.ToolCalls, state, nil),
		)...)
	}

	return s.finishTurn(ctx, req, conversation, state)
}

func (s *Service) ChatStream(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	emit func(serviceports.StreamEvent),
) error {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return err
	}

	state := &turnState{}
	messages := s.buildMessages(ctx, req, conversation)
	system := s.systemMessage(ctx, req)

	for round := range maxToolRounds {
		// A round after the first is a fresh answer, not a continuation of the
		// bubble the reader is already looking at.
		if round > 0 {
			emit(serviceports.StreamEvent{Event: "new_message", Data: nil})
		}

		result, callErr := s.completion.StreamChat(
			ctx,
			&serviceports.ChatCompletionRequest{
				TenantInfo: req.TenantInfo,
				System:     system,
				Messages:   messages,
				Tools:      buildTools(),
			},
			func(delta string) {
				emit(serviceports.StreamEvent{
					Event: "text_delta",
					Data:  map[string]string{"delta": delta},
				})
			},
		)
		if callErr != nil {
			message := friendlyCompletionError(callErr)
			s.logger.Error("import assistant completion failed", zap.Error(callErr))
			s.recordFailedTurn(
				ctx, req, conversation, conversationHandle(conversation),
				state.message(), state.toolCalls, state.actions, message,
			)
			emit(serviceports.StreamEvent{
				Event: "error",
				Data:  map[string]string{"message": message},
			})

			return nil
		}

		state.model = result.ModelIdentifier
		if strings.TrimSpace(result.Text) != "" {
			state.text.WriteString(result.Text)
		}

		if len(result.ToolCalls) == 0 {
			break
		}

		messages = append(messages, toolRound(
			result.ToolCalls,
			s.runToolCalls(ctx, req.TenantInfo, result.ToolCalls, state, emit),
		)...)
	}

	state.suggestions = normalizeSuggestions(state.suggestions)
	if len(state.suggestions) > 0 {
		emit(serviceports.StreamEvent{
			Event: "suggestions",
			Data:  map[string]any{"suggestions": state.suggestions},
		})
	}

	emit(serviceports.StreamEvent{Event: "done", Data: map[string]any{
		"conversationId": conversationHandle(conversation),
		"actions":        state.actions,
	}})

	if err = s.persistTurn(ctx, req, conversation, state); err != nil {
		return err
	}

	s.logAICall(ctx, req, state.message())

	return nil
}

// runToolCalls executes one round. emit may be nil, which is how the
// non-streaming path reuses the same body rather than keeping a second copy of
// it that drifts.
func (s *Service) runToolCalls(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	calls []serviceports.ToolCall,
	state *turnState,
	emit func(serviceports.StreamEvent),
) map[string]toolResult {
	results := make(map[string]toolResult, len(calls))

	for _, call := range calls {
		if call.Name == suggestQuickActionsTool {
			state.suggestions = readSuggestions(call.Arguments)
			results[call.ID] = toolResult{output: `{"ok":true}`, status: toolStatusCompleted}
			continue
		}

		if emit != nil {
			emit(serviceports.StreamEvent{
				Event: "tool_call_start",
				Data:  map[string]string{"name": call.Name, "callId": call.ID},
			})
		}

		output, actions := s.runToolCall(ctx, tenantInfo, call.Name, call.Arguments)
		status := toolCallStatusFromResult(output)
		state.actions = append(state.actions, actions...)
		results[call.ID] = toolResult{output: output, status: status}

		state.toolCalls = append(state.toolCalls, serviceports.ShipmentImportToolCallRecord{
			Name:   call.Name,
			CallID: call.ID,
			Status: status,
			Input:  encodeArguments(call.Arguments),
			Output: output,
		})

		if emit != nil {
			emit(serviceports.StreamEvent{Event: "tool_call_done", Data: map[string]any{
				"name":    call.Name,
				"callId":  call.ID,
				"status":  status,
				"result":  output,
				"actions": actions,
			}})
		}
	}

	return results
}

// runToolCall dispatches a call whose arguments the adapter has already
// decoded. The protocols disagree about whether arguments arrive as a JSON
// string or an object; the port settled that, so this no longer parses them.
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

	return handler(s, ctx, tenantInfo, shipmentImportToolCallArgs{m: arguments})
}

func readSuggestions(arguments map[string]any) []serviceports.ShipmentImportSuggestion {
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

func (s *Service) finishTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
	state *turnState,
) (*serviceports.ShipmentImportChatResponse, error) {
	state.suggestions = normalizeSuggestions(state.suggestions)
	if err := s.persistTurn(ctx, req, conversation, state); err != nil {
		return nil, err
	}

	s.logAICall(ctx, req, state.message())

	return &serviceports.ShipmentImportChatResponse{
		Message:        state.message(),
		ConversationID: conversationHandle(conversation),
		Actions:        state.actions,
		Suggestions:    state.suggestions,
		ToolCalls:      state.toolCalls,
	}, nil
}

func (s *Service) persistTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
	state *turnState,
) error {
	return s.persistConversationTurn(
		ctx,
		req,
		conversation,
		conversationHandle(conversation),
		state.message(),
		state.suggestions,
		state.toolCalls,
		state.actions,
		state.model,
		shipmentimportchat.TurnResultStatusCompleted,
		"",
	)
}

func (s *Service) failTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
	state *turnState,
	cause error,
) error {
	message := friendlyCompletionError(cause)
	s.logger.Error("import assistant completion failed", zap.Error(cause))
	s.recordFailedTurn(
		ctx, req, conversation, conversationHandle(conversation),
		state.message(), state.toolCalls, state.actions, message,
	)

	return errortypes.NewBusinessError(message)
}

// friendlyCompletionError turns a routing failure into something a dispatcher
// can act on. It no longer reads HTTP status codes off one vendor's error type:
// the router has already tried every provider that serves this task, so the
// distinctions that matter here are "nothing is configured", "the reply did not
// fit the schema", and "it did not work".
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
