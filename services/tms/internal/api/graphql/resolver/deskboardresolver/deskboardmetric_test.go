package deskboardresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricToModelCarriesItsFormsAndTheRecordsItNames(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	ref := boardcatalog.Ref{Kind: shipmentKind, ID: shipment}
	point := decimal.NewFromFloat(0.92)
	overdue := boardcatalog.Money(824000, "USD")
	metric := &board.FilledMetric{
		Name: billingboard.MetricReceivables,
		Form: boardcatalog.FormFigures,
		Read: &boardcatalog.MetricRead{
			Title:   "Receivables",
			Figures: map[string]*boardcatalog.Value{"arOverdue": &overdue, "arTotalOpen": nil},
			Chart: &boardcatalog.Chart{
				Unit:   boardcatalog.ValuePercent,
				X:      []boardcatalog.Tick{{Label: "W1"}, {Label: "W2", Marked: true}},
				Series: []boardcatalog.Series{{Name: "On time", Points: []*decimal.Decimal{&point, nil}}},
				Rows:   []boardcatalog.ChartRow{{Label: "S-1", Ref: &ref}},
			},
			Log: []boardcatalog.LogEntry{{At: 1_793_000_000, Text: "Tender in", Refs: []boardcatalog.Ref{ref}}},
		},
		Rows: []board.Row{{Kind: shipmentKind, ID: shipment, Readable: true, Label: "S-1"}},
	}

	model := metricToModel(metric)

	assert.Equal(t, "receivables", model.Name)
	assert.Equal(t, "figures", model.Form)
	require.Len(t, model.Figures, 2, "every figure the catalog declares that was read")
	assert.Equal(t, "arTotalOpen", model.Figures[0].Name)
	assert.Nil(t, model.Figures[0].Value)
	assert.Equal(t, "8240", model.Figures[1].Value.Number)
	require.NotNil(t, model.Chart)
	assert.True(t, model.Chart.Ticks[1].Marked)
	assert.Equal(t, "0.92", *model.Chart.Series[0].Points[0])
	assert.Nil(t, model.Chart.Series[0].Points[1])
	assert.Equal(t, "S-1", model.Chart.Rows[0].Record.Label)
	require.Len(t, model.Log, 1)
	require.Len(t, model.Log[0].Records, 1)
	assert.Equal(t, shipment.String(), model.Log[0].Records[0].ID)
	assert.Empty(t, model.Checks)
}
