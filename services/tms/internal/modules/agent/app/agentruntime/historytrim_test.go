package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The grounding guard checks the figures in a reply against what the turn read.
A figure the model quotes from a result an earlier turn read is supported, and
shortening that result in the replay must not make it look invented.
*/
func TestOpenTurn_KeepsTheFiguresOfShortenedResultsAsEvidence(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat(`{"pro":"S1"},`, 100) + `{"pro":"S9","rate":"4817.25"}`
	history := append(
		runtimetest.ReadTurn(
			"c1",
			"shipment_list",
			toolresult.Fence("shipment_list", payload),
		),
		runtimetest.QuietTurns(turnhistory.RecentToolTurns)...,
	)
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		History:    history,
		Input:      "What was S9's rate?",
	}

	turn := rt.OpenTurn(t.Context(), req)
	require.NotContains(t, strings.Join(replycheck.GroundingTexts(turn), "\n"), "4817.25",
		"the replay no longer carries the figure")

	want := decimal.RequireFromString("4817.25")
	assert.True(t, containsDecimal(replycheck.GroundingSupport(turn), want))

	restored := rt.RestoreTurn(req, new(turn.State()))
	assert.True(
		t,
		containsDecimal(replycheck.GroundingSupport(restored), want),
		"workflow turns keep it too",
	)
}

func containsDecimal(values []decimal.Decimal, want decimal.Decimal) bool {
	for _, value := range values {
		if value.Equal(want) {
			return true
		}
	}

	return false
}
