package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of EDI error
type ErrorType string

const (
	// Parsing errors
	ErrorTypeSyntax    = ErrorType("SYNTAX")
	ErrorTypeDelimiter = ErrorType("DELIMITER")
	ErrorTypeStructure = ErrorType("STRUCTURE")
	ErrorTypeEncoding  = ErrorType("ENCODING")

	// Validation errors
	ErrorTypeRequired    = ErrorType("REQUIRED")
	ErrorTypeFormat      = ErrorType("FORMAT")
	ErrorTypeValue       = ErrorType("VALUE")
	ErrorTypeLength      = ErrorType("LENGTH")
	ErrorTypeCardinality = ErrorType("CARDINALITY")

	// Business rule errors
	ErrorTypeBusinessRule = ErrorType("BUSINESS_RULE")
	ErrorTypeDependency   = ErrorType("DEPENDENCY")
	ErrorTypeConsistency  = ErrorType("CONSISTENCY")

	// System errors
	ErrorTypeIO       = ErrorType("IO")
	ErrorTypeTimeout  = ErrorType("TIMEOUT")
	ErrorTypeResource = ErrorType("RESOURCE")
)

// Severity represents the severity level of an error
type Severity int

const (
	SeverityInfo    Severity = 1
	SeverityWarning Severity = 2
	SeverityError   Severity = 3
	SeverityFatal   Severity = 4
)

// EDIError represents a comprehensive EDI error with context
type EDIError struct {
	Type     ErrorType
	Severity Severity
	Code     string
	Message  string
	Details  string
	Location *Location
	Context  map[string]any
	Recovery []RecoverySuggestion
	Caused   error
}

// Location provides detailed position information
type Location struct {
	SegmentTag     string
	SegmentIndex   int
	ElementIndex   int // 1-based
	ComponentIndex int // 1-based
	Line           int
	Column         int
	ByteOffset     int64
	FilePath       string
	PartnerID      string
}

// RecoverySuggestion provides actionable recovery steps
type RecoverySuggestion struct {
	Action      string
	Description string
	Example     string
	AutoFix     bool
	FixFunction func() error
}

// Error implements the error interface
func (e *EDIError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message))

	if e.Location != nil {
		sb.WriteString(fmt.Sprintf(" at %s", e.Location.String()))
	}

	if e.Details != "" {
		sb.WriteString(fmt.Sprintf(" - %s", e.Details))
	}

	if e.Caused != nil {
		sb.WriteString(fmt.Sprintf(" (caused by: %v)", e.Caused))
	}

	return sb.String()
}

// String formats location information
func (l *Location) String() string {
	parts := []string{}

	if l.SegmentTag != "" {
		part := l.SegmentTag
		if l.ElementIndex > 0 {
			part = fmt.Sprintf("%s-%02d", l.SegmentTag, l.ElementIndex)
			if l.ComponentIndex > 0 {
				part = fmt.Sprintf("%s-%02d", part, l.ComponentIndex)
			}
		}
		parts = append(parts, part)
	}

	if l.SegmentIndex >= 0 {
		parts = append(parts, fmt.Sprintf("segment %d", l.SegmentIndex))
	}

	if l.Line > 0 {
		parts = append(parts, fmt.Sprintf("line %d:%d", l.Line, l.Column))
	}

	if l.FilePath != "" {
		parts = append(parts, l.FilePath)
	}

	return strings.Join(parts, ", ")
}

// Unwrap returns the underlying error
func (e *EDIError) Unwrap() error {
	return e.Caused
}

// Is checks if the error is of a specific type
func (e *EDIError) Is(target error) bool {
	if t, ok := target.(*EDIError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// CanAutoFix checks if any recovery suggestions have auto-fix available
func (e *EDIError) CanAutoFix() bool {
	for _, r := range e.Recovery {
		if r.AutoFix && r.FixFunction != nil {
			return true
		}
	}
	return false
}

// TryAutoFix attempts to apply available auto-fixes
func (e *EDIError) TryAutoFix() error {
	for _, r := range e.Recovery {
		if r.AutoFix && r.FixFunction != nil {
			if err := r.FixFunction(); err != nil {
				return fmt.Errorf("auto-fix failed: %w", err)
			}
		}
	}
	return nil
}

// ErrorBuilder provides a fluent interface for constructing EDI errors
type ErrorBuilder struct {
	err *EDIError
}

// NewError creates a new error builder
func NewError(errType ErrorType, code string, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &EDIError{
			Type:     errType,
			Severity: SeverityError,
			Code:     code,
			Message:  message,
			Context:  make(map[string]any),
			Recovery: []RecoverySuggestion{},
		},
	}
}

// WithSeverity sets the error severity
func (b *ErrorBuilder) WithSeverity(s Severity) *ErrorBuilder {
	b.err.Severity = s
	return b
}

// WithDetails adds detailed error information
func (b *ErrorBuilder) WithDetails(details string) *ErrorBuilder {
	b.err.Details = details
	return b
}

// WithLocation adds location information
func (b *ErrorBuilder) WithLocation(loc *Location) *ErrorBuilder {
	b.err.Location = loc
	return b
}

// WithContext adds contextual information
func (b *ErrorBuilder) WithContext(key string, value any) *ErrorBuilder {
	b.err.Context[key] = value
	return b
}

// WithRecovery adds a recovery suggestion
func (b *ErrorBuilder) WithRecovery(suggestion RecoverySuggestion) *ErrorBuilder {
	b.err.Recovery = append(b.err.Recovery, suggestion)
	return b
}

// WithCause wraps an underlying error
func (b *ErrorBuilder) WithCause(err error) *ErrorBuilder {
	b.err.Caused = err
	return b
}

// Build returns the constructed error
func (b *ErrorBuilder) Build() *EDIError {
	return b.err
}

// Common error constructors

// NewSyntaxError creates a syntax error
func NewSyntaxError(segment string, message string) *EDIError {
	return NewError(ErrorTypeSyntax, "SYNTAX_ERROR", message).
		WithContext("segment", segment).
		WithRecovery(RecoverySuggestion{
			Action:      "Check segment format",
			Description: "Verify the segment follows X12 syntax rules",
			Example:     "Ensure proper delimiter usage and element count",
		}).
		Build()
}

// NewRequiredFieldError creates a required field error
func NewRequiredFieldError(segment string, element int, field string) *EDIError {
	return NewError(ErrorTypeRequired, "FIELD_REQUIRED", fmt.Sprintf("%s is required", field)).
		WithLocation(&Location{
			SegmentTag:   segment,
			ElementIndex: element,
		}).
		WithRecovery(RecoverySuggestion{
			Action:      "Provide required field",
			Description: fmt.Sprintf("Add the required %s value", field),
			Example:     fmt.Sprintf("%s*...%s...*", segment, field),
		}).
		Build()
}

// NewFormatError creates a format validation error
func NewFormatError(segment string, element int, expected, actual string) *EDIError {
	return NewError(ErrorTypeFormat, "FORMAT_INVALID", "Invalid format").
		WithDetails(fmt.Sprintf("Expected %s, got %s", expected, actual)).
		WithLocation(&Location{
			SegmentTag:   segment,
			ElementIndex: element,
		}).
		WithRecovery(RecoverySuggestion{
			Action:      "Fix format",
			Description: fmt.Sprintf("Change value to match %s format", expected),
			Example:     expected,
		}).
		Build()
}

// NewValueError creates a value validation error
func NewValueError(segment string, element int, value string, allowed []string) *EDIError {
	return NewError(ErrorTypeValue, "VALUE_INVALID", fmt.Sprintf("Invalid value: %s", value)).
		WithDetails(fmt.Sprintf("Allowed values: %s", strings.Join(allowed, ", "))).
		WithLocation(&Location{
			SegmentTag:   segment,
			ElementIndex: element,
		}).
		WithRecovery(RecoverySuggestion{
			Action:      "Use allowed value",
			Description: "Replace with one of the allowed values",
			Example:     allowed[0],
		}).
		Build()
}

// ErrorCollector accumulates errors during processing
type ErrorCollector struct {
	errors      []*EDIError
	maxErrors   int
	stopOnFatal bool
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxErrors int, stopOnFatal bool) *ErrorCollector {
	return &ErrorCollector{
		errors:      make([]*EDIError, 0),
		maxErrors:   maxErrors,
		stopOnFatal: stopOnFatal,
	}
}

// Add adds an error to the collection
func (c *ErrorCollector) Add(err *EDIError) error {
	c.errors = append(c.errors, err)

	if c.stopOnFatal && err.Severity == SeverityFatal {
		return fmt.Errorf("fatal error: %w", err)
	}

	if c.maxErrors > 0 && len(c.errors) > c.maxErrors {
		return fmt.Errorf("maximum error count (%d) exceeded", c.maxErrors)
	}

	return nil
}

// Errors returns all collected errors
func (c *ErrorCollector) Errors() []*EDIError {
	return c.errors
}

// HasErrors checks if any errors were collected
func (c *ErrorCollector) HasErrors() bool {
	return len(c.errors) > 0
}

// HasFatal checks if any fatal errors were collected
func (c *ErrorCollector) HasFatal() bool {
	for _, err := range c.errors {
		if err.Severity == SeverityFatal {
			return true
		}
	}
	return false
}

// GetByType returns errors of a specific type
func (c *ErrorCollector) GetByType(errType ErrorType) []*EDIError {
	var result []*EDIError
	for _, err := range c.errors {
		if err.Type == errType {
			result = append(result, err)
		}
	}
	return result
}

// GetBySeverity returns errors of a specific severity or higher
func (c *ErrorCollector) GetBySeverity(minSeverity Severity) []*EDIError {
	var result []*EDIError
	for _, err := range c.errors {
		if err.Severity >= minSeverity {
			result = append(result, err)
		}
	}
	return result
}

// Summary returns a summary of collected errors
func (c *ErrorCollector) Summary() string {
	if !c.HasErrors() {
		return "No errors"
	}

	counts := make(map[Severity]int)
	for _, err := range c.errors {
		counts[err.Severity]++
	}

	parts := []string{}
	if n := counts[SeverityFatal]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d fatal", n))
	}
	if n := counts[SeverityError]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", n))
	}
	if n := counts[SeverityWarning]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", n))
	}
	if n := counts[SeverityInfo]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d info", n))
	}

	return strings.Join(parts, ", ")
}
