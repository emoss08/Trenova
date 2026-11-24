package boardbar

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
)

func FromAnalysis(chart *analysis.Chart) (*boardcatalog.MetricRead, boardcatalog.Form) {
	unit := unitOf(chart.ValueKind)
	read := &boardcatalog.MetricRead{Title: chart.Title, Figures: map[string]*boardcatalog.Value{}}
	if chart.Kind == analysis.ChartPie {
		read.Split = splitOf(chart, unit)
		return read, boardcatalog.FormSplit
	}
	out := &boardcatalog.Chart{
		Unit:     unit,
		Currency: chart.Currency,
		X:        make([]boardcatalog.Tick, 0, len(chart.X.Values)),
		Series:   make([]boardcatalog.Series, 0, len(chart.Series)),
	}
	for _, value := range chart.X.Values {
		out.X = append(out.X, boardcatalog.Tick{Label: labelOf(value)})
	}
	role := boardcatalog.RoleNow
	if chart.Stacked {
		role = boardcatalog.RoleStack
	}
	for idx := range chart.Series {
		series := &chart.Series[idx]
		points := make([]*decimal.Decimal, 0, len(series.Values))
		for _, value := range series.Values {
			points = append(points, numberOf(value))
		}
		seriesRole := role
		if !chart.Stacked && len(chart.Series) > 1 && idx < len(chart.Series)-1 {
			seriesRole = boardcatalog.RoleWas
		}
		out.Series = append(out.Series, boardcatalog.Series{
			Name:   series.Name,
			Role:   seriesRole,
			Points: points,
		})
	}
	read.Chart = out
	if chart.Kind == analysis.ChartBar {
		return read, boardcatalog.FormBars
	}

	return read, boardcatalog.FormTrend
}

func splitOf(chart *analysis.Chart, unit boardcatalog.ValueKind) *boardcatalog.Split {
	split := &boardcatalog.Split{Total: boardcatalog.Value{Kind: unit, Currency: chart.Currency}}
	if len(chart.Series) == 0 {
		return split
	}
	series := &chart.Series[0]
	for idx, value := range series.Values {
		number := numberOf(value)
		if number == nil {
			continue
		}
		label := ""
		if idx < len(chart.X.Values) {
			label = labelOf(chart.X.Values[idx])
		}
		split.Total.Number = split.Total.Number.Add(*number)
		split.Parts = append(split.Parts, boardcatalog.Part{
			Label: label,
			Value: boardcatalog.Value{Kind: unit, Number: *number, Currency: chart.Currency},
			Kind:  boardcatalog.PartUsed,
		})
	}

	return split
}

func unitOf(kind analysis.ValueKind) boardcatalog.ValueKind {
	switch kind {
	case analysis.ValueMoney:
		return boardcatalog.ValueMoney
	case analysis.ValueCount:
		return boardcatalog.ValueCount
	case analysis.ValuePercent:
		return boardcatalog.ValuePercent
	case analysis.ValueText, analysis.ValueNumber, analysis.ValueDate, analysis.ValueDateTime,
		analysis.ValueBool:
	}

	return boardcatalog.ValueNumber
}

func numberOf(value any) *decimal.Decimal {
	var number decimal.Decimal
	switch typed := value.(type) {
	case float64:
		number = decimal.NewFromFloat(typed)
	case float32:
		number = decimal.NewFromFloat32(typed)
	case int:
		number = decimal.NewFromInt(int64(typed))
	case int64:
		number = decimal.NewFromInt(typed)
	default:
		return nil
	}

	return &number
}

func labelOf(value any) string {
	if value == nil {
		return ""
	}

	return fmt.Sprint(value)
}
