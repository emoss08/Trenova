package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// driveDirected runs a task the person handed straight to another agent
// from the conversation, in place of the conversation's own agent's reply.
//
// The turn is recorded as though the conversation's agent had called
// delegate_task with the person's words: its call, the other agent's steps
// nested under it, and the account that answers it. A later turn's model
// reads the call and the account, so it knows what was asked and what came
// of it, and the thread and the stream show the task exactly as they show
// one an agent handed out. Nothing asks the conversation's own model, so
// the turn's reply is the other agent's answer.
//
// The person's choice stands in for the conversation's agent's list; every
// other check is made again when the task opens: the agent exists, is
// enabled and talked to, the person may use it, and its budget is not spent.
func (s *Service) driveDirected(t *Turn, fx TurnEffects) *agentruntimeports.RunResult {
	directed := t.directed
	result := t.result
	call := llmports.ToolCall{
		ID:   fx.NewCallID(),
		Name: runtimetools.DelegateTaskName,
		Arguments: map[string]any{
			runtimetools.DelegateAgentParam: directed.Delegate.ID.String(),
			runtimetools.DelegateTaskParam:  directed.Task,
		},
	}
	t.ReserveCallIDs([]string{call.ID})

	opening := conversation.Message{
		Role:      conversation.RoleAssistant,
		ToolCalls: turnhistory.ToToolCallRecords([]llmports.ToolCall{call}),
		CreatedAt: fx.Now(),
	}
	result.Messages = append(result.Messages, opening)
	t.messages = append(t.messages, llmports.Message{
		Role:      llmports.RoleAssistant,
		ToolCalls: []llmports.ToolCall{call},
	})
	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventMessage,
		Data: assistantstreamports.AssistantMessageEvent{
			ToolCalls: s.access.CallsWithEffects(opening.ToolCalls),
		},
	})
	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventToolStarted,
		Data: assistantstreamports.AssistantToolStartedEvent{
			CallID:    call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
			Effect:    s.access.ToolEffect(call.Name),
		},
	})

	t.CountDelegation()
	outcome, report := delegation.Hand(t, s.handing(t, fx), &delegation.HandedTask{
		Call:     call,
		Delegate: directed.Delegate,
		Task:     directed.Task,
		Handed:   delegation.HandedRecords(directed.Records),
		Directed: true,
	})
	result.ToolCallsUsed++
	s.recordToolResult(t, fx, &call, &outcome)

	reply := delegation.DirectedReply(&report)
	fx.Emit(modelstream.DeltaEvent(reply))

	return s.finish(result, &llmports.ChatCompletionResult{Text: reply}, fx)
}
