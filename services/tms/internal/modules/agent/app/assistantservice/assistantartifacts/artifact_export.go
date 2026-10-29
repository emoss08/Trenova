package assistantartifacts

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/extractionartifact"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/recordviews"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	// exportPageRows is how many rows each read of a list asks for: the most
	// a list tool hands back at once.
	exportPageRows = 50
	// maxExportRows bounds a table's download. A list read again page by page
	// is a set of queries, not a report run, and a person who wants more than
	// this wants the report.
	maxExportRows = 10000
)

var errNotExportable = errortypes.NewValidationError(
	"artifact", errortypes.ErrInvalid, "Only a table or a report preview can be downloaded as CSV",
)

var errNotReread = errortypes.NewBusinessError(
	"This table does not say how it was read, so it cannot be read again. Ask for the list again in the conversation",
)

// exportSource is an artifact's table read again from where it came from.
type exportSource struct {
	artifact  *assistantartifact.Artifact
	arguments map[string]any
}

type tablePosition struct {
	offset int
	after  string
}

// writeTableCSV pages through a list and writes what a person reads of it:
// the columns the table shows, with their labels, and every row.
func writeTableCSV(
	sink io.Writer,
	read func(at tablePosition) (map[string]any, error),
	entity string,
) error {
	writer := csv.NewWriter(sink)
	var columns []assistantartifact.DisplayColumn
	written := 0
	at := tablePosition{}
	for written < maxExportRows {
		page, err := read(at)
		if err != nil {
			return err
		}
		rows, _ := page["items"].([]any)
		if len(rows) == 0 {
			break
		}
		projection := artifactpayload.ProjectTable(
			entity,
			artifactpayload.StringsOf(page["columns"]),
			rows,
		)
		if columns == nil {
			columns = projection.Columns
			if err = writer.Write(csvHeader(columns)); err != nil {
				return err
			}
		}
		if written, err = writeCSVRows(writer, columns, projection.Rows, written); err != nil {
			return err
		}
		if !typeutils.BoolOf(page["hasMore"]) {
			break
		}
		if at, err = nextTablePosition(page, at, len(rows)); err != nil {
			return err
		}
	}
	writer.Flush()

	return writer.Error()
}

func csvHeader(columns []assistantartifact.DisplayColumn) []string {
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, column.Label)
	}

	return header
}

func writeCSVRows(
	writer *csv.Writer,
	columns []assistantartifact.DisplayColumn,
	rows []any,
	written int,
) (int, error) {
	for _, row := range rows {
		record, _ := row.(map[string]any)
		cells := make([]string, 0, len(columns))
		for _, column := range columns {
			cells = append(cells, csvCell(record[column.Key]))
		}
		if err := writer.Write(cells); err != nil {
			return written, err
		}
		written++
		if written == maxExportRows {
			break
		}
	}

	return written, nil
}

func nextTablePosition(
	page map[string]any,
	at tablePosition,
	read int,
) (tablePosition, error) {
	if cursor := typeutils.StringOfTrimmed(page["nextCursor"]); cursor != "" {
		if cursor == at.after {
			return tablePosition{}, errors.New("the list returned the same page cursor twice")
		}

		return tablePosition{after: cursor}, nil
	}
	next := int(recordviews.NumberOf(page, "nextOffset"))
	if next <= at.offset {
		next = at.offset + read
	}

	return tablePosition{offset: next}, nil
}

// csvCell is a projected value as a spreadsheet reads it: text as itself, a
// named record by its name, a list joined.
func csvCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case map[string]any:
		for _, key := range []string{"label", "name", "value", "code"} {
			if text := typeutils.StringOfTrimmed(typed[key]); text != "" {
				return text
			}
		}

		return ""
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, csvCell(item))
		}

		return strings.Join(parts, "; ")
	default:
		return extractionartifact.DisplayValue(typed)
	}
}
