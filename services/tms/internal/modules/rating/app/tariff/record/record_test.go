package record_test

import (
	"sort"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type schemaField struct {
	Name   string        `json:"name"`
	Fields []schemaField `json:"fields"`
}

type schemaRoot struct {
	Name   string        `json:"name"`
	Fields []schemaField `json:"fields"`
}

type catalog struct {
	Schema struct {
		Roots []schemaRoot `json:"roots"`
	} `json:"schema"`
}

func fullShipment() *shipment.Shipment {
	texas := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "TX", CountryIso3: "USA"}
	illinois := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "IL", CountryIso3: "USA"}
	arrived := time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC).Unix()
	departed := arrived + 3*3600
	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "PRO-1",
		Status:    shipment.StatusInTransit,
		Weight:    new(int64(12000)),
		Pieces:    new(int64(10)),
		Moves: []*shipment.ShipmentMove{{
			Loaded:   true,
			Distance: new(812.0),
			Stops: []*shipment.Stop{
				{
					Type:            shipment.StopTypePickup,
					Status:          shipment.StopStatusCompleted,
					ActualArrival:   &arrived,
					ActualDeparture: &departed,
					Location: &location.Location{
						City:       "Dallas",
						PostalCode: "75201",
						State:      texas,
						Timezone:   "America/Chicago",
					},
				},
				{
					Type:   shipment.StopTypeDelivery,
					Status: shipment.StopStatusNew,
					Location: &location.Location{
						City:       "Chicago",
						PostalCode: "60601",
						State:      illinois,
					},
				},
			},
		}},
		Commodities: []*shipment.ShipmentCommodity{{
			Pieces:     10,
			Weight:     12000,
			LengthFeet: new(4.0),
			WidthFeet:  new(4.0),
			HeightFeet: new(5.0),
			Commodity:  &commodity.Commodity{Name: "Paint", FreightClass: commodity.FreightClass("Class70")},
		}},
	}
}

func buildRecord(t *testing.T) map[string]any {
	t.Helper()

	entity := fullShipment()
	raw, err := record.Build(&record.Input{
		Shipment:     entity,
		Origin:       record.PlaceOf(entity.ShipperStop()),
		Destination:  record.PlaceOf(entity.DeliveryStop()),
		DistanceUnit: record.DistanceMiles,
		Fuel: record.Fuel{
			Price:     decimal.NewNullDecimal(decimal.RequireFromString("3.85")),
			PriceDate: "2026-03-02",
			Index:     "DOE",
		},
		Rating: record.Rating{
			At:        time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			Mode:      record.ModeBuy,
			SellTotal: decimal.NewNullDecimal(decimal.NewFromInt(2500)),
			Currency:  "USD",
		},
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &decoded))
	return decoded
}

func keysOf(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func compare(t *testing.T, path string, fields []schemaField, value any) {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		list, isList := value.([]any)
		if isList && len(list) > 0 {
			compare(t, path+"[0]", fields, list[0])
		}
		return
	}

	want := make([]string, 0, len(fields))
	for _, field := range fields {
		want = append(want, field.Name)
	}
	sort.Strings(want)
	assert.Equal(t, want, keysOf(object), "the record's %s fields differ from the schema", path)

	for _, field := range fields {
		if len(field.Fields) == 0 {
			continue
		}
		if nested, present := object[field.Name]; present && nested != nil {
			compare(t, path+"."+field.Name, field.Fields, nested)
		}
	}
}

func TestBuildMatchesTheModulesSchemaFieldForField(t *testing.T) {
	t.Parallel()

	raw, err := rules.TariffCatalog(t.Context())
	require.NoError(t, err)
	var published catalog
	require.NoError(t, sonic.Unmarshal(raw, &published))

	built := buildRecord(t)
	roots := make([]string, 0, len(published.Schema.Roots))
	for _, root := range published.Schema.Roots {
		if root.Name == "tables" || root.Name == "bill" {
			continue
		}
		roots = append(roots, root.Name)
		value, ok := built[root.Name]
		require.Truef(t, ok, "the record has no %s root", root.Name)
		compare(t, root.Name, root.Fields, value)
	}
	sort.Strings(roots)
	assert.Equal(t, roots, keysOf(built))
}

func TestBuildDecodesInTheModule(t *testing.T) {
	t.Parallel()

	entity := fullShipment()
	raw, err := record.Build(&record.Input{
		Shipment:     entity,
		Origin:       record.PlaceOf(entity.ShipperStop()),
		Destination:  record.PlaceOf(entity.DeliveryStop()),
		DistanceUnit: record.DistanceMiles,
		Rating:       record.Rating{At: time.Unix(1_772_000_000, 0).UTC(), Mode: record.ModeSell, Currency: "USD"},
	})
	require.NoError(t, err)

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Record:  raw,
		Programs: []rules.TariffProgram{{
			ID:     "probe",
			Kind:   rules.TariffKindTariff,
			Source: "tariff \"Probe\"\n\ncharge \"Miles\" (quantity: shipment.distance, rate: $1/mi)\n",
		}},
	})
	require.NoError(t, err)
	probe := result.Result("probe")
	require.NotNil(t, probe)
	require.Equal(t, rules.TariffRated, probe.Status, "%+v", probe.Error)
	assert.Equal(t, "812", probe.Bill.Total.Value.String())
}

func TestBuildRefusesWithoutAShipment(t *testing.T) {
	t.Parallel()

	_, err := record.Build(&record.Input{})
	require.ErrorIs(t, err, record.ErrNoShipment)
}

func TestZip3TakesTheFirstThreeCharacters(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "752", record.Zip3(" 75201 "))
	assert.Equal(t, "K1A", record.Zip3("k1a 0b1"))
	assert.Equal(t, "12", record.Zip3("12"))
}
