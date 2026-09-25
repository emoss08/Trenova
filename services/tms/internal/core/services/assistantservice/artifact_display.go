package assistantservice

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/stringutils"
)

/*
A tool's result is written for the model, and the model needs what a person
does not: every record's id so the next call can name it, the tenant, the
version a write checks, which way a metric counts as worse. Handed to a person
as a table, that was a column of "inst_01M37R101VKZTB7TSKR30FJ0AT", a cell of
`[{"direction":"HigherIsWorse","label":"Workers affected",…}]` and a detected
date of 1790187600.

So an artifact keeps a projection of the result rather than the result: each
column a person can reason with, with the type that says how to draw it, and
nothing else. The model's copy is untouched. The one identifier a row keeps is
its own record's, under a key no column names, and only when that kind of
record has a page to open — the link is built from it and it is never shown.

The client holds the same rules for artifacts stored before this, so an old
table reads the same as a new one.
*/

// recordIDKey is where a projected row keeps the id its link is built from.
const recordIDKey = "id"

// leadKeys name a record, most specific first; a table leads with them.
var leadKeys = []string{
	"proNumber",
	"invoiceNumber",
	"number",
	"referenceNumber",
	"name",
	"fullName",
	"displayName",
	"title",
	"headline",
	"subject",
	"label",
	"code",
	"unitNumber",
	"licenseNumber",
}

// tableProjection is a list result as a person reads it.
type tableProjection struct {
	columns      []assistantartifact.DisplayColumn
	rows         []any
	recordEntity string
}

// projectTable reads rows against the columns the tool declared, keeping the
// columns that have something readable in them and the values that fill them.
func projectTable(entity string, declared []string, rows []any) tableProjection {
	records := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if record, ok := row.(map[string]any); ok {
			records = append(records, record)
		}
	}

	amountIsMoney := !slices.Contains(declared, "method")
	columns := make([]assistantartifact.DisplayColumn, 0, len(declared))
	values := make([]any, len(records))
	for _, key := range leadFirst(declared) {
		for i, record := range records {
			values[i] = record[key]
		}
		displayType, shown := assistantartifact.ClassifyValues(key, values, amountIsMoney)
		if !shown {
			continue
		}
		columns = append(columns, assistantartifact.DisplayColumn{
			Key:   key,
			Label: assistantartifact.DisplayLabel(key, displayType),
			Type:  displayType,
		})
	}

	recordEntity := recordEntityOf(entity)
	projected := make([]any, 0, len(records))
	for _, record := range records {
		row := make(map[string]any, len(columns)+1)
		for _, column := range columns {
			if value, ok := assistantartifact.ProjectValue(column.Type, record[column.Key]); ok {
				row[column.Key] = value
			}
		}
		if recordEntity != "" {
			if id := stringOf(record[recordIDKey]); id != "" {
				row[recordIDKey] = id
			}
		}
		projected = append(projected, row)
	}

	return tableProjection{columns: columns, rows: projected, recordEntity: recordEntity}
}

// projectRecord reads one record as labelled values, in the order the record
// declares its fields with the ones that name it first. A nested record is
// read by its name; a nested set of plain sentences — a rule's wording — is
// read field by field; anything else nested is structure, not a fact.
func projectRecord(record map[string]any, order []string) []assistantartifact.DisplayField {
	amountIsMoney := record["method"] == nil
	fields := make([]assistantartifact.DisplayField, 0, len(order))
	single := make([]any, 1)

	appendField := func(key string, value any) {
		single[0] = value
		displayType, shown := assistantartifact.ClassifyValues(key, single, amountIsMoney)
		if !shown {
			return
		}
		projected, ok := assistantartifact.ProjectValue(displayType, value)
		if !ok {
			return
		}
		fields = append(fields, assistantartifact.DisplayField{
			DisplayColumn: assistantartifact.DisplayColumn{
				Key:   key,
				Label: assistantartifact.DisplayLabel(key, displayType),
				Type:  displayType,
			},
			Value: projected,
		})
	}

	for _, key := range leadFirst(order) {
		value := record[key]
		nested, isRecord := value.(map[string]any)
		if isRecord && !assistantartifact.HiddenKey(key) &&
			assistantartifact.RecordLabel(nested) == "" {
			for _, child := range slices.Sorted(maps.Keys(nested)) {
				if text, isText := nested[child].(string); isText {
					appendField(key+stringutils.CapitalizeFirst(child), text)
				}
			}
			continue
		}
		appendField(key, value)
	}

	return fields
}

// recordEntityOf is the record-link registry's name for a list's entity, when
// its records have a page: "shipments" opens as "shipment".
func recordEntityOf(entity string) string {
	for _, candidate := range []string{entity, singular(entity)} {
		if candidate == "" {
			continue
		}
		if _, ok := productguide.Default.Record(candidate); ok {
			return candidate
		}
	}

	return ""
}

func singular(plural string) string {
	switch {
	case strings.HasSuffix(plural, "ices"):
		return strings.TrimSuffix(plural, "ices") + "ix"
	case strings.HasSuffix(plural, "ies"):
		return strings.TrimSuffix(plural, "ies") + "y"
	case strings.HasSuffix(plural, "sses"):
		return strings.TrimSuffix(plural, "es")
	case strings.HasSuffix(plural, "s"):
		return strings.TrimSuffix(plural, "s")
	default:
		return plural
	}
}

// leadFirst moves the keys that name a record to the front and keeps the rest
// where the record declared them.
func leadFirst(keys []string) []string {
	ordered := make([]string, 0, len(keys))
	for _, lead := range leadKeys {
		if slices.Contains(keys, lead) {
			ordered = append(ordered, lead)
		}
	}
	for _, key := range keys {
		if !slices.Contains(leadKeys, key) {
			ordered = append(ordered, key)
		}
	}

	return ordered
}
