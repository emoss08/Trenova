package main

import (
	"context"
	"fmt"
	"log"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/zap"
)

// Example of using the sequence generator for shipment pro numbers
func main() {
	// Initialize dependencies
	logger, _ := zap.NewProduction()

	// In production, these would be actual implementations
	// For this example, we'll use mock implementations
	store := &ExampleSequenceStore{}
	provider := &ExampleFormatProvider{}

	// Create generator
	params := seqgen.GeneratorParams{
		Store:    store,
		Provider: provider,
		Logger:   logger,
	}
	generator := seqgen.NewGenerator(params)

	ctx := context.Background()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	// Method 1: Generate a single shipment pro number (compact, no separators)
	fmt.Println("=== Single Shipment Pro Number ===")
	proNumber, err := generator.GenerateShipmentProNumber(ctx, orgID, buID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated Pro Number: %s\n", proNumber)
	// Example output: S241200011234567 (15 characters total)

	// Method 2: Generate batch of shipment pro numbers
	fmt.Println("\n=== Batch Shipment Pro Numbers ===")
	proNumbers, err := generator.GenerateShipmentProNumberBatch(ctx, orgID, buID, 5)
	if err != nil {
		log.Fatal(err)
	}
	for i, pn := range proNumbers {
		fmt.Printf("Pro Number %d: %s\n", i+1, pn)
	}

	// Method 3: Custom format with location code
	fmt.Println("\n=== Custom Format with Location ===")
	customFormat := seqgen.DefaultShipmentProNumberFormat()
	customFormat.IncludeLocationCode = true
	customFormat.LocationCode = "LAX"

	req := &seqgen.GenerateRequest{
		Type:   seqgen.SequenceTypeProNumber,
		OrgID:  orgID,
		BuID:   buID,
		Format: customFormat,
	}
	customProNumber, err := generator.Generate(ctx, req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Custom Pro Number: %s\n", customProNumber)
	// Example output: S2412LAX00011234567

	// Method 4: High-volume scenario
	fmt.Println("\n=== High Volume Configuration ===")
	highVolumeFormat := seqgen.DefaultShipmentProNumberFormat()
	highVolumeFormat.SequenceDigits = 7 // Support up to 9,999,999 per month

	req = &seqgen.GenerateRequest{
		Type:   seqgen.SequenceTypeProNumber,
		OrgID:  orgID,
		BuID:   buID,
		Count:  100, // Generate 100 at once for efficiency
		Format: highVolumeFormat,
	}
	batchProNumbers, err := generator.GenerateBatch(ctx, req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated %d pro numbers for high volume\n", len(batchProNumbers))
	fmt.Printf("First: %s\n", batchProNumbers[0])
	fmt.Printf("Last:  %s\n", batchProNumbers[len(batchProNumbers)-1])

	// Validate a pro number
	fmt.Println("\n=== Validation ===")
	validationFormat := seqgen.DefaultShipmentProNumberFormat()
	err = generator.ValidateSequence(proNumber, validationFormat)
	if err == nil {
		fmt.Println("✓ Pro number is valid")
	} else {
		fmt.Printf("✗ Invalid: %v\n", err)
	}

	// Try to validate an invalid pro number with separators
	invalidProNumber := "S-2412-0001-123456"
	err = generator.ValidateSequence(invalidProNumber, validationFormat)
	if err != nil {
		fmt.Printf("✗ Invalid (expected): %v\n", err)
	}
}

// Mock implementations for the example
type ExampleSequenceStore struct{}

func (s *ExampleSequenceStore) GetNextSequence(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) (int64, error) {
	// In production, this would query the database
	return 1, nil
}

func (s *ExampleSequenceStore) GetNextSequenceBatch(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) ([]int64, error) {
	// In production, this would query the database
	sequences := make([]int64, req.Count)
	for i := range sequences {
		sequences[i] = int64(i + 1)
	}
	return sequences, nil
}

type ExampleFormatProvider struct{}

func (p *ExampleFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType seqgen.SequenceType,
	orgID, buID pulid.ID,
) (*seqgen.Format, error) {
	// In production, this would fetch from database
	return seqgen.DefaultShipmentProNumberFormat(), nil
}
