package listtool

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
)

func TestListToolEffects(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agenttooltrust.ToolEffectLookup, agenttoolports.EffectOf(&Tool{}))
}
