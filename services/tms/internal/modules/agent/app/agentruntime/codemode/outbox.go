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
	ErrDuplicateIntent = errors.New("the script already prepared this change")
	ErrBatchTooLarge   = errors.New("the script prepared more changes than one batch holds")
	ErrIntentRan       = errors.New("a prepared change must not have run")
	ErrOutboxFiled     = errors.New("the script's changes were already filed")
)

// Intent is one write a script prepared: the call dispatch decided and
// previewed for it, held until the script ends and its batch is filed.
type Intent struct {
	Seq    int
	Action agentports.PendingAction
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
	filed         bool
}

func NewOutbox(scriptStepKey string, changeLimit int) *Outbox {
	return &Outbox{
		scriptStepKey: scriptStepKey,
		changeLimit:   changeLimit,
		intents:       make([]Intent, 0, MaxBatchSteps),
		seen:          make(map[string]int, MaxBatchSteps),
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

// Admit refuses a call before dispatch reads anything for it: one the script
// already prepared, one past the batch's size, or any once the batch is
// filed. It returns the key Hold keeps the call under.
func (o *Outbox) Admit(tool string, args map[string]any) (string, error) {
	key, err := intentKey(tool, args)
	if err != nil {
		return "", err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	return key, o.admitLocked(key)
}

func (o *Outbox) admitLocked(key string) error {
	if o.filed {
		return ErrOutboxFiled
	}
	if seq, ok := o.seen[key]; ok {
		return fmt.Errorf("%w as change %d", ErrDuplicateIntent, seq)
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
func (o *Outbox) Hold(key string, action *agentports.PendingAction) (int, error) {
	if action.Executed || action.Simulated || action.Tier == agent.TierAutoExecute {
		return 0, ErrIntentRan
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if err := o.admitLocked(key); err != nil {
		return 0, err
	}

	held := *action
	held.Arguments = maputils.CloneDeep(action.Arguments)
	held.HeldBy = slices.Clone(action.HeldBy)
	held.ScriptStepKey = o.scriptStepKey
	seq := len(o.intents) + 1
	o.intents = append(o.intents, Intent{Seq: seq, Action: held})
	o.seen[key] = seq

	return seq, nil
}

func (o *Outbox) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	return len(o.intents)
}

// Settle closes the outbox and returns what is filed: nothing for a script
// that did not finish, else every intent in the order prepared, decided
// again with the script's whole taint and never above waiting for a person.
func (o *Outbox) Settle(settlement Settlement) []agentports.PendingAction {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.filed {
		return nil
	}
	o.filed = true
	if !settlement.Completed {
		return nil
	}

	actions := make([]agentports.PendingAction, 0, len(o.intents))
	for idx := range o.intents {
		action := o.intents[idx].Action
		settle(&action, settlement)
		actions = append(actions, action)
	}

	return actions
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

func intentKey(tool string, args map[string]any) (string, error) {
	digest, err := jsonutils.CanonicalDigest(map[string]any{"tool": tool, "arguments": args})
	if err != nil {
		return "", fmt.Errorf("read the change's arguments: %w", err)
	}

	return digest, nil
}
