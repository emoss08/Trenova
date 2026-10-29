package turntaint

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func Open(
	req *agentruntimeports.RunRequest,
	rc *agentturn.RuntimeContext,
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

func contextTaint(rc *agentturn.RuntimeContext, now int64) []agent.TaintMark {
	marks := make([]agent.TaintMark, 0, 1+len(rc.Attachments)+len(rc.Memories))
	if rc.Subject != nil {
		if mark, ok := agent.SubjectTaint(rc.Subject.Type, rc.Subject.ID, now); ok {
			marks = append(marks, mark)
		} else if mark, ok = agent.OutsideAuthoredTaint(
			rc.Subject.OutsideAuthored, rc.Subject.Type, rc.Subject.ID, now,
		); ok {
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

func Call(
	policy agenttoolports.ToolPolicy,
	call llmports.ToolCall,
	data any,
	now int64,
) []agent.TaintMark {
	var refs []agent.SourcedRef
	switch policy.ReadsExternal {
	case agenttooltrust.ExternalReadAlways:
		refs = carriedRefs(&policy, data)
		if len(refs) == 0 {
			return []agent.TaintMark{callMark(call, policy.Source, nil, now)}
		}
	case agenttooltrust.ExternalReadMarked:
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

func carriedRefs(policy *agenttoolports.ToolPolicy, data any) []agent.SourcedRef {
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
	call llmports.ToolCall,
	source agent.TaintSource,
	ref *agentrecord.RecordRef,
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

func Event(mark agent.TaintMark, marks int) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantports.AssistantEventRunTainted,
		Data:  assistantports.AssistantRunTaintedEvent{Mark: mark, Marks: marks},
	}
}
