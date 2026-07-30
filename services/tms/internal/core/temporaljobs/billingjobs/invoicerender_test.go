package billingjobs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/require"
)

// A broken template and a downed renderer both surface as "the PDF failed", but
// they need opposite handling: one needs an administrator, the other needs a
// moment. Getting this wrong means either burning five attempts on a template
// that will never render, or giving up on a sidecar that was mid-restart.
func TestABrokenTemplateIsNotRetried(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add("bodyHtml", errortypes.ErrInvalid, "Line 4: unknown variable .Totl")

	classified := classifyInvoiceRenderError(multiErr)

	var appErr *temporaltype.ApplicationError
	require.ErrorAs(t, classified, &appErr)
	require.Equal(t, temporaltype.ErrorTypeTemplateInvalid, appErr.Type)
	require.False(t, appErr.Retryable)
	require.Contains(t, generateInvoicePDFRetryPolicy.NonRetryableErrorTypes,
		temporaltype.ErrorTypeTemplateInvalid.String())
}

func TestAnUnavailableRendererIsRetried(t *testing.T) {
	t.Parallel()

	classified := classifyInvoiceRenderError(
		fmt.Errorf("render invoice: %w", services.ErrPDFRendererUnavailable),
	)

	var appErr *temporaltype.ApplicationError
	require.ErrorAs(t, classified, &appErr)
	require.Equal(t, temporaltype.ErrorTypeRetryable, appErr.Type)
	require.True(t, appErr.Retryable)
	require.NotContains(t, generateInvoicePDFRetryPolicy.NonRetryableErrorTypes,
		temporaltype.ErrorTypeRetryable.String())
}

// Anything the classifier does not recognise keeps its original handling rather
// than being forced into a bucket that might be wrong.
func TestAnUnrecognisedFailurePassesThroughUnchanged(t *testing.T) {
	t.Parallel()

	original := errors.New("connection reset by peer")

	require.Equal(t, original, classifyInvoiceRenderError(original))
	require.NoError(t, classifyInvoiceRenderError(nil))
}
