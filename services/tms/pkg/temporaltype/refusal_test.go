package temporaltype

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestToDomainRefusalMarksDeliberateRefusalsFinal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		errType ErrorType
	}{
		{
			name:    "validation",
			err:     errortypes.NewValidationError("status", errortypes.ErrInvalid, "Only a draft can be posted"),
			errType: ErrorTypeInvalidInput,
		},
		{
			name:    "business rule",
			err:     errortypes.NewBusinessError("The fiscal period is closed"),
			errType: ErrorTypeInvalidInput,
		},
		{
			name:    "missing record",
			err:     errortypes.NewNotFoundError("Invoice not found"),
			errType: ErrorTypeResourceNotFound,
		},
		{
			name:    "refused permission",
			err:     errortypes.NewAuthorizationError("You cannot post invoices"),
			errType: ErrorTypePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToDomainRefusal(tt.err)

			appErr, ok := errors.AsType[*temporal.ApplicationError](got)
			require.True(t, ok)
			assert.True(t, appErr.NonRetryable())
			assert.Equal(t, tt.errType.String(), appErr.Type())
			assert.Equal(t, tt.err.Error(), appErr.Message())
			assert.ErrorIs(t, got, tt.err)
		})
	}
}

func TestToDomainRefusalLeavesRacesAndOutagesRetryable(t *testing.T) {
	t.Parallel()

	outage := errors.New("connection reset by peer")
	stale := errortypes.NewValidationError("version", errortypes.ErrVersionMismatch, "stale")
	conflict := errortypes.NewConflictError("locked by another request")

	assert.Nil(t, ToDomainRefusal(nil))
	assert.Same(t, outage, ToDomainRefusal(outage))
	assert.Equal(t, error(stale), ToDomainRefusal(stale))
	assert.Equal(t, error(conflict), ToDomainRefusal(conflict))
}
