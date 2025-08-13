package config

import (
	"context"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestConfigManager(t *testing.T) {
	manager := NewConfigManager()

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		// Create a test configuration
		config := Example204Config()

		// Save it
		err := manager.SaveConfig(config)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Load it back
		loaded, err := manager.GetConfig("204", "004010")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loaded.TransactionType != "204" {
			t.Errorf("Expected transaction type 204, got %s", loaded.TransactionType)
		}

		if len(loaded.Structure.Loops) != 2 {
			t.Errorf("Expected 2 loops, got %d", len(loaded.Structure.Loops))
		}
	})

	t.Run("CustomerConfig", func(t *testing.T) {
		// Add a configuration with customer overrides
		config := Example204Config()
		config.CustomerOverrides = map[string]CustomerConfig{
			"CUST001": {
				CustomerID:   "CUST001",
				CustomerName: "Test Customer",
				Active:       true,
				SegmentOverrides: map[string]SegmentRequirement{
					"B2A": {
						SegmentID: "B2A",
						Required:  true,
						MinOccurs: 1,
						MaxOccurs: 1,
					},
				},
				AdditionalRules: []ValidationRule{
					{
						RuleID:      "CUST_SPECIAL_REQ",
						Name:        "Customer Special Requirement",
						Description: "Customer requires special field",
						Severity:    "error",
						Type:        "field",
						Message:     "Special field required for this customer",
						ErrorCode:   "CUST_FIELD_MISSING",
					},
				},
				DefaultValues: map[string]map[int]any{
					"B2": {
						1: "TL", // Default to truckload
					},
				},
			},
		}

		err := manager.SaveConfig(config)
		if err != nil {
			t.Fatalf("Failed to save config with customer overrides: %v", err)
		}

		// Get customer config
		custConfig, err := manager.GetCustomerConfig("204", "004010", "CUST001")
		if err != nil {
			t.Fatalf("Failed to get customer config: %v", err)
		}

		if custConfig.CustomerID != "CUST001" {
			t.Errorf("Expected customer ID CUST001, got %s", custConfig.CustomerID)
		}

		if len(custConfig.AdditionalRules) != 1 {
			t.Errorf("Expected 1 additional rule, got %d", len(custConfig.AdditionalRules))
		}

		// Test default customer (no overrides)
		defaultCust, err := manager.GetCustomerConfig("204", "004010", "UNKNOWN")
		if err != nil {
			t.Fatalf("Failed to get default customer config: %v", err)
		}

		if !defaultCust.Active {
			t.Error("Default customer should be active")
		}
	})
}

func TestConfigurableBuilder(t *testing.T) {
	// Setup
	registry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}

	t.Run("Build204FromConfig", func(t *testing.T) {
		config := Example204Config()
		builder := NewConfigurableBuilder(config, nil, registry, delims)

		// Create test data matching the configuration mappings
		data := map[string]any{
			"shipment": map[string]any{
				"scac":           "TEST",
				"shipment_id":    "SHIP123",
				"payment_method": "PP",
			},
			"parties": []any{
				map[string]any{
					"entity_code": "SH",
					"name":        "Shipper Company",
					"id_code":     "SH001",
				},
				map[string]any{
					"entity_code": "CN",
					"name":        "Consignee Company",
					"id_code":     "CN001",
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

		ctx := context.Background()
		result, err := builder.BuildFromObject(ctx, data)
		if err != nil {
			t.Fatalf("Failed to build from object: %v", err)
		}

		// Verify result contains expected segments
		if !strings.Contains(result, "ST*204*") {
			t.Error("Result missing ST segment")
		}
		if !strings.Contains(result, "B2*") {
			t.Error("Result missing B2 segment")
		}
		if !strings.Contains(result, "SE*") {
			t.Error("Result missing SE segment")
		}

		t.Logf("Generated EDI:\n%s", result)
	})

	t.Run("ValidateWithConfig", func(t *testing.T) {
		config := Example204Config()
		builder := NewConfigurableBuilder(config, nil, registry, delims)

		// Test data missing required shipper
		data := map[string]any{
			"parties": []any{
				map[string]any{
					"entity_code": "CN", // Only consignee, no shipper
					"name":        "Consignee Company",
				},
			},
			"stops": []any{
				map[string]any{
					"stop_number": 1,
				},
			},
		}

		ctx := context.Background()
		issues := builder.Validate(ctx, data)

		// Should have validation errors
		if len(issues) < 2 {
			t.Errorf("Expected at least 2 validation issues, got %d", len(issues))
		}

		// Check for specific errors
		hasShipperError := false
		hasStopsError := false
		for _, issue := range issues {
			if issue.Code == "MISSING_SHIPPER" {
				hasShipperError = true
			}
			if issue.Code == "INSUFFICIENT_STOPS" {
				hasStopsError = true
			}
		}

		if !hasShipperError {
			t.Error("Expected MISSING_SHIPPER error")
		}
		if !hasStopsError {
			t.Error("Expected INSUFFICIENT_STOPS error")
		}
	})

	t.Run("CustomerOverrides", func(t *testing.T) {
		config := Example204Config()
		customer := &CustomerConfig{
			CustomerID:   "CUST001",
			CustomerName: "Test Customer",
			Active:       true,
			DefaultValues: map[string]map[int]any{
				"B2": {
					1: "TL", // Force truckload
				},
			},
			Transformations: []TransformationRule{
				{
					RuleID: "UPPER_SCAC",
					Name:   "uppercase_scac",
					Type:   "uppercase",
					Field:  "shipment.scac",
				},
			},
		}

		builder := NewConfigurableBuilder(config, customer, registry, delims)

		data := map[string]any{
			"shipment": map[string]any{
				"scac":           "test", // lowercase
				"shipment_id":    "SHIP123",
				"payment_method": "PP",
			},
			"parties": []any{
				map[string]any{
					"entity_code": "SH",
					"name":        "Shipper",
				},
				map[string]any{
					"entity_code": "CN",
					"name":        "Consignee",
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

		ctx := context.Background()
		result, err := builder.BuildFromObject(ctx, data)
		if err != nil {
			t.Fatalf("Failed to build with customer overrides: %v", err)
		}

		// Check that SCAC was uppercased
		if !strings.Contains(result, "TEST") {
			t.Error("SCAC should be uppercased to TEST")
		}

		// Check that default was applied
		if !strings.Contains(result, "B2*TL*") {
			t.Error("B2 should have TL as first element from customer default")
		}
	})
}

func TestConfigurationJSON(t *testing.T) {
	t.Run("Marshal204Config", func(t *testing.T) {
		config := Example204Config()

		data, err := sonic.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		// Unmarshal it back
		var loaded TransactionConfig
		err = sonic.Unmarshal(data, &loaded)
		if err != nil {
			t.Fatalf("Failed to unmarshal config: %v", err)
		}

		if loaded.TransactionType != "204" {
			t.Errorf("Expected transaction type 204, got %s", loaded.TransactionType)
		}

		if len(loaded.Mappings) != len(config.Mappings) {
			t.Errorf("Mappings count mismatch: expected %d, got %d",
				len(config.Mappings), len(loaded.Mappings))
		}

		// Verify mapping details preserved - find B2 mapping
		var b2Mapping *SegmentMapping
		for _, m := range loaded.Mappings {
			if m.SegmentID == "B2" {
				b2Mapping = &m
				break
			}
		}

		if b2Mapping == nil {
			t.Error("Expected B2 mapping to be present")
		} else if len(b2Mapping.Elements) != 4 {
			t.Errorf("Expected 4 elements in B2 mapping, got %d", len(b2Mapping.Elements))
		}
	})

	t.Run("Marshal997Config", func(t *testing.T) {
		config := Example997Config()

		data, err := sonic.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal 997 config: %v", err)
		}

		var loaded TransactionConfig
		err = sonic.Unmarshal(data, &loaded)
		if err != nil {
			t.Fatalf("Failed to unmarshal 997 config: %v", err)
		}

		if loaded.TransactionType != "997" {
			t.Errorf("Expected transaction type 997, got %s", loaded.TransactionType)
		}

		// Check loop structure
		if len(loaded.Structure.Loops) != 1 {
			t.Errorf("Expected 1 loop (AK2), got %d", len(loaded.Structure.Loops))
		}

		if loaded.Structure.Loops[0].LoopID != "AK2" {
			t.Errorf("Expected AK2 loop, got %s", loaded.Structure.Loops[0].LoopID)
		}
	})
}

func TestConditionEvaluation(t *testing.T) {
	registry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}

	config := Example204Config()
	builder := NewConfigurableBuilder(config, nil, registry, delims)

	testCases := []struct {
		name      string
		condition Condition
		data      any
		expected  bool
	}{
		{
			name: "Exists",
			condition: Condition{
				Type:  "exists",
				Field: "shipment.scac",
			},
			data: map[string]any{
				"shipment": map[string]any{
					"scac": "TEST",
				},
			},
			expected: true,
		},
		{
			name: "NotExists",
			condition: Condition{
				Type:  "not_exists",
				Field: "shipment.missing",
			},
			data: map[string]any{
				"shipment": map[string]any{
					"scac": "TEST",
				},
			},
			expected: true,
		},
		{
			name: "Equals",
			condition: Condition{
				Type:  "equals",
				Field: "shipment.payment_method",
				Value: "PP",
			},
			data: map[string]any{
				"shipment": map[string]any{
					"payment_method": "PP",
				},
			},
			expected: true,
		},
		{
			name: "NotEquals",
			condition: Condition{
				Type:  "not_equals",
				Field: "shipment.payment_method",
				Value: "CC",
			},
			data: map[string]any{
				"shipment": map[string]any{
					"payment_method": "PP",
				},
			},
			expected: true,
		},
		{
			name: "Contains",
			condition: Condition{
				Type:  "contains",
				Field: "shipment.id",
				Value: "SHIP",
			},
			data: map[string]any{
				"shipment": map[string]any{
					"id": "SHIP123",
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.evaluateCondition(tc.condition, tc.data)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestTransformations(t *testing.T) {
	registry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}

	config := Example204Config()
	customer := &CustomerConfig{
		CustomerID: "TEST",
		Active:     true,
		Transformations: []TransformationRule{
			{
				RuleID: "PAD_LEFT",
				Name:   "pad_shipment_id",
				Type:   "pad",
				Parameters: map[string]string{
					"length":    "10",
					"char":      "0",
					"direction": "left",
				},
			},
			{
				RuleID: "REPLACE",
				Name:   "replace_dash",
				Type:   "replace",
				Parameters: map[string]string{
					"old": "-",
					"new": "_",
				},
			},
		},
	}

	builder := NewConfigurableBuilder(config, customer, registry, delims)

	t.Run("BuiltinTransforms", func(t *testing.T) {
		tests := []struct {
			input     string
			transform string
			expected  string
		}{
			{"test", "uppercase", "TEST"},
			{"TEST", "lowercase", "test"},
			{"  test  ", "trim", "test"},
		}

		for _, test := range tests {
			result := builder.applyTransform(test.input, test.transform)
			if result != test.expected {
				t.Errorf("Transform %s: expected %s, got %s",
					test.transform, test.expected, result)
			}
		}
	})

	t.Run("CustomTransforms", func(t *testing.T) {
		// Test padding
		result := builder.applyTransform("123", "pad_shipment_id")
		if result != "0000000123" {
			t.Errorf("Expected padded value 0000000123, got %s", result)
		}

		// Test replace
		result = builder.applyTransform("test-value", "replace_dash")
		if result != "test_value" {
			t.Errorf("Expected replaced value test_value, got %s", result)
		}
	})
}
