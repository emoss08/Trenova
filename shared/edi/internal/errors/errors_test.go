package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestEDIError_Basic(t *testing.T) {
	err := NewError(ErrorTypeSyntax, "TEST001", "Test error message").
		WithSeverity(SeverityError).
		WithDetails("Additional details").
		Build()
	
	if err.Type != ErrorTypeSyntax {
		t.Errorf("Expected type %s, got %s", ErrorTypeSyntax, err.Type)
	}
	
	if err.Code != "TEST001" {
		t.Errorf("Expected code TEST001, got %s", err.Code)
	}
	
	errorStr := err.Error()
	if !strings.Contains(errorStr, "TEST001") {
		t.Error("Error string should contain error code")
	}
}

func TestEDIError_WithLocation(t *testing.T) {
	loc := &Location{
		SegmentTag:   "B2",
		SegmentIndex: 3,
		ElementIndex: 2,
		Line:         10,
		Column:       15,
	}
	
	err := NewError(ErrorTypeRequired, "REQ001", "Field required").
		WithLocation(loc).
		Build()
	
	if err.Location == nil {
		t.Fatal("Location should be set")
	}
	
	if err.Location.SegmentTag != "B2" {
		t.Errorf("Expected segment tag B2, got %s", err.Location.SegmentTag)
	}
	
	locStr := err.Location.String()
	if !strings.Contains(locStr, "B2-02") {
		t.Error("Location string should contain B2-02")
	}
	if !strings.Contains(locStr, "segment 3") {
		t.Error("Location string should contain segment index")
	}
}

func TestEDIError_WithRecovery(t *testing.T) {
	fixApplied := false
	
	err := NewError(ErrorTypeFormat, "FMT001", "Invalid date format").
		WithRecovery(RecoverySuggestion{
			Action:      "Fix date format",
			Description: "Convert to CCYYMMDD",
			Example:     "20240101",
			AutoFix:     true,
			FixFunction: func() error {
				fixApplied = true
				return nil
			},
		}).
		Build()
	
	if !err.CanAutoFix() {
		t.Error("Error should have auto-fix available")
	}
	
	if err := err.TryAutoFix(); err != nil {
		t.Errorf("Auto-fix failed: %v", err)
	}
	
	if !fixApplied {
		t.Error("Fix function should have been called")
	}
}

func TestEDIError_WithContext(t *testing.T) {
	err := NewError(ErrorTypeValue, "VAL001", "Invalid value").
		WithContext("expected", "LD").
		WithContext("actual", "XX").
		WithContext("partner", "TEST_PARTNER").
		Build()
	
	if len(err.Context) != 3 {
		t.Errorf("Expected 3 context items, got %d", len(err.Context))
	}
	
	if err.Context["partner"] != "TEST_PARTNER" {
		t.Error("Context should contain partner information")
	}
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector(10, true)
	
	// Add various severity errors
	collector.Add(NewError(ErrorTypeRequired, "REQ001", "Field required").
		WithSeverity(SeverityWarning).Build())
	
	collector.Add(NewError(ErrorTypeSyntax, "SYN001", "Syntax error").
		WithSeverity(SeverityError).Build())
	
	collector.Add(NewError(ErrorTypeFormat, "FMT001", "Format error").
		WithSeverity(SeverityError).Build())
	
	if !collector.HasErrors() {
		t.Error("Collector should have errors")
	}
	
	if collector.HasFatal() {
		t.Error("Collector should not have fatal errors")
	}
	
	errors := collector.GetBySeverity(SeverityError)
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors with severity Error, got %d", len(errors))
	}
	
	syntaxErrors := collector.GetByType(ErrorTypeSyntax)
	if len(syntaxErrors) != 1 {
		t.Errorf("Expected 1 syntax error, got %d", len(syntaxErrors))
	}
	
	summary := collector.Summary()
	if !strings.Contains(summary, "2 errors") {
		t.Errorf("Summary should contain '2 errors', got: %s", summary)
	}
}

func TestErrorCollector_StopOnFatal(t *testing.T) {
	collector := NewErrorCollector(10, true)
	
	collector.Add(NewError(ErrorTypeRequired, "REQ001", "Field required").
		WithSeverity(SeverityError).Build())
	
	// Add fatal error
	err := collector.Add(NewError(ErrorTypeStructure, "STR001", "Structure error").
		WithSeverity(SeverityFatal).Build())
	
	if err == nil {
		t.Error("Should return error when fatal error is added with stopOnFatal=true")
	}
	
	if !strings.Contains(err.Error(), "fatal error") {
		t.Error("Error should indicate fatal error")
	}
}

func TestErrorCollector_MaxErrors(t *testing.T) {
	collector := NewErrorCollector(3, false)
	
	// Add errors up to the limit
	for i := 0; i < 2; i++ {
		err := collector.Add(NewError(ErrorTypeRequired, 
			fmt.Sprintf("REQ%03d", i), "Field required").Build())
		if err != nil {
			t.Errorf("Should not return error for error %d", i)
		}
	}
	
	// Add one more to reach the limit (should succeed)
	err := collector.Add(NewError(ErrorTypeRequired, "REQ002", "Field required").Build())
	if err != nil {
		t.Error("Should not return error when at max limit")
	}
	
	// Add one more - should exceed limit
	err = collector.Add(NewError(ErrorTypeRequired, "REQ003", "Field required").Build())
	if err == nil {
		t.Error("Should return error when max errors exceeded")
	}
	
	if !strings.Contains(err.Error(), "maximum error count") {
		t.Error("Error should indicate max errors exceeded")
	}
}

func TestCommonErrorConstructors(t *testing.T) {
	// Test syntax error
	syntaxErr := NewSyntaxError("B2", "Invalid segment format")
	if syntaxErr.Type != ErrorTypeSyntax {
		t.Error("Should create syntax error")
	}
	if len(syntaxErr.Recovery) == 0 {
		t.Error("Should have recovery suggestions")
	}
	
	// Test required field error
	reqErr := NewRequiredFieldError("B2", 2, "SCAC")
	if reqErr.Type != ErrorTypeRequired {
		t.Error("Should create required error")
	}
	if reqErr.Location == nil || reqErr.Location.ElementIndex != 2 {
		t.Error("Should set location with element index")
	}
	
	// Test format error
	fmtErr := NewFormatError("DTM", 2, "CCYYMMDD", "01/01/2024")
	if fmtErr.Type != ErrorTypeFormat {
		t.Error("Should create format error")
	}
	if !strings.Contains(fmtErr.Details, "CCYYMMDD") {
		t.Error("Should include expected format in details")
	}
	
	// Test value error
	valErr := NewValueError("S5", 2, "XX", []string{"LD", "UL"})
	if valErr.Type != ErrorTypeValue {
		t.Error("Should create value error")
	}
	if !strings.Contains(valErr.Details, "LD") {
		t.Error("Should include allowed values in details")
	}
}

func TestLocation_String(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		expected []string
	}{
		{
			name: "segment only",
			location: Location{
				SegmentTag: "B2",
			},
			expected: []string{"B2"},
		},
		{
			name: "segment with element",
			location: Location{
				SegmentTag:   "B2",
				ElementIndex: 2,
			},
			expected: []string{"B2-02"},
		},
		{
			name: "full location",
			location: Location{
				SegmentTag:     "B2",
				ElementIndex:   2,
				ComponentIndex: 1,
				SegmentIndex:   5,
				Line:           10,
				Column:         15,
				FilePath:       "test.edi",
			},
			expected: []string{"B2-02-01", "segment 5", "line 10:15", "test.edi"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.location.String()
			for _, exp := range tt.expected {
				if !strings.Contains(str, exp) {
					t.Errorf("Location string should contain %s, got: %s", exp, str)
				}
			}
		})
	}
}