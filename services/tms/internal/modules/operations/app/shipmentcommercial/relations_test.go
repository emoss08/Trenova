package shipmentcommercial

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial/shipmentcommercialtest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
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
		RatingUnit:        1,
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

func (f *relationFixture) template(expression string) *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
		Name:           "relation surcharge",
		SchemaID:       "shipment",
		Expression:     expression,
		Status:         formulatemplate.StatusActive,
	}
}

func (f *relationFixture) calculator(
	t *testing.T,
	template *formulatemplate.FormulaTemplate,
) *Calculator {
	t.Helper()

	formulaService := ratingtest.NewFormulaService(t, template)
	hydrator := ratingtest.NewHydrator(t, f.store)

	return New(Params{
		Logger:     zap.NewNop(),
		RateEngine: ratingtest.NewHydratingFallbackEngine(t, formulaService, hydrator),
		Predicate:  formulaService,
		Hydrator:   hydrator,
	})
}

func TestRating_ReadsRelatedRecordsTheSaveInputOnlyNamesByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
	}{
		{name: "service type", expression: `serviceType.code == "EXP" ? 75 : 0`},
		{name: "origin state", expression: `origin.state == "CA" ? 75 : 0`},
		{name: "destination state", expression: `destination.state == "TX" ? 75 : 0`},
		{name: "customer", expression: `customer.code == "ACME" ? 75 : 0`},
		{name: "tractor type", expression: `tractorType.code == "DC" ? 75 : 0`},
		{
			name:       "friday evening pacific pickup is not a weekend",
			expression: `isWeekendPickup ? 0 : (pickupHour == 18 ? 75 : 0)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newRelationFixture(t)
			template := fixture.template(tt.expression)
			calculator := fixture.calculator(t, template)
			userID := pulid.MustNew("usr_")
			control := &tenant.ShipmentControl{}
			want := decimal.NewFromInt(75)

			created := fixture.shipmentFromInput(template.ID)
			_, err := calculator.RateAndAdoptContract(t.Context(), created, control, userID)
			require.NoError(t, err)
			require.True(t, created.FreightChargeAmount.Valid)
			assert.True(t, want.Equal(created.FreightChargeAmount.Decimal),
				"create priced %s", created.FreightChargeAmount.Decimal)

			updated := fixture.shipmentFromInput(template.ID)
			updated.ID = created.ID
			require.NoError(t, calculator.Recalculate(t.Context(), updated, control, userID))
			assert.True(t, want.Equal(updated.FreightChargeAmount.Decimal),
				"update priced %s", updated.FreightChargeAmount.Decimal)

			require.NoError(t, calculator.Recalculate(t.Context(), updated, control, userID))
			assert.True(t, want.Equal(updated.FreightChargeAmount.Decimal),
				"re-rate priced %s", updated.FreightChargeAmount.Decimal)

			totals, err := calculator.CalculateTotals(
				t.Context(),
				fixture.shipmentFromInput(template.ID),
				control,
				userID,
			)
			require.NoError(t, err)
			assert.True(t, want.Equal(totals.FreightChargeAmount),
				"totals priced %s", totals.FreightChargeAmount)
		})
	}
}

func TestRating_AnAlreadyLoadedShipmentIsNotLoadedAgain(t *testing.T) {
	t.Parallel()

	fixture := newRelationFixture(t)
	template := fixture.template(`serviceType.code == "EXP" ? 75 : 0`)
	calculator := fixture.calculator(t, template)
	entity := fixture.shipmentFromInput(template.ID)

	require.NoError(t, calculator.Recalculate(
		t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_"),
	))
	require.NoError(t, calculator.Recalculate(
		t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_"),
	))

	assert.Len(t, fixture.store.Fetches("serviceType"), 1)
	assert.Len(t, fixture.store.Fetches("location"), 1)
	assert.Len(t, fixture.store.Fetches("state"), 1)
}

func TestRateAndAdoptContract_AccessorialConditionReadsTheServiceType(t *testing.T) {
	t.Parallel()

	fixture := newRelationFixture(t)
	agreement := agreementWithAccessorial(t, decimal.NewFromInt(40))
	agreement.Accessorials[0].ApplyCondition = `serviceType.code == "EXP"`

	template := fixture.template(`100`)
	formulaService := ratingtest.NewFormulaService(t, template)
	calculator := New(Params{
		Logger:        zap.NewNop(),
		RateEngine:    shipmentcommercialtest.StubRateEngineForAgreement(t, 100, &agreement.ID),
		Predicate:     formulaService,
		AgreementRepo: agreementRepoFor(t, agreement),
		Hydrator:      ratingtest.NewHydrator(t, fixture.store),
	})
	calculator.now = func() int64 { return shipmentcommercialtest.RatedAt }

	entity := fixture.shipmentFromInput(template.ID)
	_, err := calculator.RateAndAdoptContract(
		t.Context(),
		entity,
		&tenant.ShipmentControl{},
		pulid.MustNew("usr_"),
	)
	require.NoError(t, err)

	require.Len(t, entity.AdditionalCharges, 1)
	assert.Equal(
		t,
		agreement.Accessorials[0].ID,
		*entity.AdditionalCharges[0].RateAgreementAccessorialID,
	)
	assert.True(t, decimal.NewFromInt(40).Equal(entity.AdditionalCharges[0].Amount))
}
