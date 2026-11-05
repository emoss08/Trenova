package tooldispatch

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/stretchr/testify/assert"
)

func TestUnresolvedSelection_KeepsTheErrorTextInsideTheFence(t *testing.T) {
	t.Parallel()

	refused := unresolvedSelection("shipment_update", "run", errors.New("no such record"))

	assert.True(t, refused.Failed)
	assert.Contains(t, refused.Content, toolresult.FenceInline("no such record"))
	assert.Contains(t, refused.Content, "Fix the call and try again.")
}
