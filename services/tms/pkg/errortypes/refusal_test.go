package errortypes_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestIsRefusal(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")

	assert.True(t, errortypes.IsRefusal(errortypes.NewBusinessError("no")))
	assert.True(t, errortypes.IsRefusal(errortypes.NewNotFoundError("missing")))
	assert.True(t, errortypes.IsRefusal(errortypes.NewConflictError("busy")))
	assert.True(
		t,
		errortypes.IsRefusal(errortypes.NewValidationError("f", errortypes.ErrInvalid, "bad")),
	)
	assert.True(t, errortypes.IsRefusal(multiErr))
	assert.False(t, errortypes.IsRefusal(errors.New("connection reset")))
	assert.False(t, errortypes.IsRefusal(nil))
}

func TestSummary(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("a", errortypes.ErrRequired, "First is required")
	multiErr.Add("b", errortypes.ErrInvalid, "Second is invalid")

	assert.Equal(t, "First is required; Second is invalid", errortypes.Summary(multiErr))
	assert.Equal(t, "This load has been canceled",
		errortypes.Summary(errortypes.NewBusinessError("This load has been canceled")))
	assert.Empty(t, errortypes.Summary(nil))
}

func TestSummary_SaysWhereAnIndexedFieldIsInWords(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"moves[0].stops[1].scheduledWindowEnd",
		errortypes.ErrInvalid,
		"Scheduled window end must be before the next stop's scheduled window start",
	)
	multiErr.Add("additionalCharges[2].amount", errortypes.ErrInvalid, "Amount must be positive")

	assert.Equal(
		t,
		"Move 1, stop 2: Scheduled window end must be before the next stop's scheduled window start; "+
			"Additional charge 3: Amount must be positive",
		errortypes.Summary(multiErr),
	)
}

func TestSummaryOfText_RewordsStoredValidationTextAndLeavesOtherTextAlone(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"moves[0].stops[0].scheduledWindowEnd",
		errortypes.ErrInvalid,
		"Scheduled window end must be before the next stop's scheduled window start",
	)
	multiErr.Add("bol", errortypes.ErrRequired, "BOL is required")

	assert.Equal(
		t,
		"Move 1, stop 1: Scheduled window end must be before the next stop's scheduled window start; "+
			"BOL is required",
		errortypes.SummaryOfText(multiErr.Error()),
	)
	assert.Equal(t, "The move changed since this was proposed",
		errortypes.SummaryOfText("  The move changed since this was proposed "))
	assert.Empty(t, errortypes.SummaryOfText(""))
}
