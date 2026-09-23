package toolschema

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldErrors(t *testing.T, err error) map[string]string {
	t.Helper()

	require.Error(t, err)
	multiErr, ok := err.(*errortypes.MultiError)
	require.True(t, ok, "expected field-level errors, got %T: %v", err, err)

	out := make(map[string]string, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		out[entry.Field] = entry.Message
	}

	return out
}

func TestValidate_AcceptsArgumentsThatFit(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Validate(notifyDriverSchema(), map[string]any{
		"workerId": "wrk_1", "title": "Delivery moved", "message": "Be there at 3.",
		"priority": "high", "withinDays": 3, "codes": []any{"A"},
	}))
}

// Each thing wrong is filed under the field it is wrong in, so a form can
// put the message beside the value rather than at the top.
func TestValidate_FilesEachProblemUnderItsField(t *testing.T) {
	t.Parallel()

	errs := fieldErrors(t, Validate(notifyDriverSchema(), map[string]any{
		"workerId":   "wrk_1",
		"title":      "x",
		"priority":   "urgent",
		"withinDays": 0,
		"urgent":     "yes",
		"bogus":      1,
	}))

	assert.Contains(t, errs, "message", "a missing required value is filed under its own name")
	assert.Equal(t, "This value is required", errs["message"])
	assert.Contains(t, errs, "priority")
	assert.Contains(t, errs, "withinDays")
	assert.Contains(t, errs, "urgent")
	assert.Equal(t, "This tool does not take this value", errs["bogus"])
	assert.NotContains(t, errs, "params", "the root's summary is not repeated")
}

func TestValidate_TakesGoNumbersAndSchemaLists(t *testing.T) {
	t.Parallel()

	// Arguments and schemas written in Go hold ints and []string where the
	// validator wants floats and []any; both are normalized on the way in.
	assert.NoError(t, Validate(map[string]any{
		"type":       "object",
		"properties": map[string]any{"n": map[string]any{"type": "integer", "minimum": 1}},
		"required":   []string{"n"},
	}, map[string]any{"n": int64(2)}))
}

func TestValidate_AcceptsAnythingForAnEmptySchema(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Validate(nil, map[string]any{"anything": true}))
}

func TestChanged_KeepsOnlyWhatDiffers(t *testing.T) {
	t.Parallel()

	proposed := map[string]any{
		"title":      "A",
		"withinDays": float64(3),
		"codes":      []any{"x"},
		"note":       nil,
	}
	changed := Changed(proposed, map[string]any{
		"title":      "A",
		"withinDays": 3,
		"codes":      []any{"x", "y"},
		"priority":   "high",
		"missing":    nil,
		"note":       nil,
	})

	assert.Equal(t, map[string]any{"codes": []any{"x", "y"}, "priority": "high"}, changed)
}
