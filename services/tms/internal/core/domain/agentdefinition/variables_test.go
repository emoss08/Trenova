package agentdefinition

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

func TestCronExpressionValid(t *testing.T) {
	t.Parallel()

	assert.True(t, CronExpressionValid("0 7 * * 1-5"))
	assert.True(t, CronExpressionValid("  */15 * * * *  "))
	assert.False(t, CronExpressionValid(""))
	assert.False(t, CronExpressionValid("every morning"))
	assert.False(t, CronExpressionValid("0 7 * *"))
}

func TestCronTimezoneValid(t *testing.T) {
	t.Parallel()

	assert.True(t, CronTimezoneValid("America/Chicago"))
	assert.True(t, CronTimezoneValid("UTC"))
	assert.False(t, CronTimezoneValid(""))
	assert.False(t, CronTimezoneValid("Mars/Olympus"))
}
