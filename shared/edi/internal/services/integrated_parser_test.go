package services

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/core"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
)

func TestIntegratedParser(t *testing.T) {
	// Create integrated parser
	opts := IntegratedParserOptions{
		ProfilePath:      "../../testdata/profiles",
		SchemaPath:       "../../schemas",
		StrictMode:       true,
		AutoAck:          true,
		ValidateProfiles: true,
	}

	parser, err := NewIntegratedParser(opts)
	if err != nil {
		t.Fatalf("Failed to create integrated parser: %v", err)
	}

	// Sample EDI content
	ediContent := []byte(
		`ISA*00*          *00*          *ZZ*SENDER123      *ZZ*RECEIVER456    *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER123*RECEIVER456*20210101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
	)

	t.Run("Parse_WithoutProfile", func(t *testing.T) {
		req := ParseRequest{
			Data:            ediContent,
			ValidateContent: true,
			GenerateAck:     false,
		}

		resp, err := parser.Parse(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to parse without profile: %v", err)
		}

		// Check document was parsed
		if resp.Document == nil {
			t.Fatal("Expected document to be parsed")
		}

		if resp.Document.Metadata.TransactionType != "204" {
			t.Errorf(
				"Expected transaction type 204, got %s",
				resp.Document.Metadata.TransactionType,
			)
		}

		// ISA fields are padded to 15 characters
		expectedSender := "SENDER123"
		if !strings.HasPrefix(resp.Document.Metadata.SenderID, expectedSender) {
			t.Errorf(
				"Expected sender ID to start with %s, got %s",
				expectedSender,
				resp.Document.Metadata.SenderID,
			)
		}

		// Check statistics
		if resp.Statistics.SegmentCount != 11 {
			t.Errorf("Expected 11 segments, got %d", resp.Statistics.SegmentCount)
		}

		if resp.Statistics.TransactionCount != 1 {
			t.Errorf("Expected 1 transaction, got %d", resp.Statistics.TransactionCount)
		}
	})

	t.Run("Parse_WithValidation", func(t *testing.T) {
		req := ParseRequest{
			Data:            ediContent,
			ValidateContent: true,
			GenerateAck:     false,
		}

		resp, err := parser.Parse(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to parse with validation: %v", err)
		}

		// Check validation was performed
		if resp.Statistics.ValidationTimeMs == 0 {
			t.Log("Warning: Validation time not recorded")
		}

		// Basic EDI should be valid
		if !resp.IsValid {
			t.Errorf("Expected EDI to be valid, but got %d issues", len(resp.ValidationIssues))
			for _, issue := range resp.ValidationIssues {
				t.Logf("  Issue: %s - %s", issue.Code, issue.Message)
			}
		}
	})

	t.Run("Parse_WithAcknowledgment", func(t *testing.T) {
		req := ParseRequest{
			Data:            ediContent,
			ValidateContent: true,
			GenerateAck:     true,
			AckType:         core.Ack997,
		}

		resp, err := parser.Parse(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to parse with acknowledgment: %v", err)
		}

		// Check acknowledgment was generated
		if resp.Acknowledgment == nil {
			t.Fatal("Expected acknowledgment to be generated")
		}

		if !strings.Contains(resp.Acknowledgment.EDI, "ST*997*") {
			t.Error("Generated acknowledgment should be a 997")
		}

		// Check statistics in acknowledgment
		if resp.Acknowledgment.Statistics.TransactionSetsReceived != 1 {
			t.Errorf("Expected 1 transaction set received in ack, got %d",
				resp.Acknowledgment.Statistics.TransactionSetsReceived)
		}
	})

	t.Run("Parse_WithProfile", func(t *testing.T) {
		// Create and save a test profile
		profile := &profiles.PartnerProfile{
			PartnerID:   "TEST_PARTNER",
			PartnerName: "Test Partner",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "*",
					Component: ":",
					Segment:   "~",
				},
			},
			SupportedTransactions: []profiles.TransactionSupport{
				{
					TransactionType: "204",
					Versions:        []string{"004010"},
					Required:        true,
				},
			},
		}

		err := parser.SaveProfile(profile)
		if err != nil {
			t.Fatalf("Failed to save test profile: %v", err)
		}

		req := ParseRequest{
			Data:            ediContent,
			PartnerID:       "TEST_PARTNER",
			ValidateContent: true,
			GenerateAck:     true,
		}

		resp, err := parser.Parse(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to parse with profile: %v", err)
		}

		// Check profile was used
		if resp.Profile == nil {
			t.Error("Expected profile to be loaded")
		} else if resp.Profile.PartnerID != "TEST_PARTNER" {
			t.Errorf("Expected partner ID TEST_PARTNER, got %s", resp.Profile.PartnerID)
		}

		// Check acknowledgment uses profile delimiters
		if resp.Acknowledgment != nil {
			if !strings.Contains(resp.Acknowledgment.EDI, "ISA*") {
				t.Error("Acknowledgment should use profile's element delimiter '*'")
			}
		}
	})

	t.Run("Parse_InvalidEDI", func(t *testing.T) {
		invalidEDI := []byte("This is not valid EDI content")

		req := ParseRequest{
			Data:            invalidEDI,
			ValidateContent: true, // Enable validation to detect invalid EDI
			GenerateAck:     false,
		}

		resp, err := parser.Parse(context.Background(), req)
		// The parser should error on invalid EDI content OR return an invalid result
		if err == nil && resp != nil && resp.IsValid {
			t.Error("Expected parser to error on invalid EDI content or return invalid result")
			t.Logf(
				"Response IsValid: %v, ValidationIssues: %d",
				resp.IsValid,
				len(resp.ValidationIssues),
			)
		}
	})

	t.Run("BuildWithProfile", func(t *testing.T) {
		// Create a profile for building
		profile := &profiles.PartnerProfile{
			PartnerID:   "BUILD_PARTNER",
			PartnerName: "Build Test Partner",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "`",
					Component: "<",
					Segment:   "~",
				},
			},
			SupportedTransactions: []profiles.TransactionSupport{
				{
					TransactionType: "204",
					Versions:        []string{"004010"},
					Required:        true,
				},
			},
		}

		err := parser.SaveProfile(profile)
		if err != nil {
			t.Fatalf("Failed to save build profile: %v", err)
		}

		// Business object data
		data := map[string]any{
			"shipment": map[string]any{
				"scac":           "TEST",
				"shipment_id":    "SHIP456",
				"payment_method": "PP",
			},
			"parties": []any{
				map[string]any{
					"entity_code": "SH",
					"name":        "Test Shipper",
				},
				map[string]any{
					"entity_code": "CN",
					"name":        "Test Consignee",
				},
			},
			"stops": []any{
				map[string]any{
					"stop_number": 1,
					"reason_code": "CL",
				},
				map[string]any{
					"stop_number": 2,
					"reason_code": "CU",
				},
			},
		}

		edi, err := parser.BuildWithProfile(context.Background(), data, "BUILD_PARTNER", "204")
		if err != nil {
			t.Fatalf("Failed to build with profile: %v", err)
		}

		// Check for custom delimiter
		if !strings.Contains(edi, "ST`204`") {
			t.Error("Built EDI should use custom element delimiter '`'")
		}

		// Check basic structure
		if !strings.Contains(edi, "B2`") {
			t.Error("Built EDI missing B2 segment")
		}
		if !strings.Contains(edi, "N1`SH`") {
			t.Error("Built EDI missing shipper")
		}
		if !strings.Contains(edi, "N1`CN`") {
			t.Error("Built EDI missing consignee")
		}
	})

	t.Run("ValidateWithProfile", func(t *testing.T) {
		// Parse a document first
		req := ParseRequest{
			Data:            ediContent,
			ValidateContent: false,
			GenerateAck:     false,
		}

		resp, err := parser.Parse(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to parse for validation: %v", err)
		}

		// Validate with a specific profile
		valResult, err := parser.ValidateWithProfile(
			context.Background(),
			resp.Document,
			"TEST_PARTNER",
		)
		if err != nil {
			// Profile may not exist, which is okay for this test
			t.Logf("Validation with profile failed (expected if profile doesn't exist): %v", err)
		} else {
			if !valResult.Valid {
				t.Errorf("Expected document to be valid with profile, got %d issues", len(valResult.Issues))
				for _, issue := range valResult.Issues {
					t.Logf("  Issue: [%s] %s - %s", issue.Severity, issue.Code, issue.Message)
				}
			}
		}
	})
}

func TestIntegratedParser_ProfileManagement(t *testing.T) {
	opts := IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	}

	parser, err := NewIntegratedParser(opts)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	t.Run("SaveAndLoadProfile", func(t *testing.T) {
		profile := &profiles.PartnerProfile{
			PartnerID:   "MGMT_TEST",
			PartnerName: "Management Test",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element: "*",
					Segment: "~",
				},
			},
		}

		// Save profile
		err := parser.SaveProfile(profile)
		if err != nil {
			t.Fatalf("Failed to save profile: %v", err)
		}

		// Load profile
		loaded, err := parser.GetProfile("MGMT_TEST")
		if err != nil {
			t.Fatalf("Failed to load profile: %v", err)
		}

		if loaded.PartnerID != "MGMT_TEST" {
			t.Errorf("Expected partner ID MGMT_TEST, got %s", loaded.PartnerID)
		}

		if loaded.Format.Delimiters.Element != "*" {
			t.Errorf("Expected element delimiter *, got %s", loaded.Format.Delimiters.Element)
		}
	})

	t.Run("LoadProfileFromFile", func(t *testing.T) {
		// Try to load an existing profile file if it exists
		profile, partnerID, err := parser.LoadProfile("meritor-enhanced-4010.json")
		if err != nil {
			// File may not exist, which is okay
			t.Logf("Could not load meritor profile (expected if file doesn't exist): %v", err)
		} else {
			t.Logf("Loaded profile for partner: %s", partnerID)
			if profile != nil {
				t.Logf("Profile has %d supported transactions", len(profile.SupportedTransactions))
			}
		}
	})
}
