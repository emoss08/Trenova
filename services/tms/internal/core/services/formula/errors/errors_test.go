package errors_test

import (
	"errors"
	"testing"

	formulaerrors "github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveError(t *testing.T) {
	t.Parallel()

	t.Run("NewResolveError sets all fields", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewResolveError("field.path", "Shipment", cause)

		assert.Equal(t, "field.path", err.Path)
		assert.Equal(t, "Shipment", err.EntityType)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewResolveError("weight", "Order", cause)

		assert.Equal(t, "failed to resolve weight on Order: cause", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewResolveError("path", "Entity", cause)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewResolveError("path", "Entity", cause)

		require.True(t, errors.Is(err, cause))
	})

	t.Run("errors.As works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewResolveError("path", "Entity", cause)

		var resolveErr *formulaerrors.ResolveError
		require.True(t, errors.As(err, &resolveErr))
		assert.Equal(t, "path", resolveErr.Path)
	})
}

func TestValidationError(t *testing.T) {
	t.Parallel()

	t.Run("constructor sets all fields", func(t *testing.T) {
		t.Parallel()
		err := &formulaerrors.ValidationError{
			Field:   "amount",
			Value:   -1,
			Message: "must be positive",
		}

		assert.Equal(t, "amount", err.Field)
		assert.Equal(t, -1, err.Value)
		assert.Equal(t, "must be positive", err.Message)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		err := &formulaerrors.ValidationError{
			Field:   "rate",
			Value:   0,
			Message: "cannot be zero",
		}

		assert.Equal(t, "validation failed for field rate: cannot be zero", err.Error())
	})
}

func TestTransformError(t *testing.T) {
	t.Parallel()

	t.Run("NewTransformError sets all fields", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewTransformError("string", "int", "abc", cause)

		assert.Equal(t, "string", err.SourceType)
		assert.Equal(t, "int", err.TargetType)
		assert.Equal(t, "abc", err.Value)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewTransformError("float", "decimal", 3.14, cause)

		assert.Equal(t, "failed to transform float to decimal: cause", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewTransformError("a", "b", nil, cause)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewTransformError("a", "b", nil, cause)

		require.True(t, errors.Is(err, cause))
	})

	t.Run("errors.As works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewTransformError("string", "int", "val", cause)

		var transformErr *formulaerrors.TransformError
		require.True(t, errors.As(err, &transformErr))
		assert.Equal(t, "string", transformErr.SourceType)
	})
}

func TestComputeError(t *testing.T) {
	t.Parallel()

	t.Run("NewComputeError sets all fields", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewComputeError("SUM", "LineItem", cause)

		assert.Equal(t, "SUM", err.Function)
		assert.Equal(t, "LineItem", err.EntityType)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewComputeError("AVG", "Charge", cause)

		assert.Equal(t, "failed to compute AVG for Charge: cause", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewComputeError("fn", "type", cause)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewComputeError("fn", "type", cause)

		require.True(t, errors.Is(err, cause))
	})

	t.Run("errors.As works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewComputeError("MAX", "Rate", cause)

		var computeErr *formulaerrors.ComputeError
		require.True(t, errors.As(err, &computeErr))
		assert.Equal(t, "MAX", computeErr.Function)
	})
}

func TestVariableError(t *testing.T) {
	t.Parallel()

	t.Run("NewVariableError sets all fields", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewVariableError("totalWeight", "shipment", cause)

		assert.Equal(t, "totalWeight", err.VariableName)
		assert.Equal(t, "shipment", err.Context)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewVariableError("mileage", "route", cause)

		assert.Equal(t, "failed to resolve variable 'mileage' in context route: cause", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewVariableError("var", "ctx", cause)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewVariableError("var", "ctx", cause)

		require.True(t, errors.Is(err, cause))
	})

	t.Run("errors.As works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewVariableError("distance", "leg", cause)

		var varErr *formulaerrors.VariableError
		require.True(t, errors.As(err, &varErr))
		assert.Equal(t, "distance", varErr.VariableName)
	})
}

func TestSchemaError(t *testing.T) {
	t.Parallel()

	t.Run("NewSchemaError sets all fields", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewSchemaError("schema-123", "validation", cause)

		assert.Equal(t, "schema-123", err.SchemaID)
		assert.Equal(t, "validation", err.Action)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Error returns expected format", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewSchemaError("abc-456", "parsing", cause)

		assert.Equal(t, "schema error for 'abc-456' during parsing: cause", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewSchemaError("id", "act", cause)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors.Is works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewSchemaError("id", "act", cause)

		require.True(t, errors.Is(err, cause))
	})

	t.Run("errors.As works with unwrapped error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("cause")
		err := formulaerrors.NewSchemaError("schema-xyz", "compile", cause)

		var schemaErr *formulaerrors.SchemaError
		require.True(t, errors.As(err, &schemaErr))
		assert.Equal(t, "schema-xyz", schemaErr.SchemaID)
	})
}
