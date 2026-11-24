package boardbar

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

func Fits(form boardcatalog.Form, read *boardcatalog.MetricRead) bool {
	if read == nil {
		return false
	}
	if form.Charted() {
		return chartFits(form, read.Chart)
	}
	switch form {
	case boardcatalog.FormTrend, boardcatalog.FormColumns, boardcatalog.FormBars,
		boardcatalog.FormDots, boardcatalog.FormSpans:
	case boardcatalog.FormSplit:
		return read.Split != nil && len(read.Split.Parts) > 0
	case boardcatalog.FormGrid:
		return read.Grid != nil && len(read.Grid.Rows) > 0
	case boardcatalog.FormLog:
		return len(read.Log) > 0
	case boardcatalog.FormChecks:
		return len(read.Checks) > 0
	case boardcatalog.FormTable:
		return read.Table != nil && len(read.Table.Rows) > 0
	case boardcatalog.FormFigures:
		for _, value := range read.Figures {
			if value != nil {
				return true
			}
		}
	}

	return false
}

func chartFits(form boardcatalog.Form, chart *boardcatalog.Chart) bool {
	if chart == nil {
		return false
	}
	plotted := len(chart.X) > 0 && len(chart.Series) > 0
	switch form {
	case boardcatalog.FormBars:
		return plotted || len(chart.Rows) > 0
	case boardcatalog.FormDots, boardcatalog.FormSpans:
		return len(chart.Rows) > 0
	case boardcatalog.FormTrend, boardcatalog.FormColumns, boardcatalog.FormSplit,
		boardcatalog.FormGrid, boardcatalog.FormLog, boardcatalog.FormChecks,
		boardcatalog.FormTable, boardcatalog.FormFigures:
	}

	return plotted
}
