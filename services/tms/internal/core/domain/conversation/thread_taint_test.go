package conversation

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThread_AbsorbTaintKeepsWhatTheConversationRead(t *testing.T) {
	t.Parallel()

	thread := &Thread{}
	assert.False(t, thread.Tainted())
	assert.False(t, thread.AbsorbTaint(nil, 10))
	assert.False(t, thread.AbsorbTaint(&agent.RunTaint{}, 10))

	read := &agent.RunTaint{}
	read.Add(agent.TaintMark{Source: agent.TaintSourceAttachment, CallID: "", Ref: &agent.RecordRef{
		EntityType: agent.TaintEntityDocument,
		ID:         "doc_1",
	}})
	assert.True(t, thread.AbsorbTaint(read, 20))
	assert.True(t, thread.Tainted())
	require.NotNil(t, thread.TaintedAt)
	assert.Equal(t, int64(20), *thread.TaintedAt)
	assert.NotSame(t, read, thread.Taint)

	assert.False(t, thread.AbsorbTaint(read, 30), "the same content read again is nothing new")
	assert.Equal(t, int64(20), *thread.TaintedAt)
}
