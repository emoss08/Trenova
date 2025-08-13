package ack

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestAckBuilder_Build997(t *testing.T) {
	// Create test registry and register segments
	registry := segments.NewSegmentRegistry("../../schemas")
	
	// Setup builder
	delims := x12.Delimiters{
		Element:   '*',
		Component: ':',
		Segment:   '~',
		Repetition: '^',
	}
	
	builder := NewAckBuilder(registry, "004010", delims)
	
	// Test segments (simulating a 204 transaction)
	originalSegs := []x12.Segment{
		{
			Tag: "ISA",
			Elements: [][]string{
				{"00"}, {"          "}, {"00"}, {"          "},
				{"ZZ"}, {"SENDER123      "}, {"ZZ"}, {"RECEIVER456    "},
				{"240101"}, {"1200"}, {"U"}, {"00401"},
				{"000000001"}, {"0"}, {"P"}, {":"},
			},
		},
		{
			Tag: "GS",
			Elements: [][]string{
				{"SM"}, {"SENDER123"}, {"RECEIVER456"},
				{"20240101"}, {"1200"}, {"1"}, {"X"}, {"004010"},
			},
		},
		{
			Tag: "ST",
			Elements: [][]string{
				{"204"}, {"0001"},
			},
		},
		{
			Tag: "B2",
			Elements: [][]string{
				{""}, {"SCAC"}, {"SHIPMENT123"}, {"CC"},
			},
		},
		{
			Tag: "SE",
			Elements: [][]string{
				{"3"}, {"0001"},
			},
		},
		{
			Tag: "GE",
			Elements: [][]string{
				{"1"}, {"1"},
			},
		},
		{
			Tag: "IEA",
			Elements: [][]string{
				{"1"}, {"000000001"},
			},
		},
	}
	
	// Test with no issues (successful acknowledgment)
	t.Run("Success", func(t *testing.T) {
		issues := []validation.Issue{}
		
		result, err := builder.Build997(originalSegs, issues)
		if err != nil {
			t.Fatalf("Build997 failed: %v", err)
		}
		
		// Verify result contains required segments
		if !strings.Contains(result, "ISA*") {
			t.Error("Result missing ISA segment")
		}
		if !strings.Contains(result, "GS*FA*") {
			t.Error("Result missing GS segment with FA functional code")
		}
		if !strings.Contains(result, "ST*997*") {
			t.Error("Result missing ST segment for 997")
		}
		if !strings.Contains(result, "AK1*") {
			t.Error("Result missing AK1 segment")
		}
		if !strings.Contains(result, "AK2*") {
			t.Error("Result missing AK2 segment")
		}
		if !strings.Contains(result, "AK5*") {
			t.Error("Result missing AK5 segment")
		}
		if !strings.Contains(result, "AK9*") {
			t.Error("Result missing AK9 segment")
		}
		if !strings.Contains(result, "SE*") {
			t.Error("Result missing SE segment")
		}
		if !strings.Contains(result, "GE*") {
			t.Error("Result missing GE segment")
		}
		if !strings.Contains(result, "IEA*") {
			t.Error("Result missing IEA segment")
		}
	})
	
	// Test with errors
	t.Run("WithErrors", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:     validation.Error,
				Code:         "SEG_MISSING",
				Message:      "Required segment N1 missing",
				SegmentIndex: 3,
				Tag:          "N1",
			},
		}
		
		result, err := builder.Build997(originalSegs, issues)
		if err != nil {
			t.Fatalf("Build997 failed: %v", err)
		}
		
		// Verify AK3 segment is present for error
		if !strings.Contains(result, "AK3*") {
			t.Error("Result missing AK3 segment for error reporting")
		}
		
		// Verify AK5 contains error code
		if !strings.Contains(result, "AK5*E") {
			t.Error("AK5 should indicate errors were noted")
		}
	})
}

func TestAckBuilder_Build999(t *testing.T) {
	// Create test registry and register segments
	registry := segments.NewSegmentRegistry("../../schemas")
	
	// Setup builder
	delims := x12.Delimiters{
		Element:   '*',
		Component: ':',
		Segment:   '~',
		Repetition: '^',
	}
	
	builder := NewAckBuilder(registry, "004010", delims)
	
	// Test segments
	originalSegs := []x12.Segment{
		{
			Tag: "ISA",
			Elements: [][]string{
				{"00"}, {"          "}, {"00"}, {"          "},
				{"ZZ"}, {"SENDER123      "}, {"ZZ"}, {"RECEIVER456    "},
				{"240101"}, {"1200"}, {"U"}, {"00401"},
				{"000000001"}, {"0"}, {"P"}, {":"},
			},
		},
		{
			Tag: "GS",
			Elements: [][]string{
				{"SM"}, {"SENDER123"}, {"RECEIVER456"},
				{"20240101"}, {"1200"}, {"1"}, {"X"}, {"004010"},
			},
		},
		{
			Tag: "ST",
			Elements: [][]string{
				{"204"}, {"0001"},
			},
		},
		{
			Tag: "B2",
			Elements: [][]string{
				{""}, {"SCAC"}, {"SHIPMENT123"}, {"CC"},
			},
		},
		{
			Tag: "SE",
			Elements: [][]string{
				{"3"}, {"0001"},
			},
		},
		{
			Tag: "GE",
			Elements: [][]string{
				{"1"}, {"1"},
			},
		},
		{
			Tag: "IEA",
			Elements: [][]string{
				{"1"}, {"000000001"},
			},
		},
	}
	
	// Test with segment-level error
	t.Run("SegmentError", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:        validation.Error,
				Code:            "SEG_UNRECOGNIZED",
				Message:         "Unrecognized segment XYZ",
				SegmentIndex:    3,
				Tag:             "XYZ",
				Level:           "segment",
			},
		}
		
		result, err := builder.Build999(originalSegs, issues)
		if err != nil {
			t.Fatalf("Build999 failed: %v", err)
		}
		
		// Verify required segments
		if !strings.Contains(result, "ST*999*") {
			t.Error("Result missing ST segment for 999")
		}
		if !strings.Contains(result, "IK3*") {
			t.Error("Result missing IK3 segment for segment error")
		}
		if !strings.Contains(result, "IK5*") {
			t.Error("Result missing IK5 segment")
		}
	})
	
	// Test with element-level error
	t.Run("ElementError", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:        validation.Error,
				Code:            "ELEM_INVALID_CODE",
				Message:         "Invalid code value",
				SegmentIndex:    3,
				Tag:             "B2",
				Level:           "segment",
				ElementPosition: 2,
				ElementRef:      "B202",
				BadValue:        "INVALID",
			},
		}
		
		result, err := builder.Build999(originalSegs, issues)
		if err != nil {
			t.Fatalf("Build999 failed: %v", err)
		}
		
		// Verify IK4 segment is present for element error
		if !strings.Contains(result, "IK4*") {
			t.Error("Result missing IK4 segment for element error")
		}
	})
	
	// Test with context information
	t.Run("WithContext", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:        validation.Error,
				Code:            "SEG_CONDITIONAL",
				Message:         "Conditional requirement not met",
				SegmentIndex:    3,
				Tag:             "N1",
				Level:           "segment",
				Context:         "When B201=TL, N1*SH is required",
			},
		}
		
		result, err := builder.Build999(originalSegs, issues)
		if err != nil {
			t.Fatalf("Build999 failed: %v", err)
		}
		
		// Verify CTX segment is present for context
		if !strings.Contains(result, "CTX*") {
			t.Error("Result missing CTX segment for context information")
		}
	})
}

func TestAckBuilder_ExtractValues(t *testing.T) {
	delims := x12.Delimiters{
		Element:   '*',
		Component: ':',
		Segment:   '~',
		Repetition: '^',
	}
	
	registry := segments.NewSegmentRegistry("../../schemas")
	builder := NewAckBuilder(registry, "004010", delims)
	
	segments := []x12.Segment{
		{
			Tag: "ISA",
			Elements: [][]string{
				{"00"}, {"          "}, {"00"}, {"          "},
				{"ZZ"}, {"SENDER123      "}, {"ZZ"}, {"RECEIVER456    "},
				{"240101"}, {"1200"}, {"U"}, {"00401"},
				{"000000001"}, {"0"}, {"P"}, {":"},
			},
		},
		{
			Tag: "GS",
			Elements: [][]string{
				{"SM"}, {"SENDER123"}, {"RECEIVER456"},
				{"20240101"}, {"1200"}, {"1"}, {"X"}, {"004010"},
			},
		},
	}
	
	t.Run("ExtractISA", func(t *testing.T) {
		values := builder.extractISAValues(segments)
		
		if values["senderID"] != "SENDER123      " {
			t.Errorf("Expected sender ID 'SENDER123      ', got '%s'", values["senderID"])
		}
		if values["receiverID"] != "RECEIVER456    " {
			t.Errorf("Expected receiver ID 'RECEIVER456    ', got '%s'", values["receiverID"])
		}
		if values["version"] != "00401" {
			t.Errorf("Expected version '00401', got '%s'", values["version"])
		}
	})
	
	t.Run("ExtractGS", func(t *testing.T) {
		values := builder.extractGSValues(segments)
		
		if values["functionalID"] != "SM" {
			t.Errorf("Expected functional ID 'SM', got '%s'", values["functionalID"])
		}
		if values["senderCode"] != "SENDER123" {
			t.Errorf("Expected sender code 'SENDER123', got '%s'", values["senderCode"])
		}
		if values["controlNumber"] != "1" {
			t.Errorf("Expected control number '1', got '%s'", values["controlNumber"])
		}
	})
}

func TestAckBuilder_ErrorCodeMapping(t *testing.T) {
	delims := x12.Delimiters{
		Element:   '*',
		Component: ':',
		Segment:   '~',
		Repetition: '^',
	}
	
	registry := segments.NewSegmentRegistry("../../schemas")
	builder := NewAckBuilder(registry, "004010", delims)
	
	// Test IK304 code mapping
	t.Run("IK304Codes", func(t *testing.T) {
		testCases := []struct {
			issueCode string
			expected  string
		}{
			{"SEG_UNRECOGNIZED", "1"},
			{"SEG_UNEXPECTED", "2"},
			{"SEG_MISSING", "3"},
			{"SEG_LOOP_ERROR", "4"},
			{"SEG_NOT_IN_POSITION", "5"},
			{"SEG_NOT_IN_DEFINED", "6"},
			{"SEG_NOT_IN_PROPER", "7"},
			{"SEG_HAS_DATA_ERRORS", "8"},
			{"OTHER", "8"}, // Default
		}
		
		for _, tc := range testCases {
			issue := validation.Issue{Code: tc.issueCode}
			result := builder.getIK304Code(issue)
			if result != tc.expected {
				t.Errorf("For code %s, expected %s, got %s", tc.issueCode, tc.expected, result)
			}
		}
	})
	
	// Test IK403 code mapping
	t.Run("IK403Codes", func(t *testing.T) {
		testCases := []struct {
			issueCode string
			expected  string
		}{
			{"ELEM_MISSING", "1"},
			{"ELEM_CONDITIONAL_MISSING", "2"},
			{"ELEM_TOO_SHORT", "3"},
			{"ELEM_TOO_LONG", "4"},
			{"ELEM_INVALID_CODE", "5"},
			{"ELEM_INVALID_CHAR", "6"},
			{"ELEM_INVALID_CODE_VALUE", "7"},
			{"ELEM_INVALID_DATE", "8"},
			{"ELEM_INVALID_TIME", "9"},
			{"ELEM_EXCLUSION_CONDITION", "10"},
			{"OTHER", "12"}, // Default
		}
		
		for _, tc := range testCases {
			issue := validation.Issue{Code: tc.issueCode}
			result := builder.getIK403Code(issue)
			if result != tc.expected {
				t.Errorf("For code %s, expected %s, got %s", tc.issueCode, tc.expected, result)
			}
		}
	})
}