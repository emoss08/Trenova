package validators_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/util/validators"
	"github.com/stretchr/testify/require"
)

func TestValidateConditionalLogic_Valid(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.NoError(t, err)
}

func TestValidateConditionalLogic_MissingRequiredKey_Name(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Logic is missing required key: 'name'")
}

func TestValidateConditionalLogic_InvalidOperation(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "invalid_op",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid operation 'invalid_op'")
}

func TestValidateConditionalLogic_InvalidDataType(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "invalid_type",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid data type 'invalid_type'")
}

func TestValidateConditionalLogic_OperationInExpectsList(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "in",
				Value:     "not_a_list",
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'in' expects a list value")
}

func TestValidateConditionalLogic_OperationIsNullShouldNotHaveValue(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "isnull",
				Value:     "should_not_be_here",
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'isnull or not_isnull' should not have a value")
}

func TestValidateConditionalLogic_OperationContainsExpectsString(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "contains",
				Value:     12345, // Not a string
				DataType:  "string",
			},
		},
	}

	err := validators.ValidateConditionalLogic(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Operation 'contains or icontains' expects a string value")
}

func TestValidateModelFieldsExist_Valid(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "example_column",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.NoError(t, err)
}

func TestValidateModelFieldsExist_FieldDoesNotExist(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "nonexistent_column",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Field 'nonexistent_column' does not exist")
}

func TestValidateModelFieldsExist_FieldNotAllowed(t *testing.T) {
	data := validators.TableChangeAlertConditionalLogic{
		Name:        "Example Logic",
		Description: "This is an example",
		TableName:   "example_table",
		Conditions: []validators.TableChangeAlertCondition{
			{
				ID:        1,
				Column:    "organization_id",
				Operation: "eq",
				Value:     "example_value",
				DataType:  "string",
			},
		},
	}

	modelFields := []string{"example_column", "another_column", "organization_id"}

	err := validators.ValidateModelFieldsExist(data, modelFields)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Conditional Field 'organization_id' is not allowed")
}
