package segments

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/errors"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestSegmentProcessor_ProcessSegments(t *testing.T) {
	// Create test registry with ISA schema
	registry := NewSegmentRegistry("../../schemas")
	
	// Register test ISA schema
	registry.RegisterSegment(&SegmentSchema{
		ID:      "ISA",
		Name:    "Interchange Control Header",
		Version: "004010",
		Elements: []ElementSchema{
			{
				Position:    1,
				RefID:       "I01",
				Name:        "Authorization Information Qualifier",
				Type:        "ID",
				Requirement: "Mandatory",
				MinLength:   2,
				MaxLength:   2,
				Codes: []CodeValue{
					{Code: "00", Description: "No Authorization Information Present"},
				},
			},
			{
				Position:    2,
				RefID:       "I02",
				Name:        "Authorization Information",
				Type:        "AN",
				Requirement: "Mandatory",
				MinLength:   10,
				MaxLength:   10,
			},
		},
	})

	// Register B2 schema
	registry.RegisterSegment(&SegmentSchema{
		ID:      "B2",
		Name:    "Beginning Segment for Shipment Information Transaction",
		Version: "004010",
		Elements: []ElementSchema{
			{
				Position:    1,
				RefID:       "B201",
				Name:        "Tariff Service Code",
				Type:        "AN",
				Requirement: "Optional",
				MinLength:   1,
				MaxLength:   2,
			},
			{
				Position:    2,
				RefID:       "B202",
				Name:        "Standard Carrier Alpha Code",
				Type:        "AN",
				Requirement: "Mandatory",
				MinLength:   2,
				MaxLength:   4,
			},
			{
				Position:    3,
				RefID:       "B203",
				Name:        "Shipment Identification Number",
				Type:        "AN",
				Requirement: "Optional",
				MinLength:   1,
				MaxLength:   30,
			},
		},
	})

	processor := NewSegmentProcessor(registry)

	// Test segments
	segments := []x12.Segment{
		{
			Tag: "ISA",
			Elements: [][]string{
				{"00"},
				{"          "},
				{"00"},
				{"          "},
				{"ZZ"},
				{"SENDER         "},
				{"ZZ"},
				{"RECEIVER       "},
				{"240101"},
				{"1200"},
				{"U"},
				{"00401"},
				{"000000001"},
				{"0"},
				{"P"},
				{">"},
			},
			Index: 0,
		},
		{
			Tag: "B2",
			Elements: [][]string{
				{""},
				{"SCAC"},
				{"ABC123"},
				{"CC"},
			},
			Index: 1,
		},
	}

	ctx := context.Background()
	processed, err := processor.ProcessSegments(ctx, segments, "004010")
	if err != nil {
		t.Fatalf("ProcessSegments failed: %v", err)
	}

	if len(processed) != 2 {
		t.Errorf("Expected 2 processed segments, got %d", len(processed))
	}

	// Verify ISA processing
	isaSegment := processed[0]
	if isaSegment.Schema.ID != "ISA" {
		t.Errorf("Expected ISA segment, got %s", isaSegment.Schema.ID)
	}

	// Check ISA01 value
	if val, ok := isaSegment.Data["ISA01"]; !ok || val != "00" {
		t.Errorf("ISA01 not processed correctly: %v", val)
	}

	// Verify B2 processing
	b2Segment := processed[1]
	if b2Segment.Schema.ID != "B2" {
		t.Errorf("Expected B2 segment, got %s", b2Segment.Schema.ID)
	}

	// Check B202 (SCAC) value
	if val, ok := b2Segment.Data["B202"]; !ok || val != "SCAC" {
		t.Errorf("B202 not processed correctly: %v", val)
	}
}

func TestSegmentProcessor_CustomerOverlay(t *testing.T) {
	registry := NewSegmentRegistry("../../schemas")
	
	// Register B2 schema
	registry.RegisterSegment(&SegmentSchema{
		ID:      "B2",
		Name:    "Beginning Segment",
		Version: "004010",
		Elements: []ElementSchema{
			{
				Position:    1,
				RefID:       "B201",
				Name:        "Tariff Service Code",
				Type:        "AN",
				Requirement: "Optional",
				MinLength:   1,
				MaxLength:   2,
			},
			{
				Position:    2,
				RefID:       "B202",
				Name:        "Standard Carrier Alpha Code",
				Type:        "AN",
				Requirement: "Optional",
				MinLength:   2,
				MaxLength:   4,
			},
		},
	})

	processor := NewSegmentProcessor(registry)

	// Set customer requirements
	customerReq := &CustomerRequirements{
		PartnerID:       "TEST_PARTNER",
		Version:         "004010",
		TransactionType: "204",
		SegmentRules: map[string]SegmentOverlay{
			"B2": {
				SegmentID: "B2",
				Elements: map[int]ElementOverlay{
					1: {
						DefaultValue: "TL", // Default to Truckload
					},
					2: {
						Transform: "uppercase",
					},
				},
			},
		},
	}

	processor.SetCustomerRequirements(customerReq)

	segments := []x12.Segment{
		{
			Tag: "B2",
			Elements: [][]string{
				{""},     // Empty B201
				{"scac"}, // Lowercase SCAC
			},
			Index: 0,
		},
	}

	ctx := context.Background()
	processed, err := processor.ProcessSegments(ctx, segments, "004010")
	if err != nil {
		t.Fatalf("ProcessSegments failed: %v", err)
	}

	b2Segment := processed[0]

	// Check default value was applied
	if val, ok := b2Segment.Data["B201"]; !ok || val != "TL" {
		t.Errorf("Default value not applied to B201: %v", val)
	}

	// Check transform was applied
	if val, ok := b2Segment.Data["B202"]; !ok || val != "SCAC" {
		t.Errorf("Transform not applied to B202: %v", val)
	}

	// Check customer ID was set
	if b2Segment.CustomerID != "TEST_PARTNER" {
		t.Errorf("Customer ID not set: %s", b2Segment.CustomerID)
	}
}

func TestSegmentProcessor_ConditionalRules(t *testing.T) {
	registry := NewSegmentRegistry("../../schemas")
	
	// Register schemas
	registry.RegisterSegment(&SegmentSchema{
		ID:      "B2",
		Name:    "Beginning Segment",
		Version: "004010",
		Elements: []ElementSchema{
			{Position: 1, RefID: "B201", Name: "Tariff Service Code", Type: "AN", Requirement: "Optional"},
			{Position: 2, RefID: "B202", Name: "SCAC", Type: "AN", Requirement: "Optional"},
			{Position: 3, RefID: "B203", Name: "Shipment ID", Type: "AN", Requirement: "Optional"},
		},
	})

	registry.RegisterSegment(&SegmentSchema{
		ID:      "N1",
		Name:    "Name",
		Version: "004010",
		Elements: []ElementSchema{
			{Position: 1, RefID: "N101", Name: "Entity Identifier Code", Type: "ID", Requirement: "Mandatory"},
			{Position: 2, RefID: "N102", Name: "Name", Type: "AN", Requirement: "Optional"},
		},
	})

	processor := NewSegmentProcessor(registry)

	// Set conditional rule: If B201 = "TL", then N1*SH must be present
	customerReq := &CustomerRequirements{
		PartnerID:       "TEST_PARTNER",
		Version:         "004010",
		TransactionType: "204",
		Conditionals: []ConditionalRule{
			{
				ID:          "RULE_TL_SHIPPER",
				Description: "Truckload shipments require shipper information",
				When: Condition{
					Segment:  "B2",
					Element:  1,
					Operator: "equals",
					Value:    "TL",
				},
				Then: Requirement{
					Segment: "N1",
					Element: 1,
					MustBe:  "equal_to",
					Value:   "SH",
				},
				Severity: "error",
			},
		},
	}

	processor.SetCustomerRequirements(customerReq)

	// Test case 1: B201 = "TL" but no N1*SH segment
	segments1 := []x12.Segment{
		{Tag: "B2", Elements: [][]string{{"TL"}, {"SCAC"}, {"123"}}, Index: 0},
		{Tag: "N1", Elements: [][]string{{"BT"}, {"Bill To Name"}}, Index: 1},
	}

	ctx := context.Background()
	_, err := processor.ProcessSegments(ctx, segments1, "004010")
	if err == nil {
		t.Error("Expected error for missing N1*SH when B201=TL")
	}

	// Test case 2: B201 = "TL" with N1*SH segment present
	segments2 := []x12.Segment{
		{Tag: "B2", Elements: [][]string{{"TL"}, {"SCAC"}, {"123"}}, Index: 0},
		{Tag: "N1", Elements: [][]string{{"SH"}, {"Shipper Name"}}, Index: 1},
	}

	// Reset error collector for second test case
	processor.errorCollector = errors.NewErrorCollector(100, false)
	
	processed, err := processor.ProcessSegments(ctx, segments2, "004010")
	if err != nil {
		// Should succeed
		if processor.errorCollector.HasErrors() {
			// Check if it's just warnings
			errors := processor.errorCollector.GetBySeverity(3) // Error severity
			if len(errors) > 0 {
				t.Errorf("Unexpected errors when N1*SH is present: %v", err)
			}
		}
	}

	if len(processed) != 2 {
		t.Errorf("Expected 2 processed segments, got %d", len(processed))
	}
}

func TestSegmentProcessor_LoopTracking(t *testing.T) {
	registry := NewSegmentRegistry("../../schemas")
	
	// Register minimal schemas for testing
	registry.RegisterSegment(&SegmentSchema{ID: "ISA", Name: "Interchange", Version: "004010"})
	registry.RegisterSegment(&SegmentSchema{ID: "GS", Name: "Group", Version: "004010"})
	registry.RegisterSegment(&SegmentSchema{ID: "ST", Name: "Transaction", Version: "004010"})
	registry.RegisterSegment(&SegmentSchema{ID: "B2", Name: "Beginning", Version: "004010"})

	processor := NewSegmentProcessor(registry)

	segments := []x12.Segment{
		{Tag: "ISA", Elements: [][]string{}, Index: 0},
		{Tag: "GS", Elements: [][]string{}, Index: 1},
		{Tag: "ST", Elements: [][]string{}, Index: 2},
		{Tag: "B2", Elements: [][]string{}, Index: 3},
	}

	ctx := context.Background()
	processed, err := processor.ProcessSegments(ctx, segments, "004010")
	if err != nil {
		t.Fatalf("ProcessSegments failed: %v", err)
	}

	// Check position tracking
	for i, seg := range processed {
		if seg.Position.Index != i {
			t.Errorf("Segment %d: expected index %d, got %d", i, i, seg.Position.Index)
		}

		// ISA should set interchange
		if i == 0 && seg.Position.Interchange == "" {
			t.Error("ISA segment should set interchange control number")
		}

		// GS should set functional group
		if i == 1 && seg.Position.Functional == "" {
			t.Error("GS segment should set functional group control number")
		}

		// ST should set transaction
		if i == 2 && seg.Position.Transaction == "" {
			t.Error("ST segment should set transaction control number")
		}
	}
}

// Benchmark tests

func BenchmarkSegmentProcessor_Small(b *testing.B) {
	registry := NewSegmentRegistry("../../schemas")
	registry.RegisterSegment(&SegmentSchema{
		ID:      "B2",
		Name:    "Beginning Segment",
		Version: "004010",
		Elements: []ElementSchema{
			{Position: 1, Name: "Field1", Type: "AN", Requirement: "Optional"},
			{Position: 2, Name: "Field2", Type: "AN", Requirement: "Optional"},
			{Position: 3, Name: "Field3", Type: "AN", Requirement: "Optional"},
		},
	})

	processor := NewSegmentProcessor(registry)
	segments := []x12.Segment{
		{Tag: "B2", Elements: [][]string{{"A"}, {"B"}, {"C"}}, Index: 0},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessSegments(ctx, segments, "004010")
	}
}

func BenchmarkSegmentProcessor_Large(b *testing.B) {
	registry := NewSegmentRegistry("../../schemas")
	
	// Register common segment schemas
	for _, tag := range []string{"ISA", "GS", "ST", "B2", "N1", "N3", "N4", "S5", "SE", "GE", "IEA"} {
		registry.RegisterSegment(&SegmentSchema{
			ID:      tag,
			Name:    tag + " Segment",
			Version: "004010",
		})
	}

	processor := NewSegmentProcessor(registry)

	// Create a large set of segments
	segments := make([]x12.Segment, 0, 1000)
	for i := 0; i < 1000; i++ {
		segments = append(segments, x12.Segment{
			Tag:      "B2",
			Elements: [][]string{{"A"}, {"B"}, {"C"}},
			Index:    i,
		})
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessSegments(ctx, segments, "004010")
	}
}