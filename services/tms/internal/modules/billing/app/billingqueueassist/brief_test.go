package billingqueueassist

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, 0, openCount(&briefFacts{}))
	assert.Equal(t, 32, openCount(briefFixture()))
}
