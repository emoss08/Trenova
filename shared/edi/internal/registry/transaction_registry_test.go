package registry

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/transactions"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func TestTransactionRegistry(t *testing.T) {
	// Setup
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}
	
	registry := NewTransactionRegistry(segRegistry, delims)
	
	t.Run("RegisterAndRetrieveBuilder", func(t *testing.T) {
		// Register a 997 builder
		builder997 := transactions.NewAck997Builder(segRegistry, "004010", delims)
		err := registry.RegisterBuilder("997", "004010", builder997)
		if err != nil {
			t.Fatalf("Failed to register builder: %v", err)
		}
		
		// Retrieve it
		retrieved, err := registry.GetBuilder("997", "004010")
		if err != nil {
			t.Fatalf("Failed to retrieve builder: %v", err)
		}
		
		if retrieved.GetTransactionType() != "997" {
			t.Errorf("Expected transaction type 997, got %s", retrieved.GetTransactionType())
		}
		
		// Try to retrieve non-existent
		_, err = registry.GetBuilder("999", "005010")
		if err == nil {
			t.Error("Expected error for non-existent builder")
		}
	})
	
	t.Run("RegisterAndRetrieveConfig", func(t *testing.T) {
		// Register a 204 config
		cfg204 := config.Example204Config()
		err := registry.RegisterConfig(cfg204)
		if err != nil {
			t.Fatalf("Failed to register config: %v", err)
		}
		
		// Retrieve builder created from config
		builder, err := registry.GetBuilder("204", "004010")
		if err != nil {
			t.Fatalf("Failed to get builder from config: %v", err)
		}
		
		if builder.GetTransactionType() != "204" {
			t.Errorf("Expected transaction type 204, got %s", builder.GetTransactionType())
		}
	})
	
	t.Run("CustomerSpecificBuilder", func(t *testing.T) {
		// Register config with customer overrides
		cfg := config.Example204Config()
		cfg.CustomerOverrides = map[string]config.CustomerConfig{
			"CUST001": {
				CustomerID:   "CUST001",
				CustomerName: "Test Customer",
				Active:       true,
				DefaultValues: map[string]map[int]any{
					"B2": {
						1: "TL",
					},
				},
			},
		}
		
		err := registry.RegisterConfig(cfg)
		if err != nil {
			t.Fatalf("Failed to register config with customer overrides: %v", err)
		}
		
		// Get builder for customer
		builder, err := registry.GetBuilderForCustomer("204", "004010", "CUST001")
		if err != nil {
			t.Fatalf("Failed to get customer builder: %v", err)
		}
		
		if builder.GetTransactionType() != "204" {
			t.Errorf("Expected transaction type 204, got %s", builder.GetTransactionType())
		}
		
		// Get builder for unknown customer (should use defaults)
		builder2, err := registry.GetBuilderForCustomer("204", "004010", "UNKNOWN")
		if err != nil {
			t.Fatalf("Failed to get builder for unknown customer: %v", err)
		}
		
		if builder2.GetTransactionType() != "204" {
			t.Errorf("Expected transaction type 204, got %s", builder2.GetTransactionType())
		}
	})
	
	t.Run("LoadDefaultTransactions", func(t *testing.T) {
		err := registry.LoadDefaultTransactions()
		if err != nil {
			t.Fatalf("Failed to load default transactions: %v", err)
		}
		
		// Check that defaults are loaded
		types := registry.ListTransactionTypes()
		if len(types) < 3 {
			t.Errorf("Expected at least 3 transaction types, got %d", len(types))
		}
		
		// Verify specific types
		has997 := false
		has999 := false
		has204 := false
		
		for _, info := range types {
			switch info.Type {
			case "997":
				has997 = true
				if !info.HasBuilder {
					t.Error("997 should have a builder")
				}
			case "999":
				has999 = true
				if !info.HasBuilder {
					t.Error("999 should have a builder")
				}
			case "204":
				has204 = true
				if !info.HasBuilder {
					t.Error("204 should have a builder")
				}
			}
		}
		
		if !has997 {
			t.Error("Missing 997 transaction type")
		}
		if !has999 {
			t.Error("Missing 999 transaction type")
		}
		if !has204 {
			t.Error("Missing 204 transaction type")
		}
	})
	
	t.Run("BuildTransaction", func(t *testing.T) {
		// Register a 997 builder
		builder997 := transactions.NewAck997Builder(segRegistry, "004010", delims)
		err := registry.RegisterBuilder("997", "004010", builder997)
		if err != nil {
			t.Fatalf("Failed to register builder: %v", err)
		}
		
		// Build a 997
		req := &transactions.Ack997Request{
			OriginalISA: transactions.InterchangeEnvelope{
				AuthQualifier:     "00",
				AuthInfo:          "          ",
				SecurityQualifier: "00",
				SecurityInfo:      "          ",
				SenderQualifier:   "ZZ",
				SenderID:          "SENDER",
				ReceiverQualifier: "ZZ",
				ReceiverID:        "RECEIVER",
				StandardsID:       "U",
				Version:           "00401",
				ControlNumber:     "000000001",
				AckRequested:      "0",
				Usage:             "P",
			},
			OriginalGS: transactions.FunctionalGroupEnvelope{
				FunctionalID:      "SM",
				SenderCode:        "SENDER",
				ReceiverCode:      "RECEIVER",
				ControlNumber:     "1",
				ResponsibleAgency: "X",
				Version:           "004010",
			},
			Transactions: []transactions.TransactionInfo{
				{
					SetID:      "204",
					Control:    "0001",
					StartIndex: 2,
					EndIndex:   10,
				},
			},
		}
		
		ctx := context.Background()
		result, err := registry.Build(ctx, "997", "004010", req)
		if err != nil {
			t.Fatalf("Failed to build 997: %v", err)
		}
		
		if result == "" {
			t.Error("Expected non-empty result")
		}
		
		// Should contain key segments
		if !contains(result, "ISA") {
			t.Error("Result should contain ISA segment")
		}
		if !contains(result, "ST*997") {
			t.Error("Result should contain ST*997 segment")
		}
		if !contains(result, "AK1") {
			t.Error("Result should contain AK1 segment")
		}
	})
}

func TestConfigurableTransactionBuilder(t *testing.T) {
	segRegistry := segments.NewSegmentRegistry("../../schemas")
	delims := x12.Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}
	
	t.Run("ImplementsInterface", func(t *testing.T) {
		cfg := config.Example204Config()
		configBuilder := config.NewConfigurableBuilder(cfg, nil, segRegistry, delims)
		
		builder := &ConfigurableTransactionBuilder{
			builder: configBuilder,
			config:  cfg,
		}
		
		// Verify it implements TransactionBuilder interface
		var _ transactions.TransactionBuilder = builder
		
		if builder.GetTransactionType() != "204" {
			t.Errorf("Expected transaction type 204, got %s", builder.GetTransactionType())
		}
		
		if builder.GetVersion() != "004010" {
			t.Errorf("Expected version 004010, got %s", builder.GetVersion())
		}
	})
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}