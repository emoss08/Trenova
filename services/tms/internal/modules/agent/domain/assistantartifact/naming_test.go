package assistantartifact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeTitle(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"  late loads for Peak Distributing ": "Late loads for Peak Distributing",
		"**Shipment SEED-SHP-007 brief**":     "Shipment SEED-SHP-007 brief",
		"## Storm impact on today's loads":    "Storm impact on today's loads",
		"\"Weekly summary\".":                 "Weekly summary",
		"Queue [draft]":                       "Queue draft",
		"Line one\nline two":                  "Line one line two",
		"`code` title:":                       "Code title",
		"":                                    untitledTitle,
		"   ":                                 untitledTitle,
		"**":                                  untitledTitle,
		"S26100090038417 · Peak Distributing": "S26100090038417 · Peak Distributing",
	}
	for input, want := range cases {
		assert.Equal(t, want, NormalizeTitle(input), input)
	}

	long := NormalizeTitle(strings.Repeat("a", MaxTitleRunes+40))
	assert.Equal(t, MaxTitleRunes, len([]rune(long)))
}

func TestRecordTitle(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Formula template Per CWT (Hundredweight)",
		RecordTitle("formula_template", "Per CWT (Hundredweight)"))
	assert.Equal(t, "Shipment S26100090038417", RecordTitle("shipment", "S26100090038417"))
	assert.Equal(t, "DOT random pool Drivers 2026", RecordTitle("dot_random_pool", "Drivers 2026"))
	assert.Equal(t, "Billing queue item", RecordTitle("billing_queue_item", ""))
}

func TestRecordsTitle(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"shipments":             "Shipments",
		"shipment":              "Shipments",
		"formula_templates":     "Formula templates",
		"billing_queue_items":   "Billing queue items",
		"clearinghouse_queries": "Clearinghouse queries",
		"workers":               "Workers",
		"dot_random_pools":      "DOT random pools",
		"invoices":              "Invoices",
		"rate_matrices":         "Rate matrices",
	}
	for input, want := range cases {
		assert.Equal(t, want, RecordsTitle(input), input)
	}
}

func TestPhraseTitle(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Assign move", PhraseTitle("assign_move"))
	assert.Equal(t, "Send EDI status", PhraseTitle("send_edi_status"))
	assert.Equal(t, "IFTA quarterly", PhraseTitle("ifta_quarterly"))
}
