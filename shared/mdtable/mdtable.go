// Package mdtable finds the tables a markdown text holds.
package mdtable

import (
	"regexp"
	"strings"
)

// Table is one markdown table: where it sits in the text, as byte offsets
// covering its header, separator and body lines with their line breaks,
// what its header and body cells say, and the line of prose just above it.
type Table struct {
	Start  int
	End    int
	Header []string
	Rows   [][]string
	Lead   string
}

var (
	separatorRow = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?\s*$`)
	fence        = regexp.MustCompile("^\\s*(```|~~~)")
	linkText     = regexp.MustCompile(`\[([^\[\]]*)\]\([^()]*\)`)
	emphasis     = regexp.MustCompile("[*_`~]+")
	spaces       = regexp.MustCompile(`\s+`)
)

type line struct {
	text  string
	start int
	end   int
}

// Find is every table in the text, in order. A table inside a fenced code
// block is code, not a table, and is left out.
func Find(text string) []Table {
	lines := splitLines(text)
	tables := make([]Table, 0, 1)
	inFence := false
	for idx := 0; idx < len(lines); idx++ {
		current := lines[idx].text
		if fence.MatchString(current) {
			inFence = !inFence
			continue
		}
		if inFence || idx+1 >= len(lines) || !opensTable(current, lines[idx+1].text) {
			continue
		}

		table := Table{Start: lines[idx].start, Header: Cells(current)}
		if idx > 0 {
			table.Lead = strings.TrimSpace(lines[idx-1].text)
		}
		end := idx + 1
		for next := idx + 2; next < len(lines); next++ {
			row := lines[next].text
			if strings.TrimSpace(row) == "" || !strings.Contains(row, "|") {
				break
			}
			table.Rows = append(table.Rows, Cells(row))
			end = next
		}
		table.End = lines[end].end
		tables = append(tables, table)
		idx = end
	}

	return tables
}

// opensTable reports a header row followed by the separator row under it.
func opensTable(header, separator string) bool {
	return strings.Contains(header, "|") && strings.Contains(separator, "|") &&
		separatorRow.MatchString(separator)
}

// Cells is a table row's cells, trimmed, with the pipes at its edges gone.
func Cells(row string) []string {
	trimmed := strings.TrimSpace(row)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}

	return cells
}

// Plain is a cell as its words read: a link as its text, emphasis and code
// marks gone, spaces closed up, in lower case, so the same value written in
// a table and held in a record compare equal.
func Plain(cell string) string {
	text := linkText.ReplaceAllString(cell, "$1")
	text = emphasis.ReplaceAllString(text, "")
	text = spaces.ReplaceAllString(strings.TrimSpace(text), " ")

	return strings.ToLower(text)
}

func splitLines(text string) []line {
	lines := make([]line, 0, strings.Count(text, "\n")+1)
	start := 0
	for start <= len(text) {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			lines = append(lines, line{text: text[start:], start: start, end: len(text)})
			break
		}
		lines = append(lines, line{
			text:  text[start : start+end],
			start: start,
			end:   start + end + 1,
		})
		start += end + 1
	}

	return lines
}
