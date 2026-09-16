// Package assistantservice runs the assistant's turn: guard the request, let the
// model call tools, guard the answer.
package assistantservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// maxIterations bounds the tool loop. A model that keeps calling tools without
	// concluding would otherwise spend an organization's budget on one question,
	// and in practice a genuine answer needs two or three lookups.
	maxIterations = 6
	// maxToolResultChars bounds what one tool contributes to the context. A broad
	// search can return far more than the model can use, and truncating here is
	// better than a context-length failure that loses the whole conversation.
	maxToolResultChars = 12000
)

// TurnRequest is one exchange.
type TurnRequest struct {
	Definition *agentdefinition.Definition
	Actor      *serviceports.RequestActor
	// History is the thread so far, oldest first, excluding the new message.
	History []conversation.Message
	Input   string
}

// TurnResult is what the turn produced, including the audit trail.
type TurnResult struct {
	// Reply is what to show the person.
	Reply string
	// Decision is the guard's verdict. A refusal carries the reason.
	Decision agentguard.Decision
	// Messages are the turns to persist, in order, starting with the user's.
	Messages []conversation.Message
	// Proposals are actions the agent asked for that need a human decision. They
	// are returned rather than executed.
	Proposals []PendingAction
	Model     string
	Provider  pulid.ID
}

// PendingAction is a write the agent proposed. Nothing here has run.
type PendingAction struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
	Rationale string         `json:"rationale"`
	// Tier is the effective autonomy tier for this call: the more restrictive of
	// the tool's own default and the agent's ceiling.
	Tier agent.AutonomyTier `json:"tier"`
	// ToolCallID ties the proposal back to the assistant message that asked for
	// it, so the persisted proposal can point at the turn it came out of.
	ToolCallID string `json:"toolCallId"`
}

// Run executes one turn without an audience.
func (s *Service) Run(ctx context.Context, req *TurnRequest) (*TurnResult, error) {
	return s.RunObserved(ctx, req, nil)
}

// RunObserved executes one turn and reports progress to emit as it happens.
//
// The order matters. The request is guarded before a provider is chosen, so an
// out-of-scope question costs one cheap classification rather than a frontier
// model call plus a tool loop. The answer is guarded again on the way out,
// because the layers in between are advisory.
func (s *Service) RunObserved(
	ctx context.Context,
	req *TurnRequest,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	tenantInfo := req.Actor.TenantInfo()

	decision := s.guard.Evaluate(ctx, tenantInfo, req.Input)
	userMessage := conversation.Message{
		Role:          conversation.RoleUser,
		Content:       req.Input,
		ScopeStage:    string(decision.Stage),
		ScopeCategory: string(decision.Category),
		ScopeReason:   string(decision.Reason),
	}

	if !decision.Allowed {
		userMessage.Refused = true
		emit(refusedEvent(decision))

		return &TurnResult{
			Reply:    decision.Message,
			Decision: decision,
			Messages: []conversation.Message{
				userMessage,
				{
					Role:          conversation.RoleAssistant,
					Content:       decision.Message,
					Refused:       true,
					ScopeStage:    string(decision.Stage),
					ScopeCategory: string(decision.Category),
					ScopeReason:   string(decision.Reason),
				},
			},
		}, nil
	}

	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventAccepted,
		Data: serviceports.AssistantAcceptedEvent{
			Content:       req.Input,
			ScopeStage:    string(decision.Stage),
			ScopeCategory: string(decision.Category),
		},
	})

	return s.runLoop(ctx, req, decision, userMessage, emit)
}

func (s *Service) runLoop(
	ctx context.Context,
	req *TurnRequest,
	decision agentguard.Decision,
	userMessage conversation.Message,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	result := &TurnResult{
		Decision: decision,
		Messages: []conversation.Message{userMessage},
	}

	tools := s.toolSpecsFor(req.Definition)
	messages := toAdapterMessages(req.History)
	messages = append(messages, modeladapter.Message{
		Role:    modeladapter.RoleUser,
		Content: req.Input,
	})

	sink := func(delta string) {
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventDelta,
			Data:  serviceports.AssistantDeltaEvent{Text: delta},
		})
	}

	for iteration := range maxIterations {
		completion, err := s.completion.StreamChat(ctx, &serviceports.ChatCompletionRequest{
			TenantInfo: req.Actor.TenantInfo(),
			System:     req.Definition.BuildSystemPrompt(),
			Messages:   messages,
			Tools:      tools,
		}, sink)
		if err != nil {
			return nil, err
		}

		result.Model = completion.ModelIdentifier
		result.Provider = completion.ProviderID

		if len(completion.ToolCalls) == 0 {
			return s.finish(result, completion, emit)
		}

		assistantTurn := conversation.Message{
			Role:         conversation.RoleAssistant,
			Content:      completion.Text,
			ToolCalls:    toToolCallRecords(completion.ToolCalls),
			Model:        completion.ModelIdentifier,
			ProviderID:   completion.ProviderID,
			InputTokens:  completion.InputTokens,
			OutputTokens: completion.OutputTokens,
		}
		result.Messages = append(result.Messages, assistantTurn)
		messages = append(messages, modeladapter.Message{
			Role:      modeladapter.RoleAssistant,
			Content:   completion.Text,
			ToolCalls: completion.ToolCalls,
		})
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventMessage,
			Data: serviceports.AssistantMessageEvent{
				Content:   assistantTurn.Content,
				ToolCalls: assistantTurn.ToolCalls,
				Model:     assistantTurn.Model,
			},
		})

		for _, call := range completion.ToolCalls {
			emit(serviceports.StreamEvent{
				Event: serviceports.AssistantEventToolStarted,
				Data: serviceports.AssistantToolStartedEvent{
					CallID:    call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
				},
			})

			outcome := s.dispatch(ctx, req, call, completion.Text)
			if outcome.proposal != nil {
				result.Proposals = append(result.Proposals, *outcome.proposal)
			}

			emit(serviceports.StreamEvent{
				Event: serviceports.AssistantEventToolFinished,
				Data: serviceports.AssistantToolFinishedEvent{
					CallID:   call.ID,
					Name:     call.Name,
					Failed:   outcome.failed,
					Proposed: outcome.proposal != nil,
					Content:  outcome.content,
				},
			})

			result.Messages = append(result.Messages, conversation.Message{
				Role:       conversation.RoleTool,
				Content:    outcome.content,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				ToolFailed: outcome.failed,
			})
			messages = append(messages, modeladapter.Message{
				Role:       modeladapter.RoleTool,
				Content:    outcome.content,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				IsError:    outcome.failed,
			})
		}

		s.logger.Debug("assistant tool iteration",
			zap.Int("iteration", iteration+1),
			zap.Int("calls", len(completion.ToolCalls)),
		)
	}

	// Falling out of the loop means the model kept calling tools without
	// concluding. Saying so is better than returning the last tool payload as
	// though it were an answer.
	exhausted := "I gathered information but could not finish answering within the " +
		"allowed number of lookups. Try narrowing the question."
	result.Reply = exhausted
	result.Messages = append(result.Messages, conversation.Message{
		Role:    conversation.RoleAssistant,
		Content: exhausted,
	})
	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventDelta,
		Data:  serviceports.AssistantDeltaEvent{Text: exhausted},
	})

	return result, nil
}

func (s *Service) finish(
	result *TurnResult,
	completion *serviceports.ChatCompletionResult,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	outputDecision := agentguard.EvaluateOutput(completion.Text)
	if !outputDecision.Allowed {
		// Reaching here means something upstream let a code request through, so it
		// is logged as a signal that the guard or the prompt needs attention.
		s.logger.Warn("assistant reply refused by output guard",
			zap.String("rule", outputDecision.MatchedRule),
			zap.String("model", completion.ModelIdentifier),
		)

		result.Decision = outputDecision
		result.Reply = outputDecision.Message
		result.Messages = append(result.Messages, conversation.Message{
			Role:          conversation.RoleAssistant,
			Content:       outputDecision.Message,
			Refused:       true,
			ScopeStage:    string(outputDecision.Stage),
			ScopeCategory: string(outputDecision.Category),
			ScopeReason:   string(outputDecision.Reason),
			Model:         completion.ModelIdentifier,
			ProviderID:    completion.ProviderID,
			InputTokens:   completion.InputTokens,
			OutputTokens:  completion.OutputTokens,
		})
		// The reader has already seen the text stream in; this tells them to let
		// go of it.
		emit(refusedEvent(outputDecision))

		return result, nil
	}

	result.Reply = completion.Text
	result.Messages = append(result.Messages, conversation.Message{
		Role:         conversation.RoleAssistant,
		Content:      completion.Text,
		Model:        completion.ModelIdentifier,
		ProviderID:   completion.ProviderID,
		InputTokens:  completion.InputTokens,
		OutputTokens: completion.OutputTokens,
	})

	return result, nil
}

func refusedEvent(decision agentguard.Decision) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventRefused,
		Data: serviceports.AssistantRefusedEvent{
			Message:  decision.Message,
			Stage:    string(decision.Stage),
			Category: string(decision.Category),
			Reason:   string(decision.Reason),
		},
	}
}

type toolOutcome struct {
	content  string
	failed   bool
	proposal *PendingAction
}

// dispatch runs one tool call.
//
// A tool the agent was not configured with is refused here even though it was
// never offered, because the offered list is a prompt and prompts are
// suggestions. The registry lookup is the enforcement.
func (s *Service) dispatch(
	ctx context.Context,
	req *TurnRequest,
	call modeladapter.ToolCall,
	completionText string,
) toolOutcome {
	if !req.Definition.AllowsTool(call.Name) && !s.isQueryTool(call.Name) {
		return toolOutcome{
			content: fmt.Sprintf(
				"Tool %q is not available to this agent. Use one of the tools you were given.",
				call.Name,
			),
			failed: true,
		}
	}

	if tool, ok := s.queryTools.Get(call.Name); ok {
		return s.runQueryTool(ctx, req, tool, call)
	}

	actionTool, ok := s.actionTools.Get(call.Name)
	if !ok {
		return toolOutcome{
			content: fmt.Sprintf("Tool %q does not exist.", call.Name),
			failed:  true,
		}
	}

	// A write never runs inline. The effective tier is the more restrictive of
	// the tool's own and the agent's ceiling, and every tier short of
	// auto-execute means a person decides.
	tier := req.Definition.EffectiveTier(actionTool.DefaultAutonomyTier())

	return toolOutcome{
		content: fmt.Sprintf(
			"Recorded a proposal to run %q. It is awaiting review at the %s tier and has not run.",
			call.Name, tier,
		),
		proposal: &PendingAction{
			ToolName:   call.Name,
			Arguments:  call.Arguments,
			Rationale:  proposalRationale(completionText, call.Name),
			Tier:       tier,
			ToolCallID: call.ID,
		},
	}
}

func (s *Service) runQueryTool(
	ctx context.Context,
	req *TurnRequest,
	tool serviceports.AgentQueryTool,
	call modeladapter.ToolCall,
) toolOutcome {
	data, err := tool.Query(ctx, serviceports.QueryToolParams{
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		Actor:          req.Actor,
		Params:         call.Arguments,
	})
	if err != nil {
		// The message goes back to the model rather than aborting the turn, so it
		// can correct a bad argument and try again.
		return toolOutcome{
			content: fmt.Sprintf("Tool %q failed: %s", call.Name, err.Error()),
			failed:  true,
		}
	}

	encoded, err := sonic.Marshal(data)
	if err != nil {
		return toolOutcome{
			content: fmt.Sprintf("Tool %q returned data that could not be encoded.", call.Name),
			failed:  true,
		}
	}

	return toolOutcome{content: fenceToolResult(call.Name, string(encoded))}
}

func (s *Service) isQueryTool(name string) bool {
	_, ok := s.queryTools.Get(name)
	return ok
}

// fenceToolResult wraps a result as untrusted data.
//
// Records carry customer-authored text — a shipment comment, a customer name, a
// document note — and any of it can contain something shaped like an
// instruction. The model is already told to treat fenced content as data, and
// this is the same fence the billing agent uses.
func fenceToolResult(toolName, payload string) string {
	truncated := payload
	if len(truncated) > maxToolResultChars {
		truncated = truncated[:maxToolResultChars] + "\n…(truncated)"
	}

	var builder strings.Builder
	builder.WriteString("Result from ")
	builder.WriteString(toolName)
	builder.WriteString(":\n")
	builder.WriteString("<untrusted_data>\n")
	builder.WriteString(strings.ReplaceAll(truncated, "</untrusted_data>", "<\\/untrusted_data>"))
	builder.WriteString("\n</untrusted_data>")

	return builder.String()
}

func toToolCallRecords(calls []modeladapter.ToolCall) []conversation.ToolCallRecord {
	if len(calls) == 0 {
		return nil
	}

	records := make([]conversation.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		records = append(records, conversation.ToolCallRecord{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return records
}

func toAdapterMessages(history []conversation.Message) []modeladapter.Message {
	messages := make([]modeladapter.Message, 0, len(history))

	for _, msg := range history {
		switch msg.Role {
		case conversation.RoleAssistant:
			// A refused turn is a boundary message rather than something the model
			// said, and replaying it invites the model to argue with it.
			if msg.Refused {
				continue
			}
			messages = append(messages, modeladapter.Message{
				Role:      modeladapter.RoleAssistant,
				Content:   msg.Content,
				ToolCalls: fromToolCallRecords(msg.ToolCalls),
			})
		case conversation.RoleTool:
			messages = append(messages, modeladapter.Message{
				Role:       modeladapter.RoleTool,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				IsError:    msg.ToolFailed,
			})
		default:
			if msg.Refused {
				continue
			}
			messages = append(messages, modeladapter.Message{
				Role:    modeladapter.RoleUser,
				Content: msg.Content,
			})
		}
	}

	return messages
}

func fromToolCallRecords(records []conversation.ToolCallRecord) []modeladapter.ToolCall {
	if len(records) == 0 {
		return nil
	}

	calls := make([]modeladapter.ToolCall, 0, len(records))
	for _, record := range records {
		calls = append(calls, modeladapter.ToolCall{
			ID:        record.ID,
			Name:      record.Name,
			Arguments: record.Arguments,
		})
	}

	return calls
}

// maxRationaleChars bounds what the model's own words contribute to a proposal's
// rationale. An approver reads this next to the parameters; several paragraphs
// of narration would bury the one thing they need to check.
const maxRationaleChars = 600

// proposalRationale is what an approver sees as the reason for a proposed write.
//
// The model's own text is used when it said anything, because "reassigning this
// move to the Dallas terminal because the original driver is out of hours" is
// more useful than a generic line. It is the model's claim rather than a
// verified fact, so it is truncated and never treated as more than narration:
// the parameters shown beside it are what would actually run.
func proposalRationale(completionText, toolName string) string {
	trimmed := strings.TrimSpace(completionText)
	if trimmed == "" {
		return fmt.Sprintf(
			"The assistant asked to run %s during a conversation without explaining why.",
			toolName,
		)
	}

	return truncateRunes(trimmed, maxRationaleChars)
}
