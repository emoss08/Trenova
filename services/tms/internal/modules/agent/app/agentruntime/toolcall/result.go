package toolcall

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
)

type Result struct {
	Content string
	Failed  bool
	Action  *agentruntimeports.PendingAction
	// Batch is what a script prepared and filed for a person as one batch;
	// none of it ran.
	Batch []agentruntimeports.PendingAction
	// Continuation is how a later call of the turn may continue a script.
	Continuation codemode.Continuation
	// Unread is a script that stopped on an error before it read anything.
	Unread bool
	// Data is what a query tool returned, before it was encoded for the
	// model. The caller may turn it into something a person sees.
	Data    any
	RanWith map[string]any
	// Publishes marks a document the observer has to keep; its result is
	// written once the observer says what it kept.
	Publishes bool
	Summary   string
	// DelegateReport is the bounded account of a delegate_task call, kept on
	// the call's result message.
	DelegateReport *conversation.DelegateReport
	// Taint is the outside content the call read.
	Taint []agent.TaintMark
	Found []string
	// HandOff names the person's other agents a find_tools answer named as
	// holding what it matched, for the Desk's hand-off menu.
	HandOff []pulid.ID
	Verdict string
	Reason  string
	// Dispatched marks an outcome dispatch produced, which wrote its own
	// step; any other refusal is the loop's own (RunResult.LoopRefusals).
	Dispatched bool
	// Memories are the memories a recall read back, and saved the one a
	// remember kept or offered.
	Memories []pulid.ID
	Saved    *agentmemoryports.SavedMemory
	// Shown is what the observer put in front of the person for the call.
	Shown *agentruntimeports.ShownArtifact
}

func Failed(format string, args ...any) Result {
	return Result{Content: fmt.Sprintf(format, args...), Failed: true}
}

func Refused(verdict, reason, format string, args ...any) Result {
	outcome := Failed(format, args...)
	outcome.Verdict = verdict
	outcome.Reason = reason

	return outcome
}

func (o *Result) VerdictOrDerived() string {
	if o.Verdict != "" {
		return o.Verdict
	}

	return derivedVerdict(o.Failed, o.Action)
}

func (o Result) Outcome() Outcome {
	return Outcome{
		Content:        o.Content,
		Failed:         o.Failed,
		Action:         o.Action,
		Batch:          o.Batch,
		Publishes:      o.Publishes,
		Summary:        o.Summary,
		DelegateReport: o.DelegateReport,
		Taint:          o.Taint,
		Found:          o.Found,
		HandOff:        o.HandOff,
		Verdict:        o.Verdict,
		Memories:       o.Memories,
		SavedMemory:    o.Saved,
		Shown:          o.Shown,
		Continuation:   o.Continuation,
		Unread:         o.Unread,
		Data:           o.Data,
		RanWith:        o.RanWith,
	}
}
