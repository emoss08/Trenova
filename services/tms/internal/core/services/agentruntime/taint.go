package agentruntime

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func openTaint(
	req *serviceports.RunRequest,
	rc *agentdefinition.RuntimeContext,
	now int64,
) (*agent.RunTaint, []agent.TaintMark) {
	if req.Delegation != nil && req.Taint == nil {
		return nil, nil
	}

	taint := req.Taint.Clone()
	if taint == nil {
		taint = &agent.RunTaint{}
	}

	opened := taint.Absorb(contextTaint(rc, now))

	return taint, opened
}

func contextTaint(rc *agentdefinition.RuntimeContext, now int64) []agent.TaintMark {
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
	var refs []agent.SourcedRef
	switch policy.ReadsExternal {
	case agent.ExternalReadAlways:
		refs = carriedRefs(&policy, data)
		if len(refs) == 0 {
			return []agent.TaintMark{callMark(call, policy.Source, nil, now)}
		}
	case agent.ExternalReadMarked:
		refs = carriedRefs(&policy, data)
		if len(refs) == 0 {
			return nil
		}
	default:
		return nil
	}

	marks := make([]agent.TaintMark, 0, len(refs))
	for idx := range refs {
		marks = append(marks, callMark(call, refs[idx].Source, &refs[idx].Ref, now))
	}

	return marks
}

func carriedRefs(policy *serviceports.ToolPolicy, data any) []agent.SourcedRef {
	if carrier, ok := data.(agent.SourcedTaintCarrier); ok {
		sourced := carrier.TaintedMarks()
		refs := make([]agent.SourcedRef, 0, len(sourced))
		for _, ref := range sourced {
			if !ref.Source.IsValid() || !policy.MarksFrom(ref.Source) {
				ref.Source = policy.Source
			}
			refs = append(refs, ref)
		}

		return refs
	}

	carrier, ok := data.(agent.TaintCarrier)
	if !ok {
		return nil
	}
	records := carrier.TaintedRecords()
	refs := make([]agent.SourcedRef, 0, len(records))
	for _, record := range records {
		refs = append(refs, agent.SourcedRef{Source: policy.Source, Ref: record})
	}

	return refs
}

func callMark(
	call serviceports.ToolCall,
	source agent.TaintSource,
	ref *agent.RecordRef,
	now int64,
) agent.TaintMark {
	return agent.TaintMark{
		Source:   source,
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
