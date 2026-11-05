package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptRead_SettlesWithWhatTheScriptsOwnCallRead(t *testing.T) {
	t.Parallel()

	own := agent.TaintMark{
		Source: agent.TaintSourceRecordNote,
		Ref:    &agentrecord.RecordRef{EntityType: agent.TaintEntityReport, ID: "rrun_1"},
	}

	turn := &agent.RunTaint{}
	read := scriptRead(turn, []agent.TaintMark{own})

	require.True(t, read.Tainted())
	assert.True(t, read.Has(agent.TaintSourceRecordNote))
	assert.False(t, turn.Tainted(), "the turn's own taint is left as it was")

	assert.True(t, scriptRead(nil, []agent.TaintMark{own}).Tainted())
	assert.Nil(t, scriptRead(nil, nil), "a taint nobody knows stays unknown")
}
