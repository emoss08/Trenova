package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func (t *Turn) Taint() *agent.RunTaint { return t.result.Taint }

func (t *Turn) absorbTaint(fx TurnEffects, marks []agent.TaintMark) {
	taint := t.result.Taint
	for _, mark := range taint.Absorb(marks) {
		fx.Emit(turntaint.Event(mark, len(taint.Marks)))
	}
}

func (t *Turn) announceOpened(fx TurnEffects) {
	opened := t.opened
	t.opened = nil
	for idx, mark := range opened {
		fx.Emit(turntaint.Event(mark, len(t.result.Taint.Marks)-len(opened)+idx+1))
	}
}

// InheritDelegateTaint folds into the turn whatever outside content a
// delegate read, announcing each new mark through emit.
func (t *Turn) InheritDelegateTaint(
	emit func(kernelports.StreamEvent),
	result *agentruntimeports.RunResult,
) {
	if result == nil {
		return
	}
	if result.Taint == nil {
		t.result.Taint = nil

		return
	}
	taint := t.result.Taint
	for _, mark := range taint.Inherit(result.Taint) {
		emit(turntaint.Event(mark, len(taint.Marks)))
	}
}
