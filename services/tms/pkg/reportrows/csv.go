package reportrows

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/bytedance/sonic"
)

// WriteCSV writes the rows as CSV for a program to read, not a person: the
// header is each column's id and the values stay raw, so a decimal keeps every
// digit and an instant stays unix seconds. Totals are left out; a program
// computes its own.
func (e *Envelope) WriteCSV(w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(e.ColumnIDs()); err != nil {
		return err
	}

	record := make([]string, len(e.Schema))
	for i := range e.Rows {
		for j, value := range e.Rows[i] {
			cell, err := csvCell(value)
			if err != nil {
				return fmt.Errorf("row %d column %s: %w", i, e.Schema[j].ID, err)
			}
			record[j] = cell
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()

	return writer.Error()
}

func csvCell(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int:
		return strconv.Itoa(v), nil
	default:
		return sonic.MarshalString(v)
	}
}
