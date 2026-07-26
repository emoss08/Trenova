package compiler

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapDef builds a location-level report carrying coordinates, which is the
// shape a map tile actually plots — locations and customers are the entities
// the catalog exposes latitude and longitude on.
func mapDef(chart report.ChartSpec) *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "location",
		Columns: []report.ColumnSpec{
			dim("c_lat", "latitude"),
			dim("c_lng", "longitude"),
			dim("c_name", "name"),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
		Charts: []report.ChartSpec{chart},
	}
}

// A map is positioned by coordinates alone — no category axis, and a measure
// is optional because the pin exists whether or not anything scales it.
func TestCompileMapChart(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	// Charts are validated by the compiler and carried to the client on the
	// definition itself, so a clean compile is the assertion that matters.
	compiled := mustCompile(t, c, mapDef(report.ChartSpec{
		ID:            "ch_map",
		Type:          report.ChartMap,
		Title:         "Stop Density",
		LatColumnID:   "c_lat",
		LngColumnID:   "c_lng",
		LabelColumnID: "c_name",
	}))

	assert.Contains(t, compiled.SQL, "latitude")
	assert.Contains(t, compiled.SQL, "longitude")
}

// A measure sizes the marker, so it is allowed but never required.
func TestCompileMapChartWithMeasure(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, mapDef(report.ChartSpec{
		ID:          "ch_map",
		Type:        report.ChartMap,
		LatColumnID: "c_lat",
		LngColumnID: "c_lng",
		SeriesIDs:   []string{"c_count"},
	}))

	assert.Contains(t, compiled.SQL, "COUNT(")
}

func TestMapChartValidation(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	tests := []struct {
		name    string
		chart   report.ChartSpec
		wantErr string
	}{
		{
			name:    "missing latitude",
			chart:   report.ChartSpec{ID: "m", Type: report.ChartMap, LngColumnID: "c_lng"},
			wantErr: "needs a latitude column",
		},
		{
			name:    "missing longitude",
			chart:   report.ChartSpec{ID: "m", Type: report.ChartMap, LatColumnID: "c_lat"},
			wantErr: "needs a longitude column",
		},
		{
			name: "unknown coordinate column",
			chart: report.ChartSpec{
				ID: "m", Type: report.ChartMap, LatColumnID: "c_lat", LngColumnID: "c_nope",
			},
			wantErr: "unknown column",
		},
		{
			// Both coordinates from one column puts every row on a diagonal,
			// which looks like data rather than the mistake it is.
			name: "same column for both coordinates",
			chart: report.ChartSpec{
				ID: "m", Type: report.ChartMap, LatColumnID: "c_lat", LngColumnID: "c_lat",
			},
			wantErr: "cannot be the same column",
		},
		{
			name: "category axis on a map",
			chart: report.ChartSpec{
				ID: "m", Type: report.ChartMap, XColumnID: "c_name",
				LatColumnID: "c_lat", LngColumnID: "c_lng",
			},
			wantErr: "has no category axis",
		},
		{
			name: "unknown label column",
			chart: report.ChartSpec{
				ID: "m", Type: report.ChartMap, LatColumnID: "c_lat", LngColumnID: "c_lng",
				LabelColumnID: "c_nope",
			},
			wantErr: "unknown column",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Compile(context.Background(), newRequest(mapDef(tt.chart)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
