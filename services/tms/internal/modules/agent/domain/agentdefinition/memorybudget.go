package agentdefinition

import "github.com/emoss08/trenova/pkg/errortypes"

const (
	DefaultMemoryTokenBudget = 6000
	MinMemoryTokenBudget     = 1000
	MaxMemoryTokenBudget     = 16000
)

// MaxPromptMemories bounds how many memories one prompt carries, whatever
// the token budget leaves room for. Past a few dozen short lines the model
// weighs each one less, and the rest are a recall_memory call away.
const MaxPromptMemories = 50

func (d *Definition) EffectiveMemoryTokenBudget() int {
	if d == nil || d.MemoryTokenBudget == nil {
		return DefaultMemoryTokenBudget
	}

	return *d.MemoryTokenBudget
}

func (d *Definition) validateMemoryBudget(multiErr *errortypes.MultiError) {
	if d.MemoryTokenBudget == nil {
		return
	}

	budget := *d.MemoryTokenBudget
	if budget < MinMemoryTokenBudget || budget > MaxMemoryTokenBudget {
		multiErr.Add(
			"memoryTokenBudget",
			errortypes.ErrInvalid,
			"Memory in the prompt must be between {0} and {1} tokens; leave it empty for the default of {2}",
			MinMemoryTokenBudget,
			MaxMemoryTokenBudget,
			DefaultMemoryTokenBudget,
		)
	}
}
