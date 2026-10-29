package turnscripts

import (
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// Ledger is what a turn keeps of the scripts it ran.
type Ledger struct {
	// Continuable is every script of the turn a later call may continue, by
	// its call id, with how; Continuing is every call that continues one, by
	// its own call id, with the id of the script it continues.
	Continuable map[string]codemode.Continuation
	Continuing  map[string]string
	// Unread are the turn's scripts that stopped on an error before they
	// read anything, which the turn's script count leaves out.
	Unread map[string]bool
}

// Position is how many calls to the script tool the turn has made up to and
// including this one, zero for any other tool, reading the calls the turn
// made from its messages. A call that continues a script is not counted: it
// finishes what that script began. Nor is an earlier script that stopped on
// an error before it read anything, up to ScriptsPerTurn of them: correcting
// a read's name is not another analysis.
func (l *Ledger) Position(call *llmports.ToolCall, messages []conversation.Message) int {
	if call.Name != codemode.ToolRunScript {
		return 0
	}
	position, unread := 0, 0
	for idx := range messages {
		for _, made := range messages[idx].ToolCalls {
			if made.Name != codemode.ToolRunScript {
				continue
			}
			counted := !l.finishes(made.ID)
			if made.ID != call.ID && l.Unread[made.ID] {
				unread++
				counted = unread > codemode.ScriptsPerTurn
			}
			if counted {
				position++
			}
			if made.ID == call.ID {
				return position
			}
		}
	}

	return position + 1
}

// finishes reports whether the call with id continues a script that read a
// dataset and showed nothing: it finishes that analysis rather than starting
// another, so the turn's script count leaves it out.
func (l *Ledger) finishes(id string) bool {
	earlier, continues := l.Continuing[id]

	return continues && l.Continuable[earlier] == codemode.ContinueFree
}

// Continue is the arguments of the script of this turn that call continues,
// its code the whole of what ran before: one whose answer said it may be
// continued and that no other call has continued. It claims that script for
// call, so a second call naming it runs nothing. Nil when call continues
// nothing it may.
func (l *Ledger) Continue(call *llmports.ToolCall, messages []conversation.Message) map[string]any {
	if call.Name != codemode.ToolRunScript {
		return nil
	}
	earlier := strings.TrimSpace(argcheck.String(call.Arguments, codemode.ParamScriptContinues))
	if earlier == "" {
		return nil
	}
	if l.Continuable[earlier] == codemode.ContinueNone {
		return nil
	}
	for other, continued := range l.Continuing {
		if continued == earlier && other != call.ID {
			return nil
		}
	}
	arguments := l.continuedArguments(earlier, 0, messages)
	if arguments == nil {
		return nil
	}
	if l.Continuing == nil {
		l.Continuing = make(map[string]string, 1)
	}
	l.Continuing[call.ID] = earlier

	return arguments
}

// continuedArguments is what the script call with id ran: its own arguments,
// or for a call that itself continued one, the code of every script before it
// and then its own, over the datasets and inputs of the first. A chain is
// never longer than the scripts a turn runs.
func (l *Ledger) continuedArguments(
	id string,
	depth int,
	messages []conversation.Message,
) map[string]any {
	arguments := scriptArguments(id, messages)
	earlier, chained := l.Continuing[id]
	if arguments == nil || !chained {
		return arguments
	}
	if depth > 2*codemode.ScriptsPerTurn {
		return nil
	}
	base := l.continuedArguments(earlier, depth+1, messages)
	if base == nil {
		return nil
	}
	whole := maps.Clone(base)
	whole[codemode.ParamScriptCode] = strings.TrimRight(
		argcheck.String(base, codemode.ParamScriptCode),
		"\n",
	) + "\n\n" + argcheck.String(
		arguments,
		codemode.ParamScriptCode,
	)
	for _, key := range []string{codemode.ParamScriptTitle, codemode.ParamScriptPurpose} {
		if own := strings.TrimSpace(argcheck.String(arguments, key)); own != "" {
			whole[key] = own
		}
	}

	return whole
}

// scriptArguments is what the turn's script call with id was asked to run.
func scriptArguments(id string, messages []conversation.Message) map[string]any {
	for idx := range messages {
		for _, made := range messages[idx].ToolCalls {
			if made.ID == id && made.Name == codemode.ToolRunScript {
				return made.Arguments
			}
		}
	}

	return nil
}

// Note keeps a script a later call of the turn may continue, and one that
// read nothing.
func (l *Ledger) Note(call *llmports.ToolCall, outcome *toolcall.Result) {
	if outcome.Unread && call.ID != "" {
		if l.Unread == nil {
			l.Unread = make(map[string]bool, 1)
		}
		l.Unread[call.ID] = true
	}
	if outcome.Continuation == codemode.ContinueNone || call.ID == "" {
		return
	}
	if l.Continuable == nil {
		l.Continuable = make(map[string]codemode.Continuation, 1)
	}
	l.Continuable[call.ID] = outcome.Continuation
}
