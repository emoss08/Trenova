package agentinstructions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructionVariablesListsEachPlaceholderOnceInOrder(t *testing.T) {
	t.Parallel()

	text := "Greet {{user.name}} of {{organization}}. Today is {{today}}; " +
		"remind {{user.name}} again. Ignore { single } and {{}}."

	assert.Equal(
		t,
		[]string{"{{user.name}}", "{{organization}}", "{{today}}"},
		InstructionVariables(text),
	)
}

func TestInstructionVariablesIsEmptyWithoutPlaceholders(t *testing.T) {
	t.Parallel()

	assert.Empty(t, InstructionVariables("Answer billing questions."))
}

func TestChangedInstructionVariablesNamesWhatWasLostAndAdded(t *testing.T) {
	t.Parallel()

	missing, added := ChangedInstructionVariables(
		"Help {{user.name}} at {{organization}} on {{today}}.",
		"Help {{ user.name }} at {{organization}} for {{user.role}}.",
	)

	assert.Equal(t, []string{"{{user.name}}", "{{today}}"}, missing)
	assert.Equal(t, []string{"{{ user.name }}", "{{user.role}}"}, added)
}

func TestChangedInstructionVariablesIsEmptyWhenKept(t *testing.T) {
	t.Parallel()

	missing, added := ChangedInstructionVariables(
		"Help {{user.name}} today, {{today}}.",
		"On {{today}}, help {{user.name}}.",
	)

	assert.Empty(t, missing)
	assert.Empty(t, added)
}
