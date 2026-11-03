package temporaltype

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/keel"
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

func TestToActivityFailureMakesFinalRefusalsNonRetryable(t *testing.T) {
	t.Parallel()

	missing := errortypes.NewNotFoundError("The document's file could not be found in storage")
	tests := []struct {
		name    string
		err     error
		errType ErrorType
	}{
		{
			name:    "missing record, wrapped",
			err:     fmt.Errorf("probe the PDF before converting it: %w", missing),
			errType: ErrorTypeResourceNotFound,
		},
		{
			name:    "validation",
			err:     errortypes.NewValidationError("status", errortypes.ErrInvalid, "Only a draft can be posted"),
			errType: ErrorTypeInvalidInput,
		},
		{
			name:    "refused permission",
			err:     errortypes.NewAuthorizationError("You cannot post invoices"),
			errType: ErrorTypePermissionDenied,
		},
		{
			name:    "list without a limit",
			err:     fmt.Errorf("list narrated insights: %w", keel.ErrUnbounded),
			errType: ErrorTypeNonRetryable,
		},
		{
			name:    "cross-tenant statement without AllTenants",
			err:     fmt.Errorf("roll up usage: %w: carrier_intel_usage_ledger", keel.ErrAllTenantsRequired),
			errType: ErrorTypeNonRetryable,
		},
		{
			name:    "plan quota",
			err:     errortypes.NewQuotaExceededError("documents", 10, 10, "starter"),
			errType: ErrorTypeQuotaExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToActivityFailure(tt.err)

			appErr, ok := errors.AsType[*temporal.ApplicationError](got)
			require.True(t, ok)
			assert.True(t, appErr.NonRetryable())
			assert.Equal(t, tt.errType.String(), appErr.Type())
		})
	}
}

func TestToActivityFailureLeavesWhatMightChangeRetryable(t *testing.T) {
	t.Parallel()

	outage := errors.New("connection reset by peer")
	stale := errortypes.NewValidationError("version", errortypes.ErrVersionMismatch, "stale")
	conflict := errortypes.NewConflictError("locked by another request")
	business := errortypes.NewBusinessError("Nova could not reach an AI provider just now. Try again in a moment.")
	database := errortypes.NewDatabaseError("Failed to download document")

	assert.Nil(t, ToActivityFailure(nil))
	for name, err := range map[string]error{
		"outage":           outage,
		"stale version":    stale,
		"conflict":         conflict,
		"business message": business,
		"database":         database,
	} {
		assert.Same(t, err, ToActivityFailure(err), name)
	}
}

func TestToActivityFailureKeepsTheActivitysOwnClassification(t *testing.T) {
	t.Parallel()

	missing := errortypes.NewNotFoundError("Invoice not found")
	retryable := temporal.NewApplicationErrorWithCause("converter restarting", "converter_unavailable", missing)
	assert.Same(t, retryable, ToActivityFailure(retryable),
		"a Temporal ApplicationError already states whether it retries")

	own := NewRetryableError("Failed to list accounting controls", missing)
	got := ToActivityFailure(own)
	appErr, ok := errors.AsType[*temporal.ApplicationError](got)
	require.True(t, ok)
	assert.False(t, appErr.NonRetryable(), "a temporaltype error keeps its Retryable choice")
	assert.Equal(t, ErrorTypeRetryable.String(), appErr.Type())
}
