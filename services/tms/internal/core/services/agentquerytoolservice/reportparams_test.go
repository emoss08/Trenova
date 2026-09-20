package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/stretchr/testify/assert"
)

func unbilledEntry() *canned.Entry {
	return &canned.Entry{
		Name: "Delivered, Not Billed",
		Definition: &report.Definition{
			Parameters: []report.ParameterDef{
				{Name: "windowDays", Type: "int"},
				{Name: "statuses", Type: "enum", Multi: true},
			},
		},
	}
}

// The payload from the transcript, sent five times against a report that would
// otherwise have run. The compiler is right to reject it; this is the boundary
// where a model's guess at the container gets straightened out.
func TestNormalizeReportParameters_UnwrapsTheShapeAModelInvents(t *testing.T) {
	t.Parallel()

	normalized := normalizeReportParameters(unbilledEntry(), map[string]any{
		"windowDays": "90",
		"statuses": map[string]any{
			"item": []any{"Completed", "ReadyToInvoice", "PartiallyCompleted"},
		},
	})

	assert.Equal(
		t,
		[]any{"Completed", "ReadyToInvoice", "PartiallyCompleted"},
		normalized["statuses"],
	)
	assert.Equal(t, "90", normalized["windowDays"], "only the container is corrected")
}

func TestNormalizeReportParameters_AcceptsTheOtherWrappersAndAScalar(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]any{
		"items":  map[string]any{"items": []any{"Completed"}},
		"values": map[string]any{"values": []any{"Completed"}},
		"list":   map[string]any{"list": []any{"Completed"}},
		"scalar": "Completed",
		"single": map[string]any{"value": "Completed"},
		"csv":    "Completed, ReadyToInvoice",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			normalized := normalizeReportParameters(
				unbilledEntry(),
				map[string]any{"statuses": value},
			)
			list, ok := normalized["statuses"].([]any)
			assert.True(t, ok, "a multi parameter always reaches the compiler as a list")
			assert.Contains(t, list, "Completed")
		})
	}
}

func TestNormalizeReportParameters_LeavesAProperListAlone(t *testing.T) {
	t.Parallel()

	normalized := normalizeReportParameters(unbilledEntry(), map[string]any{
		"statuses": []any{"Completed", "Invoiced"},
	})

	assert.Equal(t, []any{"Completed", "Invoiced"}, normalized["statuses"])
}

// An object with more than one key is a value the model meant as an object.
// Reaching into it would be guessing about the report, not about the shape.
func TestNormalizeReportParameters_WillNotReachIntoARealObject(t *testing.T) {
	t.Parallel()

	value := map[string]any{"item": []any{"Completed"}, "mode": "any"}
	normalized := normalizeReportParameters(unbilledEntry(), map[string]any{"statuses": value})

	assert.Equal(t, value, normalized["statuses"])
}

// The mirror image, which the same models produce just as often.
func TestNormalizeReportParameters_UnwrapsAOneItemListForASingleValue(t *testing.T) {
	t.Parallel()

	normalized := normalizeReportParameters(unbilledEntry(), map[string]any{
		"windowDays": []any{90},
	})

	assert.Equal(t, 90, normalized["windowDays"])
}

// Nothing is invented: a parameter the model did not send stays unsent, so the
// missing-parameter answer still fires and still asks the person.
func TestNormalizeReportParameters_AddsNothing(t *testing.T) {
	t.Parallel()

	normalized := normalizeReportParameters(unbilledEntry(), map[string]any{"windowDays": 90})

	assert.NotContains(t, normalized, "statuses")
	assert.Len(t, normalized, 1)
}

func TestNormalizeReportParameters_SurvivesAReportWithNoDefinition(t *testing.T) {
	t.Parallel()

	values := map[string]any{"statuses": "Completed"}
	assert.Equal(t, values, normalizeReportParameters(&canned.Entry{}, values))
	assert.Equal(t, values, normalizeReportParameters(nil, values))
}

func TestDescribeParameterShape_SaysWhatToSend(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"a JSON array of enum",
		describeParameterShape(report.ParameterDef{Type: "enum", Multi: true}),
	)
	assert.Equal(t, "a single int", describeParameterShape(report.ParameterDef{Type: "int"}))
	assert.Equal(t, "a single value", describeParameterShape(report.ParameterDef{}))
}
