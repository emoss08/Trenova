package core

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestAcknowledgmentHandler(t *testing.T) {
	registry := segments.NewSegmentRegistry("../../schemas")
	profileManager := profiles.NewProfileManager("../../testdata/profiles")
	handler := NewAcknowledgmentHandler(registry, profileManager)

	// Create a sample document
	doc := &x12.Document{
		Segments: []x12.Segment{
			{Tag: "ISA", Elements: [][]string{
				{"00"},
				{"          "},
				{"00"},
				{"          "},
				{"ZZ"},
				{"SENDER123      "},
				{"ZZ"},
				{"RECEIVER456    "},
				{"210101"},
				{"1200"},
				{"U"},
				{"00401"},
				{"000000001"},
				{"0"},
				{"P"},
				{">"},
			}},
			{Tag: "GS", Elements: [][]string{
				{
					"SM",
				}, {"SENDER123"}, {"RECEIVER456"}, {"20210101"}, {"1200"}, {"1"}, {"X"}, {"004010"},
			}},
			{Tag: "ST", Elements: [][]string{{"204"}, {"0001"}}},
			{Tag: "B2", Elements: [][]string{{"", "SCAC", "SHIP123", "PP"}}},
			{Tag: "SE", Elements: [][]string{{"4"}, {"0001"}}},
			{Tag: "GE", Elements: [][]string{{"1"}, {"1"}}},
			{Tag: "IEA", Elements: [][]string{{"1"}, {"000000001"}}},
		},
		Metadata: x12.DocumentMetadata{
			SenderID:         "SENDER123",
			ReceiverID:       "RECEIVER456",
			ISAControlNumber: "000000001",
			GSControlNumber:  "1",
			STControlNumber:  "0001",
			TransactionType:  "204",
		},
	}

	t.Run("Generate997_Accepted", func(t *testing.T) {
		req := GenerateRequest{
			Type:     Ack997,
			Original: doc,
			Issues:   []validation.Issue{},
			Accepted: true,
		}

		resp, err := handler.Generate(req)
		if err != nil {
			t.Fatalf("Failed to generate 997: %v", err)
		}

		// Check basic structure
		if !strings.Contains(resp.EDI, "ISA*") {
			t.Error("Generated 997 missing ISA segment")
		}
		if !strings.Contains(resp.EDI, "ST*997*") {
			t.Error("Generated 997 missing ST segment")
		}
		if !strings.Contains(resp.EDI, "AK1*") {
			t.Error("Generated 997 missing AK1 segment")
		}
		if !strings.Contains(resp.EDI, "AK9*A*") {
			t.Error("Generated 997 should have acceptance code A")
		}

		// Check statistics
		if resp.Statistics.TransactionSetsReceived != 1 {
			t.Errorf(
				"Expected 1 transaction set received, got %d",
				resp.Statistics.TransactionSetsReceived,
			)
		}
		if resp.Statistics.TransactionSetsAccepted != 1 {
			t.Errorf(
				"Expected 1 transaction set accepted, got %d",
				resp.Statistics.TransactionSetsAccepted,
			)
		}
	})

	t.Run("Generate997_WithErrors", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:     validation.Error,
				Code:         "MISSING_SEGMENT",
				Message:      "Required segment N1 is missing",
				Tag:          "N1",
				SegmentIndex: 4,
			},
		}

		req := GenerateRequest{
			Type:     Ack997,
			Original: doc,
			Issues:   issues,
			Accepted: false,
		}

		resp, err := handler.Generate(req)
		if err != nil {
			t.Fatalf("Failed to generate 997 with errors: %v", err)
		}

		// Check for error acknowledgment (group level, simplified approach)
		if !strings.Contains(resp.EDI, "AK9*E*") {
			t.Error("Generated 997 should have group-level error code E")
		}

		// Check statistics
		if resp.Statistics.ErrorsReported != 1 {
			t.Errorf("Expected 1 error reported, got %d", resp.Statistics.ErrorsReported)
		}
		if resp.Statistics.TransactionSetsAccepted != 0 {
			t.Errorf(
				"Expected 0 transaction sets accepted, got %d",
				resp.Statistics.TransactionSetsAccepted,
			)
		}
	})

	t.Run("Generate999_Accepted", func(t *testing.T) {
		req := GenerateRequest{
			Type:     Ack999,
			Original: doc,
			Issues:   []validation.Issue{},
			Accepted: true,
		}

		resp, err := handler.Generate(req)
		if err != nil {
			t.Fatalf("Failed to generate 999: %v", err)
		}

		// Check basic structure
		if !strings.Contains(resp.EDI, "ST*999*") {
			t.Error("Generated 999 missing ST segment")
		}
		if !strings.Contains(resp.EDI, "AK1*") {
			t.Error("Generated 999 missing AK1 segment")
		}
		if !strings.Contains(resp.EDI, "AK2*") {
			t.Error("Generated 999 missing AK2 segment")
		}
		if !strings.Contains(resp.EDI, "IK5*A") {
			t.Error("Generated 999 should have acceptance code A")
		}
	})

	t.Run("Generate999_WithWarnings", func(t *testing.T) {
		issues := []validation.Issue{
			{
				Severity:     validation.Warning,
				Code:         "DEPRECATED_ELEMENT",
				Message:      "Element B2-01 is deprecated",
				Tag:          "B2",
				SegmentIndex: 3,
				ElementIndex: 1,
			},
		}

		req := GenerateRequest{
			Type:     Ack999,
			Original: doc,
			Issues:   issues,
			Accepted: true, // Warnings don't cause rejection
		}

		resp, err := handler.Generate(req)
		if err != nil {
			t.Fatalf("Failed to generate 999 with warnings: %v", err)
		}

		// Check for acceptance despite warnings
		if !strings.Contains(resp.EDI, "IK5*A") {
			t.Error("Generated 999 should accept with warnings")
		}

		// Check statistics
		if resp.Statistics.WarningsReported != 1 {
			t.Errorf("Expected 1 warning reported, got %d", resp.Statistics.WarningsReported)
		}
		if resp.Statistics.TransactionSetsAccepted != 1 {
			t.Errorf(
				"Expected 1 transaction set accepted with warnings, got %d",
				resp.Statistics.TransactionSetsAccepted,
			)
		}
	})

	t.Run("Generate_UnsupportedType", func(t *testing.T) {
		req := GenerateRequest{
			Type:     AcknowledgmentType("998"), // Invalid type
			Original: doc,
			Issues:   []validation.Issue{},
			Accepted: true,
		}

		_, err := handler.Generate(req)
		if err == nil {
			t.Fatal("Expected error for unsupported acknowledgment type")
		}
		if !strings.Contains(err.Error(), "unsupported acknowledgment type") {
			t.Errorf("Expected unsupported type error, got: %v", err)
		}
	})

	t.Run("PartnerProfile_CustomDelimiters", func(t *testing.T) {
		// Create a test profile with custom delimiters
		profile := &profiles.PartnerProfile{
			PartnerID:   "TEST_PARTNER",
			PartnerName: "Test Partner",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "`",
					Component: "<",
					Segment:   "~",
				},
			},
		}

		// Save profile to manager
		err := profileManager.SaveProfile(profile)
		if err != nil {
			t.Fatalf("Failed to save profile: %v", err)
		}

		req := GenerateRequest{
			Type:      Ack997,
			PartnerID: "TEST_PARTNER",
			Original:  doc,
			Issues:    []validation.Issue{},
			Accepted:  true,
		}

		resp, err := handler.Generate(req)
		if err != nil {
			t.Fatalf("Failed to generate 997 with partner profile: %v", err)
		}

		// Check for custom delimiters
		if !strings.Contains(resp.EDI, "ISA`") {
			t.Error("Generated 997 should use custom element delimiter '`'")
		}
		if !strings.Contains(resp.EDI, "~") {
			t.Error("Generated 997 should use custom segment delimiter '~'")
		}
	})
}
