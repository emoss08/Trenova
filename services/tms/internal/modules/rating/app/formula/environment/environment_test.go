package environment_test

import (
	"os"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func builder(t *testing.T) *environment.Builder {
	t.Helper()

	registry := schema.NewRegistry()
	definition, err := os.ReadFile("../schema/definitions/shipment.schema.json")
	require.NoError(t, err)
	require.NoError(t, registry.Register("shipment", definition))

	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)
	return environment.NewBuilder(environment.Params{Registry: registry, Resolver: res})
}

func encodedEnv(t *testing.T, env map[string]any) map[string]sonic.NoCopyRawMessage {
	t.Helper()

	encoded, err := rules.EncodeEnv(env)
	require.NoError(t, err, "every value the builder produces encodes exactly")

	var fields map[string]sonic.NoCopyRawMessage
	require.NoError(t, sonic.Unmarshal(encoded, &fields))
	return fields
}

func TestBuild_EncodesARealShipmentExactlyWithEachDateInItsStopsOffset(t *testing.T) {
	t.Parallel()

	weight := int64(4_000)
	length, width, height := 4.1, 2.0, 2.0
	pickupAt := time.Date(2026, time.March, 2, 14, 0, 0, 0, time.UTC).Unix()
	entity := &shipment.Shipment{
		ServiceType: &servicetype.ServiceType{Code: "STD"},
		Weight:      &weight,
		BaseRate:    decimal.NewNullDecimal(decimal.RequireFromString("2.1500")),
		RatingUnit:  1,
		Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{{
			Type:                 shipment.StopTypePickup,
			ScheduledWindowStart: pickupAt,
			Location: &location.Location{
				City:     "Chicago",
				Timezone: "America/Chicago",
			},
		}}}},
		Commodities: []*shipment.ShipmentCommodity{{
			CommodityID: pulid.MustNew("com_"),
			Weight:      400,
			Pieces:      2,
			LengthFeet:  &length,
			WidthFeet:   &width,
			HeightFeet:  &height,
			Commodity:   &commodity.Commodity{Name: "Paint", FreightClass: commodity.FreightClass85},
		}},
	}

	built, err := builder(t).Build(entity, "shipment", map[string]any{
		"ratingTime": time.Unix(pickupAt, 0).UTC(),
		"fuelPrice":  3.85,
	}, map[string]any{"miles": 120.5})
	require.NoError(t, err)
	fields := encodedEnv(t, built.Env)

	assert.Equal(t, `4000`, string(fields["weight"]), "an integer field is a whole number")
	assert.Equal(t, `2.15`, string(fields["baseRate"]), "a decimal is its exact text")
	assert.Equal(t, `3.85`, string(fields["fuelPrice"]), "a provided value becomes exact")
	assert.Equal(t, `120.5`, string(fields["miles"]), "a variable from JSON becomes exact")
	assert.Equal(t, `{"$date":"2026-03-02T08:00:00-06:00"}`, string(fields["pickupDate"]),
		"the pickup keeps the offset of its stop")
	assert.Equal(t, `{"$date":"2026-03-02T14:00:00Z"}`, string(fields["ratingTime"]))
	assert.Equal(t, `null`, string(fields["tractorType"]), "an absent optional relation is nil")
	assert.Equal(t, `null`, string(fields["pieces"]), "an empty nullable field is nil")
	assert.Equal(t, `1`, string(fields["totalStops"]))

	var commodities []map[string]sonic.NoCopyRawMessage
	require.NoError(t, sonic.Unmarshal(fields["commodities"], &commodities))
	require.Len(t, commodities, 1)
	assert.Equal(t, `4.1`, string(commodities[0]["lengthFeet"]), "stored feet keep the digits typed")
	assert.Equal(t, `400`, string(commodities[0]["weight"]))
	assert.Equal(t, `"85"`, string(commodities[0]["freightClass"]))

	assert.Equal(t, formulatypes.ValueSourceProvided, built.Sources["fuelPrice"])
	assert.Equal(t, formulatypes.ValueSourceInput, built.Sources["miles"])
}

func TestBuild_ANonNullableFieldTheRecordLeavesEmptyReadsAsItsZero(t *testing.T) {
	t.Parallel()

	built, err := builder(t).Build(&shipment.Shipment{
		ServiceType: &servicetype.ServiceType{Code: "STD"},
	}, "shipment", nil, nil)
	require.NoError(t, err)
	fields := encodedEnv(t, built.Env)

	assert.Equal(t, `0`, string(fields["baseRate"]))
	assert.Equal(t, `0`, string(fields["freightChargeAmount"]))
}

func TestBuild_AFieldThatCannotBeReadIsNilAndRecorded(t *testing.T) {
	t.Parallel()

	built, err := builder(t).Build(&shipment.Shipment{}, "shipment", nil, nil)
	require.NoError(t, err)

	assert.Nil(t, environment.ValueAt(built.Env, "serviceType.code"))
	assert.Contains(t, built.Failures, "serviceType.code")
	assert.NotContains(t, built.Failures, "tractorType.code", "a nullable relation is no failure")
}

func TestZero_ThePreviewEnvironmentWithoutAShipmentIsAnExactValueTree(t *testing.T) {
	t.Parallel()

	built, err := builder(t).Zero("shipment", nil, map[string]any{
		"pickupDate": "2026-01-02",
		"rate":       2.5,
	})
	require.NoError(t, err)
	fields := encodedEnv(t, built.Env)

	assert.Equal(t, `0`, string(fields["totalDistance"]))
	assert.Equal(t, `""`, string(fields["status"]))
	assert.Equal(t, `false`, string(fields["hasHazmat"]))
	assert.Equal(t, `{"$date":"2026-01-02T00:00:00Z"}`, string(fields["pickupDate"]),
		"a sample date typed as text becomes a date")
	assert.Equal(t, `2.5`, string(fields["rate"]))
}

func TestMerge_RefusesANumberThatIsNotFinite(t *testing.T) {
	t.Parallel()

	env := map[string]any{}
	err := environment.Merge(env, map[string]any{
		"rates": []any{map[string]any{"value": 0.0 / zero()}},
	}, environment.Sources{}, formulatypes.ValueSourceInput)
	require.ErrorIs(t, err, environment.ErrNonFiniteValue)
}

func zero() float64 {
	return 0
}
