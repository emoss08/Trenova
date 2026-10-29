package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/stretchr/testify/assert"
)

func TestAfterExternalContentOnlyLowersTheTier(t *testing.T) {
	t.Parallel()

	tier, held := AfterExternalContent(agent.TierAutoExecute, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.True(t, held)

	tier, held = AfterExternalContent(agent.TierActWithApproval, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.True(t, held)

	tier, held = AfterExternalContent(agent.TierPropose, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.False(t, held)

	tier, held = AfterExternalContent(agent.TierAutoExecute, false)
	assert.Equal(t, agent.TierAutoExecute, tier)
	assert.False(t, held)
}

func TestReadsExternalContent_NamesTheWebAndPastedPartnerText(t *testing.T) {
	t.Parallel()

	assert.True(t, ReadsExternalContent(agentextension.ToolWebSearch))
	assert.True(t, ReadsExternalContent(agentextension.ToolWebRead))
	assert.True(t, ReadsExternalContent("edi_inspect_x12"))
	assert.True(t, ReadsExternalContent("edi_draft_connection"),
		"a partner's implementation guide is their text too")
	assert.False(t, ReadsExternalContent("edi_get_message"),
		"a stored partner document taints the run; only text handed to the agent caps writes")
}
