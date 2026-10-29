package agentdefinition

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
