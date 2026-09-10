package fuelimport

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/rateimport"
)

// ErrNoFixedWidthRecords means the file held nothing but blank lines.
var ErrNoFixedWidthRecords = errors.New("this file has no records in it")

// FixedWidthColumn is one field of a fixed-width record, as a half-open byte
// range over the line. Start and End come from the vendor's record layout and are
// one-indexed and inclusive, which is how those documents are always written.
type FixedWidthColumn struct {
	Name  string
	Start int
	End   int
}

// FixedWidthLayout describes a vendor's record layout. Trenova does not ship a
// layout for any particular file, because the networks publish those under their
// own agreements and they differ by account. The layout is configured per
// connection instead, and the column names are then matched to fields by the same
// header dictionary a spreadsheet upload uses.
type FixedWidthLayout struct {
	Columns []FixedWidthColumn
}

func (l FixedWidthLayout) Headers() []string {
	headers := make([]string, 0, len(l.Columns))
	for _, column := range l.Columns {
		headers = append(headers, column.Name)
	}

	return headers
}

// ParseFixedWidthLayout reads a layout written as `Name:start-end` entries
// separated by commas or newlines. Lines starting with # are comments, so a
// layout can carry a note about which vendor document it came from.
func ParseFixedWidthLayout(spec string) (FixedWidthLayout, error) {
	layout := FixedWidthLayout{}
	seen := make(map[string]struct{}, 16)

	for _, entry := range splitLayoutEntries(spec) {
		column, err := parseFixedWidthColumn(entry)
		if err != nil {
			return FixedWidthLayout{}, err
		}

		key := strings.ToLower(column.Name)
		if _, duplicate := seen[key]; duplicate {
			return FixedWidthLayout{}, fmt.Errorf(
				"the column %q appears twice in the layout",
				column.Name,
			)
		}
		seen[key] = struct{}{}

		layout.Columns = append(layout.Columns, column)
	}

	if len(layout.Columns) == 0 {
		return FixedWidthLayout{}, errors.New("the layout is empty")
	}

	return layout, nil
}

func splitLayoutEntries(spec string) []string {
	lines := strings.FieldsFunc(spec, func(r rune) bool {
		return r == '\n' || r == '\r'
	})

	entries := make([]string, 0, len(lines))
	for _, line := range lines {
		// A comment runs to the end of its line, so it has to be cut before the
		// line is split on commas; otherwise a comma inside the comment turns its
		// tail into an entry of its own.
		if index := strings.Index(line, "#"); index >= 0 {
			line = line[:index]
		}

		for _, entry := range strings.Split(line, ",") {
			if entry = strings.TrimSpace(entry); entry != "" {
				entries = append(entries, entry)
			}
		}
	}

	return entries
}

func parseFixedWidthColumn(entry string) (FixedWidthColumn, error) {
	name, span, ok := strings.Cut(entry, ":")
	if !ok {
		return FixedWidthColumn{}, layoutFormatError(entry)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return FixedWidthColumn{}, fmt.Errorf("the layout entry %q needs a name", entry)
	}

	startText, endText, ok := strings.Cut(strings.TrimSpace(span), "-")
	if !ok {
		return FixedWidthColumn{}, layoutFormatError(entry)
	}

	start, err := strconv.Atoi(strings.TrimSpace(startText))
	if err != nil {
		return FixedWidthColumn{}, layoutFormatError(entry)
	}

	end, err := strconv.Atoi(strings.TrimSpace(endText))
	if err != nil {
		return FixedWidthColumn{}, layoutFormatError(entry)
	}

	switch {
	case start < 1:
		return FixedWidthColumn{}, fmt.Errorf(
			"the column %q starts at column %d; a record starts at column 1",
			name, start,
		)
	case end < start:
		return FixedWidthColumn{}, fmt.Errorf(
			"the column %q ends before it starts",
			name,
		)
	}

	return FixedWidthColumn{Name: name, Start: start, End: end}, nil
}

func layoutFormatError(entry string) error {
	return fmt.Errorf(
		"the layout entry %q must be written as Name:start-end, for example Card Number:9-24",
		entry,
	)
}

// ReadFixedWidth slices every record of a fixed-width export into the columns the
// layout declares, producing the same sheet a delimited file would. A record
// shorter than the layout yields blank cells for the fields it does not reach,
// because vendors trim trailing padding inconsistently.
func ReadFixedWidth(content []byte, layout FixedWidthLayout) (*rateimport.Sheet, error) {
	if len(layout.Columns) == 0 {
		return nil, errors.New("the layout is empty")
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	rows := make([][]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, sliceFixedWidthRecord(line, layout))
	}

	if len(rows) == 0 {
		return nil, ErrNoFixedWidthRecords
	}

	return &rateimport.Sheet{
		Headers:      layout.Headers(),
		Rows:         rows,
		FirstDataRow: 1,
	}, nil
}

func sliceFixedWidthRecord(line string, layout FixedWidthLayout) []string {
	runes := []rune(line)
	cells := make([]string, 0, len(layout.Columns))

	for _, column := range layout.Columns {
		start := column.Start - 1
		if start >= len(runes) {
			cells = append(cells, "")
			continue
		}

		end := min(column.End, len(runes))
		cells = append(cells, strings.TrimSpace(string(runes[start:end])))
	}

	return cells
}
