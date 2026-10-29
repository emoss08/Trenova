package agentruntime

import (
	"cmp"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// ReplayAware is implemented by effects that can re-run the loop over a
// recorded history. Workflow code re-executes the loop whenever a worker
// rebuilds an execution it no longer holds, and a counter bumped on every
// pass would count each call once per rebuild.
type ReplayAware interface {
	Replaying() bool
}

// countToolOutcome counts one finished call by tool and verdict. Every call
// the loop answers comes through recordToolResult, refusals included, so this
// is the one place a call is counted. A call with no verdict of its own, a
// find_tools answer or a repeated read, is counted as the verdict its result
// implies.
func (s *Service) countToolOutcome(
	t *Turn,
	fx TurnEffects,
	name string,
	outcome *toolcall.Result,
) {
	if aware, ok := fx.(ReplayAware); ok && aware.Replaying() {
		return
	}

	s.metrics.RecordToolOutcome(aimetrics.ToolOutcomeLabels{
		Tool:     s.access.ToolMetricLabel(name),
		Verdict:  outcome.VerdictOrDerived(),
		Model:    aianalytics.CanonicalModel(t.result.Model),
		Provider: string(t.providerKind),
	})
}

const (
	ReaskLoop        = "loop"
	ReaskCutOff      = "cut_off"
	ReaskEmptyReply  = "empty_reply"
	ReaskCommand     = "command"
	ReaskGrounding   = "grounding"
	ReaskPlacement   = "placement"
	ReaskBudgetSpent = "budget_spent"
	ReaskStuck       = "stuck"
)

func (s *Service) reask(
	t *Turn,
	fx TurnEffects,
	guard string,
	completion *llmports.ChatCompletionResult,
) {
	t.reask = guard
	if aware, ok := fx.(ReplayAware); ok && aware.Replaying() {
		return
	}

	model, provider := t.result.Model, t.providerKind
	if completion != nil {
		model = cmp.Or(completion.ModelIdentifier, model)
		provider = cmp.Or(completion.ProviderKind, provider)
	}
	s.metrics.RecordGuardReask(aimetrics.GuardReaskLabels{
		Guard:    guard,
		Model:    aianalytics.CanonicalModel(model),
		Provider: string(provider),
	})
}

func (t *Turn) takeReask() string {
	guard := t.reask
	t.reask = ""

	return guard
}
