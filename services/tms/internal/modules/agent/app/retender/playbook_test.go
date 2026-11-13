package retender

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
)

// The person is shown which guide the move goes down and the carriers in
// the order the guide asks them, whatever order the offers were read in.
func TestSummaryNamesTheGuideAndTheCarriersInOrder(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("car_"), pulid.MustNew("car_")
	planned := &tendercreate.TenderPreview{
		Guide: &tender.RoutingGuide{Name: "Midwest dry van"},
		Tender: &tender.Tender{Offers: []*tender.TenderOffer{
			{CarrierID: second, Rank: 2},
			{CarrierID: first, Rank: 1},
		}},
		Carriers: map[pulid.ID]string{first: "Acme Freight", second: "Northline Carriers"},
		Screening: &tendercreate.GuideScreeningSummary{
			Skipped: []tendercreate.GuideEntryScreeningResult{{CarrierName: "Lapsed Lines"}},
		},
	}

	carriers := carriersInOrder(planned)
	assert.Equal(t, []string{"Acme Freight", "Northline Carriers"}, carriers)
	assert.Equal(t,
		"Offers the move down routing guide Midwest dry van, one carrier at a time in this order: "+
			"Acme Freight, Northline Carriers. Each has the guide's time to accept before the next "+
			"is asked. Left out as ineligible: Lapsed Lines.",
		summary(planned, carriers))
	assert.Equal(t, "routing guide Midwest dry van", guideName(planned))
}

// A guide with nobody left to ask offers no carriers, which the playbook
// turns into a reason instead of an empty tender.
func TestCarriersInOrderIsEmptyWhenTheGuideHasNobodyLeft(t *testing.T) {
	t.Parallel()

	assert.Empty(t, carriersInOrder(&tendercreate.TenderPreview{}))
	assert.Empty(t, carriersInOrder(&tendercreate.TenderPreview{Tender: &tender.Tender{}}))
	assert.Equal(t, "its routing guide", guideName(&tendercreate.TenderPreview{}))
}
