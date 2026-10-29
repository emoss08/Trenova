package guidequerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
)

func TestGuideToolEffects(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agenttooltrust.ToolEffectNavigate, agenttoolports.EffectOf(&openPageTool{}))
	assert.Equal(
		t,
		agenttooltrust.ToolEffectDiscover,
		agenttoolports.EffectOf(&findInTrenovaTool{}),
	)
}
