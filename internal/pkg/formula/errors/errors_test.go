// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveError(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		entityType string
		cause      error
		wantMsg    string
	}{
		{
			name:       "basic resolve error",
			path:       "Customer.Name",
			entityType: "*shipment.Shipment",
			cause:      fmt.Errorf("field not found"),
			wantMsg:    "failed to resolve Customer.Name on *shipment.Shipment: field not found",
		},
		{
			name:       "nested path error",
			path:       "TractorType.Equipment.CostPerMile",
			entityType: "*shipment.Shipment",
			cause:      fmt.Errorf("nil pointer"),
			wantMsg:    "failed to resolve TractorType.Equipment.CostPerMile on *shipment.Shipment: nil pointer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewResolveError(tt.path, tt.entityType, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())

			// * Test unwrapping
			unwrapped := err.Unwrap()
			assert.Equal(t, tt.cause, unwrapped)
		})
	}
}

func TestTransformError(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		targetType string
		value      any
		cause      error
		wantMsg    string
	}{
		{
			name:       "decimal to float64 error",
			sourceType: "decimal.Decimal",
			targetType: "float64",
			value:      nil,
			cause:      fmt.Errorf("invalid decimal"),
			wantMsg:    "failed to transform decimal.Decimal to float64: invalid decimal",
		},
		{
			name:       "int64 to float64 error",
			sourceType: "*int64",
			targetType: "float64",
			value:      nil,
			cause:      fmt.Errorf("nil pointer"),
			wantMsg:    "failed to transform *int64 to float64: nil pointer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewTransformError(tt.sourceType, tt.targetType, tt.value, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())

			// * Test unwrapping
			unwrapped := err.Unwrap()
			assert.Equal(t, tt.cause, unwrapped)
		})
	}
}

func TestComputeError(t *testing.T) {
	tests := []struct {
		name       string
		function   string
		entityType string
		cause      error
		wantMsg    string
	}{
		{
			name:       "temperature differential error",
			function:   "computeTemperatureDifferential",
			entityType: "*shipment.Shipment",
			cause:      fmt.Errorf("missing temperature values"),
			wantMsg:    "failed to compute computeTemperatureDifferential for *shipment.Shipment: missing temperature values",
		},
		{
			name:       "hazmat check error",
			function:   "computeHasHazmat",
			entityType: "*shipment.Shipment",
			cause:      fmt.Errorf("commodities not loaded"),
			wantMsg:    "failed to compute computeHasHazmat for *shipment.Shipment: commodities not loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewComputeError(tt.function, tt.entityType, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())

			// * Test unwrapping
			unwrapped := err.Unwrap()
			assert.Equal(t, tt.cause, unwrapped)
		})
	}
}

func TestVariableError(t *testing.T) {
	tests := []struct {
		name         string
		variableName string
		context      string
		cause        error
		wantMsg      string
	}{
		{
			name:         "variable not found",
			variableName: "temperature_differential",
			context:      "shipment",
			cause:        fmt.Errorf("variable not registered"),
			wantMsg:      "failed to resolve variable 'temperature_differential' in context shipment: variable not registered",
		},
		{
			name:         "variable resolution failed",
			variableName: "hazmat_class",
			context:      "shipment",
			cause:        fmt.Errorf("no hazmat data"),
			wantMsg:      "failed to resolve variable 'hazmat_class' in context shipment: no hazmat data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewVariableError(tt.variableName, tt.context, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())

			// * Test unwrapping
			unwrapped := err.Unwrap()
			assert.Equal(t, tt.cause, unwrapped)
		})
	}
}

func TestSchemaError(t *testing.T) {
	tests := []struct {
		name     string
		schemaID string
		action   string
		cause    error
		wantMsg  string
	}{
		{
			name:     "schema compilation error",
			schemaID: "shipment-schema-v1",
			action:   "compile",
			cause:    fmt.Errorf("invalid JSON schema syntax"),
			wantMsg:  "schema error for 'shipment-schema-v1' during compile: invalid JSON schema syntax",
		},
		{
			name:     "schema validation error",
			schemaID: "shipment-schema-v1",
			action:   "validate",
			cause:    fmt.Errorf("required field missing: ProNumber"),
			wantMsg:  "schema error for 'shipment-schema-v1' during validate: required field missing: ProNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.NewSchemaError(tt.schemaID, tt.action, tt.cause)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())

			// * Test unwrapping
			unwrapped := err.Unwrap()
			assert.Equal(t, tt.cause, unwrapped)
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &errors.ValidationError{
		Field:   "Weight",
		Value:   -100,
		Message: "weight must be positive",
	}

	assert.Equal(t, "validation failed for field Weight: weight must be positive", err.Error())
}

func TestErrorChaining(t *testing.T) {
	// * Test that errors can be chained properly
	baseErr := fmt.Errorf("base error")
	computeErr := errors.NewComputeError("computeHasHazmat", "*shipment.Shipment", baseErr)
	resolveErr := errors.NewResolveError("Commodities", "*shipment.Shipment", computeErr)

	// * Check the full error message
	fullMsg := resolveErr.Error()
	assert.True(t, strings.Contains(fullMsg, "failed to resolve Commodities"))

	// * Check unwrapping works
	unwrapped1 := resolveErr.Unwrap()
	assert.Equal(t, computeErr, unwrapped1)

	unwrapped2 := unwrapped1.(*errors.ComputeError).Unwrap()
	assert.Equal(t, baseErr, unwrapped2)
}
