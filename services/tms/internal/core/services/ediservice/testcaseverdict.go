package ediservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
)

type TestCaseRunVerdict struct {
	Passed                 bool     `json:"passed"`
	Warnings               int      `json:"warnings"`
	Errors                 int      `json:"errors"`
	MissingWarningCodes    []string `json:"missingWarningCodes"`
	UnexpectedWarningCodes []string `json:"unexpectedWarningCodes"`
	MissingErrorCodes      []string `json:"missingErrorCodes"`
	UnexpectedErrorCodes   []string `json:"unexpectedErrorCodes"`
}

func EvaluateTestCaseRun(
	testCase *edi.EDITestCase,
	diagnostics []edix12.Diagnostic,
) TestCaseRunVerdict {
	verdict := TestCaseRunVerdict{}
	warningCodes := make([]string, 0, len(diagnostics))
	errorCodes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		switch diagnostic.Severity {
		case edi.ValidationSeverityWarning:
			verdict.Warnings++
			warningCodes = append(warningCodes, diagnostic.Code)
		case edi.ValidationSeverityError:
			verdict.Errors++
			errorCodes = append(errorCodes, diagnostic.Code)
		case edi.ValidationSeverityInfo:
		}
	}

	verdict.MissingWarningCodes, verdict.UnexpectedWarningCodes = diffDiagnosticCodes(
		testCase.ExpectedWarningCodes,
		warningCodes,
	)
	verdict.MissingErrorCodes, verdict.UnexpectedErrorCodes = diffDiagnosticCodes(
		testCase.ExpectedErrorCodes,
		errorCodes,
	)
	verdict.Passed = verdict.Warnings == testCase.ExpectedWarnings &&
		verdict.Errors == testCase.ExpectedErrors &&
		len(verdict.MissingWarningCodes) == 0 &&
		len(verdict.UnexpectedWarningCodes) == 0 &&
		len(verdict.MissingErrorCodes) == 0 &&
		len(verdict.UnexpectedErrorCodes) == 0
	return verdict
}

func diffDiagnosticCodes(expected, actual []string) (missing, unexpected []string) {
	missing = []string{}
	unexpected = []string{}
	if len(expected) == 0 {
		return missing, unexpected
	}
	expectedSet := normalizeDiagnosticCodes(expected)
	actualSet := normalizeDiagnosticCodes(actual)
	for _, code := range expectedSet {
		if !slices.Contains(actualSet, code) {
			missing = append(missing, code)
		}
	}
	for _, code := range actualSet {
		if !slices.Contains(expectedSet, code) {
			unexpected = append(unexpected, code)
		}
	}
	return missing, unexpected
}
