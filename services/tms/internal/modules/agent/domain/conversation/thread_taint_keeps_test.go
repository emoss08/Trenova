package conversation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThread_AbsorbTaintKeepsSourcesAndOverflowItHasNoMarkFor(t *testing.T) {
	t.Parallel()

	read := agent.TaintMark{
		Source: agent.TaintSourceEDI,
		Ref:    &agentrecord.RecordRef{EntityType: agent.TaintEntityShipment, ID: "shp_1"},
	}
	thread := &Thread{}
	first := &agent.RunTaint{}
	first.Add(read)
	require.True(t, thread.AbsorbTaint(first, 10))

	assert.True(t, thread.AbsorbTaint(&agent.RunTaint{
		Marks: []agent.TaintMark{read},
		Seen:  []agent.TaintSource{agent.TaintSourceEDI, agent.TaintSourceWeb},
	}, 20), "a source read after the marks filled is news")
	assert.True(t, thread.Taint.Has(agent.TaintSourceWeb))

	assert.True(t, thread.AbsorbTaint(&agent.RunTaint{
		Marks:      []agent.TaintMark{read},
		Overflowed: true,
	}, 30), "so is learning that some reads were not kept")
	assert.True(t, thread.Taint.Overflowed)

	assert.False(t, thread.AbsorbTaint(thread.Taint.Clone(), 40))
}
