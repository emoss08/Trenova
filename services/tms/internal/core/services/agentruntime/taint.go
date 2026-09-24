package agentruntime

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func openTaint(
	req *serviceports.RunRequest,
	rc agentdefinition.RuntimeContext,
	now int64,
) (*agent.RunTaint, []agent.TaintMark) {
	if req.Delegation != nil && req.Taint == nil {
		return nil, nil
	}

	taint := req.Taint.Clone()
	if taint == nil {
		taint = &agent.RunTaint{}
	}

	return taint, taint.Absorb(contextTaint(rc, now))
}

func contextTaint(rc agentdefinition.RuntimeContext, now int64) []agent.TaintMark {
	marks := make([]agent.TaintMark, 0, 1+len(rc.Attachments)+len(rc.Memories))
	if rc.Subject != nil {
		if mark, ok := agent.SubjectTaint(rc.Subject.Type, rc.Subject.ID, now); ok {
			marks = append(marks, mark)
		}
	}
	for idx := range rc.Attachments {
		if id := rc.Attachments[idx].DocumentID; id != "" {
			marks = append(marks, agent.AttachmentTaint(id, now))
		}
	}
	for _, memory := range rc.Memories {
		for _, ref := range memory.TaintedRecords() {
			marks = append(marks, agent.TaintMark{
				Source: agent.TaintSourceMemory,
				Ref:    &ref,
				At:     now,
			})
		}
	}

	return marks
}

func callTaint(
	policy serviceports.ToolPolicy,
	call serviceports.ToolCall,
	data any,
	now int64,
) []agent.TaintMark {
	var refs []agent.RecordRef
	if carrier, ok := data.(agent.TaintCarrier); ok {
		refs = carrier.TaintedRecords()
	}

	switch policy.ReadsExternal {
	case agent.ExternalReadAlways:
		if len(refs) == 0 {
			return []agent.TaintMark{callMark(policy, call, nil, now)}
		}
	case agent.ExternalReadMarked:
		if len(refs) == 0 {
			return nil
		}
	default:
		return nil
	}

	marks := make([]agent.TaintMark, 0, len(refs))
	for idx := range refs {
		marks = append(marks, callMark(policy, call, &refs[idx], now))
	}

	return marks
}

func callMark(
	policy serviceports.ToolPolicy,
	call serviceports.ToolCall,
	ref *agent.RecordRef,
	now int64,
) agent.TaintMark {
	return agent.TaintMark{
		Source:   policy.Source,
		ToolName: call.Name,
		CallID:   call.ID,
		Ref:      ref,
		At:       now,
	}
}

func taintedEvent(mark agent.TaintMark, marks int) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventRunTainted,
		Data:  serviceports.AssistantRunTaintedEvent{Mark: mark, Marks: marks},
	}
}

func (t *Turn) Taint() *agent.RunTaint { return t.result.Taint }

func (t *Turn) absorbTaint(fx TurnEffects, marks []agent.TaintMark) {
	taint := t.result.Taint
	for _, mark := range taint.Absorb(marks) {
		fx.Emit(taintedEvent(mark, len(taint.Marks)))
	}
}

func (t *Turn) announceOpened(fx TurnEffects) {
	opened := t.opened
	t.opened = nil
	for idx, mark := range opened {
		fx.Emit(taintedEvent(mark, len(t.result.Taint.Marks)-len(opened)+idx+1))
	}
}

func (t *Turn) inheritDelegateTaint(fx TurnEffects, result *serviceports.RunResult) {
	if result == nil {
		return
	}
	if result.Taint == nil {
		t.result.Taint = nil

		return
	}
	t.absorbTaint(fx, result.Taint.Marks)
}
