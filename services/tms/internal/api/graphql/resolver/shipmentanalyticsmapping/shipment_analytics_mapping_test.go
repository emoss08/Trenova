package shipmentanalyticsmapping

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentAnalyticsToModel_MapsTypedCards(t *testing.T) {
	t.Parallel()

	target := 96.5
	model, err := ToModel(kernelports.AnalyticsData{
		"activeShipments": map[string]any{
			"count":               12,
			"changeFromYesterday": -1,
			"sparkline": []any{
				map[string]any{"hour": "08:00", "value": 3.5},
			},
			"breakdown": map[string]any{
				"inTransit": 6,
				"atRisk":    2,
				"loading":   3,
				"done":      1,
			},
		},
		"onTimePercent": map[string]any{
			"percent":         94.2,
			"onTimeCount":     94,
			"totalCount":      100,
			"target":          target,
			"deltaPp":         -1.1,
			"sevenDayPercent": 95.3,
		},
		"tomorrowsPickups": map[string]any{
			"date": "2026-06-07",
			"pickups": []any{
				map[string]any{
					"shipmentId":        "shp_1",
					"proNumber":         "PRO-1",
					"pickupWindowStart": 10,
					"customer":          "Acme",
					"origin":            "CHI",
					"destination":       "DAL",
					"driver":            "Alex",
					"status":            "scheduled",
				},
			},
		},
		"laneHeatmap": map[string]any{
			"windowDays": 7,
			"total":      1,
			"cells": []any{
				map[string]any{"origin": "IL", "destination": "TX", "count": 1},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, model.ActiveShipments)
	assert.Equal(t, 12, model.ActiveShipments.Count)
	require.Len(t, model.ActiveShipments.Sparkline, 1)
	assert.Equal(t, "08:00", model.ActiveShipments.Sparkline[0].Hour)
	require.NotNil(t, model.OnTimePercent)
	assert.Equal(t, target, *model.OnTimePercent.Target)
	require.NotNil(t, model.TomorrowsPickups)
	assert.Equal(t, "PRO-1", model.TomorrowsPickups.Pickups[0].ProNumber)
	require.NotNil(t, model.LaneHeatmap)
	assert.Equal(t, "TX", model.LaneHeatmap.Cells[0].Destination)
}

func TestShipmentAnalyticsToModel_KeepsWithheldValuesNull(t *testing.T) {
	t.Parallel()

	model, err := ToModel(kernelports.AnalyticsData{
		"customerMix": nil,
		"detentionWatchlist": map[string]any{
			"items": []any{map[string]any{"shipmentId": "PRO-1", "customer": nil}},
		},
		"tomorrowsPickups": map[string]any{
			"pickups": []any{map[string]any{
				"proNumber":   "PRO-1",
				"customer":    nil,
				"origin":      "CHI",
				"destination": nil,
				"driver":      "",
			}},
		},
		"profitability": map[string]any{"avgCpm": 1.9, "avgMarginPct": nil, "hasMargin": true},
	})

	require.NoError(t, err)
	assert.Nil(t, model.CustomerMix, "a reader who cannot read customers gets no customer mix")
	assert.Nil(t, model.DetentionWatchlist.Items[0].Customer)
	pickup := model.TomorrowsPickups.Pickups[0]
	assert.Nil(t, pickup.Customer)
	assert.Equal(t, "CHI", *pickup.Origin)
	assert.Nil(t, pickup.Destination)
	require.NotNil(t, pickup.Driver, "an unassigned pickup has no driver, which is not withheld")
	assert.Empty(t, *pickup.Driver)
	assert.Nil(t, model.Profitability.AvgMarginPct)
	assert.True(t, model.Profitability.HasMargin)
}
