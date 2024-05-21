package validators_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/validators"
	"github.com/stretchr/testify/require"
)

func TestValidateConditionalLogic_Valid(t *testing.T) {
	data := map[string]interface{}{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "EQUALS",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.NoError(t, err)
}

func TestValidateConditionalLogic_MissingRequiredKey_Name(t *testing.T) {
	data := map[string]any{
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "eq",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Logic is missing required key: 'name'")
}

func TestValidateConditionalLogic_InvalidOperation(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "UNSUPPORTED",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid operation 'UNSUPPORTED'")
}

func TestValidateConditionalLogic_InvalidDataType(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "EQUALS",
				"value":     "example_value",
				"dataType":  "invalid_type",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid data type 'invalid_type'")
}

func TestValidateConditionalLogic_OperationInExpectsList(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "IN",
				"value":     "not_a_list",
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'in' expects a list value")
}

func TestValidateConditionalLogic_OperationIsNullShouldNotHaveValue(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "IS_NULL",
				"value":     "should_not_be_here",
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'isnull or not_isnull' should not have a value")
}

func TestValidateConditionalLogic_OperationContainsExpectsString(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "CONTAINS",
				"value":     12345, // Not a string
				"dataType":  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'contains or icontains' expects a string value")
}

func TestValidateModelFieldsExist_Valid(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "example_column",
				"operation": "EQUALS",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.NoError(t, err)
}

func TestValidateModelFieldsExist_FieldDoesNotExist(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "nonexistent_column",
				"operation": "EQUALS",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Field 'nonexistent_column' does not exist")
}

func TestValidateModelFieldsExist_FieldNotAllowed(t *testing.T) {
	data := map[string]any{
		"name":        "Example Logic",
		"description": "This is an example",
		"tableName":   "example_table",
		"conditions": []any{
			map[string]any{
				"id":        1,
				"column":    "organization_id",
				"operation": "EQUALS",
				"value":     "example_value",
				"dataType":  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column", "organization_id"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Field 'organization_id' is not allowed")
}
