package conform

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	eventTenderOffered = "TenderOffered"
	firstControl       = "0001"
)

type flowInstance struct {
	flow  string
	key   string
	state jsonutils.RawJSON
}

func unopened(flow, key string) *flowInstance {
	return &flowInstance{flow: flow, key: key, state: jsonutils.RawJSON("{}")}
}

func (r *runner) offered(
	ctx context.Context,
	s *subject,
	c *Case,
	step *Step,
	now time.Time,
) *flowInstance {
	flow, clause := s.check.Sending(step.Set, eventTenderOffered)
	if flow == nil || clause == nil {
		return nil
	}
	event := lading.FlowEvent{
		Kind:     lading.EventTrenova,
		Event:    eventTenderOffered,
		Document: step.Document,
	}
	key, ok := r.flowKey(ctx, s, c, step, flow.Name, event, now)
	if !ok {
		return nil
	}
	instance := unopened(flow.Name, key)
	actions, ok := r.flowStep(ctx, s, c, step, instance, event, now)
	if !ok {
		return nil
	}
	for index := range actions {
		action := &actions[index]
		if action.Kind != lading.ActionSend || action.Set != step.Set {
			continue
		}
		sent := lading.FlowEvent{Kind: lading.EventSent, ID: action.ID, Control: firstControl}
		if _, ok = r.flowStep(ctx, s, c, step, instance, sent, now); !ok {
			return nil
		}
	}
	return instance
}

func (r *runner) received(
	ctx context.Context,
	s *subject,
	c *Case,
	step *Step,
	conversation *flowInstance,
	transaction *lading.TransactionResult,
	now time.Time,
) {
	flow, clause := s.check.Receiving(step.Set)
	if flow == nil || clause == nil {
		return
	}
	event := lading.FlowEvent{
		Kind:     lading.EventReceived,
		Set:      step.Set,
		Document: transaction.Document,
		Outcomes: outcomeInputs(transaction),
	}
	key, ok := r.flowKey(ctx, s, c, step, flow.Name, event, now)
	if !ok {
		return
	}
	instance := unopened(flow.Name, key)
	if conversation != nil && conversation.flow == flow.Name && conversation.key == key {
		instance = conversation
	}
	actions, ok := r.flowStep(ctx, s, c, step, instance, event, now)
	if !ok {
		return
	}
	for index := range actions {
		action := &actions[index]
		switch {
		case action.Kind == lading.ActionAcknowledge && action.Rejected && action.Message != nil:
			transaction.Outcomes = append(transaction.Outcomes, lading.Outcome{
				Kind:    lading.OutcomeReject,
				Message: action.Message.String(),
				Rule:    "flow " + flow.Name,
			})
		case action.Kind == lading.ActionHold:
			transaction.Outcomes = append(transaction.Outcomes, lading.Outcome{
				Kind:    lading.OutcomeHold,
				Message: actionMessage(action),
				Rule:    "flow " + flow.Name,
			})
		}
	}
}

func (r *runner) flowKey(
	ctx context.Context,
	s *subject,
	c *Case,
	step *Step,
	flow string,
	event lading.FlowEvent,
	now time.Time,
) (string, bool) {
	key, err := r.client.FlowKey(ctx, s.project, &lading.FlowRequest{
		Flow:  flow,
		Event: event,
		Now:   instant(now),
	})
	if err != nil {
		c.fail(step, &Failure{Source: SourceLading, Rule: "flow " + flow, Message: err.Error()})
		return "", false
	}
	return key, true
}

func (r *runner) flowStep(
	ctx context.Context,
	s *subject,
	c *Case,
	step *Step,
	instance *flowInstance,
	event lading.FlowEvent,
	now time.Time,
) ([]lading.FlowAction, bool) {
	result, err := r.client.FlowStep(ctx, s.project, &lading.FlowRequest{
		Flow:  instance.flow,
		State: instance.state,
		Event: event,
		Now:   instant(now),
	})
	if err != nil {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "flow " + instance.flow,
			Message: err.Error(),
		})
		return nil, false
	}
	if result.Error != nil {
		c.fail(step, &Failure{
			Source:  SourceLading,
			Rule:    "flow " + instance.flow,
			Message: result.Error.String(),
		})
		return nil, false
	}
	instance.state = result.State
	for index := range result.Actions {
		action := &result.Actions[index]
		step.Lading = append(step.Lading, LadingFinding{
			Kind:    "flow " + action.Kind,
			Code:    action.Set,
			Rule:    instance.flow,
			Message: actionMessage(action),
		})
	}
	return result.Actions, true
}

func actionMessage(action *lading.FlowAction) string {
	if action.Message == nil {
		return ""
	}
	return action.Message.String()
}

func outcomeInputs(transaction *lading.TransactionResult) []lading.OutcomeInput {
	inputs := make([]lading.OutcomeInput, 0, len(transaction.Outcomes)+1)
	rejected := false
	for _, outcome := range transaction.Outcomes {
		rejected = rejected || outcome.Kind == lading.OutcomeReject
		inputs = append(inputs, lading.OutcomeInput{
			Kind:    outcome.Kind,
			Message: outcome.Message,
			Code:    outcome.Code,
		})
	}
	if transaction.Status == lading.TransactionRejected && !rejected {
		inputs = append(inputs, lading.OutcomeInput{Kind: lading.OutcomeReject})
	}
	return inputs
}
