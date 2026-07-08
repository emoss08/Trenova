package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateTestCaseRun(t *testing.T) {
	t.Parallel()

	diagnostics := []edix12.Diagnostic{
		{Severity: edi.ValidationSeverityWarning, Code: "missing_optional_element"},
		{Severity: edi.ValidationSeverityError, Code: "invalid_date_format"},
		{Severity: edi.ValidationSeverityInfo, Code: "informational"},
	}

	countsOnly := EvaluateTestCaseRun(&edi.EDITestCase{
		ExpectedWarnings: 1,
		ExpectedErrors:   1,
	}, diagnostics)
	assert.True(t, countsOnly.Passed)
	assert.Equal(t, 1, countsOnly.Warnings)
	assert.Equal(t, 1, countsOnly.Errors)
	assert.Empty(t, countsOnly.MissingWarningCodes)
	assert.Empty(t, countsOnly.UnexpectedErrorCodes)

	setMismatch := EvaluateTestCaseRun(&edi.EDITestCase{
		ExpectedWarnings:     1,
		ExpectedErrors:       1,
		ExpectedWarningCodes: []string{"missing_optional_element"},
		ExpectedErrorCodes:   []string{"segment_out_of_order"},
	}, diagnostics)
	assert.False(t, setMismatch.Passed)
	assert.Equal(t, []string{"segment_out_of_order"}, setMismatch.MissingErrorCodes)
	assert.Equal(t, []string{"invalid_date_format"}, setMismatch.UnexpectedErrorCodes)

	countMismatch := EvaluateTestCaseRun(&edi.EDITestCase{
		ExpectedWarnings: 0,
		ExpectedErrors:   1,
	}, diagnostics)
	assert.False(t, countMismatch.Passed)
}
