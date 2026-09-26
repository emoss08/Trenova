package assistantservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	insightID       = "inst_01M37R101VKZTB7TSKR30FJ0AT"
	detectedOn      = float64(1790187600)
	windowStart     = float64(1787613600)
	windowEnd       = float64(1790205600)
	shipmentPulid   = "shp_01M37R101VKZTB7TSKR30FJ0AT"
	workerPulid     = "wrk_01M37R101VKZTB7TSKR30FJ0AT"
	customerPulid   = "cus_01M37R101VKZTB7TSKR30FJ0AT"
	organizationID  = "org_01M37R101VKZTB7TSKR30FJ0AT"
	businessUnitID  = "bu_01M37R101VKZTB7TSKR30FJ0AT"
	medicalCardNote = "none on file"
)

func columnKeys(columns []assistantartifact.DisplayColumn) []string {
	keys := make([]string, 0, len(columns))
	for _, column := range columns {
		keys = append(keys, column.Key)
	}

	return keys
}

func columnByKey(
	t *testing.T,
	columns []assistantartifact.DisplayColumn,
	key string,
) assistantartifact.DisplayColumn {
	t.Helper()

	for _, column := range columns {
		if column.Key == key {
			return column
		}
	}
	require.Failf(t, "no column", "no column %q", key)

	return assistantartifact.DisplayColumn{}
}

// insightListResult is list_insights as the model received it, row for row.
func insightListResult() map[string]any {
	return map[string]any{
		"count":       float64(1),
		"searchedFor": []any{"status is Active"},
		"columns": []any{
			"id", "category", "severity", "status", "subject", "headline", "narrative",
			"recommendation", "metrics", "links", "windowStart", "windowEnd", "detectedOn",
			"stale", "dismissReason",
		},
		"items": []any{map[string]any{
			"id":             insightID,
			"category":       "Compliance",
			"severity":       "Critical",
			"status":         "Active",
			"subject":        "Medical cards",
			"headline":       "3 workers' medical cards expire within 14 days",
			"narrative":      "Three drivers on the active roster hold medical certificates that lapse before the end of the month; the first lapses today.",
			"recommendation": "Schedule DOT physicals for the three drivers this week.",
			"metrics": []any{
				map[string]any{
					"direction": "HigherIsWorse",
					"label":     "Workers affected",
					"unit":      "Count",
					"value":     "3",
				},
				map[string]any{
					"direction": "LowerIsWorse",
					"label":     "First expiry in",
					"unit":      "Days",
					"value":     "0",
				},
			},
			"links":       []any{map[string]any{"count": float64(3), "label": "Workers", "path": "/hr/workers"}},
			"windowStart": windowStart,
			"windowEnd":   windowEnd,
			"detectedOn":  detectedOn,
			"stale":       false,
		}},
	}
}

/*
The table the owner saw beside the conversation: an id column of PULIDs, a
metrics cell of raw JSON with "HigherIsWorse" in it, a detected date of
1790187600 and a "stale: false" on every row. The finding reads as the finding
now — what it is, how bad, when — and nothing the model needed and a person
does not survives into the stored payload.
*/
func TestTableArtifact_ReadsAFindingAsAPersonDoes(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("list_insights", insightListResult()))
	require.NotNil(t, artifact)

	columns, ok := artifact.Payload["columns"].([]assistantartifact.DisplayColumn)
	require.True(t, ok)
	assert.Equal(t, []string{
		"headline", "subject", "category", "severity", "status", "narrative",
		"recommendation", "metrics", "links", "windowStart", "windowEnd", "detectedOn",
	}, columnKeys(columns), "the finding leads; ids, the stale flag and empty columns go")

	assert.Equal(t, assistantartifact.DisplayEnum, columnByKey(t, columns, "category").Type)
	assert.Equal(t, assistantartifact.DisplayStatus, columnByKey(t, columns, "severity").Type)
	assert.Equal(t, assistantartifact.DisplayLongText, columnByKey(t, columns, "narrative").Type)
	assert.Equal(t, assistantartifact.DisplayMetrics, columnByKey(t, columns, "metrics").Type)
	assert.Equal(t, assistantartifact.DisplayLinks, columnByKey(t, columns, "links").Type)
	assert.Equal(t, assistantartifact.DisplayDateTime, columnByKey(t, columns, "windowStart").Type)
	assert.Equal(t, "Window start", columnByKey(t, columns, "windowStart").Label)
	detected := columnByKey(t, columns, "detectedOn")
	assert.Equal(t, assistantartifact.DisplayDate, detected.Type)
	assert.Equal(t, "Detected", detected.Label)

	rows, ok := artifact.Payload["rows"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, row, "id", "an insight has no page, so its row keeps no id at all")
	assert.Equal(t, int64(1790187600), row["detectedOn"])
	assert.Equal(t, []map[string]any{
		{"label": "Workers affected", "value": "3", "unit": "Count"},
		{"label": "First expiry in", "value": "0", "unit": "Days"},
	}, row["metrics"])
	assert.Equal(t, []map[string]any{
		{"label": "Workers", "path": "/hr/workers", "count": float64(3)},
	}, row["links"])
	assert.NotContains(t, artifact.Payload, "recordEntity")

	encoded, err := sonic.Marshal(artifact.Payload)
	require.NoError(t, err)
	for _, unreadable := range []string{insightID, "HigherIsWorse", "LowerIsWorse", "direction", `"stale"`} {
		assert.NotContains(t, string(encoded), unreadable)
	}
}

// A row keeps its record's id only where the registry knows the record's
// page, and only under the key the link is built from; no column names it.
func TestTableArtifact_KeepsARecordIDOnlyToOpenIt(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("list_shipments", map[string]any{
		"count":   float64(1),
		"columns": []any{"id", "proNumber", "customerId", "status", "totalCharge", "actualShipDate", "assignedTo"},
		"items": []any{map[string]any{
			"id":             shipmentPulid,
			"proNumber":      "PRO-778",
			"customerId":     customerPulid,
			"status":         "InTransit",
			"totalCharge":    "2840.50",
			"actualShipDate": medicalCardNote,
			"assignedTo":     workerPulid,
		}},
	}))
	require.NotNil(t, artifact)

	columns, ok := artifact.Payload["columns"].([]assistantartifact.DisplayColumn)
	require.True(t, ok)
	assert.Equal(t, []string{"proNumber", "status", "totalCharge", "actualShipDate"}, columnKeys(columns),
		"an id under any name is not a column, and neither is a column of nothing but ids")
	assert.Equal(t, assistantartifact.DisplayMoney, columnByKey(t, columns, "totalCharge").Type)
	assert.Equal(t, "shipment", artifact.Payload["recordEntity"])

	rows, ok := artifact.Payload["rows"].([]any)
	require.True(t, ok)
	row, ok := rows[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, shipmentPulid, row["id"])
	assert.Equal(t, "2840.50", row["totalCharge"], "money stays the decimal the tool wrote")
	assert.NotContains(t, row, "customerId")
	assert.NotContains(t, row, "assignedTo")
}

func TestClassifyValues_ReadsEachColumnByItsNameAndItsValues(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		key    string
		values []any
		want   assistantartifact.DisplayType
		shown  bool
	}{
		{"a flag that never holds is noise", "stale", []any{false, false}, "", false},
		{"a flag that holds is said", "stale", []any{false, true}, assistantartifact.DisplayFlag, true},
		{"a yes or no", "hazardous", []any{true, false}, assistantartifact.DisplayBoolean, true},
		{"a count named like a date stays a count", "daysUntilExpiry", []any{float64(21)}, assistantartifact.DisplayNumber, true},
		{"an unset date is nothing", "medicalCardExpiry", []any{float64(0)}, "", false},
		{"the phrase for an unset date is words", "medicalCardExpiry", []any{medicalCardNote}, assistantartifact.DisplayText, true},
		{"a phrase beside real dates is a date column", "medicalCardExpiry", []any{medicalCardNote, detectedOn}, assistantartifact.DisplayDate, true},
		{"a date the runtime already wrote", "windowStart", []any{"2026-08-24 11:20 PDT (30 days ago)"}, assistantartifact.DisplayDateTime, true},
		{"an amount due is not a date", "amountDue", []any{"12.00"}, assistantartifact.DisplayText, true},
		{"reason is not a date", "reason", []any{"Driver out sick"}, assistantartifact.DisplayText, true},
		{"a total written as a decimal is money", "total", []any{"12.00"}, assistantartifact.DisplayMoney, true},
		{"a total counted is a number", "total", []any{float64(4)}, assistantartifact.DisplayNumber, true},
		{"a percentage", "onTimePercent", []any{"92.5"}, assistantartifact.DisplayPercent, true},
		{"a year is not a quantity", "year", []any{float64(2019)}, assistantartifact.DisplayText, true},
		{"a state is two letters, not a status", "state", []any{"CA"}, assistantartifact.DisplayText, true},
		{"a status with spaces is words", "assignmentBlocked", []any{"Medical card lapsed"}, assistantartifact.DisplayText, true},
		{"a type is a category", "driverType", []any{"OTR", "Local"}, assistantartifact.DisplayEnum, true},
		{"prose by its name", "recommendation", []any{"Call them."}, assistantartifact.DisplayLongText, true},
		{"words in a list", "tags", []any{[]any{"hazmat", "team"}}, assistantartifact.DisplayText, true},
		{"records nested in a row", "stops", []any{[]any{map[string]any{"city": "Reno"}}}, "", false},
		{"a nested record by its name", "customer", []any{map[string]any{"id": customerPulid, "name": "Acme"}}, assistantartifact.DisplayText, true},
		{"a nested record with no name", "rating", []any{map[string]any{"score": float64(3)}}, "", false},
		{"the tenant", "organizationId", []any{organizationID}, "", false},
		{"a version", "version", []any{float64(3)}, "", false},
		{"a snake-case id", "business_unit_id", []any{businessUnitID}, "", false},
		{"a word ending in id letters is not an id", "paid", []any{true}, assistantartifact.DisplayBoolean, true},
		{"which way is worse", "direction", []any{"HigherIsWorse"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, shown := assistantartifact.ClassifyValues(tc.key, tc.values, true)
			assert.Equal(t, tc.shown, shown)
			if tc.shown {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// A bare amount beside a method may be a percentage of the linehaul, so it
// is a figure rather than money.
func TestClassifyValues_AnAmountBesideAMethodIsNotMoney(t *testing.T) {
	t.Parallel()

	got, shown := assistantartifact.ClassifyValues("amount", []any{"12.50"}, false)
	require.True(t, shown)
	assert.NotEqual(t, assistantartifact.DisplayMoney, got)

	got, _ = assistantartifact.ClassifyValues("amount", []any{"12.50"}, true)
	assert.Equal(t, assistantartifact.DisplayMoney, got)
}

func TestDisplayLabel_SaysWhatHappenedRatherThanWhen(t *testing.T) {
	t.Parallel()

	for key, want := range map[string]string{
		"detectedOn":   "Detected",
		"createdAt":    "Created",
		"windowStart":  "Window start",
		"lastMvrCheck": "Last MVR check",
		"cdlClass":     "CDL class",
		"proNumber":    "Pro number",
	} {
		_, displayType := assistantartifact.DatedKey(key)
		if displayType == "" {
			displayType = assistantartifact.DisplayText
		}
		assert.Equal(t, want, assistantartifact.DisplayLabel(key, displayType), key)
	}
}

type cardRecord struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organizationId"`
	Version        int64          `json:"version"`
	Status         string         `json:"status"`
	ProNumber      string         `json:"proNumber"`
	Customer       map[string]any `json:"customer"`
	TotalCharge    string         `json:"totalCharge"`
	Moves          []any          `json:"moves"`
	Rule           map[string]any `json:"rule"`
}

// A card is the record's readable fields in the order the record declares
// them, with its name first: never its id, its tenancy or its version, and a
// nested record by its name rather than its structure.
func TestEntityCardArtifact_KeepsTheRecordsReadableFields(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("get_shipment", cardRecord{
		ID:             shipmentPulid,
		OrganizationID: organizationID,
		Version:        4,
		Status:         "InTransit",
		ProNumber:      "PRO-778",
		Customer:       map[string]any{"id": customerPulid, "name": "Acme"},
		TotalCharge:    "2840.50",
		Moves:          []any{map[string]any{"id": "mv_1", "sequence": float64(1)}},
		Rule:           map[string]any{"measures": "Days until the card lapses", "threshold": "14 days"},
	}))
	require.NotNil(t, artifact)
	assert.Equal(t, "Shipment PRO-778", artifact.Title)
	assert.NotContains(t, artifact.Payload, "record", "the raw record is not kept for a person")
	assert.Contains(t, artifact.Payload["path"], shipmentPulid, "the id is how the card opens")

	fields, ok := artifact.Payload["fields"].([]assistantartifact.DisplayField)
	require.True(t, ok)
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.Key)
	}
	assert.Equal(t, []string{
		"proNumber", "status", "customer", "totalCharge", "ruleMeasures", "ruleThreshold",
	}, keys)
	assert.Equal(t, "Acme", fields[2].Value)
	assert.Equal(t, "Rule measures", fields[4].Label)
}

// A record with nothing that names it is titled by its kind alone, never by
// its id.
func TestEntityCardArtifact_NeverTitlesARecordByItsID(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("get_bank_receipt", map[string]any{
		"id":     "brc_01M37R101VKZTB7TSKR30FJ0AT",
		"amount": "120.00",
	}))
	require.NotNil(t, artifact)
	assert.Equal(t, "Bank receipt", artifact.Title)
}

func TestRecordEntityOf_FindsTheRegistrysName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment", recordEntityOf("shipments"))
	assert.Equal(t, "worker", recordEntityOf("worker"))
	assert.Equal(t, "rate_matrix", recordEntityOf("rate_matrices"))
	assert.Empty(t, recordEntityOf("insights"))
	assert.Empty(t, recordEntityOf(""))
}
