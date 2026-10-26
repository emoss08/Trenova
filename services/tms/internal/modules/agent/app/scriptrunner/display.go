package scriptrunner

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
)

const (
	pageBytes = 512 << 10
	pageRows  = 5000

	showFigure = "figure"
	showTable  = "table"
	showChart  = "chart"
)

type datasetEntry struct {
	Name    string                      `json:"name"`
	Label   string                      `json:"label"`
	Rows    int                         `json:"rows"`
	Columns []scriptports.DatasetColumn `json:"columns"`
	Text    *string                     `json:"text,omitempty"`
}

// datasetIndex is what the SDK binds a global to each dataset from: its name,
// columns and size. The rows are paged in through the rows host function.
func datasetIndex(datasets []scriptports.Dataset) []datasetEntry {
	entries := make([]datasetEntry, 0, len(datasets))
	for idx := range datasets {
		dataset := &datasets[idx]
		columns := dataset.Columns
		if columns == nil {
			columns = []scriptports.DatasetColumn{}
		}
		entries = append(entries, datasetEntry{
			Name:    dataset.Name,
			Label:   dataset.Label,
			Rows:    len(dataset.Rows),
			Columns: columns,
			Text:    dataset.Text,
		})
	}

	return entries
}

type rowsCall struct {
	Dataset string `json:"dataset"`
	Offset  int    `json:"offset"`
}

type rowsPage struct {
	Rows []sonic.NoCopyRawMessage `json:"rows"`
	Done bool                     `json:"done"`
}

// rows is the next page of a dataset from offset: as many rows as fit in a
// page's bytes, so a wide table and a narrow one cost the interpreter alike.
func (h *host) rows(payload []byte) ([]byte, error) {
	var call rowsCall
	if err := sonic.Unmarshal(payload, &call); err != nil {
		return sonic.Marshal(rowsPage{Rows: []sonic.NoCopyRawMessage{}, Done: true})
	}
	dataset := h.dataset(call.Dataset)
	if dataset == nil || call.Offset < 0 || call.Offset >= len(dataset.Rows) {
		return sonic.Marshal(rowsPage{Rows: []sonic.NoCopyRawMessage{}, Done: true})
	}

	page := rowsPage{Rows: make([]sonic.NoCopyRawMessage, 0, min(pageRows, len(dataset.Rows)))}
	size := 0
	next := call.Offset
	for ; next < len(dataset.Rows) && len(page.Rows) < pageRows; next++ {
		row, err := sonic.Marshal(dataset.Rows[next])
		if err != nil {
			return nil, fmt.Errorf("encode row %d of dataset %s: %w", next, dataset.Name, err)
		}
		if size+len(row) > pageBytes && len(page.Rows) > 0 {
			break
		}
		size += len(row)
		page.Rows = append(page.Rows, row)
	}
	page.Done = next >= len(dataset.Rows)

	return sonic.Marshal(page)
}

func (h *host) dataset(name string) *scriptports.Dataset {
	for idx := range h.datasets {
		if h.datasets[idx].Name == name {
			return &h.datasets[idx]
		}
	}

	return nil
}

type showCall struct {
	Type string `json:"type"`

	Label    string `json:"label"`
	Value    any    `json:"value"`
	Kind     string `json:"kind"`
	Note     string `json:"note"`
	Currency string `json:"currency"`

	Title     string                 `json:"title"`
	Columns   []analysis.TableColumn `json:"columns"`
	Rows      [][]any                `json:"rows"`
	TotalRows int                    `json:"totalRows"`

	X          analysis.ChartAxis     `json:"x"`
	Series     []analysis.ChartSeries `json:"series"`
	ValueKind  string                 `json:"valueKind"`
	Stacked    bool                   `json:"stacked"`
	Horizontal bool                   `json:"horizontal"`
}

type showReply struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// show keeps one figure, table or chart the script showed the person, within
// what a run may keep. Past that it is not shown, and the script is told.
func (h *host) show(payload []byte) showReply {
	var call showCall
	if err := sonic.Unmarshal(payload, &call); err != nil {
		return showReply{Status: "invalid", Message: "what was shown could not be read"}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var (
		kept  bool
		limit string
	)
	switch call.Type {
	case showFigure:
		kept = h.shown.AddFigure(&analysis.Figure{
			Label:    call.Label,
			Value:    call.Value,
			Kind:     analysis.ValueKind(call.Kind),
			Currency: call.Currency,
			Note:     call.Note,
		})
		limit = fmt.Sprintf("a run shows at most %d figures, each with a label",
			analysis.MaxFigures)
	case showTable:
		kept = h.shown.AddTable(&analysis.Table{
			Title:     call.Title,
			Columns:   h.columnKinds(call.Columns),
			Rows:      call.Rows,
			TotalRows: call.TotalRows,
		})
		limit = fmt.Sprintf("a run shows at most %d tables, each with a column", analysis.MaxTables)
	case showChart:
		kept = h.shown.AddChart(&analysis.Chart{
			Kind:       analysis.ChartKind(call.Kind),
			Title:      call.Title,
			X:          call.X,
			Series:     call.Series,
			ValueKind:  analysis.ValueKind(call.ValueKind),
			Currency:   call.Currency,
			Stacked:    call.Stacked,
			Horizontal: call.Horizontal,
		})
		limit = fmt.Sprintf(
			"a run shows at most %d charts, each a bar, line, area, scatter or pie chart with values",
			analysis.MaxCharts,
		)
	default:
		return showReply{Status: "invalid", Message: "show a figure, a table or a chart"}
	}
	if !kept {
		h.notShown++
		return showReply{Status: "limit", Message: "not shown: " + limit}
	}

	return showReply{Status: "ok"}
}

// columnKinds gives a column the script named no kind for the kind of the
// dataset column it shares a name with, so money read from a report is shown
// as money.
func (h *host) columnKinds(columns []analysis.TableColumn) []analysis.TableColumn {
	for idx := range columns {
		if columns[idx].Kind.IsValid() {
			continue
		}
		for dsIdx := range h.datasets {
			for _, column := range h.datasets[dsIdx].Columns {
				if column.Name == columns[idx].Name {
					columns[idx].Kind = column.Kind
					columns[idx].Currency = column.Currency
				}
			}
		}
	}

	return columns
}
