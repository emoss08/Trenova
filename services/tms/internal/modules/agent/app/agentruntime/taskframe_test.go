package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttaskframe"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenTurn_RanksTheOpeningToolsOnTheWrittenNeed(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	vague := &agentruntimeports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "which ones are due?",
	}
	require.False(t, service.OpenTurn(t.Context(), vague).tools.offers("equipment_list_trailers"),
		"the bare follow-up names nothing a tool covers")

	framed := *vague
	framed.Context.Frame = agenttaskframe.ChatFrame(
		"List the trailers whose inspection is due.",
		"list trailers by inspection date",
	)
	turn := service.OpenTurn(t.Context(), &framed)

	assert.True(t, turn.tools.offers("equipment_list_trailers"), "the need names what the work is")
	assert.Equal(
		t,
		"list trailers by inspection date\nwhich ones are due?",
		turnQuery(&framed).Text,
		"the semantic ranking reads the need too, with the message kept beside it",
	)
	assert.Contains(t, turn.system, "What the person wants: List the trailers")
}

func TestOpenTurn_WithoutANeedRanksOnTheMessage(t *testing.T) {
	t.Parallel()

	req := &agentruntimeports.RunRequest{Actor: testActor(), Input: "trailer inspections"}
	assert.Equal(t, agentruntimeports.TurnQueryRequest(req).Text, turnQuery(req).Text)
	assert.Empty(t, frameNeed(req))
}
