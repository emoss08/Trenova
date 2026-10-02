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
	assert.True(t, errortypes.IsRefusal(errortypes.NewValidationError("f", errortypes.ErrInvalid, "bad")))
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
