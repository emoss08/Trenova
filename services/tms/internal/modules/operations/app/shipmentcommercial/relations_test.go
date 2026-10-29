package shipmentcommercial

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type relationFixture struct {
	store         *ratingtest.RelationStore
	orgID         pulid.ID
	buID          pulid.ID
	serviceTypeID pulid.ID
	customerID    pulid.ID
	tractorTypeID pulid.ID
	fresnoID      pulid.ID
	dallasID      pulid.ID
	pickupAt      int64
}

func newRelationFixture(t *testing.T) *relationFixture {
	t.Helper()

	store := ratingtest.NewRelationStore()
	california := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "CA"}
	texas := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "TX"}
	store.States[california.ID] = california
	store.States[texas.ID] = texas

	fresno := &location.Location{
		ID:       pulid.MustNew("loc_"),
		City:     "Fresno",
		StateID:  california.ID,
		Timezone: "America/Los_Angeles",
	}
	dallas := &location.Location{
		ID:       pulid.MustNew("loc_"),
		City:     "Dallas",
		StateID:  texas.ID,
		Timezone: "America/Chicago",
	}
	store.Locations[fresno.ID] = fresno
	store.Locations[dallas.ID] = dallas

	expedited := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	store.ServiceTypes[expedited.ID] = expedited
	acme := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME", Name: "Acme"}
	store.Customers[acme.ID] = acme
	dayCab := &equipmenttype.EquipmentType{ID: pulid.MustNew("et_"), Code: "DC"}
	store.Equipment[dayCab.ID] = dayCab

	pacific, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	fridayEvening := time.Date(2026, time.October, 9, 18, 0, 0, 0, pacific)
	require.Equal(t, time.Friday, fridayEvening.Weekday())
	require.Equal(t, time.Saturday, fridayEvening.UTC().Weekday())

	return &relationFixture{
		store:         store,
		orgID:         pulid.MustNew("org_"),
		buID:          pulid.MustNew("bu_"),
		serviceTypeID: expedited.ID,
		customerID:    acme.ID,
		tractorTypeID: dayCab.ID,
		fresnoID:      fresno.ID,
		dallasID:      dallas.ID,
		pickupAt:      fridayEvening.Unix(),
	}
}

func (f *relationFixture) shipmentFromInput(templateID pulid.ID) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    f.orgID,
		BusinessUnitID:    f.buID,
		FormulaTemplateID: templateID,
		ServiceTypeID:     f.serviceTypeID,
		CustomerID:        f.customerID,
		TractorTypeID:     f.tractorTypeID,
		Moves: []*shipment.ShipmentMove{{
			Status: shipment.MoveStatusNew,
			Stops: []*shipment.Stop{
				{
					Status:               shipment.StopStatusNew,
					Type:                 shipment.StopTypePickup,
					LocationID:           f.fresnoID,
					ScheduledWindowStart: f.pickupAt,
				},
				{
					Status:               shipment.StopStatusNew,
					Type:                 shipment.StopTypeDelivery,
					LocationID:           f.dallasID,
					Sequence:             1,
					ScheduledWindowStart: f.pickupAt + int64(48*time.Hour/time.Second),
				},
			},
		}},
	}
}

func relationTemplate(expression string) *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Name:          "Relation surcharge",
		Source:        "tariff \"Relation surcharge\"\n\ncharge \"Freight\" = " + expression + "\n",
	}
}

func (f *relationFixture) calculator(
	t *testing.T,
	template *tariffports.Template,
) (*Calculator, *ratingtest.Stack) {
	t.Helper()

	hydrator := ratingtest.NewHydrator(t, f.store)
	stack := ratingtest.NewStack(t, ratingtest.StackOptions{
		Templates: []*tariffports.Template{template},
		Hydrator:  hydrator,
	})

	calculator := New(Params{
		Logger:     zap.NewNop(),
		RateEngine: stack.Engine,
		Hydrator:   hydrator,
	})
	return calculator, stack
}

func TestRating_ReadsRelatedRecordsTheSaveInputOnlyNamesByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
	}{
		{
			name:       "service type",
			expression: `if shipment.serviceType.code == "EXP" { $75 } else { $0 }`,
		},
		{
			name:       "origin state",
			expression: `if lane.origin.state == "CA" { $75 } else { $0 }`,
		},
		{
			name:       "destination state",
			expression: `if lane.destination.state == "TX" { $75 } else { $0 }`,
		},
		{
			name:       "customer",
			expression: `if customer.code == "ACME" { $75 } else { $0 }`,
		},
		{
			name: "tractor type",
			expression: `if let tractor = shipment.tractorType { ` +
				`if tractor.code == "DC" { $75 } else { $0 } } else { $0 }`,
		},
		{
			name: "friday evening pacific pickup is not a weekend",
			expression: `if let pickup = shipment.pickup { ` +
				`if let start = pickup.window.start { ` +
				`if start.weekend { $0 } else if start.hour == 18 { $75 } else { $0 } ` +
				`} else { $0 } } else { $0 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRelationFixture(t)
			template := relationTemplate(tt.expression)
			calculator, stack := fixture.calculator(t, template)
			userID := pulid.MustNew("usr_")
			control := &shipmentcontrol.ShipmentControl{}
			want := decimal.NewFromInt(75)

			created := fixture.shipmentFromInput(template.TemplateID)
			_, err := calculator.RateAndAdoptContract(t.Context(), created, control, userID)
			require.NoError(t, err)
			require.True(t, created.FreightChargeAmount.Valid)
			assert.True(t, want.Equal(created.FreightChargeAmount.Decimal),
				"create priced %s", created.FreightChargeAmount.Decimal)

			updated := fixture.shipmentFromInput(template.TemplateID)
			updated.ID = created.ID
			require.NoError(t, calculator.Recalculate(t.Context(), updated, control, userID))
			assert.True(t, want.Equal(updated.FreightChargeAmount.Decimal),
				"update priced %s", updated.FreightChargeAmount.Decimal)

			require.NoError(t, calculator.Recalculate(t.Context(), updated, control, userID))
			assert.True(t, want.Equal(updated.FreightChargeAmount.Decimal),
				"re-rate priced %s", updated.FreightChargeAmount.Decimal)

			totals, err := calculator.CalculateTotals(
				t.Context(),
				fixture.shipmentFromInput(template.TemplateID),
				control,
				userID,
			)
			require.NoError(t, err)
			assert.True(t, want.Equal(totals.FreightChargeAmount),
				"totals priced %s", totals.FreightChargeAmount)
			assert.Positive(t, stack.Evaluations())
		})
	}
}

func TestRating_AnAlreadyLoadedShipmentIsNotLoadedAgain(t *testing.T) {
	t.Parallel()

	fixture := newRelationFixture(t)
	template := relationTemplate(`if shipment.serviceType.code == "EXP" { $75 } else { $0 }`)
	calculator, _ := fixture.calculator(t, template)
	entity := fixture.shipmentFromInput(template.TemplateID)

	require.NoError(t, calculator.Recalculate(
		t.Context(), entity, &shipmentcontrol.ShipmentControl{}, pulid.MustNew("usr_"),
	))
	require.NoError(t, calculator.Recalculate(
		t.Context(), entity, &shipmentcontrol.ShipmentControl{}, pulid.MustNew("usr_"),
	))

	assert.Len(t, fixture.store.Fetches("serviceType"), 1)
	assert.Len(t, fixture.store.Fetches("location"), 1)
	assert.Len(t, fixture.store.Fetches("state"), 1)
}
