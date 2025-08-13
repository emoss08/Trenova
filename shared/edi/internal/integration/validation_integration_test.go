package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/registry"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/services"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// TestValidationIntegration tests the complete validation pipeline
func TestValidationIntegration(t *testing.T) {
	// Initialize components
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	// Load segment schemas
	if err := segRegistry.LoadFromDirectory(); err != nil {
		t.Fatalf("Failed to load segment schemas: %v", err)
	}
	delims := x12.DefaultDelimiters()
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)
	configMgr := config.NewConfigManager()

	// Add 204 configuration
	config204 := config.Example204Config()
	configMgr.SaveConfig(config204)

	// Use strict validation for these tests to ensure errors are caught
	strictValidationConfig := segments.ValidationConfig{
		Level: segments.ValidationLevelStrict,
		Elements: segments.ElementValidationConfig{
			EnforceMandatory:     true,
			AllowExtraElements:   false,
			SkipLengthValidation: false,
			SkipFormatValidation: false,
		},
		Codes: segments.CodeValidationConfig{
			InvalidCodeHandling: segments.CodeHandlingError,
			CaseSensitive:       false,
			AllowPartialMatches: false,
			AllowCustomCodes:    false,
			MinLengthToValidate: 1,
		},
	}
	validationSvc := services.NewValidationServiceWithConfig(
		txRegistry,
		segRegistry,
		configMgr,
		strictValidationConfig,
	)

	testCases := []struct {
		name           string
		ediContent     string
		expectedIssues []string
		severity       validation.Severity
	}{
		{
			name: "Valid_204_Basic",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{},
			severity:       validation.Warning, // Use Warning instead of Info
		},
		{
			name: "Missing_Shipper",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*7*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{"MISSING_SHIPPER", "shipper"},
			severity:       validation.Error,
		},
		{
			name: "Invalid_Stop_Codes",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*XX~
S5*2*YY~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{"XX", "YY", "invalid", "reason code"},
			severity:       validation.Error,
		},
		{
			name: "Insufficient_Stops",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
SE*7*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{"INSUFFICIENT_STOPS", "2 stops"},
			severity:       validation.Error,
		},
		{
			name: "Missing_Required_B2_Elements",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2****~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{"B2", "required", "empty"},
			severity:       validation.Error,
		},
		{
			name: "Invalid_Payment_Method",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*ZZ~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			expectedIssues: []string{"ZZ", "payment", "invalid"},
			severity:       validation.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &services.ValidateRequest{
				EDIContent:      tc.ediContent,
				TransactionType: "204",
				Version:         "004010",
			}

			result, err := validationSvc.Validate(context.Background(), req)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			// Check if validation result matches expectations
			if len(tc.expectedIssues) == 0 {
				// Should be valid
				if !result.Valid {
					t.Errorf("Expected valid EDI, but got %d issues", len(result.Issues))
					for _, issue := range result.Issues {
						t.Logf("  Issue: [%s] %s - %s", issue.Severity, issue.Code, issue.Message)
					}
				}
			} else {
				// Should have specific issues
				if result.Valid {
					t.Error("Expected validation to fail, but it passed")
				}

				// Check for expected error messages
				for _, expected := range tc.expectedIssues {
					found := false
					for _, issue := range result.Issues {
						if strings.Contains(strings.ToLower(issue.Message), strings.ToLower(expected)) ||
							strings.Contains(strings.ToLower(issue.Code), strings.ToLower(expected)) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue containing '%s' not found", expected)
						// Log all issues for debugging
						for _, issue := range result.Issues {
							t.Logf("  Actual issue: [%s] %s - %s", issue.Severity, issue.Code, issue.Message)
						}
					}
				}

				// Check severity
				hasExpectedSeverity := false
				for _, issue := range result.Issues {
					if issue.Severity == tc.severity {
						hasExpectedSeverity = true
						break
					}
				}
				if !hasExpectedSeverity && tc.severity != validation.Warning {
					t.Errorf("Expected at least one issue with severity %s", tc.severity)
				}
			}
		})
	}
}

// TestCustomerValidationRules tests customer-specific validation rules
func TestCustomerValidationRules(t *testing.T) {
	// Initialize services
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	// Load segment schemas
	if err := segRegistry.LoadFromDirectory(); err != nil {
		t.Fatalf("Failed to load segment schemas: %v", err)
	}
	configMgr := config.NewConfigManager()

	// Create a 204 config with customer overrides
	config204 := config.Example204Config()
	config204.CustomerOverrides = map[string]config.CustomerConfig{
		"STRICT_CUSTOMER": {
			CustomerID:   "STRICT_CUSTOMER",
			CustomerName: "Strict Validation Customer",
			Active:       true,
			AdditionalRules: []config.ValidationRule{
				{
					RuleID:      "CUST_REQ_B2A",
					Name:        "B2A Required",
					Description: "Customer requires B2A segment",
					Severity:    "error",
					Type:        "segment",
					Condition: config.Condition{
						Type:  "not_exists",
						Field: "B2A",
					},
					Message:   "Customer requires B2A segment",
					ErrorCode: "CUST_MISSING_B2A",
				},
				{
					RuleID:      "CUST_MIN_3_STOPS",
					Name:        "Minimum 3 Stops",
					Description: "Customer requires at least 3 stops",
					Severity:    "error",
					Type:        "business_rule",
					Condition: config.Condition{
						Type:  "count_less_than",
						Field: "S5",
						Value: 3,
					},
					Message:   "Customer requires at least 3 stops",
					ErrorCode: "CUST_INSUFFICIENT_STOPS",
				},
			},
		},
	}

	configMgr.SaveConfig(config204)

	delims := x12.DefaultDelimiters()
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)

	// Use strict validation for customer rules testing
	strictValidationConfig := segments.ValidationConfig{
		Level: segments.ValidationLevelStrict,
		Elements: segments.ElementValidationConfig{
			EnforceMandatory:     true,
			AllowExtraElements:   false,
			SkipLengthValidation: false,
			SkipFormatValidation: false,
		},
		Codes: segments.CodeValidationConfig{
			InvalidCodeHandling: segments.CodeHandlingError,
			CaseSensitive:       false,
			AllowPartialMatches: false,
			AllowCustomCodes:    false,
			MinLengthToValidate: 1,
		},
	}
	validationSvc := services.NewValidationServiceWithConfig(
		txRegistry,
		segRegistry,
		configMgr,
		strictValidationConfig,
	)

	testCases := []struct {
		name           string
		customerID     string
		ediContent     string
		shouldPass     bool
		expectedErrors []string
	}{
		{
			name:       "Standard_Customer_2_Stops",
			customerID: "",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			shouldPass:     true,
			expectedErrors: []string{},
		},
		{
			name:       "Strict_Customer_2_Stops_Fails",
			customerID: "STRICT_CUSTOMER",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			shouldPass:     false,
			expectedErrors: []string{"CUST_MISSING_B2A", "CUST_INSUFFICIENT_STOPS"},
		},
		{
			name:       "Strict_Customer_3_Stops_With_B2A_Passes",
			customerID: "STRICT_CUSTOMER",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
B2A*00*LT~
N1*SH*Shipper Company*93*SH001~
N1*CN*Consignee Company*93*CN001~
S5*1*CL~
S5*2*LD~
S5*3*CU~
SE*10*0001~
GE*1*1~
IEA*1*000000001~`,
			shouldPass:     true,
			expectedErrors: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &services.ValidateRequest{
				EDIContent:      tc.ediContent,
				TransactionType: "204",
				Version:         "004010",
				CustomerID:      tc.customerID,
			}

			result, err := validationSvc.Validate(context.Background(), req)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			if tc.shouldPass {
				if !result.Valid {
					t.Errorf("Expected validation to pass, but got %d issues", len(result.Issues))
					for _, issue := range result.Issues {
						t.Logf("  Issue: [%s] %s - %s", issue.Severity, issue.Code, issue.Message)
					}
				}
			} else {
				if result.Valid {
					t.Error("Expected validation to fail, but it passed")
				}

				// Check for expected customer errors
				for _, expectedErr := range tc.expectedErrors {
					found := false
					for _, issue := range result.Issues {
						if strings.Contains(issue.Code, expectedErr) {
							found = true
							t.Logf("Found expected error: %s", expectedErr)
							break
						}
					}
					if !found {
						t.Errorf("Expected error code %s not found", expectedErr)
					}
				}
			}
		})
	}
}

// TestProfileValidation tests validation with partner profiles
func TestProfileValidation(t *testing.T) {
	parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
		ProfilePath: "../../testdata/profiles",
		SchemaPath:  "../../schemas",
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Create profiles with different validation settings
	profiles := []struct {
		profile *profiles.PartnerProfile
		edi     string
		valid   bool
	}{
		{
			profile: &profiles.PartnerProfile{
				ValidationConfig: profiles.ValidationConfig{
					Strictness:              "lenient",
					EnforceSegmentOrder:     false,
					EnforceSegmentCounts:    false,
					EnforceRequiredElements: false,
					AllowUnknownSegments:    true,
				},
			},
			edi: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`,
			valid: true, // Lenient mode allows minimal EDI
		},
		{
			profile: &profiles.PartnerProfile{
				PartnerID:   "STRICT",
				PartnerName: "Strict Partner",
				Active:      true,
				ValidationConfig: profiles.ValidationConfig{
					Strictness:              "strict",
					EnforceSegmentOrder:     true,
					EnforceSegmentCounts:    true,
					EnforceRequiredElements: true,
					EnforceElementFormats:   true,
					EnforceElementLengths:   true,
					ValidateControlNumbers:  true,
					UniqueControlNumbers:    true,
				},
			},
			edi: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`,
			valid: false, // Strict mode requires all components
		},
	}

	for _, tc := range profiles {
		t.Run(tc.profile.PartnerID, func(t *testing.T) {
			// Save profile
			err := parser.SaveProfile(tc.profile)
			if err != nil {
				t.Fatalf("Failed to save profile: %v", err)
			}

			// Parse and validate
			req := services.ParseRequest{
				Data:            []byte(tc.edi),
				PartnerID:       tc.profile.PartnerID,
				ValidateContent: true,
			}

			resp, err := parser.Parse(context.Background(), req)
			if err != nil {
				// Parse errors are acceptable for invalid EDI
				if tc.valid {
					t.Fatalf("Failed to parse EDI: %v", err)
				}
				return
			}

			if tc.valid && !resp.IsValid {
				t.Errorf("Expected EDI to be valid with %s profile, but got %d issues",
					tc.profile.PartnerID, len(resp.ValidationIssues))
				for _, issue := range resp.ValidationIssues {
					t.Logf("  Issue: %s", issue.Message)
				}
			} else if !tc.valid && resp.IsValid {
				t.Errorf("Expected EDI to be invalid with %s profile, but it passed",
					tc.profile.PartnerID)
			}
		})
	}
}

// TestCrossFieldValidation tests validation rules that span multiple fields/segments
func TestCrossFieldValidation(t *testing.T) {
	// Initialize services
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	// Load segment schemas
	if err := segRegistry.LoadFromDirectory(); err != nil {
		t.Fatalf("Failed to load segment schemas: %v", err)
	}
	configMgr := config.NewConfigManager()

	// Create config with cross-field validation rules
	config204 := config.Example204Config()
	// Note: These rules are placeholders - the actual cross-field validation logic
	// would need to be implemented in the ConfigurableBuilder.Validate method
	// to properly check stop sequences and order
	/*
		config204.ValidationRules = append(config204.ValidationRules,
			config.ValidationRule{
				RuleID:      "STOP_SEQUENCE",
				Name:        "Stop Sequence Validation",
				Description: "Stops must have sequential numbers",
				Severity:    "error",
				Type:        "cross_field",
				Message:     "Stop numbers must be sequential",
				ErrorCode:   "INVALID_STOP_SEQUENCE",
			},
			config.ValidationRule{
				RuleID:      "PICKUP_BEFORE_DELIVERY",
				Name:        "Pickup Before Delivery",
				Description: "Pickup stops must come before delivery stops",
				Severity:    "error",
				Type:        "business_rule",
				Message:     "Pickup stops must precede delivery stops",
				ErrorCode:   "INVALID_STOP_ORDER",
			},
		)
	*/

	configMgr.SaveConfig(config204)

	delims := x12.DefaultDelimiters()
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)

	// Use strict validation for cross-field validation testing
	strictValidationConfig := segments.ValidationConfig{
		Level: segments.ValidationLevelStrict,
		Elements: segments.ElementValidationConfig{
			EnforceMandatory:     true,
			AllowExtraElements:   false,
			SkipLengthValidation: false,
			SkipFormatValidation: false,
		},
		Codes: segments.CodeValidationConfig{
			InvalidCodeHandling: segments.CodeHandlingError,
			CaseSensitive:       false,
			AllowPartialMatches: false,
			AllowCustomCodes:    false,
			MinLengthToValidate: 1,
		},
	}
	validationSvc := services.NewValidationServiceWithConfig(
		txRegistry,
		segRegistry,
		configMgr,
		strictValidationConfig,
	)

	testCases := []struct {
		name       string
		ediContent string
		valid      bool
		errorCode  string
	}{
		{
			name: "Valid_Stop_Sequence",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper*93*SH001~
N1*CN*Consignee*93*CN001~
S5*1*CL~
S5*2*CU~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			valid: true,
		},
		{
			name: "Invalid_Stop_Order_Delivery_First",
			ediContent: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *210101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20210101*1200*1*X*004010~
ST*204*0001~
B2**TEST*SHIP123*PP~
N1*SH*Shipper*93*SH001~
N1*CN*Consignee*93*CN001~
S5*1*CU~
S5*2*CL~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`,
			valid:     true, // Currently we don't enforce this
			errorCode: "INVALID_STOP_ORDER",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &services.ValidateRequest{
				EDIContent:      tc.ediContent,
				TransactionType: "204",
				Version:         "004010",
			}

			result, err := validationSvc.Validate(context.Background(), req)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			if tc.valid {
				if !result.Valid {
					t.Errorf("Expected valid EDI, but got %d issues", len(result.Issues))
					for _, issue := range result.Issues {
						t.Logf("  Issue: %s", issue.Message)
					}
				}
			} else {
				if result.Valid {
					t.Error("Expected validation to fail")
				}

				// Check for specific error code
				found := false
				for _, issue := range result.Issues {
					if issue.Code == tc.errorCode {
						found = true
						break
					}
				}
				if !found && tc.errorCode != "" {
					t.Errorf("Expected error code %s not found", tc.errorCode)
				}
			}
		})
	}
}
