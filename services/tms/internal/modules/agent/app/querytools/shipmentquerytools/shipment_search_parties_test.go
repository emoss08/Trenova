package shipmentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartyGroups_PrefersWhatAWordNamesWholeOverWhatItOnlyBegins(t *testing.T) {
	t.Parallel()

	freshHaul := pulid.MustNew("cus_")
	parties := &shipmentParties{
		terms:     []string{"freshhaul"},
		customers: []partyMatch{{term: "freshhaul", id: freshHaul, name: "FreshHaul Foods"}},
		commodities: []partyMatch{
			{term: "freshhaul", id: pulid.MustNew("com_"), name: "Fresh Meat"},
			{term: "freshhaul", id: pulid.MustNew("com_"), name: "Fresh Seafood"},
		},
	}

	groups := partyGroups(parties, filtercatalog.NewCriteria("shipments"))

	require.Len(t, groups, 1)
	assert.Equal(t, []pulid.ID{freshHaul}, groups[0].CustomerIDs)
	assert.Empty(t, groups[0].CommodityIDs)
}

func TestPartyGroups_KeepsEveryReadingWhenNoneNamesTheWordWhole(t *testing.T) {
	t.Parallel()

	parties := &shipmentParties{
		terms:       []string{"fres"},
		customers:   []partyMatch{{term: "fres", id: pulid.MustNew("cus_"), name: "FreshHaul Foods"}},
		commodities: []partyMatch{{term: "fres", id: pulid.MustNew("com_"), name: "Fresh Meat"}},
	}

	groups := partyGroups(parties, filtercatalog.NewCriteria("shipments"))

	require.Len(t, groups, 1)
	assert.Len(t, groups[0].CustomerIDs, 1)
	assert.Len(t, groups[0].CommodityIDs, 1)
}
