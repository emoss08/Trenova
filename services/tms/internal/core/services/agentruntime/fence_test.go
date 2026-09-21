package agentruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stored result is read back for the transcript exactly as the tool
// returned it, close tag and all, and a line that is not a fenced result
// says so rather than yielding an empty payload.
func TestUnfenceToolResult_ReadsBackWhatWasFenced(t *testing.T) {
	t.Parallel()

	payload := `{"note":"see </untrusted_data> in the docs","rows":[1,2]}`
	name, got, ok := UnfenceToolResult(FenceToolResult("list_shipments", payload))

	require.True(t, ok)
	assert.Equal(t, "list_shipments", name)
	assert.Equal(t, payload, got)

	_, _, ok = UnfenceToolResult(`Tool "run_report" failed: no such report`)
	assert.False(t, ok)
	_, _, ok = UnfenceToolResult("")
	assert.False(t, ok)
}
