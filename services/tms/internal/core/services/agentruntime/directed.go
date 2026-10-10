package agentruntime

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// directedTask is the task the person handed another agent, on a turn that
// can run one: the person's own, read as it runs, in a conversation that
// keeps what the other agent does.
func directedTask(req *serviceports.RunRequest) *serviceports.DirectedTask {
	if req.Directed == nil || req.Delegation != nil || req.Unattended ||
		req.ThreadID.IsNil() {
		return nil
	}
	task := *req.Directed
	if len(task.Records) == 0 {
		task.Records = recordsInPlay(req)
	}

	return &task
}

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
func (s *Service) driveDirected(t *Turn, fx TurnEffects) *serviceports.RunResult {
	directed := t.directed
	result := t.result
	call := serviceports.ToolCall{
		ID:   fx.NewCallID(),
		Name: delegateTaskName,
		Arguments: map[string]any{
			delegateAgentParam: directed.Delegate.ID.String(),
			delegateTaskParam:  directed.Task,
		},
	}
	t.ReserveCallIDs([]string{call.ID})

	opening := conversation.Message{
		Role:      conversation.RoleAssistant,
		ToolCalls: toToolCallRecords([]serviceports.ToolCall{call}),
		CreatedAt: fx.Now(),
	}
	result.Messages = append(result.Messages, opening)
	t.messages = append(t.messages, serviceports.Message{
		Role:      serviceports.RoleAssistant,
		ToolCalls: []serviceports.ToolCall{call},
	})
	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventMessage,
		Data: serviceports.AssistantMessageEvent{
			ToolCalls: s.callsWithEffects(opening.ToolCalls),
		},
	})
	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventToolStarted,
		Data: serviceports.AssistantToolStartedEvent{
			CallID:    call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
			Effect:    s.ToolEffect(call.Name),
		},
	})

	t.delegations++
	outcome, report := s.handTask(t, fx, &handedTask{
		call:     call,
		delegate: directed.Delegate,
		task:     directed.Task,
		handed:   handedRecords(directed.Records),
		directed: true,
	})
	result.ToolCallsUsed++
	s.recordToolResult(t, fx, call, outcome)

	reply := directedReply(&report)
	fx.Emit(deltaEvent(reply))

	return s.finish(result, &serviceports.ChatCompletionResult{Text: reply}, fx)
}

// recordsInPlay hands the conversation's subject and the records it has been
// working on to the agent the person turned to. Without them "bill it" reached
// the billing agent as two words, and it went looking for which load.
func recordsInPlay(req *serviceports.RunRequest) []agent.RecordRef {
	watched := watchedRecords(req)
	records := make([]agent.RecordRef, 0, min(len(watched), maxDelegateRecords))
	for _, record := range watched {
		if len(records) == maxDelegateRecords {
			break
		}
		if record.Type == "" || record.ID == "" {
			continue
		}
		records = append(records, agent.RecordRef{EntityType: record.Type, ID: record.ID})
	}
	if len(records) == 0 {
		return nil
	}

	return records
}

func handedRecords(records []agent.RecordRef) *DelegateContext {
	if len(records) == 0 {
		return nil
	}

	return &DelegateContext{Records: records}
}

// directedReply is what closes a directed turn: the other agent's answer,
// or why there is none, so the conversation never ends on a tool result
// and the person reads the outcome where they read every reply.
func directedReply(report *serviceports.AssistantDelegateFinishedEvent) string {
	reply := strings.TrimSpace(report.Reply)
	switch report.Status {
	case serviceports.DelegateStatusCompleted:
		if reply != "" {
			return reply
		}
		return report.AgentName + " finished without saying anything more."
	case serviceports.DelegateStatusExhausted:
		if reply != "" {
			return reply + "\n\n" + report.Reason
		}
		return report.Reason
	case serviceports.DelegateStatusDeclined,
		serviceports.DelegateStatusRefused,
		serviceports.DelegateStatusStopped,
		serviceports.DelegateStatusFailed:
		if report.Reason != "" {
			return report.Reason
		}
	}

	return report.AgentName + " did not report back."
}
