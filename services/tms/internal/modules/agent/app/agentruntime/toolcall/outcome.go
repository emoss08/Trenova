package toolcall

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
)

// Call is one tool call the loop has decided to run.
type Call struct {
	Call           llmports.ToolCall                 `json:"call"`
	CompletionText string                            `json:"completionText"`
	ProposedSoFar  []agentruntimeports.PendingAction `json:"proposedSoFar,omitempty"`
	// Ordinal numbers this exact call within the run, so the step key of a
	// legitimately repeated call differs from the first one's.
	Ordinal int `json:"ordinal"`
	// Taint is the outside content the turn had read when it made the call.
	// Nil is a turn opened before taint was kept.
	Taint *agent.RunTaint `json:"taint,omitempty"`
	// AfterExternalContent says the turn has read content from outside the
	// organization, so a write is proposed rather than run.
	AfterExternalContent bool     `json:"afterExternalContent,omitempty"`
	Earlier              []string `json:"earlier,omitempty"`
	// MoneySpent is what the turn's automatic money writes, its delegates'
	// and the delegating turn's, have moved and not yet recorded.
	MoneySpent agenttooltrust.MoneyUse `json:"moneySpent,omitzero"`
	Beyond     agent.AutonomyTier      `json:"beyond,omitempty"`
	// Model, Provider and ProviderID are what answered the completion that
	// asked for the call, and ScriptPosition is how many scripts the turn has
	// asked for up to and including this one; the code-mode gate reads the
	// answering provider's settings and the count. ScriptContinues is the
	// arguments of the script of the turn the call continues, which the count
	// leaves out.
	Model           string          `json:"model,omitempty"`
	Provider        aiprovider.Kind `json:"provider,omitempty"`
	ProviderID      pulid.ID        `json:"providerId,omitempty"`
	ScriptPosition  int             `json:"scriptPosition,omitempty"`
	ScriptContinues map[string]any  `json:"scriptContinues,omitempty"`
	// ScriptReach is, for the call that runs a script, the reads the turn
	// reached beyond the agent's own through find_tools, each with its tier:
	// the script may call them as the model may.
	ScriptReach map[string]agent.AutonomyTier `json:"scriptReach,omitempty"`
}

// Outcome is what one tool call came to.
type Outcome struct {
	Content string                           `json:"content"`
	Failed  bool                             `json:"failed"`
	Action  *agentruntimeports.PendingAction `json:"action,omitempty"`
	// Batch is what a script prepared, filed as one batch for a person.
	Batch []agentruntimeports.PendingAction `json:"batch,omitempty"`
	// Publishes marks a document still to be kept. What the model reads is
	// written once it has been.
	Publishes bool   `json:"publishes,omitempty"`
	Summary   string `json:"summary,omitempty"`
	// DelegateReport is the bounded account of a delegate_task call.
	DelegateReport *conversation.DelegateReport `json:"delegateReport,omitempty"`
	// Taint is the outside content the call read.
	Taint   []agent.TaintMark `json:"taint,omitempty"`
	Found   []string          `json:"found,omitempty"`
	HandOff []pulid.ID        `json:"handOff,omitempty"`
	Verdict string            `json:"verdict,omitempty"`
	// Memories are what a recall read back; SavedMemory is what a remember
	// kept or offered to keep.
	Memories    []pulid.ID                    `json:"memories,omitempty"`
	SavedMemory *agentmemoryports.SavedMemory `json:"savedMemory,omitempty"`
	// Shown is what the call put in front of the person, which the loop
	// keeps when it is a table, for the pass over the final reply.
	Shown *agentruntimeports.ShownArtifact `json:"shown,omitempty"`
	// Continuation is how a later call of the turn may continue the script
	// the call ran.
	Continuation codemode.Continuation `json:"continuation,omitempty"`
	// Unread is a script that stopped on an error before it read anything,
	// which the turn's script count leaves out.
	Unread bool `json:"unread,omitempty"`
	// Data is what a query tool returned before it was encoded for the model.
	// It never crosses a durable boundary: whatever needs it runs where the
	// tool ran.
	Data    any            `json:"-"`
	RanWith map[string]any `json:"-"`
}

func (o *Outcome) Result() Result {
	return Result{
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
		Saved:          o.SavedMemory,
		Shown:          o.Shown,
		Continuation:   o.Continuation,
		Unread:         o.Unread,
		Data:           o.Data,
		RanWith:        o.RanWith,
	}
}
