package ratinghydration_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratinghydration"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func realSchema(t *testing.T) []byte {
	t.Helper()

	raw, err := os.ReadFile("../formula/schema/definitions/shipment.schema.json")
	require.NoError(t, err)
	return raw
}

func environmentFor(t *testing.T, schemaJSON []byte) *environment.Builder {
	t.Helper()

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register("shipment", schemaJSON))

	return environment.NewBuilder(environment.Params{
		Registry: registry,
		Resolver: resolver.NewResolver(),
	})
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestNew_CoversEveryRelationTheShipmentSchemaReads(t *testing.T) {
	t.Parallel()

	env := environmentFor(t, realSchema(t))
	hydrator, err := ratinghydration.New(
		ratingtest.HydratorParams(env, ratingtest.NewRelationStore()),
	)
	require.NoError(t, err)

	inline := map[string]bool{
		"Moves": true, "Moves.Stops": true, "Commodities": true, "AdditionalCharges": true,
	}
	loaded := hydrator.Relations()
	for _, relation := range env.RequiredPreloads("shipment") {
		if inline[relation] {
			continue
		}
		assert.Contains(t, loaded, relation, "the hydrator must load %s", relation)
	}
	assert.Less(
		t,
		slices.Index(loaded, "Moves.Stops.Location"),
		slices.Index(loaded, "Moves.Stops.Location.State"),
		"a relation loads before the relations under it",
	)
}

func TestNew_RefusesASchemaRelationItCannotLoad(t *testing.T) {
	t.Parallel()

	schemaJSON := strings.Replace(
		string(realSchema(t)),
		`"properties": {`,
		`"properties": {
        "billToName": {
            "type": "string",
            "description": "The bill-to customer's name",
            "x-source": { "path": "BillToCustomer.Name" }
        },`,
		1,
	)

	_, err := ratinghydration.New(
		ratingtest.HydratorParams(
			environmentFor(t, []byte(schemaJSON)),
			ratingtest.NewRelationStore(),
		),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BillToCustomer")
}

func TestHydrateShipment_LoadsEveryMissingRelationOneQueryPerKind(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	tx := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "TX"}
	ca := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "CA"}
	w.States[tx.ID], w.States[ca.ID] = tx, ca

	dallas := &location.Location{ID: pulid.MustNew("loc_"), City: "Dallas", StateID: tx.ID}
	fresno := &location.Location{ID: pulid.MustNew("loc_"), City: "Fresno", StateID: ca.ID}
	w.Locations[dallas.ID], w.Locations[fresno.ID] = dallas, fresno

	cust := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME"}
	w.Customers[cust.ID] = cust
	svc := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	w.ServiceTypes[svc.ID] = svc
	tractor := &equipmenttype.EquipmentType{ID: pulid.MustNew("et_"), Code: "DC"}
	trailer := &equipmenttype.EquipmentType{ID: pulid.MustNew("et_"), Code: "DV"}
	w.Equipment[tractor.ID], w.Equipment[trailer.ID] = tractor, trailer

	hazmat := &hazardousmaterial.HazardousMaterial{ID: pulid.MustNew("hm_"), Name: "Paint"}
	w.Hazmats[hazmat.ID] = hazmat
	paint := &commodity.Commodity{
		ID:                  pulid.MustNew("com_"),
		Name:                "Paint",
		HazardousMaterialID: hazmat.ID,
	}
	w.Commodities[paint.ID] = paint

	entity := &shipment.Shipment{
		CustomerID:    cust.ID,
		ServiceTypeID: svc.ID,
		TractorTypeID: tractor.ID,
		TrailerTypeID: trailer.ID,
		Moves: []*shipment.ShipmentMove{
			{Stops: []*shipment.Stop{{LocationID: dallas.ID}, {LocationID: fresno.ID}}},
			{Stops: []*shipment.Stop{{LocationID: fresno.ID}, nil, {LocationID: dallas.ID}}},
		},
		Commodities: []*shipment.ShipmentCommodity{{CommodityID: paint.ID}},
	}

	err := ratingtest.NewHydrator(t, w).
		HydrateShipment(t.Context(), &ratequoteports.HydrateShipmentRequest{
			Shipment:   entity,
			TenantInfo: tenant(),
		})
	require.NoError(t, err)

	require.NotNil(t, entity.Customer)
	assert.Equal(t, "ACME", entity.Customer.Code)
	require.NotNil(t, entity.ServiceType)
	assert.Equal(t, "EXP", entity.ServiceType.Code)
	require.NotNil(t, entity.TractorType)
	assert.Equal(t, "DC", entity.TractorType.Code)
	require.NotNil(t, entity.TrailerType)
	assert.Equal(t, "DV", entity.TrailerType.Code)
	assert.Nil(t, entity.ShipmentType)

	for _, move := range entity.Moves {
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			require.NotNil(t, stop.Location)
			require.NotNil(t, stop.Location.State)
		}
	}
	assert.Equal(t, "TX", entity.Moves[0].Stops[0].Location.State.Abbreviation)
	assert.Equal(t, "CA", entity.Moves[0].Stops[1].Location.State.Abbreviation)

	require.NotNil(t, entity.Commodities[0].Commodity)
	require.NotNil(t, entity.Commodities[0].Commodity.HazardousMaterial)
	assert.Equal(t, "Paint", entity.Commodities[0].Commodity.HazardousMaterial.Name)

	require.Len(t, w.Fetches("location"), 1, "one query for every stop's location")
	assert.Len(t, w.Fetches("location")[0], 2, "each location is asked for once")
	require.Len(t, w.Fetches("state"), 1)
	assert.Len(t, w.Fetches("state")[0], 2)
	assert.Len(t, w.Fetches("equipmentType"), 2, "tractor and trailer are separate relations")
	assert.Empty(t, w.Fetches("shipmentType"), "no shipment type id, nothing to load")
}

func TestHydrateShipment_KeepsMatchingRelationsAndReloadsStaleOnes(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	svc := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	w.ServiceTypes[svc.ID] = svc
	cust := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME"}

	entity := &shipment.Shipment{
		CustomerID:    cust.ID,
		Customer:      cust,
		ServiceTypeID: svc.ID,
		ServiceType:   &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "STD"},
		TractorTypeID: pulid.Nil,
		TractorType:   &equipmenttype.EquipmentType{ID: pulid.MustNew("et_"), Code: "DC"},
	}

	err := ratingtest.NewHydrator(t, w).
		HydrateShipment(t.Context(), &ratequoteports.HydrateShipmentRequest{
			Shipment:   entity,
			TenantInfo: tenant(),
		})
	require.NoError(t, err)

	assert.Same(t, cust, entity.Customer, "a loaded relation matching its id is kept")
	assert.Empty(t, w.Fetches("customer"))
	require.NotNil(t, entity.ServiceType)
	assert.Equal(t, "EXP", entity.ServiceType.Code, "a relation whose id changed is reloaded")
	assert.Nil(t, entity.TractorType, "a relation whose id was cleared is dropped")
}

func TestHydrateShipment_RefreshDiscardsEveryLoadedRelation(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	cust := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME"}
	w.Customers[cust.ID] = cust
	state := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "TX"}
	w.States[state.ID] = state
	loc := &location.Location{ID: pulid.MustNew("loc_"), City: "Dallas", StateID: state.ID}
	w.Locations[loc.ID] = loc

	entity := &shipment.Shipment{
		CustomerID: cust.ID,
		Customer:   &customer.Customer{ID: cust.ID, Code: "FORGED"},
		Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{{
			LocationID: loc.ID,
			Location: &location.Location{
				ID:      loc.ID,
				City:    "Elsewhere",
				StateID: state.ID,
				State:   &usstate.UsState{ID: state.ID, Abbreviation: "ZZ"},
			},
		}}}},
	}

	err := ratingtest.NewHydrator(t, w).
		HydrateShipment(t.Context(), &ratequoteports.HydrateShipmentRequest{
			Shipment:   entity,
			TenantInfo: tenant(),
			Refresh:    true,
		})
	require.NoError(t, err)

	assert.Equal(t, "ACME", entity.Customer.Code)
	assert.Equal(t, "Dallas", entity.Moves[0].Stops[0].Location.City)
	assert.Equal(t, "TX", entity.Moves[0].Stops[0].Location.State.Abbreviation)
}

func TestHydrateShipment_UnknownReferenceIsAValidationError(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	missing := pulid.MustNew("svt_")
	entity := &shipment.Shipment{ServiceTypeID: missing}

	err := ratingtest.NewHydrator(t, w).
		HydrateShipment(t.Context(), &ratequoteports.HydrateShipmentRequest{
			Shipment:   entity,
			TenantInfo: tenant(),
		})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "serviceTypeId", validation.Field)
	assert.Contains(t, err.Error(), missing.String())
	assert.Nil(t, entity.ServiceType)
}

func TestHydrateShipment_FallsBackToTheShipmentsTenant(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	svc := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	w.ServiceTypes[svc.ID] = svc
	owner := tenant()

	entity := &shipment.Shipment{
		OrganizationID: owner.OrgID,
		BusinessUnitID: owner.BuID,
		ServiceTypeID:  svc.ID,
	}

	err := ratingtest.NewHydrator(t, w).
		HydrateShipment(t.Context(), &ratequoteports.HydrateShipmentRequest{
			Shipment: entity,
		})
	require.NoError(t, err)
	require.NotNil(t, entity.ServiceType)
	require.Len(t, w.Tenants("serviceType"), 1)
	assert.Equal(t, owner.OrgID, w.Tenants("serviceType")[0].OrgID)
	assert.Equal(t, owner.BuID, w.Tenants("serviceType")[0].BuID)
}

func TestHydrateShipments_LoadsABatchWithOneQueryPerRelationKind(t *testing.T) {
	t.Parallel()

	w := ratingtest.NewRelationStore()
	expedited := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "EXP"}
	standard := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "STD"}
	w.ServiceTypes[expedited.ID], w.ServiceTypes[standard.ID] = expedited, standard

	batch := []*shipment.Shipment{
		{ServiceTypeID: expedited.ID},
		{ServiceTypeID: standard.ID},
		{ServiceTypeID: expedited.ID},
		nil,
	}

	err := ratingtest.NewHydrator(t, w).HydrateShipments(
		t.Context(),
		&ratequoteports.HydrateShipmentsRequest{Shipments: batch, TenantInfo: tenant()},
	)
	require.NoError(t, err)

	require.Len(t, w.Fetches("serviceType"), 1)
	assert.Len(t, w.Fetches("serviceType")[0], 2)
	assert.Equal(t, "EXP", batch[0].ServiceType.Code)
	assert.Equal(t, "STD", batch[1].ServiceType.Code)
	assert.Equal(t, "EXP", batch[2].ServiceType.Code)
}

func TestHydrateShipments_RefusesABatchSpanningOrganizations(t *testing.T) {
	t.Parallel()

	first, second := tenant(), tenant()
	batch := []*shipment.Shipment{
		{OrganizationID: first.OrgID, BusinessUnitID: first.BuID},
		{OrganizationID: second.OrgID, BusinessUnitID: second.BuID},
	}

	err := ratingtest.NewHydrator(t, ratingtest.NewRelationStore()).HydrateShipments(
		t.Context(),
		&ratequoteports.HydrateShipmentsRequest{Shipments: batch},
	)
	require.ErrorIs(t, err, ratinghydration.ErrMixedTenants)
}
