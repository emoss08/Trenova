package deskbench

import (
	"time"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

type eventCollector struct {
	live   *liveTurn
	events []Event
	deltas int
	tools  []*ToolCallRecord
	byCall map[string]*ToolCallRecord
}

func newEventCollector(live *liveTurn) *eventCollector {
	return &eventCollector{
		live:   live,
		events: make([]Event, 0, 32),
		tools:  make([]*ToolCallRecord, 0, 8),
		byCall: make(map[string]*ToolCallRecord, 8),
	}
}

func streamedText(event string) bool {
	switch event {
	case serviceports.AssistantEventDelta,
		serviceports.AssistantEventReasoning,
		serviceports.AssistantEventDelegateDelta,
		serviceports.AssistantEventDelegateReasoning:
		return true
	}

	return false
}

func (e *eventCollector) onFrame(frame serviceports.TurnStreamFrame) error {
	e.live.frame(frame)
	if streamedText(frame.Event) {
		e.deltas++
		return nil
	}

	at := time.Now()
	event := Event{At: at, Name: frame.Event}
	if len(frame.Data) > 0 {
		var data any
		if err := sonic.Unmarshal(frame.Data, &data); err != nil {
			event.Data = string(frame.Data)
		} else {
			event.Data = data
		}
	}
	e.events = append(e.events, event)

	switch frame.Event {
	case serviceports.AssistantEventToolStarted:
		var started serviceports.AssistantToolStartedEvent
		if sonic.Unmarshal(frame.Data, &started) == nil {
			e.start(at, &started)
		}
	case serviceports.AssistantEventToolFinished:
		var finished serviceports.AssistantToolFinishedEvent
		if sonic.Unmarshal(frame.Data, &finished) == nil {
			e.finish(at, &finished)
		}
	}

	return nil
}

func callKey(callID, delegateCallID string) string {
	return delegateCallID + "/" + callID
}

func (e *eventCollector) start(at time.Time, started *serviceports.AssistantToolStartedEvent) {
	record := &ToolCallRecord{
		CallID:         started.CallID,
		Name:           started.Name,
		Arguments:      started.Arguments,
		Why:            started.Why,
		Effect:         string(started.Effect),
		AgentID:        started.AgentID,
		DelegateCallID: started.DelegateCallID,
		StartedAt:      at,
	}
	e.tools = append(e.tools, record)
	e.byCall[callKey(started.CallID, started.DelegateCallID)] = record
}

func (e *eventCollector) finish(at time.Time, finished *serviceports.AssistantToolFinishedEvent) {
	key := callKey(finished.CallID, finished.DelegateCallID)
	record, ok := e.byCall[key]
	if !ok {
		record = &ToolCallRecord{
			CallID:         finished.CallID,
			Name:           finished.Name,
			AgentID:        finished.AgentID,
			DelegateCallID: finished.DelegateCallID,
			StartedAt:      at,
		}
		e.tools = append(e.tools, record)
		e.byCall[key] = record
	}

	record.FinishedAt = at
	record.Finished = true
	record.Failed = finished.Failed
	record.Proposed = finished.Proposed
	record.Verdict = finished.Verdict
	record.Summary = finished.Summary
	record.Result = finished.Content
	if record.Effect == "" {
		record.Effect = string(finished.Effect)
	}
}

func (e *eventCollector) into(record *TurnRecord) {
	record.Events = e.events
	record.StreamDeltas = e.deltas
	record.Tools = e.tools
}
