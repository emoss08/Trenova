package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryVectorState_RefusesATornVector(t *testing.T) {
	t.Parallel()

	state := packQueryVector(runtimetest.UsableQuery())
	require.NotNil(t, state)

	state.Vector = state.Vector[:len(state.Vector)-4]
	assert.False(t, state.unpack().Usable())

	var missing *QueryVectorState
	assert.False(t, missing.unpack().Usable())
}
