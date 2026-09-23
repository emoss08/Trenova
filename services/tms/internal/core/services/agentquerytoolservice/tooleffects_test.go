package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

func TestDeclaredToolEffects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tool any
		want agent.ToolEffect
	}{
		{tool: &openPageTool{}, want: agent.ToolEffectNavigate},
		{tool: &findInTrenovaTool{}, want: agent.ToolEffectDiscover},
		{tool: &runReportTool{}, want: agent.ToolEffectPresent},
		{tool: &compareReportRunsTool{}, want: agent.ToolEffectPresent},
		{tool: &composeTableViewTool{}, want: agent.ToolEffectPresent},
		{tool: &planDispatchTool{}, want: agent.ToolEffectPresent},
		{tool: &previewReportTool{}, want: agent.ToolEffectLookup},
		{tool: &listTool{}, want: agent.ToolEffectLookup},
		{tool: &getTool{}, want: agent.ToolEffectLookup},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, serviceports.EffectOf(tc.tool))
	}
}
