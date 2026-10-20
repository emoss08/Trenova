package codemode

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/maputils"
)

// MaxBatchSteps is the most writes one script may prepare: the size of a
// batch the decision queue decides together.
const MaxBatchSteps = 50

var (
	ErrConflictingIntents = errors.New("the script prepared two different changes to one record")
	ErrBatchTooLarge      = errors.New("the script prepared more changes than one batch holds")
	ErrIntentRan          = errors.New("a prepared change must not have run")
	ErrOutboxFiled        = errors.New("the script's changes were already filed")
)

// Intent is one write a script prepared: the call dispatch decided and
// previewed for it, held until the script ends and its batch is filed.
type Intent struct {
	Seq    int
	Action agentports.PendingAction
}

// Call is one write a script asked for, with the record its tool says it
// changes; Target is nil for a tool that names none.
type Call struct {
	Tool      string
	Arguments map[string]any
	Target    *agentports.ToolTarget
}

// Admission says what to do with a call. A call identical to one the script
// already prepared is folded into it: Folded names that change, and nothing
// is dispatched for it.
type Admission struct {
	key    string
	target string
	Folded int
}

// Redecider decides an intent again once the script has ended, with the
// taint of everything the script read, wherever in the script it read it.
type Redecider func(action *agentports.PendingAction) agenttoolpolicy.Decision

// Settlement is how the script ended, as the broker saw it.
type Settlement struct {
	// Completed is false for a script that raised, timed out, was stopped by
	// a limit or lost its sandbox: a batch it did not finish is never filed.
	Completed bool
	// Tainted is the script's taint, and ExternalContent whether it, or the
	// turn before it, read the web or pasted partner text.
	Tainted         bool
	ExternalContent bool
	Redecide        Redecider
}

// Outbox holds the writes one script prepared. Nothing in it has run, and
// nothing in it is ever sent into the sandbox: the script holds only the
// sequence numbers it was given.
type Outbox struct {
	mu            sync.Mutex
	scriptStepKey string
	changeLimit   int
	intents       []Intent
	seen          map[string]int
	targets       map[string]targetUse
	conflict      error
	filed         bool
}

type targetUse struct {
	seq int
	key string
}

func NewOutbox(scriptStepKey string, changeLimit int) *Outbox {
	return &Outbox{
		scriptStepKey: scriptStepKey,
		changeLimit:   changeLimit,
		intents:       make([]Intent, 0, MaxBatchSteps),
		seen:          make(map[string]int, MaxBatchSteps),
		targets:       make(map[string]targetUse, MaxBatchSteps),
	}
}

// Expand is the single-record calls one write of a script is made of: a
// call naming several records of a batch tool is that tool's call for each
// record alone, so every record is its own step with its own preview.
func Expand(tool agentports.AgentTool, args map[string]any) ([]map[string]any, error) {
	if splitter, ok := tool.(agentports.ToolRecordSplitter); ok {
		return splitter.SplitRecords(args)
	}

	return []map[string]any{args}, nil
}

// Admit decides a call before dispatch reads anything for it. A call the
// script already prepared word for word is folded into that change. A
// different change to a record the script already changes through the same
// tool refuses the whole batch, naming both: which of the two the script
// meant is never picked for it. A call past the batch's size is refused,
// and so is any once the batch is filed or refused.
func (o *Outbox) Admit(call Call) (Admission, error) {
	key, err := intentKey(call.Tool, call.Arguments)
	if err != nil {
		return Admission{}, err
	}
	admission := Admission{key: key, target: targetKey(call)}

	o.mu.Lock()
	defer o.mu.Unlock()

	if seq, ok := o.seen[key]; ok && o.usable() == nil {
		admission.Folded = seq

		return admission, nil
	}

	return admission, o.admitLocked(admission)
}

func (o *Outbox) usable() error {
	if o.filed {
		return ErrOutboxFiled
	}

	return o.conflict
}

func (o *Outbox) admitLocked(admission Admission) error {
	if err := o.usable(); err != nil {
		return err
	}
	if admission.target != "" {
		if used, ok := o.targets[admission.target]; ok && used.key != admission.key {
			o.conflict = fmt.Errorf(
				"%w: changes %d and %d both change the same record with the same tool, "+
					"differently; nothing in this batch is filed",
				ErrConflictingIntents, used.seq, len(o.intents)+1,
			)

			return o.conflict
		}
	}
	if len(o.intents) >= o.limit() {
		return fmt.Errorf("%w: at most %d", ErrBatchTooLarge, o.limit())
	}

	return nil
}

func (o *Outbox) limit() int {
	if o.changeLimit > 0 && o.changeLimit < MaxBatchSteps {
		return o.changeLimit
	}

	return MaxBatchSteps
}

// Hold keeps a call dispatch decided and previewed, and returns the number
// the script refers to it by. A call that ran, or would run, is refused: an
// intent waits for a person whatever its tool would do in a turn.
func (o *Outbox) Hold(admission Admission, action *agentports.PendingAction) (int, error) {
	if admission.Folded > 0 {
		return admission.Folded, nil
	}
	if action.Executed || action.Simulated || action.Tier == agent.TierAutoExecute {
		return 0, ErrIntentRan
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if seq, ok := o.seen[admission.key]; ok && o.usable() == nil {
		return seq, nil
	}
	if err := o.admitLocked(admission); err != nil {
		return 0, err
	}

	held := *action
	held.Arguments = maputils.CloneDeep(action.Arguments)
	held.HeldBy = slices.Clone(action.HeldBy)
	held.ScriptStepKey = o.scriptStepKey
	seq := len(o.intents) + 1
	o.intents = append(o.intents, Intent{Seq: seq, Action: held})
	o.seen[admission.key] = seq
	if admission.target != "" {
		o.targets[admission.target] = targetUse{seq: seq, key: admission.key}
	}

	return seq, nil
}

func (o *Outbox) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	return len(o.intents)
}

// Settle closes the outbox and returns what is filed: nothing for a script
// that did not finish or whose batch was refused, else every intent in the
// order prepared, decided again with the script's whole taint and never
// above waiting for a person.
func (o *Outbox) Settle(settlement Settlement) ([]agentports.PendingAction, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.filed {
		return nil, ErrOutboxFiled
	}
	o.filed = true
	if o.conflict != nil {
		return nil, o.conflict
	}
	if !settlement.Completed {
		return nil, nil
	}

	actions := make([]agentports.PendingAction, 0, len(o.intents))
	for idx := range o.intents {
		action := o.intents[idx].Action
		settle(&action, settlement)
		actions = append(actions, action)
	}

	return actions, nil
}

func settle(action *agentports.PendingAction, settlement Settlement) {
	if settlement.Redecide != nil {
		decision := settlement.Redecide(action)
		if action.Tier.Above(decision.Tier) {
			action.Tier = decision.Tier
		}
		for _, key := range decision.HeldBy {
			hold(action, key)
		}
	}
	if settlement.Tainted || settlement.ExternalContent {
		action.Tainted = true
	}
	if settlement.ExternalContent && action.Tier.Above(agent.TierPropose) {
		action.Tier = agent.TierPropose
		hold(action, agenttoolpolicy.HeldByTainted)
	}
	if action.Tier.Above(agent.TierActWithApproval) {
		action.Tier = agent.TierActWithApproval
	}
	hold(action, agenttoolpolicy.HeldByScriptIntent)
	action.MoneyMinor = nil
	action.MoneyCurrency = ""
}

func hold(action *agentports.PendingAction, key string) {
	if !slices.Contains(action.HeldBy, key) {
		action.HeldBy = append(action.HeldBy, key)
	}
}

func targetKey(call Call) string {
	if call.Target == nil || call.Target.ID.IsNil() {
		return ""
	}

	return call.Tool + "\x00" + string(call.Target.Resource) + "\x00" + call.Target.ID.String()
}

func intentKey(tool string, args map[string]any) (string, error) {
	digest, err := jsonutils.CanonicalDigest(map[string]any{"tool": tool, "arguments": args})
	if err != nil {
		return "", fmt.Errorf("read the change's arguments: %w", err)
	}

	return digest, nil
}
