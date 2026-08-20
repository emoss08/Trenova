package rateimport

import (
	"encoding/csv"
	"strings"
)

// TemplateFileName is what the starter sheet downloads as.
const TemplateFileName = "rate-sheet-template.csv"

// templateColumn is one column of the starter sheet.
//
// The headers here are the canonical spellings of the fields ParseRow reads,
// so a sheet built from the template maps without a single unplaced column.
// The test suite holds this file to that: every header must map, and every
// example row must parse.
type templateColumn struct {
	header   string
	examples [2]string
}

var templateColumns = []templateColumn{
	{header: "Lane Name", examples: [2]string{"Southeast per-mile", "Chicago to Dallas"}},
	{header: "Origin Type", examples: [2]string{"State", "City"}},
	{header: "Origin", examples: [2]string{"GA", "IL"}},
	{header: "Origin City", examples: [2]string{"", "Chicago"}},
	{header: "Destination Type", examples: [2]string{"State", "City"}},
	{header: "Destination", examples: [2]string{"FL", "TX"}},
	{header: "Destination City", examples: [2]string{"", "Dallas"}},
	{header: "Rate Type", examples: [2]string{"Per Mile", "Flat"}},
	{header: "Rate", examples: [2]string{"2.85", "1450"}},
	{header: "Currency", examples: [2]string{"USD", "USD"}},
	{header: "Minimum Charge", examples: [2]string{"350", ""}},
	{header: "Maximum Charge", examples: [2]string{"", ""}},
	{header: "Discount Percent", examples: [2]string{"", ""}},
}

// TemplateCSV renders the starter sheet somebody fills in instead of guessing
// at column names.
//
// The two example rows are meant to be replaced; they exist because a header
// row alone does not show that "Origin Type" takes the kind of place and
// "Origin" takes the place itself.
func TemplateCSV() string {
	records := make([][]string, 0, 3)

	headers := make([]string, 0, len(templateColumns))
	for _, column := range templateColumns {
		headers = append(headers, column.header)
	}
	records = append(records, headers)

	for exampleIndex := range 2 {
		row := make([]string, 0, len(templateColumns))
		for _, column := range templateColumns {
			row = append(row, column.examples[exampleIndex])
		}
		records = append(records, row)
	}

	var b strings.Builder
	writer := csv.NewWriter(&b)

	// WriteAll only fails when the underlying writer does, and a Builder never
	// does.
	_ = writer.WriteAll(records)

	return b.String()
}
