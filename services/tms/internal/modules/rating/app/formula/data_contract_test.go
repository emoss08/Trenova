package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func evaluate(
	t *testing.T,
	svc *formula.Service,
	entity *shipment.Shipment,
	expression string,
) decimal.Decimal {
	t.Helper()
	return evaluateWith(t, svc, entity, expression, nil)
}

func evaluateWith(
	t *testing.T,
	svc *formula.Service,
	entity *shipment.Shipment,
	expression string,
	variables map[string]any,
) decimal.Decimal {
	t.Helper()

	preview, err := svc.Preview(t.Context(), &formula.PreviewRequest{
		Expression: expression,
		Entity:     entity,
		SchemaID:   "shipment",
		Variables:  variables,
		TenantInfo: testTenant(),
	})
	require.NoError(t, err, expression)
	for _, diagnostic := range preview.Diagnostics {
		require.False(t, diagnostic.IsError(), "%s: %s", expression, diagnostic.Message)
	}
	require.NotNil(t, preview.Evaluated, expression)
	require.Nil(t, preview.Error, "%s: %v", expression, preview.Error)
	return preview.Evaluated.Amount
}

func assertAmount(t *testing.T, want string, got decimal.Decimal, expression string) {
	t.Helper()
	assert.True(
		t,
		decimal.RequireFromString(want).Equal(got),
		"%s: want %s, got %s",
		expression,
		want,
		got,
	)
}

func ratedShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ServiceType: &servicetype.ServiceType{Code: "STD"},
		RatingUnit:  1,
	}
}

func commodityLine(class commodity.FreightClass, weight, pieces int64) *shipment.ShipmentCommodity {
	return &shipment.ShipmentCommodity{
		CommodityID: pulid.MustNew("com_"),
		Weight:      weight,
		Pieces:      pieces,
		Commodity: &commodity.Commodity{
			Name:         string(class),
			FreightClass: class,
		},
	}
}

func TestDataContract_FreightClassesReadAsTheirNumbers(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	entity := ratedShipment()
	entity.Commodities = []*shipment.ShipmentCommodity{
		commodityLine(commodity.FreightClass85, 200, 4),
		commodityLine(commodity.FreightClass100, 900, 2),
		commodityLine(commodity.FreightClass92_5, 300, 1),
	}

	cases := map[string]string{
		`highestFreightClass == "100" ? 1 : 0`:              "1",
		`primaryFreightClass == "100" ? 1 : 0`:              "1",
		`any(commodities, .freightClass == "85") ? 1 : 0`:   "1",
		`any(commodities, .freightClass == "92.5") ? 1 : 0`: "1",
	}
	for expression, want := range cases {
		assertAmount(t, want, evaluate(t, svc, entity, expression), expression)
	}
}

func TestDataContract_FreightClassLooksUpAClassKeyedTable(t *testing.T) {
	t.Parallel()

	svc := setupServiceWithMatrixData(t, freightClassMatrix())
	entity := ratedShipment()
	entity.Commodities = []*shipment.ShipmentCommodity{
		commodityLine(commodity.FreightClass85, 200, 4),
		commodityLine(commodity.FreightClass100, 900, 2),
	}

	expression := `lookup("class_rates", highestFreightClass)`
	assertAmount(t, "31.5", evaluate(t, svc, entity, expression), expression)
}

func stopAt(
	stopType shipment.StopType,
	status shipment.StopStatus,
	sequence int64,
	city string,
	window int64,
) *shipment.Stop {
	windowEnd := window + 3600
	return &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Type:                 stopType,
		Status:               status,
		Sequence:             sequence,
		ScheduledWindowStart: window,
		ScheduledWindowEnd:   &windowEnd,
		Location: &location.Location{
			City:  city,
			State: &usstate.UsState{Abbreviation: "TX"},
		},
	}
}

func TestDataContract_StopsAreReadInSequenceAndCanceledStopsDoNotCount(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	pickup := stopAt(shipment.StopTypePickup, shipment.StopStatusNew, 0, "Dallas", 1_800_000_000)
	canceled := stopAt(
		shipment.StopTypeDelivery,
		shipment.StopStatusCanceled,
		1,
		"Waco",
		1_800_010_000,
	)
	final := stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 2, "Austin", 1_800_020_000)

	entity := ratedShipment()
	entity.Moves = []*shipment.ShipmentMove{{
		Sequence: 0,
		Stops:    []*shipment.Stop{final, pickup, canceled},
	}}

	cases := map[string]string{
		`totalStops`:                                 "2",
		`origin.city == "Dallas" ? 1 : 0`:            "1",
		`destination.city == "Austin" ? 1 : 0`:       "1",
		`stops[0].city == "Dallas" ? 1 : 0`:          "1",
		`stops[1].status == "Canceled" ? 1 : 0`:      "1",
		`len(stops)`:                                 "3",
		`totalStops > 2 ? (totalStops - 2) * 50 : 0`: "0",
	}
	for expression, want := range cases {
		assertAmount(t, want, evaluate(t, svc, entity, expression), expression)
	}
}

func TestDataContract_DeliveryDateFallsBackToTheScheduledWindowStart(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	pickup := stopAt(shipment.StopTypePickup, shipment.StopStatusNew, 0, "Dallas", 1_800_000_000)
	delivery := stopAt(
		shipment.StopTypeDelivery,
		shipment.StopStatusNew,
		1,
		"Austin",
		1_800_020_000,
	)
	delivery.ScheduledWindowEnd = nil

	entity := ratedShipment()
	entity.Moves = []*shipment.ShipmentMove{{Stops: []*shipment.Stop{pickup, delivery}}}

	assertAmount(
		t,
		"1",
		evaluate(t, svc, entity, `deliveryDate == nil ? 0 : 1`),
		"deliveryDate",
	)
}

func TestDataContract_PointerFieldsReachTheFormulaAsNumbers(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	pieces := int64(10)
	entity := ratedShipment()
	entity.Pieces = &pieces

	cases := map[string]string{
		`pieces * 2`:                 "20",
		`max(pieces, 1)`:             "10",
		`pieces == 10 ? 1 : 0`:       "1",
		`(pieces ?? 0) + 1`:          "11",
		`pieces in [10, 20] ? 1 : 0`: "1",
	}
	for expression, want := range cases {
		assertAmount(t, want, evaluate(t, svc, entity, expression), expression)
	}
}

func TestDataContract_AnAbsentOptionalRelationIsNil(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	without := ratedShipment()
	with := ratedShipment()
	with.TractorType = &equipmenttype.EquipmentType{Code: "DC"}

	expression := `tractorType == nil ? 0 : 50`
	assertAmount(t, "0", evaluate(t, svc, without, expression), expression)
	assertAmount(t, "50", evaluate(t, svc, with, expression), expression)

	guarded := `(tractorType?.code ?? "") == "DC" ? 50 : 0`
	assertAmount(t, "0", evaluate(t, svc, without, guarded), guarded)
	assertAmount(t, "50", evaluate(t, svc, with, guarded), guarded)
}

func TestDataContract_UnguardedOptionalRelationMemberIsWarned(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	checked, err := svc.Check(t.Context(), &formula.CheckRequest{
		SchemaID: "shipment",
		Expect:   rules.ExpectAmount,
		Sources: []formula.CheckSource{{
			Scope:  formula.MainScope,
			Source: `hasPrefix(tractorType.code, "DC") ? 1 : 0`,
		}},
	})
	require.NoError(t, err)
	require.Len(t, checked, 1)
	assert.True(t, checked[0].Check.Accepted())
	assert.Equal(t, []string{"tractorType.code"}, checked[0].Check.Unguarded())

	diagnostic := checked[0].Check.Diagnostics[0]
	require.NotEmpty(t, diagnostic.Fixes)
	require.NotEmpty(t, diagnostic.Fixes[0].Changes)
	assert.Equal(t, `(tractorType?.code ?? "")`, diagnostic.Fixes[0].Changes[0].Insert)
}

func TestDataContract_AggregatesAgreeWithTheShipment(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	headerWeight := int64(5_000)
	headerPieces := int64(40)
	length, width, height := 40.0, 8.0, 8.0
	entity := ratedShipment()
	entity.Weight = &headerWeight
	entity.Pieces = &headerPieces
	entity.EnvelopeLengthFeet = &length
	entity.EnvelopeWidthFeet = &width
	entity.EnvelopeHeightFeet = &height
	paint := commodityLine(commodity.FreightClass85, 1_200, 6)
	paint.Commodity.HazardousMaterialID = pulid.MustNew("hm_")
	entity.Commodities = []*shipment.ShipmentCommodity{
		paint,
		commodityLine(commodity.FreightClass100, 1_360, 4),
	}

	assertAmount(t, "2560", evaluate(t, svc, entity, `totalWeight`), "totalWeight")
	assertAmount(t, "10", evaluate(t, svc, entity, `totalPieces`), "totalPieces")
	assertAmount(t, "2560", evaluate(t, svc, entity, `totalCubicFeet`), "totalCubicFeet")
	assertAmount(t, "1", evaluate(t, svc, entity, `density`), "density")
	assertAmount(t, "1", evaluate(t, svc, entity, `hasHazmat ? 1 : 0`), "hasHazmat")

	withHazmatRecord := ratedShipment()
	line := commodityLine(commodity.FreightClass85, 100, 1)
	line.Commodity.HazardousMaterial = &hazardousmaterial.HazardousMaterial{Name: "Paint"}
	withHazmatRecord.Commodities = []*shipment.ShipmentCommodity{line}
	assertAmount(
		t,
		"1",
		evaluate(t, svc, withHazmatRecord, `hasHazmat ? 1 : 0`),
		"hasHazmat from the loaded record",
	)
}

func freightClassMatrix() []*ratematrixports.RateMatrixLookupData {
	table := exactLookupMatrix("class_rates", map[string]string{
		string(commodity.FreightClass85):  "20",
		string(commodity.FreightClass100): "31.5",
	})
	table.Matrix.Dimensions[0].Kind = ratematrix.DimensionKindFreightClass
	return []*ratematrixports.RateMatrixLookupData{table}
}
