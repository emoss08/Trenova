package templatediag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticsSortedPutsErrorsFirst(t *testing.T) {
	diags := Diagnostics{
		{Severity: SeverityWarning, Code: CodeUnknownVariable, Field: "a", Line: 1},
		{Severity: SeverityError, Code: CodeForbiddenElement, Field: "a", Line: 9},
		{Severity: SeverityError, Code: CodeForbiddenElement, Field: "a", Line: 2},
	}

	sorted := diags.Sorted()

	assert.Equal(t, SeverityError, sorted[0].Severity)
	assert.Equal(t, 2, sorted[0].Line)
	assert.Equal(t, SeverityError, sorted[1].Severity)
	assert.Equal(t, SeverityWarning, sorted[2].Severity)
}

func TestDiagnosticStringIncludesPosition(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityError,
		Code:     CodeSyntaxError,
		Field:    "bodyHtml",
		Message:  "unexpected {{end}}",
		Line:     12,
		Column:   4,
	}

	s := d.String()
	assert.Contains(t, s, "error")
	assert.Contains(t, s, CodeSyntaxError)
	assert.Contains(t, s, "bodyHtml")
	assert.Contains(t, s, "line 12")
	assert.Contains(t, s, "col 4")
	assert.Contains(t, s, "unexpected {{end}}")
}
