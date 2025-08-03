/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentvalidator_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
)

func TestWeightComplianceRules(t *testing.T) {
	// Create a validation engine
	engine := framework.NewValidationEngine()

	// Create a shipment with weight exceeding the limit
	shp := &shipment.Shipment{
		Weight: new(int64),
	}
	*shp.Weight = 90000 // Exceeds the 80,000 lbs limit

	// Add weight compliance rules
	shipmentvalidator.AddWeightComplianceRules(engine, shp)

	// Validate and check for errors
	multiErr := engine.Validate(context.Background())

	// Should have an error
	if multiErr == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	// Check for specific error
	expectedError := struct {
		Field   string
		Code    errors.ErrorCode
		Message string
	}{
		Field:   "weight",
		Code:    errors.ErrInvalid,
		Message: "Total weight exceeds maximum allowed (80,000 lbs) for interstate transport. Current: 90000 lbs",
	}

	matcher := testutils.NewErrorMatcher(t, multiErr)
	matcher.HasError(expectedError.Field, expectedError.Code, expectedError.Message)
}

func TestHazmatComplianceRules(t *testing.T) {
	// Create a validation engine
	engine := framework.NewValidationEngine()

	// Create hazmat ID
	hazmatID := pulid.MustNew("haz_")

	// Create a shipment with hazmat commodities
	comm := &shipment.ShipmentCommodity{
		Commodity: &commodity.Commodity{
			HazardousMaterialID: &hazmatID,
		},
	}

	shp := &shipment.Shipment{
		Commodities: []*shipment.ShipmentCommodity{comm},
	}

	// Add hazmat compliance rules
	shipmentvalidator.AddHazmatComplianceRules(engine, shp)

	// Validate and check for errors
	multiErr := engine.Validate(context.Background())

	// Should have an error
	if multiErr == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	// Check for specific error
	expectedError := struct {
		Field   string
		Code    errors.ErrorCode
		Message string
	}{
		Field:   "hazmat",
		Code:    errors.ErrInvalid,
		Message: "Hazardous materials documentation is required for shipments containing hazardous materials",
	}

	matcher := testutils.NewErrorMatcher(t, multiErr)
	matcher.HasError(expectedError.Field, expectedError.Code, expectedError.Message)
}
