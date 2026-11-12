package filterwords

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	filterCustomerA = "cus_01JW3K18A8TPC7T7SPX7745V1H"
	filterCustomerB = "cus_01JSV4G0Q80KNYKQ254QWT5T0V"
	filterCustomerC = "cus_01J3TA9AYRY47X0A6SSB8D0M0H"
)

func shown(filters []Filter) [][2]string {
	out := make([][2]string, 0, len(filters))
	for _, filter := range filters {
		out = append(out, [2]string{filter.Label, filter.Value})
	}

	return out
}

func TestOf_AFieldsFilterReadsAsItsLabelAndWords(t *testing.T) {
	t.Parallel()

	filters := Of(
		[]string{
			"status one of New, PartiallyAssigned, InTransit",
			"serviceType equals LessThanTruckload",
			"name contains McDonald",
			"createdAt within the last 7 days",
		},
		map[string]any{"filters": []any{
			map[string]any{"field": "status", "operator": "in"},
			map[string]any{"field": "serviceType", "operator": "eq"},
			map[string]any{"field": "name", "operator": "contains"},
			map[string]any{"field": "createdAt", "operator": "lastndays"},
		}},
	)

	assert.Equal(t, [][2]string{
		{"Status", "New, Partially assigned, In transit"},
		{"Service type", "Less than truckload"},
		{"Name", "contains McDonald"},
		{"Created at", "within the last 7 days"},
	}, shown(filters))
}

func TestOf_AToolsOwnPhraseKeepsItsWords(t *testing.T) {
	t.Parallel()

	filters := Of(
		[]string{
			`text matching "granite"`,
			"with a hold in force for any reason",
			"late running late: a stop past its window with no arrival",
			"",
		},
		map[string]any{"query": "granite", "late": true},
	)

	assert.Equal(t, [][2]string{
		{"", `text matching "granite"`},
		{"", "with a hold in force for any reason"},
		{"Late", "running late: a stop past its window with no arrival"},
	}, shown(filters))
}

func TestOfResult_ReadsTheTermsAListReported(t *testing.T) {
	t.Parallel()

	arguments := map[string]any{"status": "Delayed"}

	decoded := OfResult(map[string]any{"searchedFor": []any{"status equals Delayed", 7}}, arguments)
	assert.Equal(t, [][2]string{{"Status", "Delayed"}}, shown(decoded))

	typed := OfResult(map[string]any{"searchedFor": []string{"status equals Delayed"}}, arguments)
	assert.Equal(t, [][2]string{{"Status", "Delayed"}}, shown(typed))

	assert.Empty(t, OfResult(map[string]any{"name": "Granite Textiles"}, arguments))
	assert.Empty(t, OfResult([]any{"status equals Delayed"}, arguments))
}

func TestOf_AnUnfilteredListHasNoFilters(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Of([]string{"no filters: the most recent shipments"}, nil))
}

func TestOf_ARecordIDIsNeverShown(t *testing.T) {
	t.Parallel()

	arguments := map[string]any{"customerId": []any{filterCustomerA, filterCustomerB}}

	filters := Of(
		[]string{"customerId one of " + filterCustomerA + ", " + filterCustomerB},
		arguments,
	)
	assert.Equal(t, [][2]string{{"Customer", "2 selected"}}, shown(filters))

	phrase := Of([]string{"billed to " + filterCustomerA}, arguments)
	assert.Equal(t, [][2]string{{"", "billed to (1 selected)"}}, shown(phrase))
}

func TestName_RecordsAreNamedAndTheRestCounted(t *testing.T) {
	t.Parallel()

	filters := Of(
		[]string{
			"customerId one of " + filterCustomerA + ", " + filterCustomerB + ", " + filterCustomerC,
			"status equals Delayed",
		},
		map[string]any{"customerId": []any{}, "status": "Delayed"},
	)

	assert.Equal(t, []string{filterCustomerA, filterCustomerB, filterCustomerC}, Records(filters))

	names := map[string]string{
		filterCustomerA: "Granite Textiles",
		filterCustomerC: "Granite Brands",
	}
	Name(filters, func(id string) string { return names[id] })

	assert.Equal(t, [][2]string{
		{"Customer", "Granite Textiles, Granite Brands and 1 more"},
		{"Status", "Delayed"},
	}, shown(filters))
	assert.Equal(t,
		"Customer Granite Textiles, Granite Brands and 1 more · Status Delayed",
		Line(filters),
	)
}

func TestRecords_NoFiltersNameNone(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Records(nil))
	assert.Empty(t, Line(nil))
}
