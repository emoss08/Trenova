package toolaccess

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
)

func TestOpenTurn_WithoutANeedRanksOnTheMessage(t *testing.T) {
	t.Parallel()

	req := &agentruntimeports.RunRequest{Actor: runtimetest.Actor(), Input: "trailer inspections"}
	assert.Equal(t, agentruntimeports.TurnQueryRequest(req).Text, TurnQuery(req).Text)
	assert.Empty(t, FrameNeed(req))
}
