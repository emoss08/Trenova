package templateengine

import (
	"fmt"
	"sort"
	"strings"
)

// Severity separates "this will not ship" from "you probably meant something
// else". Draft saves tolerate warnings; publishing does not.
type Severity string

const (
	SeverityError   = Severity("error")
	SeverityWarning = Severity("warning")
)

func (s Severity) String() string { return string(s) }

// Diagnostic codes. These are stable identifiers the admin UI keys help text and
// remediation hints off, so renaming one is a breaking change to the client.
const (
	// CodeTemplateTooLarge means the source exceeded the configured byte cap.
	CodeTemplateTooLarge = "template_too_large"

	// CodeSyntaxError means the template did not parse.
	CodeSyntaxError = "syntax_error"

	// CodeForbiddenElement means the markup contains an element we never allow.
	CodeForbiddenElement = "forbidden_element"

	// CodeForbiddenAttribute means the markup contains an attribute we never
	// allow, which in practice means an inline event handler.
	CodeForbiddenAttribute = "forbidden_attribute"

	// CodeForbiddenURLScheme means a URL attribute used a scheme that can
	// execute or read local state.
	CodeForbiddenURLScheme = "forbidden_url_scheme"

	// CodeForbiddenCSS means a stylesheet used a construct that can fetch or
	// execute, such as @import or a non-data url().
	CodeForbiddenCSS = "forbidden_css"

	// CodeUnknownVariable means the template referenced a path the kind does not
	// publish. A warning while drafting, an error when publishing: a typo here
	// renders as blank space on a customer's invoice.
	CodeUnknownVariable = "unknown_variable"

	// CodeRequiredVariableMissing means the template dropped a path the kind
	// declares mandatory, such as an invoice total.
	CodeRequiredVariableMissing = "required_variable_missing"

	// CodeNestedTemplateForbidden means the source used define/template/block. A
	// version is exactly one template.
	CodeNestedTemplateForbidden = "nested_template_forbidden"

	// CodeUnknownFunction means the template called a function outside the
	// allowed set.
	CodeUnknownFunction = "unknown_function"

	// CodeUnsafeInterpolationContext means a variable sits somewhere the HTML
	// escaper refuses to reason about, so it would render as ZgotmplZ.
	CodeUnsafeInterpolationContext = "unsafe_interpolation_context"

	// CodeSampleRenderFailed means the template parsed but could not execute
	// against the kind's sample data.
	CodeSampleRenderFailed = "sample_render_failed"

	// CodeOutputTooLarge means rendering exceeded the output byte budget.
	CodeOutputTooLarge = "output_too_large"

	// CodeRemoteAssetBlocked means an asset reference was dropped because it
	// pointed somewhere we will not fetch from.
	CodeRemoteAssetBlocked = "remote_asset_blocked"

	// CodeAssetBudgetExceeded means an asset was dropped because the document
	// already used its allowance.
	CodeAssetBudgetExceeded = "asset_budget_exceeded"
)

// Diagnostic is one problem found in a template, addressed to the person editing
// it. Line and Column are 1-based and zero when the problem is not positional.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Field    string   `json:"field"`
	Message  string   `json:"message"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
}

func (d Diagnostic) String() string {
	var b strings.Builder
	b.WriteString(string(d.Severity))
	b.WriteString(": ")
	b.WriteString(d.Code)
	if d.Field != "" {
		b.WriteString(" [")
		b.WriteString(d.Field)
		b.WriteString("]")
	}
	if d.Line > 0 {
		fmt.Fprintf(&b, " (line %d", d.Line)
		if d.Column > 0 {
			fmt.Fprintf(&b, ", col %d", d.Column)
		}
		b.WriteString(")")
	}
	b.WriteString(": ")
	b.WriteString(d.Message)
	return b.String()
}

// Diagnostics is an ordered set of findings.
type Diagnostics []Diagnostic

// HasErrors reports whether anything blocks use of the template.
func (d Diagnostics) HasErrors() bool {
	for i := range d {
		if d[i].Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only the blocking findings.
func (d Diagnostics) Errors() Diagnostics {
	out := make(Diagnostics, 0, len(d))
	for i := range d {
		if d[i].Severity == SeverityError {
			out = append(out, d[i])
		}
	}
	return out
}

// Warnings returns only the advisory findings.
func (d Diagnostics) Warnings() Diagnostics {
	out := make(Diagnostics, 0, len(d))
	for i := range d {
		if d[i].Severity == SeverityWarning {
			out = append(out, d[i])
		}
	}
	return out
}

// Sorted returns the diagnostics ordered for display: errors first, then by
// position, so the editor can jump to the first thing worth fixing.
func (d Diagnostics) Sorted() Diagnostics {
	out := make(Diagnostics, len(d))
	copy(out, d)

	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Severity == SeverityError) != (out[j].Severity == SeverityError) {
			return out[i].Severity == SeverityError
		}
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Column < out[j].Column
	})

	return out
}

// Summary renders the blocking findings as a single message, for the error paths
// that cannot carry structure.
//
// Deliberately not named Error: Diagnostics is a findings list, not an error, and
// giving it an Error method would let it satisfy the error interface and be
// returned where a real error belongs — including when it holds only warnings.
func (d Diagnostics) Summary() string {
	errs := d.Errors()
	if len(errs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(errs))
	for i := range errs {
		parts = append(parts, errs[i].String())
	}
	return strings.Join(parts, "; ")
}

func errorDiag(code, field, message string) Diagnostic {
	return Diagnostic{Severity: SeverityError, Code: code, Field: field, Message: message}
}

// positionedError is an errorDiag that can point the editor at a line.
func positionedError(code, field, message string, line, column int) Diagnostic {
	return Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Field:    field,
		Message:  message,
		Line:     line,
		Column:   column,
	}
}
