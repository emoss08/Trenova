package turntaint_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_AResumedWaitLeavesTheNextTurnCappedAndTainted(t *testing.T) {
	t.Parallel()

	resumed, opened := turntaint.Open(
		&agentruntimeports.RunRequest{AfterExternalContent: true},
		&agentturn.RuntimeContext{},
		10,
	)
	require.Len(t, opened, 1)
	assert.Equal(t, agent.TaintSourceWeb, opened[0].Source)

	thread := &conversation.Thread{}
	require.True(t, thread.AbsorbTaint(resumed, 10), "the clean thread keeps the mark")

	next, reopened := turntaint.Open(
		&agentruntimeports.RunRequest{Taint: thread.Taint},
		&agentturn.RuntimeContext{},
		20,
	)
	assert.Empty(t, reopened, "the next turn reads nothing new")
	assert.True(t, next.Tainted())
	assert.True(t, toolaccess.CarriesExternalContent(next))
}

func TestOpen_ATurnWithoutOutsideInputOpensClean(t *testing.T) {
	t.Parallel()

	taint, opened := turntaint.Open(
		&agentruntimeports.RunRequest{},
		&agentturn.RuntimeContext{},
		10,
	)

	assert.Empty(t, opened)
	assert.False(t, taint.Tainted())
}
