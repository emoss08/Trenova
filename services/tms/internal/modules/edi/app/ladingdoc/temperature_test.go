package ladingdoc_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func degrees(value int16) *int16 {
	return &value
}

func TestAMultiTemperatureTenderKeepsEachDropsAndLinesRange(t *testing.T) {
	t.Parallel()

	payload, _, err := ladingdoc.ToLoadTender(
		jsonutils.RawJSON(`{"key":"G-1","stops":[`+
			`{"sequence":1,"type":"Pickup","window":{"start":{"at":"2026-01-10T14:00:00Z"}}},`+
			`{"sequence":2,"type":"Delivery","window":{"start":{"at":"2026-01-11T14:00:00Z"}},`+
			`"temperature":{"min":{"value":"-10","unit":"°F"},"max":{"value":"0","unit":"°F"}}},`+
			`{"sequence":3,"type":"Delivery","window":{"start":{"at":"2026-01-11T18:00:00Z"}},`+
			`"temperature":{"min":{"value":"34","unit":"°F"},"max":{"value":"38.4","unit":"°F"}}}],`+
			`"commodities":[{"description":"Ice cream",`+
			`"temperature":{"max":{"value":"-5","unit":"°F"}}}]}`),
		ladingdoc.Unmapped{},
	)

	require.NoError(t, err)
	require.Len(t, payload.Moves, 1)
	stops := payload.Moves[0].Stops
	require.Len(t, stops, 3)
	assert.Nil(t, stops[0].TemperatureMin)
	assert.Nil(t, stops[0].TemperatureMax)
	assert.Equal(t, degrees(-10), stops[1].TemperatureMin)
	assert.Equal(t, degrees(0), stops[1].TemperatureMax)
	assert.Equal(t, degrees(34), stops[2].TemperatureMin)
	assert.Equal(t, degrees(38), stops[2].TemperatureMax)
	require.Len(t, payload.Commodities, 1)
	assert.Nil(t, payload.Commodities[0].TemperatureMin)
	assert.Equal(t, degrees(-5), payload.Commodities[0].TemperatureMax)
	assert.Equal(t, degrees(-10), payload.TemperatureMin,
		"drops that share no range give the load the span from the coldest to the warmest bound")
	assert.Equal(t, degrees(38), payload.TemperatureMax)
}

func TestATenderWithOnlyALoadTemperatureKeepsIt(t *testing.T) {
	t.Parallel()

	payload, _, err := ladingdoc.ToLoadTender(
		jsonutils.RawJSON(`{"key":"G-2",`+
			`"temperature":{"min":{"value":"33","unit":"°F"},"max":{"value":"36","unit":"°F"}},`+
			`"stops":[{"sequence":1,"type":"Pickup","window":{"start":{"at":"2026-01-10T14:00:00Z"}}},`+
			`{"sequence":2,"type":"Delivery","window":{"start":{"at":"2026-01-11T14:00:00Z"}}}]}`),
		ladingdoc.Unmapped{},
	)

	require.NoError(t, err)
	assert.Equal(t, degrees(33), payload.TemperatureMin)
	assert.Equal(t, degrees(36), payload.TemperatureMax)
	assert.Nil(t, payload.Moves[0].Stops[1].TemperatureMin)
}

func TestAnOutboundTenderWritesEachStopsAndLinesRange(t *testing.T) {
	t.Parallel()

	raw, err := ladingdoc.FromLoadTender(&editransfer.LoadTenderPayload{
		TemperatureMin: degrees(-10),
		TemperatureMax: degrees(38),
		Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
			{Type: "Pickup", ScheduledWindowStart: 1_768_053_600},
			{
				Type:                 "Delivery",
				ScheduledWindowStart: 1_768_147_200,
				TemperatureMin:       degrees(-10),
				TemperatureMax:       degrees(0),
			},
			{
				Type:                 "Delivery",
				ScheduledWindowStart: 1_768_160_000,
				TemperatureMin:       degrees(34),
			},
		}}},
		Commodities: []editransfer.LoadTenderCommodity{
			{CommodityLabel: "Ice cream", Weight: 1000, Pieces: 10, TemperatureMax: degrees(-5)},
			{CommodityLabel: "Dry goods", Weight: 500, Pieces: 5},
		},
	}, "G-3", ladingdoc.Zones{Default: "America/Chicago"}, nil)
	require.NoError(t, err)

	var document ladingdoc.LoadTender
	require.NoError(t, sonic.Unmarshal(raw, &document))
	require.NotNil(t, document.Temperature)
	assert.Equal(
		t,
		&ladingdoc.Measure{Value: "-10", Unit: ladingdoc.UnitFahrenheit},
		document.Temperature.Min,
	)
	require.Len(t, document.Stops, 3)
	assert.Nil(t, document.Stops[0].Temperature)
	require.NotNil(t, document.Stops[1].Temperature)
	assert.Equal(t, "-10", document.Stops[1].Temperature.Min.Value)
	assert.Equal(t, "0", document.Stops[1].Temperature.Max.Value)
	require.NotNil(t, document.Stops[2].Temperature)
	assert.Equal(t, "34", document.Stops[2].Temperature.Min.Value)
	assert.Nil(t, document.Stops[2].Temperature.Max)
	require.Len(t, document.Commodities, 2)
	require.NotNil(t, document.Commodities[0].Temperature)
	assert.Nil(t, document.Commodities[0].Temperature.Min)
	assert.Equal(t, "-5", document.Commodities[0].Temperature.Max.Value)
	assert.Nil(t, document.Commodities[1].Temperature)
}
