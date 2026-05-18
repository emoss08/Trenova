package edistarlark

import (
	"time"

	"go.starlark.net/starlark"
)

const (
	DefaultMaxExecutionSteps = uint64(100_000)
	DefaultTimeout           = 100 * time.Millisecond
	timeoutCancelGrace       = 10 * time.Millisecond

	defaultFunctionName = "value"
	defaultFilename     = "edi_element.star"
)

const (
	DiagnosticCodeLibraryDuplicateFunction = "script_library_duplicate_function"
	DiagnosticCodeLibraryReservedFunction  = "script_library_reserved_function"
	DiagnosticCodeLibrarySyntaxError       = "script_library_syntax_error"
	DiagnosticCodeFunctionNotFound         = "script_function_not_found"
	DiagnosticCodeFunctionNotCallable      = "script_function_not_callable"
)

type Options struct {
	MaxExecutionSteps uint64
	Timeout           time.Duration
}

type Evaluator struct {
	options     Options
	predeclared starlark.StringDict
}

type EvalRequest struct {
	Script          string
	FunctionName    string
	Libraries       []ScriptLibrary
	Context         map[string]any
	Item            any
	SegmentID       string
	ElementPosition int
	Path            string
}

type ScriptLibrary struct {
	Name   string
	Script string
}

type EvalResult struct {
	Value          string
	Raw            starlark.Value
	Diagnostics    []Diagnostic
	ExecutionSteps uint64
}

type Diagnostic struct {
	Severity        DiagnosticSeverity `json:"severity"`
	Code            string             `json:"code"`
	SegmentID       string             `json:"segmentId"`
	ElementPosition int                `json:"elementPosition"`
	Path            string             `json:"path"`
	Message         string             `json:"message"`
	SuggestedFix    string             `json:"suggestedFix"`
}

type DiagnosticSeverity string

const (
	DiagnosticSeverityError   = DiagnosticSeverity("Error")
	DiagnosticSeverityWarning = DiagnosticSeverity("Warning")
	DiagnosticSeverityInfo    = DiagnosticSeverity("Info")
)
