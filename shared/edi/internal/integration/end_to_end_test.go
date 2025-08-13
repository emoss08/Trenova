package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/core"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/services"
)

// TestEndToEndPipeline tests the complete EDI processing pipeline
func TestEndToEndPipeline(t *testing.T) {
	// Initialize integrated parser with lenient validation
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
		StrictMode:  false,
		AutoAck:     true,
	})
	if err != nil {
		t.Fatalf("Failed to create integrated parser: %v", err)
	}

	testCases := []struct {
		name        string
		ediFile     string
		profileFile string
		expectValid bool
		checks      func(t *testing.T, resp *services.ParseResponse)
	}{
		{
			name:        "Meritor_204_BacktickDelimiters",
			ediFile:     "../../testdata/204/meritor-sample.edi",
			profileFile: "meritor-enhanced-4010.json",
			expectValid: false, // Business rules fail (missing consignee), but parsing should work
			checks: func(t *testing.T, resp *services.ParseResponse) {
				// Check backtick element delimiter was detected
				if resp.Profile != nil {
					delims := resp.Profile.GetDelimiters()
					if delims.Element != '`' {
						t.Errorf("Expected backtick element delimiter, got %v", delims.Element)
					}
				}
				
				// Verify that parsing worked (should have some validation issues but parsing should succeed)
				if len(resp.ValidationIssues) == 0 {
					t.Error("Expected some validation issues for business rules")
				}
				
				// Check that the validation issues are business rule related, not parsing related
				for _, issue := range resp.ValidationIssues {
					if issue.Code == "PARSE_ERROR" || issue.Code == "INVALID_STRUCTURE" {
						t.Errorf("Should not have parsing errors, got: %s - %s", issue.Code, issue.Message)
					}
				}
			},
		},
		{
			name:        "MultiStop_204_StandardDelimiters",
			ediFile:     "../../testdata/204/multi-stop-sample.edi",
			profileFile: "multistop-4010.json",
			expectValid: true,
			checks: func(t *testing.T, resp *services.ParseResponse) {
				// Check multiple stops were parsed
				if resp.Statistics.SegmentCount < 20 {
					t.Errorf("Expected at least 20 segments for multi-stop, got %d", resp.Statistics.SegmentCount)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load EDI file
			ediContent, err := os.ReadFile(tc.ediFile)
			if err != nil {
				t.Skipf("Skipping test - EDI file not found: %v", err)
				return
			}

			// Load profile if specified
			var partnerID string
			if tc.profileFile != "" {
				profile, err := parser.LoadProfile(tc.profileFile)
				if err != nil {
					t.Logf("Warning: Could not load profile %s: %v", tc.profileFile, err)
				} else {
					partnerID = profile.PartnerID
					// Save profile for use
					parser.SaveProfile(profile)
				}
			}

			// Parse with validation and acknowledgment generation
			req := services.ParseRequest{
				Data:            ediContent,
				PartnerID:       partnerID,
				ValidateContent: true,
				GenerateAck:     true,
				AckType:         core.Ack997,
			}

			resp, err := parser.Parse(context.Background(), req)
			if err != nil {
				t.Fatalf("Failed to parse EDI: %v", err)
			}

			// Check validation results
			if tc.expectValid && !resp.IsValid {
				t.Errorf("Expected EDI to be valid, but got %d issues", len(resp.ValidationIssues))
				for _, issue := range resp.ValidationIssues {
					t.Logf("  Issue: [%s] %s - %s", issue.Severity, issue.Code, issue.Message)
				}
			}

			// Check acknowledgment was generated
			if resp.Acknowledgment == nil {
				t.Error("Expected acknowledgment to be generated")
				t.Logf("Response IsValid: %v, ValidationIssues: %d", resp.IsValid, len(resp.ValidationIssues))
			} else {
				// Log the acknowledgment for debugging
				t.Logf("Generated acknowledgment (first 200 chars): %.200s", resp.Acknowledgment.EDI)
				
				// Verify acknowledgment structure based on delimiter
				// Check for 997 acknowledgment with appropriate delimiter
				hasValidAck := false
				if strings.Contains(resp.Acknowledgment.EDI, "ST*997*") || 
				   strings.Contains(resp.Acknowledgment.EDI, "ST`997`") ||
				   strings.Contains(resp.Acknowledgment.EDI, "ST:997:") {
					hasValidAck = true
				}
				
				if !hasValidAck {
					t.Errorf("Acknowledgment should contain ST segment with 997 transaction, got: %.200s", resp.Acknowledgment.EDI)
				}
				
				// Check acceptance status with appropriate delimiter
				if tc.expectValid {
					hasAcceptance := false
					if strings.Contains(resp.Acknowledgment.EDI, "AK9*A*") ||
					   strings.Contains(resp.Acknowledgment.EDI, "AK9`A`") ||
					   strings.Contains(resp.Acknowledgment.EDI, "AK9:A:") {
						hasAcceptance = true
					}
					if !hasAcceptance {
						t.Error("Valid EDI should produce accepted acknowledgment")
					}
				}
			}

			// Run custom checks
			if tc.checks != nil {
				tc.checks(t, resp)
			}
		})
	}
}

// TestCompleteWorkflow tests a complete EDI workflow from receipt to acknowledgment
func TestCompleteWorkflow(t *testing.T) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Step 1: Create a partner profile
	profile := &profiles.PartnerProfile{
		PartnerID:   "WORKFLOW_TEST",
		PartnerName: "Workflow Test Partner",
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
		ValidationConfig: profiles.ValidationConfig{
			Strictness:              "lenient",
			EnforceSegmentOrder:     false,
			EnforceSegmentCounts:    false,
			EnforceRequiredElements: false,
			EnforceElementFormats:   false,
			ValidateControlNumbers:  false,
		},
	}

	err = parser.SaveProfile(profile)
	if err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Step 2: Build EDI from business object
	businessData := map[string]any{
		"shipment": map[string]any{
			"scac":           "TEST",
			"shipment_id":    "WF123456",
			"payment_method": "PP",
		},
		"parties": []any{
			map[string]any{
				"entity_code": "SH",
				"name":        "Workflow Shipper Inc",
				"id_code":     "SH001",
			},
			map[string]any{
				"entity_code": "CN",
				"name":        "Workflow Consignee LLC",
				"id_code":     "CN001",
			},
		},
		"stops": []any{
			map[string]any{
				"stop_number": 1,
				"reason_code": "CL", // Complete Load (pickup)
			},
			map[string]any{
				"stop_number": 2,
				"reason_code": "CU", // Complete Unload (delivery)
			},
		},
	}

	ediContent, err := parser.BuildWithProfile(context.Background(), businessData, "WORKFLOW_TEST", "204")
	if err != nil {
		t.Fatalf("Failed to build EDI: %v", err)
	}

	t.Logf("Built EDI:\n%s", ediContent)

	// Step 3: Parse the built EDI
	parseReq := services.ParseRequest{
		Data:            []byte(ediContent),
		PartnerID:       "WORKFLOW_TEST",
		ValidateContent: true,
		GenerateAck:     true,
		AckType:         core.Ack997,
	}

	parseResp, err := parser.Parse(context.Background(), parseReq)
	if err != nil {
		t.Fatalf("Failed to parse built EDI: %v", err)
	}

	// Step 4: Verify round-trip
	if !parseResp.IsValid {
		t.Errorf("Built EDI should be valid, got %d issues", len(parseResp.ValidationIssues))
		for _, issue := range parseResp.ValidationIssues {
			t.Logf("  Issue: %s", issue.Message)
		}
	}

	// Step 5: Verify acknowledgment
	if parseResp.Acknowledgment == nil {
		t.Fatal("Expected acknowledgment to be generated")
	}

	t.Logf("Generated 997 Acknowledgment:\n%s", parseResp.Acknowledgment.EDI)

	// Verify acknowledgment indicates acceptance
	if !strings.Contains(parseResp.Acknowledgment.EDI, "AK9*A*") {
		t.Error("Acknowledgment should indicate acceptance")
	}

	// Step 6: Parse the acknowledgment itself
	ackParseReq := services.ParseRequest{
		Data:            []byte(parseResp.Acknowledgment.EDI),
		ValidateContent: true,
		GenerateAck:     false, // Don't acknowledge the acknowledgment
	}

	ackParseResp, err := parser.Parse(context.Background(), ackParseReq)
	if err != nil {
		t.Fatalf("Failed to parse acknowledgment: %v", err)
	}

	if ackParseResp.Document.Metadata.TransactionType != "997" {
		t.Errorf("Expected 997 transaction, got %s", ackParseResp.Document.Metadata.TransactionType)
	}
}

// TestErrorHandlingAndRecovery tests error scenarios and recovery
func TestErrorHandlingAndRecovery(t *testing.T) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
		StrictMode:  true, // Use strict mode to catch validation errors
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	testCases := []struct {
		name           string
		ediContent     string
		expectedErrors []string
	}{
		{
			name: "Missing_Required_Segments",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedErrors: []string{
				"N1", // Missing shipper/consignee
				"S5", // Missing stops
			},
		},
		{
			name: "Invalid_Segment_Values",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*XX~
N1*SH*Shipper~
N1*CN*Consignee~
S5*1*XX~
S5*2*YY~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedErrors: []string{
				"XX", // Invalid payment method
				"XX", // Invalid stop reason codes
				"YY",
			},
		},
		{
			name: "Malformed_Structure",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
SE*1*0001~
B2**TEST*SHIP123*PP~
GE*1*1~
IEA*1*000000001~`,
			expectedErrors: []string{
				"N1", // Missing required N1 segments
				"S5", // Missing required S5 segments
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a strict validation profile for error testing
			strictProfile := &profiles.PartnerProfile{
				PartnerID:   "STRICT_TEST",
				PartnerName: "Strict Test Partner",
				Active:      true,
				ValidationConfig: profiles.ValidationConfig{
					Strictness:              "strict",
					EnforceSegmentOrder:     true,
					EnforceSegmentCounts:    true,
					EnforceRequiredElements: true,
					EnforceElementFormats:   true,
					ValidateControlNumbers:  true,
				},
			}
			parser.SaveProfile(strictProfile)
			
			req := services.ParseRequest{
				Data:            []byte(tc.ediContent),
				PartnerID:       "STRICT_TEST",
				ValidateContent: true,
				GenerateAck:     true,
				AckType:         core.Ack997,
			}

			resp, err := parser.Parse(context.Background(), req)
			if err != nil {
				// Some errors are expected
				t.Logf("Parse error (expected): %v", err)
				return
			}

			// Check that validation caught the errors
			if resp.IsValid {
				t.Error("Expected validation to fail")
			}

			// Verify expected errors were detected
			for _, expectedErr := range tc.expectedErrors {
				found := false
				for _, issue := range resp.ValidationIssues {
					if strings.Contains(issue.Message, expectedErr) || strings.Contains(issue.Code, expectedErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error related to %s not found", expectedErr)
					// Log all actual issues for debugging
					for _, issue := range resp.ValidationIssues {
						t.Logf("  Actual issue: [%s] %s", issue.Code, issue.Message)
					}
				}
			}

			// Verify rejection acknowledgment
			if resp.Acknowledgment != nil {
				if strings.Contains(resp.Acknowledgment.EDI, "AK9*A*") {
					t.Error("Invalid EDI should produce rejected acknowledgment")
				}
			}
		})
	}
}

// TestProfileCompatibility tests profile compatibility across different EDI formats
func TestProfileCompatibility(t *testing.T) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Create profiles with different configurations
	profiles := []*profiles.PartnerProfile{
		{
			PartnerID:   "STANDARD",
			PartnerName: "Standard Format",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "*",
					Component: ":",
					Segment:   "~",
				},
			},
			ValidationConfig: profiles.ValidationConfig{
				Strictness: "lenient",
			},
		},
		{
			PartnerID:   "NEWLINE",
			PartnerName: "Newline Segments",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "*",
					Component: ":",
					Segment:   "\n",
				},
			},
			ValidationConfig: profiles.ValidationConfig{
				Strictness: "lenient",
			},
		},
		{
			PartnerID:   "CUSTOM",
			PartnerName: "Custom Delimiters",
			Active:      true,
			Format: profiles.FormatConfig{
				Delimiters: profiles.DelimiterConfig{
					Element:   "`",
					Component: "<",
					Segment:   "~",
				},
			},
			ValidationConfig: profiles.ValidationConfig{
				Strictness: "lenient",
			},
		},
	}

	// Test data that should work with all profiles
	testData := map[string]any{
		"shipment": map[string]any{
			"scac":           "TEST",
			"shipment_id":    "COMPAT123",
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

	for _, prof := range profiles {
		t.Run(prof.PartnerID, func(t *testing.T) {
			// Save profile
			err := parser.SaveProfile(prof)
			if err != nil {
				t.Fatalf("Failed to save profile: %v", err)
			}

			// Build EDI with profile
			edi, err := parser.BuildWithProfile(context.Background(), testData, prof.PartnerID, "204")
			if err != nil {
				t.Fatalf("Failed to build with profile %s: %v", prof.PartnerID, err)
			}
			

			// Verify delimiters are used correctly
			delims := prof.Format.Delimiters
			if delims.Element != "*" && !strings.Contains(edi, delims.Element) {
				t.Errorf("Built EDI should use element delimiter %s", delims.Element)
			}
			if delims.Segment != "~" && !strings.Contains(edi, delims.Segment) {
				t.Errorf("Built EDI should use segment delimiter %s", delims.Segment)
			}

			// Parse it back
			parseReq := services.ParseRequest{
				Data:            []byte(edi),
				PartnerID:       prof.PartnerID,
				ValidateContent: true,
			}

			parseResp, err := parser.Parse(context.Background(), parseReq)
			if err != nil {
				t.Fatalf("Failed to parse EDI built with profile %s: %v", prof.PartnerID, err)
			}
			

			if !parseResp.IsValid {
				t.Errorf("EDI built with profile %s should be valid", prof.PartnerID)
			}
		})
	}
}

// TestConcurrentProcessing tests concurrent EDI processing
func TestConcurrentProcessing(t *testing.T) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Create sample EDI
	ediContent := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company~
N1*CN*Consignee Company~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`

	// Process multiple EDI documents concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			req := services.ParseRequest{
				Data:            []byte(ediContent),
				ValidateContent: true,
				GenerateAck:     true,
			}

			resp, err := parser.Parse(context.Background(), req)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
				return
			}

			if !resp.IsValid {
				errors <- fmt.Errorf("goroutine %d: EDI not valid", id)
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Errorf("Concurrent processing error: %v", err)
		}
	}

	if successCount != numGoroutines {
		t.Errorf("Expected %d successful processes, got %d", numGoroutines, successCount)
	}
}

// BenchmarkEndToEndProcessing benchmarks the complete processing pipeline
func BenchmarkEndToEndProcessing(b *testing.B) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}

	ediContent := []byte(`ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company~
N1*CN*Consignee Company~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := services.ParseRequest{
			Data:            ediContent,
			ValidateContent: true,
			GenerateAck:     true,
		}

		_, err := parser.Parse(context.Background(), req)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// TestRealWorldSamples tests against real EDI samples if available
func TestRealWorldSamples(t *testing.T) {
	samplesDir := "../../testdata/204"
	if _, err := os.Stat(samplesDir); os.IsNotExist(err) {
		t.Skip("Samples directory not found")
	}

	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Read all .edi files in the samples directory
	files, err := filepath.Glob(filepath.Join(samplesDir, "*.edi"))
	if err != nil {
		t.Fatalf("Failed to list EDI files: %v", err)
	}

	for _, file := range files {
		filename := filepath.Base(file)
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", filename, err)
			}

			req := services.ParseRequest{
				Data:            content,
				ValidateContent: true,
				GenerateAck:     true,
			}

			resp, err := parser.Parse(context.Background(), req)
			if err != nil {
				t.Logf("Failed to parse %s: %v", filename, err)
				return
			}

			t.Logf("Parsed %s: %d segments, %d transactions, valid=%v",
				filename,
				resp.Statistics.SegmentCount,
				resp.Statistics.TransactionCount,
				resp.IsValid)

			if len(resp.ValidationIssues) > 0 {
				t.Logf("Validation issues in %s:", filename)
				for _, issue := range resp.ValidationIssues {
					t.Logf("  [%s] %s", issue.Severity, issue.Message)
				}
			}
		})
	}
}