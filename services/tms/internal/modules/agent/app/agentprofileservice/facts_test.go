package agentprofileservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func factText(facts agentprofile.Facts, key string) string {
	for _, fact := range facts {
		if fact.Key == key {
			return fact.Text
		}
	}

	return ""
}

func sampleActivity() *agentprofileports.OrganizationActivity {
	return &agentprofileports.OrganizationActivity{
		Volume: agentprofileports.ShipmentVolume{
			Total: 1234, Canceled: 37, Moves: 1300, Brokered: 520, Customers: 118,
		},
		TopCustomers: []agentprofileports.NamedCount{
			{Name: "Northwind Foods", Count: 410},
			{Name: "Contoso Retail", Count: 180},
		},
		TopLanes: []agentprofileports.LaneCount{
			{Origin: "TX", Destination: "CA", Count: 240},
		},
		TopPickupLocations: []agentprofileports.NamedCount{{Name: "Northwind DC, Dallas TX", Count: 300}},
		Tractors: agentprofileports.FleetCounts{
			Active: 46, OutOfService: 3,
			ByType: []agentprofileports.NamedCount{{Name: "Sleeper", Count: 30}, {Name: "Day cab", Count: 16}},
		},
		Workers: agentprofileports.WorkforceCounts{
			Active: 52, Contractors: 8,
			ByDriverType: []agentprofileports.NamedCount{{Name: "OTR", Count: 30}, {Name: "Regional", Count: 22}},
		},
		CarriersUsed:          63,
		Integrations:          []string{"PCMiler", "Samsara"},
		AccountingConnections: []agentprofileports.NamedStatus{{Name: "QuickBooksOnline", Status: "Degraded"}},
		EDIPartners:           1,
	}
}

func TestOrganizationFacts_SayTheWorkInRoundedWords(t *testing.T) {
	t.Parallel()

	facts := OrganizationFacts(&OrganizationInputs{
		Organization: &tenant.Organization{
			AssetOperationsEnabled: true,
			BrokerageEnabled:       true,
			City:                   "Dallas",
			Timezone:               "America/Chicago",
			State:                  &usstate.UsState{Abbreviation: "TX"},
		},
		Activity:        sampleActivity(),
		DispatchControl: &dispatchcontrol.DispatchControl{EnforceHOSCompliance: true},
		WindowDays:      90,
	})

	assert.Equal(t,
		"Runs its own fleet and also brokers freight to outside carriers.",
		factText(facts, "organization.operations"))
	assert.Equal(t, "Based in Dallas, TX; keeps time in America/Chicago.",
		factText(facts, "organization.home"))
	assert.Equal(t,
		"Entered about 1,200 shipments in the last 90 days (about 100 a week). About 5% were "+
			"canceled. About 120 customers shipped.",
		factText(facts, "shipments.volume"))
	assert.Equal(t,
		"About 40% of moves were covered by outside carriers, using about 65 carriers.",
		factText(facts, "shipments.coverage"))
	assert.Equal(t,
		"Busiest customers by shipments: Northwind Foods (about 35%), Contoso Retail (about 15%).",
		factText(facts, "customers.top"))
	assert.Equal(t, "Busiest lanes, state to state: TX to CA (about 20%).", factText(facts, "lanes.top"))
	assert.Equal(t,
		"About 45 tractors, 3 of them out of service or in the shop; mostly Sleeper about 65%, Day cab about 35%.",
		factText(facts, "fleet.tractors"))
	assert.Empty(t, factText(facts, "fleet.trailers"), "nothing to say is left out")
	assert.Equal(t, "Accounting system: QuickBooksOnline (degraded).",
		factText(facts, "integrations.accounting"))
	assert.Equal(t, "Exchanges EDI with one trading partner.", factText(facts, "integrations.edi"))
	assert.Contains(t, factText(facts, "settings.dispatch"), "hours of service enforced")
}

func TestOrganizationFacts_ASmallMoveInTheWorkKeepsTheSameWords(t *testing.T) {
	t.Parallel()

	before := sampleActivity()
	after := sampleActivity()
	after.Volume.Total += 9
	after.TopCustomers[0].Count += 3
	after.CarriersUsed++

	org := &tenant.Organization{AssetOperationsEnabled: true}
	a := OrganizationFacts(&OrganizationInputs{Organization: org, Activity: before})
	b := OrganizationFacts(&OrganizationInputs{Organization: org, Activity: after})

	assert.Equal(t, Digest(a, nil), Digest(b, nil),
		"a profile is written again when the work changes, not every night")
}

func TestUserFacts_NeedWorkToSayAnything(t *testing.T) {
	t.Parallel()

	quiet := UserFacts(&agentprofileports.UserActivity{JobTitle: "Dispatcher"}, 30)
	assert.False(t, HasActivity(quiet), "a title alone is not a profile")

	busy := UserFacts(&agentprofileports.UserActivity{
		JobTitle:       "Dispatcher",
		OwnedShipments: []agentprofileports.NamedCount{{Name: "InTransit", Count: 12}},
		Customers:      []agentprofileports.NamedCount{{Name: "Northwind Foods", Count: 9}},
		ChangedRecords: []agentprofileports.NamedCount{{Name: "shipment", Count: 140}},
		ToolsUsed:      []agentprofileports.NamedCount{{Name: "dispatch_assign_move", Count: 30}},
	}, 30)
	require.True(t, HasActivity(busy))
	assert.Equal(t, "Job title: Dispatcher.", factText(busy, "person.title"))
	assert.Equal(t, "Owns about 10 shipments entered in the last 30 days (InTransit about 100%).",
		factText(busy, "work.shipments.owned"))
	assert.Equal(t, "Changes these kinds of records most, over the last 30 days: shipment (140).",
		factText(busy, "work.records"))
}
