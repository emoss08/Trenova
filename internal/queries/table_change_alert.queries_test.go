package queries_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateInsertFieldString(t *testing.T) {
	fields := []string{"id", "name", "created_at", "email", "updated_at", "organization_id", "address"}
	maxLength := 10
	expected := "'name', new.name, 'email', new.email, 'address', new.address"

	result := queries.CreateInsertFieldString(fields, maxLength)
	require.Equal(t, expected, result)
}

func TestFormatValueForOperation(t *testing.T) {
	tests := []struct {
		operation types.Operation
		value     any
		expected  string
		hasError  bool
	}{
		{types.OperationEquals, "example", "'example'", false},
		{types.OperationNotEquals, "example", "'example'", false},
		{types.OperationGreaterThan, 5, "'5'", false},
		{types.OperationGreaterThanOrEqual, 5, "'5'", false},
		{types.OperationLessThan, 5, "'5'", false},
		{types.OperationLessThanOrEqual, 5, "'5'", false},
		{types.OperationContains, "example", "'%example%'", false},
		{types.OperationIcontains, "example", "'%example%'", false},
		{types.OperationIn, []any{"a", "b", "c"}, "('a','b','c')", false},
		{types.OperationNotIn, []any{"a", "b", "c"}, "('a','b','c')", false},
		{types.OperationIsNull, nil, "", false},
		{types.OperationIsNotNull, nil, "", false},
		{types.Operation("UNSUPPORTED"), "example", "", true},
	}

	for _, test := range tests {
		result, err := queries.FormatValueForOperation(test.operation, test.value)
		if test.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		}
	}
}

func TestBuildConditionString(t *testing.T) {
	tests := []struct {
		column    string
		operation types.Operation
		value     any
		expected  string
		hasError  bool
	}{
		{"name", types.OperationEquals, "example", "name = 'example'", false},
		{"age", types.OperationGreaterThan, 30, "age > '30'", false},
		{"email", types.OperationContains, "example", "email LIKE '%example%'", false},
		{"status", types.OperationIsNull, nil, "status IS NULL", false},
		{"category", types.OperationIn, []any{"a", "b", "c"}, "category IN ('a','b','c')", false},
		{"unsupported", types.Operation("UNSUPPORTED"), "example", "", true},
	}

	for _, test := range tests {
		result, err := queries.BuildConditionString(test.column, test.operation, test.value)
		if test.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		}
	}
}

func TestBuildConditionalLogicSQL(t *testing.T) {
	tests := []struct {
		data     map[string]any
		expected string
		hasError bool
	}{
		{
			data: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "name",
						"operation": "EQUALS",
						"value":     "example",
					},
					map[string]any{
						"column":    "age",
						"operation": "GREATER_THAN",
						"value":     30,
					},
				},
			},
			expected: "new.name = 'example' AND new.age > '30'",
			hasError: false,
		},
		{
			data: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "email",
						"operation": "CONTAINS",
						"value":     "example",
					},
					map[string]any{
						"column":    "status",
						"operation": "IS_NULL",
						"value":     nil,
					},
				},
			},
			expected: "new.email LIKE '%example%' AND new.status IS NULL",
			hasError: false,
		},
		{
			data: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "category",
						"operation": "IN",
						"value":     []any{"a", "b", "c"},
					},
				},
			},
			expected: "new.category IN ('a','b','c')",
			hasError: false,
		},
		{
			data: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "unsupported",
						"operation": "UNSUPPORTED",
						"value":     "example",
					},
				},
			},
			expected: "",
			hasError: true,
		},
	}

	for _, test := range tests {
		result, err := queries.BuildConditionalLogicSQL(test.data)
		if test.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		}
	}
}

func TestCreateInsertFunctionString(t *testing.T) {
	tests := []struct {
		listenerName      string
		functionName      string
		fields            []string
		orgID             uuid.UUID
		conditionalLogic  map[string]any
		expectedSubstring string
	}{
		{
			listenerName: "my_listener",
			functionName: "my_insert_function",
			fields:       []string{"id", "name", "created_at", "email", "updated_at", "organization_id", "address"},
			orgID:        uuid.New(),
			conditionalLogic: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "name",
						"operation": "EQUALS",
						"value":     "example",
					},
					map[string]any{
						"column":    "age",
						"operation": "GREATER_THAN",
						"value":     30,
					},
				},
			},
			expectedSubstring: "CREATE OR REPLACE FUNCTION my_insert_function()",
		},
		{
			listenerName: "another_listener",
			functionName: "another_insert_function",
			fields:       []string{"id", "description", "timestamp", "email", "modified_at", "org_id", "location"},
			orgID:        uuid.New(),
			conditionalLogic: map[string]any{
				"conditions": []any{
					map[string]any{
						"column":    "description",
						"operation": "CONTAINS",
						"value":     "test",
					},
					map[string]any{
						"column":    "status",
						"operation": "IS_NULL",
						"value":     nil,
					},
				},
			},
			expectedSubstring: "CREATE OR REPLACE FUNCTION another_insert_function()",
		},
	}

	for _, test := range tests {
		t.Run(test.functionName, func(t *testing.T) {
			result := queries.CreateInsertFunctionString(
				test.listenerName,
				test.functionName,
				test.fields,
				test.orgID,
				test.conditionalLogic,
			)

			t.Logf("Result: %s", result)

			require.Contains(t, result, test.expectedSubstring)
			require.Contains(t, result, test.listenerName)
			require.Contains(t, result, test.functionName)
			require.Contains(t, result, test.orgID.String())
		})
	}
}
