package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
)

func TestDeclaredToolEffects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tool any
		want agenttooltrust.ToolEffect
	}{
		{tool: &openPageTool{}, want: agenttooltrust.ToolEffectNavigate},
		{tool: &findInTrenovaTool{}, want: agenttooltrust.ToolEffectDiscover},
		{tool: &runReportTool{}, want: agenttooltrust.ToolEffectPresent},
		{tool: &compareReportRunsTool{}, want: agenttooltrust.ToolEffectPresent},
		{tool: &composeTableViewTool{}, want: agenttooltrust.ToolEffectPresent},
		{tool: &planDispatchTool{}, want: agenttooltrust.ToolEffectPresent},
		{tool: &previewReportTool{}, want: agenttooltrust.ToolEffectLookup},
		{tool: &listTool{}, want: agenttooltrust.ToolEffectLookup},
		{tool: &getTool{}, want: agenttooltrust.ToolEffectLookup},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, agenttoolports.EffectOf(tc.tool))
	}
}
