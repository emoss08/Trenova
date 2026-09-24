package agentdefinition

import (
	"strconv"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	DefaultMemoryTokenBudget = 6000
	MinMemoryTokenBudget     = 1000
	MaxMemoryTokenBudget     = 16000
)

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
			"Memory in the prompt must be between "+strconv.Itoa(MinMemoryTokenBudget)+
				" and "+strconv.Itoa(MaxMemoryTokenBudget)+" tokens; leave it empty for the default of "+
				strconv.Itoa(DefaultMemoryTokenBudget),
		)
	}
}
