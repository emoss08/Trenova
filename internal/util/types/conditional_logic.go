package types

import "fmt"

// Operation represents a type of SQL operation.
type Operation string

// Constants for available operations.
const (
	OperationEquals             Operation = "EQUALS"
	OperationNotEquals          Operation = "NOT_EQUALS"
	OperationGreaterThan        Operation = "GREATER_THAN"
	OperationGreaterThanOrEqual Operation = "GREATER_THAN_OR_EQUAL"
	OperationLessThan           Operation = "LESS_THAN"
	OperationLessThanOrEqual    Operation = "LESS_THAN_OR_EQUAL"
	OperationContains           Operation = "CONTAINS"
	OperationIcontains          Operation = "ICONTAINS"
	OperationIn                 Operation = "IN"
	OperationNotIn              Operation = "NOT_IN"
	OperationIsNull             Operation = "IS_NULL"
	OperationIsNotNull          Operation = "IS_NOT_NULL"
)

// AvailableOperations lists all supported operations.
var AvailableOperations = map[Operation]struct{}{
	OperationEquals:             {},
	OperationNotEquals:          {},
	OperationGreaterThan:        {},
	OperationGreaterThanOrEqual: {},
	OperationLessThan:           {},
	OperationLessThanOrEqual:    {},
	OperationContains:           {},
	OperationIcontains:          {},
	OperationIn:                 {},
	OperationNotIn:              {},
	OperationIsNull:             {},
	OperationIsNotNull:          {},
}

// OperationMapping maps operations to their SQL representations.
var OperationMapping = map[Operation]string{
	OperationEquals:             "=",
	OperationNotEquals:          "!=",
	OperationGreaterThan:        ">",
	OperationGreaterThanOrEqual: ">=",
	OperationLessThan:           "<",
	OperationLessThanOrEqual:    "<=",
	OperationContains:           "LIKE",
	OperationIcontains:          "ILIKE",
	OperationIn:                 "IN",
	OperationNotIn:              "NOT IN",
	OperationIsNull:             "IS NULL",
	OperationIsNotNull:          "IS NOT NULL",
}

// InvalidOperationError represents an error for unsupported operations.
type InvalidOperationError struct {
	Operation Operation
}

func (e InvalidOperationError) Error() string {
	return fmt.Sprintf("operation %s is not supported", e.Operation)
}

// TableChangeAlertCondition represents a condition in the table change alert.
type TableChangeAlertCondition struct {
	ID        int    `json:"id"`
	Column    string `json:"column"`
	Operation string `json:"operation"`
	Value     any    `json:"value"`
	DataType  string `json:"dataType"`
}

// TableChangeAlertConditionalLogic represents the conditional logic for a table change alert.
type TableChangeAlertConditionalLogic struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	TableName   string                      `json:"tableName"`
	Conditions  []TableChangeAlertCondition `json:"conditions"`
}
