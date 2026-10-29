package reportquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
)

func TestReportToolEffects(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agenttooltrust.ToolEffectPresent, agenttoolports.EffectOf(&runReportTool{}))
	assert.Equal(
		t,
		agenttooltrust.ToolEffectPresent,
		agenttoolports.EffectOf(&compareReportRunsTool{}),
	)
	assert.Equal(t, agenttooltrust.ToolEffectLookup, agenttoolports.EffectOf(&previewReportTool{}))
}
