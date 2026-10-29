package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryCommandHasAHold(t *testing.T) {
	t.Parallel()

	for _, command := range agentextension.AllCommands() {
		hold, ok := CommandHolds[command]
		require.True(t, ok, "%s has no hold", command)
		assert.NotEmpty(t, hold.Directive, command)
		assert.NotEmpty(t, hold.Correction, command)
		assert.NotEmpty(t, hold.RetryReason, command)
		assert.Contains(t, hold.Directive, command.Text(), command)
	}
	assert.Len(t, CommandHolds, len(agentextension.AllCommands()),
		"a hold for a command that does not exist")
}
