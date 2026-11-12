package billingqueueassist

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func briefFixture() *briefFacts {
	return &briefFacts{
		Lanes: []*billingqueueports.BillingQueueLaneStat{
			{Lane: billingqueue.LaneReady, Count: 21, Total: decimal.RequireFromString("104880")},
			{
				Lane:  billingqueue.LaneNeedsYou,
				Count: 11,
				Total: decimal.RequireFromString("58302.50"),
			},
		},
		Concerns: []*billingqueueports.AgentConcern{
			{Shipment: "S-24129", Summary: "Fuel is 20.6% of linehaul"},
		},
	}
}

func TestABriefMayOnlyUseTheFiguresItWasGiven(t *testing.T) {
	t.Parallel()

	allowed := allowedNumbers(briefFixture())
	good := "21 are ready to approve, $104,880.00 in all, and 11 need you ($58,302.50). Fuel on S-24129 is 20.6% of linehaul."
	assert.Empty(t, unknownNumbers(good, allowed))
	assert.Empty(t, unknownNumbers("About $104,880 is ready.", allowed))

	bad := "32 items are waiting, $163,182.50 in all."
	assert.Equal(t, []string{"32", "$163,182.50"}, unknownNumbers(bad, allowed))
}

func TestABriefIsToldTheQueueInTheBillersWordsWithEachShipmentByItsNumber(t *testing.T) {
	t.Parallel()

	facts := briefFixture()
	facts.Issues = []*billingqueueports.OpenIssueCount{{Code: billingqueue.IssuePODUnsigned, Count: 3}}
	facts.Batches = []*billingqueueports.Recommendation{{
		Kind:  billingqueueports.RecommendApproveReady,
		Count: 4,
		Total: decimal.RequireFromString("16735.08"),
	}}

	assert.Equal(t, "Ready to approve: 21 items, $104,880.00\n"+
		"Needs the biller: 11 items, $58,302.50\n"+
		"An unsigned proof of delivery: 3 items\n"+
		"Can be done in one step, approve every item that is ready: 4 items, $16,735.08\n"+
		"You questioned shipment S-24129: Fuel is 20.6% of linehaul", briefLines(facts))
}

func TestAnEmptyQueueNeedsNoModel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, laneCount(nil))
	assert.Equal(t, 32, laneCount(briefFixture().Lanes))
}

func storedBrief(t *testing.T, lanes []*billingqueueports.BillingQueueLaneStat) *billingqueue.QueueBrief {
	t.Helper()

	hash, err := lanesHash(lanes)
	require.NoError(t, err)

	return &billingqueue.QueueBrief{
		Text:        "21 are ready to approve and 11 need you.",
		LanesHash:   hash,
		GeneratedAt: 1_000,
	}
}

func lanesNow(t *testing.T, lanes []*billingqueueports.BillingQueueLaneStat) string {
	t.Helper()

	hash, err := lanesHash(lanes)
	require.NoError(t, err)

	return hash
}

func TestABriefWrittenForTheseLanesIsShownAsWritten(t *testing.T) {
	t.Parallel()

	lanes := briefFixture().Lanes
	view, current := briefView(storedBrief(t, lanes), lanes, lanesNow(t, lanes), 5_000)

	assert.True(t, current)
	assert.Equal(t, &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        "21 are ready to approve and 11 need you.",
		GeneratedAt: 1_000,
	}, view)
}

func TestAnEmptiedQueueSaysNothingIsWaitingWhateverWasStored(t *testing.T) {
	t.Parallel()

	view, current := briefView(storedBrief(t, briefFixture().Lanes), nil, lanesNow(t, nil), 5_000)

	assert.False(t, current, "the stored brief is rewritten for the empty queue")
	assert.Equal(t, &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        "Nothing is waiting to be billed.",
		GeneratedAt: 1_000,
	}, view)
}

func TestAnEmptyQueueWithNoBriefSaysNothingIsWaitingNow(t *testing.T) {
	t.Parallel()

	view, current := briefView(nil, nil, lanesNow(t, nil), 5_000)

	assert.False(t, current)
	assert.Equal(t, &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        "Nothing is waiting to be billed.",
		GeneratedAt: 5_000,
	}, view)
}

func TestABriefWrittenForOtherLanesIsShownStale(t *testing.T) {
	t.Parallel()

	moved := []*billingqueueports.BillingQueueLaneStat{
		{Lane: billingqueue.LaneReady, Count: 4, Total: decimal.RequireFromString("16735.08")},
	}
	view, current := briefView(storedBrief(t, briefFixture().Lanes), moved, lanesNow(t, moved), 5_000)

	assert.False(t, current)
	assert.Equal(t, &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        "21 are ready to approve and 11 need you.",
		GeneratedAt: 1_000,
		Stale:       true,
	}, view)
}

func TestAQueueWithItemsAndNoBriefHasNoTextYet(t *testing.T) {
	t.Parallel()

	lanes := briefFixture().Lanes
	view, current := briefView(nil, lanes, lanesNow(t, lanes), 5_000)

	assert.False(t, current)
	assert.Equal(t, &billingqueueports.QueueBriefView{Available: true}, view)
}

func TestADraftThatKeepsToItsFiguresIsTheBrief(t *testing.T) {
	t.Parallel()

	allowed := allowedNumbers(briefFixture())

	assert.Empty(t, draftProblem("21 are ready to approve and 11 need you.", allowed))
}

func TestADraftIsSentBackWithWhatWasWrong(t *testing.T) {
	t.Parallel()

	allowed := allowedNumbers(briefFixture())

	assert.Equal(t, "The draft was empty. Write two to four sentences.", draftProblem("", allowed))
	assert.Equal(t,
		"The draft was too long. Keep it under 700 characters.",
		draftProblem(strings.Repeat("a", 701), allowed),
	)
	assert.Equal(t,
		"The draft used figures you were not given: 32. Use only the figures listed, "+
			"written exactly as given; never count, add or compute one.",
		draftProblem("32 items are waiting.", allowed),
	)
}

func TestABriefNovaCannotWriteIsWrittenFromTheFigures(t *testing.T) {
	t.Parallel()

	facts := briefFixture()
	facts.Lanes = append(facts.Lanes, &billingqueueports.BillingQueueLaneStat{
		Lane:  billingqueue.LaneHeld,
		Count: 1,
		Total: decimal.RequireFromString("450"),
	})

	assert.Equal(t,
		"Ready to approve: 21 items, $104,880.00. Needs the biller: 11 items, $58,302.50. "+
			"On hold: 1 item, $450.00.",
		factsBrief(facts),
	)
}
