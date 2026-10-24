package reportjobs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestRenderFailure_ARowLimitFailsTheRunWithoutRetrying(t *testing.T) {
	t.Parallel()

	err := renderFailure(fmt.Errorf("render: %w", &render.RowLimitError{MaxRows: 500}), 1024)

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeReportTooExpensive, appErr.Type())

	runErr := runErrorFrom(err)
	assert.Equal(t, ErrTypeReportTooExpensive, runErr.Code)
	assert.Equal(t,
		"PDF output is limited to 500 rows; export this report as CSV or XLSX instead",
		runErr.Message)
}

func TestRenderFailure_AnOversizedArtifactFailsTheRunWithoutRetrying(t *testing.T) {
	t.Parallel()

	err := renderFailure(errArtifactTooLarge, 1024)

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.True(t, appErr.NonRetryable())
	assert.Equal(t, ErrTypeReportTooExpensive, appErr.Type())
}

func TestRenderFailure_LeavesOtherFailuresRetryable(t *testing.T) {
	t.Parallel()

	cause := errors.New("storage unavailable")
	err := renderFailure(cause, 1024)

	assert.Same(t, cause, err)
}
