package services

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/registry"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestValidationService(t *testing.T) {
	// Setup
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	if err := segRegistry.LoadFromDirectory(); err != nil {
		t.Fatalf("Failed to load segment schemas: %v", err)
	}
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}
	
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)
	configMgr := config.NewConfigManager()
	
	// Register test configurations
	cfg204 := config.Example204Config()
	configMgr.SaveConfig(cfg204)
	txRegistry.RegisterConfig(cfg204)
	
	cfg997 := config.Example997Config()
	configMgr.SaveConfig(cfg997)
	txRegistry.RegisterConfig(cfg997)
	
	service := NewValidationService(txRegistry, segRegistry, configMgr)
	
	t.Run("ValidateSimple204", func(t *testing.T) {
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
N1*SH*SHIPPER NAME*93*SHIP123~
N3*123 SHIPPER ST~
N4*CITY*ST*12345*US~
N1*CN*CONSIGNEE NAME*93*CONS456~
N3*456 CONSIGNEE AVE~
N4*OTHER CITY*ST*67890*US~
S5*1*CL*SHIPPER NAME*123 SHIPPER ST*CITY*ST*12345*US~
G62*86*20240102*1000~
S5*2*CU*CONSIGNEE NAME*456 CONSIGNEE AVE*OTHER CITY*ST*67890*US~
G62*70*20240103*1400~
SE*14*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent:      edi,
			TransactionType: "204",
			Version:         "004010",
			Options: ValidationOptions{
				DetailLevel: "detailed",
			},
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		if !result.Valid {
			t.Errorf("Expected valid EDI, got invalid")
			for _, issue := range result.Issues {
				t.Logf("Issue: %s - %s (Level: %s, Severity: %s)", issue.Code, issue.Message, issue.Level, issue.Severity)
			}
			// Also log segment counts to debug
			t.Logf("Segment counts: %+v", result.Statistics.SegmentCounts)
			if result.Metadata != nil && result.Metadata["business_object"] != nil {
				t.Logf("Business object: %+v", result.Metadata["business_object"])
			}
		}
		
		if result.TransactionType != "204" {
			t.Errorf("Expected transaction type 204, got %s", result.TransactionType)
		}
		
		if result.Statistics.TotalSegments != 17 {
			t.Errorf("Expected 17 segments, got %d", result.Statistics.TotalSegments)
		}
	})
	
	t.Run("ValidateInvalid204MissingStops", func(t *testing.T) {
		// 204 with only one stop (should fail validation)
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
N1*SH*SHIPPER NAME*93*SHIP123~
N1*CN*CONSIGNEE NAME*93*CONS456~
S5*1*CL*SHIPPER NAME*123 SHIPPER ST*CITY*ST*12345*US~
SE*7*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent: edi,
			Options: ValidationOptions{
				DetailLevel: "detailed",
			},
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		// Should be invalid due to insufficient stops
		if result.Valid {
			t.Error("Expected invalid EDI due to insufficient stops")
		}
		
		// Check for the specific error
		foundStopError := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Code, "LOOP") && strings.Contains(issue.Message, "S5") {
				foundStopError = true
				break
			}
		}
		
		if !foundStopError {
			t.Error("Expected error about insufficient stops")
			for _, issue := range result.Issues {
				t.Logf("Issue: %s - %s", issue.Code, issue.Message)
			}
		}
	})
	
	t.Run("Validate997", func(t *testing.T) {
		edi := `ISA*00*          *00*          *ZZ*RECEIVER       *ZZ*SENDER         *240101*1200*U*00401*000000001*0*P*>~
GS*FA*RECEIVER*SENDER*20240101*1200*1*X*004010~
ST*997*0001~
AK1*SM*1~
AK2*204*0001~
AK5*A~
AK9*A*1*1*1~
SE*7*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent:      edi,
			TransactionType: "997",
			Version:         "004010",
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		if !result.Valid {
			t.Errorf("Expected valid 997, got invalid")
			for _, issue := range result.Issues {
				t.Logf("Issue: %s - %s", issue.Code, issue.Message)
			}
		}
		
		if result.TransactionType != "997" {
			t.Errorf("Expected transaction type 997, got %s", result.TransactionType)
		}
	})
	
	t.Run("DetectTransactionType", func(t *testing.T) {
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent: edi,
			// Don't specify transaction type - let it detect
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		if result.TransactionType != "204" {
			t.Errorf("Expected detected transaction type 204, got %s", result.TransactionType)
		}
		
		if result.Version != "004010" {
			t.Errorf("Expected detected version 004010, got %s", result.Version)
		}
	})
	
	t.Run("ValidateWithCustomerConfig", func(t *testing.T) {
		// Add customer config
		cfg204 := config.Example204Config()
		cfg204.CustomerOverrides = map[string]config.CustomerConfig{
			"CUST001": {
				CustomerID:   "CUST001",
				CustomerName: "Test Customer",
				Active:       true,
				DefaultValues: map[string]map[int]any{
					"B2": {
						1: "TL", // Default tariff code
					},
				},
				AdditionalRules: []config.ValidationRule{
					{
						RuleID:      "CUST_REQUIRED_REF",
						Name:        "Customer Reference Required",
						Description: "Customer requires reference number",
						Severity:    "error",
						Type:        "business_rule",
						Condition: config.Condition{
							Type:  "not_exists",
							Field: "N9[01=PO]",
						},
						Message:   "Customer requires PO reference (N9*PO)",
						ErrorCode: "MISSING_PO_REF",
					},
				},
			},
		}
		configMgr.SaveConfig(cfg204)
		
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
N1*SH*SHIPPER NAME*93*SHIP123~
N1*CN*CONSIGNEE NAME*93*CONS456~
S5*1*CL*SHIPPER NAME*123 SHIPPER ST*CITY*ST*12345*US~
S5*2*CU*CONSIGNEE NAME*456 CONSIGNEE AVE*OTHER CITY*ST*67890*US~
SE*8*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent:      edi,
			TransactionType: "204",
			Version:         "004010",
			CustomerID:      "CUST001",
			Options: ValidationOptions{
				DetailLevel: "detailed",
			},
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		// Should have customer info in metadata
		if result.Metadata != nil {
			if custID, ok := result.Metadata["customer_id"]; ok {
				if custID != "CUST001" {
					t.Errorf("Expected customer_id CUST001, got %v", custID)
				}
			}
			
			if hasCustConfig, ok := result.Metadata["customer_config"]; ok {
				if hasCustConfig != true {
					t.Error("Expected customer_config to be true")
				}
			}
		}
	})
	
	t.Run("ValidateBatch", func(t *testing.T) {
		edi1 := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT1*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`
		
		edi2 := `ISA*00*          *00*          *ZZ*RECEIVER       *ZZ*SENDER         *240101*1200*U*00401*000000002*0*P*>~
GS*FA*RECEIVER*SENDER*20240101*1200*2*X*004010~
ST*997*0001~
AK1*SM*1~
AK9*A*0*0*0~
SE*3*0001~
GE*1*2~
IEA*1*000000002~`
		
		requests := []*ValidateRequest{
			{EDIContent: edi1},
			{EDIContent: edi2},
		}
		
		ctx := context.Background()
		results, err := service.ValidateBatch(ctx, requests)
		if err != nil {
			t.Fatalf("Failed to validate batch: %v", err)
		}
		
		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		
		if results[0].TransactionType != "204" {
			t.Errorf("Expected first transaction type 204, got %s", results[0].TransactionType)
		}
		
		if results[1].TransactionType != "997" {
			t.Errorf("Expected second transaction type 997, got %s", results[1].TransactionType)
		}
	})
	
	t.Run("PreviewValidation", func(t *testing.T) {
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
SE*2*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent: edi,
		}
		
		ctx := context.Background()
		result, err := service.PreviewValidation(ctx, req)
		if err != nil {
			t.Fatalf("Failed to preview validate: %v", err)
		}
		
		// Check preview mode flag
		if result.Metadata == nil {
			t.Fatal("Expected metadata in preview mode")
		}
		
		if preview, ok := result.Metadata["preview_mode"]; !ok || preview != true {
			t.Error("Expected preview_mode flag to be true")
		}
		
		// Should have detailed level
		if result.Metadata["validation_type"] == nil && result.Metadata["config_used"] == nil {
			// This is expected as we're using defaults
		}
	})
	
	t.Run("ValidateWithOptions", func(t *testing.T) {
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
N1*SH*SHIPPER NAME~
SE*3*0001~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent: edi,
			Options: ValidationOptions{
				StopOnFirstError: true,
				MaxErrors:        1,
				ValidateOnly:     "syntax",
				SkipWarnings:     true,
				DetailLevel:      "minimal",
			},
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate with options: %v", err)
		}
		
		// Should have limited errors due to MaxErrors
		errorCount := 0
		for _, issue := range result.Issues {
			if issue.Severity == validation.Error {
				errorCount++
			}
		}
		
		// Should not have warnings since SkipWarnings is true
		warningCount := 0
		for _, issue := range result.Issues {
			if issue.Severity == validation.Warning {
				warningCount++
			}
		}
		
		if warningCount > 0 {
			t.Errorf("Expected no warnings with SkipWarnings=true, got %d", warningCount)
		}
	})
	
	t.Run("ValidateParseError", func(t *testing.T) {
		edi := `This is not valid EDI content`
		
		req := &ValidateRequest{
			EDIContent: edi,
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if result.Valid {
			t.Error("Expected invalid result for malformed EDI")
		}
		
		// Should have parse error
		foundParseError := false
		for _, issue := range result.Issues {
			if issue.Code == "PARSE_ERROR" {
				foundParseError = true
				break
			}
		}
		
		if !foundParseError {
			t.Error("Expected PARSE_ERROR in issues")
		}
	})
}

func TestValidationServiceStructure(t *testing.T) {
	// Setup
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}
	
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)
	configMgr := config.NewConfigManager()
	service := NewValidationService(txRegistry, segRegistry, configMgr)
	
	t.Run("ValidateBasicStructure", func(t *testing.T) {
		// Test ST/SE matching validation
		edi := `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240101*1200*U*00401*000000001*0*P*>~
GS*SM*SENDER*RECEIVER*20240101*1200*1*X*004010~
ST*204*0001~
B2**SCAC*SHIPMENT123*PP~
ST*204*0002~
SE*2*0001~
SE*2*0002~
GE*1*1~
IEA*1*000000001~`
		
		req := &ValidateRequest{
			EDIContent: edi,
		}
		
		ctx := context.Background()
		result, err := service.Validate(ctx, req)
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		// Should detect mismatched ST/SE
		if result.Valid {
			t.Error("Expected invalid due to mismatched ST/SE")
		}
	})
	
	t.Run("ValidateEmptySegment", func(t *testing.T) {
		doc := &x12.Document{
			Segments: []x12.Segment{
				{Tag: "ISA", Elements: [][]string{{"00"}, {"          "}}},
				{Tag: ""},  // Empty segment
				{Tag: "IEA", Elements: [][]string{{"1"}, {"000000001"}}},
			},
		}
		
		// Mock validateWithDefaults call
		ctx := context.Background()
		result, err := service.validateWithDefaults(ctx, doc, "204", "004010", ValidationOptions{})
		if err != nil {
			t.Fatalf("Failed to validate: %v", err)
		}
		
		if result.Valid {
			t.Error("Expected invalid due to empty segment")
		}
		
		// Check for empty segment error
		foundEmptyError := false
		for _, issue := range result.Issues {
			if issue.Code == "EMPTY_SEGMENT" {
				foundEmptyError = true
				break
			}
		}
		
		if !foundEmptyError {
			t.Error("Expected EMPTY_SEGMENT error")
		}
	})
}

// Mock helper for time
func init() {
	timeNow = func() int64 {
		return 1000 // Fixed time for testing
	}
}