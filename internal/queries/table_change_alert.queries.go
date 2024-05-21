package queries

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/google/uuid"
)

// CreateInsertFieldString creates a string of field-value pairs for use in a SQL INSERT statement.
// This function generates a string representation of field-value pairs, excluding certain fields like 'id',
// 'created', 'modified', and 'organization_id'. It is used in constructing a dynamic SQL INSERT statement.
//
// Parameters:
//
//	fields ([]string): A list of field names to include in the INSERT statement.
//
// Returns:
//
//	string: A string of field-value pairs for a SQL INSERT statement.
func CreateInsertFieldString(fields []string, maxLength int) string {
	excludedFields := map[string]struct{}{
		"id":               {},
		"created_at":       {},
		"updated_at":       {},
		"organization_id":  {},
		"business_unit_id": {},
	}

	var fieldStrings []string
	for _, field := range fields {
		if _, excluded := excludedFields[field]; !excluded {
			truncatedField := util.TruncateName(field, maxLength)
			fieldStrings = append(fieldStrings, fmt.Sprintf("'%s', new.%s", truncatedField, truncatedField))
		}
	}

	if len(fieldStrings) == 0 {
		return ""
	}

	return strings.Join(fieldStrings[:len(fieldStrings)-1], ", ") + (", " + fieldStrings[len(fieldStrings)-1])
}

// FormatValueForOperation formats a value for SQL operations based on the specified operation type.
func FormatValueForOperation(operation types.Operation, value any) (string, error) {
	if _, ok := types.AvailableOperations[operation]; !ok {
		return "", types.InvalidOperationError{Operation: operation}
	}

	switch operation {
	case types.OperationEquals, types.OperationNotEquals, types.OperationGreaterThan, types.OperationGreaterThanOrEqual, types.OperationLessThan, types.OperationLessThanOrEqual:
		return fmt.Sprintf("'%v'", value), nil
	case types.OperationContains, types.OperationIcontains:
		return fmt.Sprintf("'%%%v%%'", value), nil
	case types.OperationIn, types.OperationNotIn:
		switch v := value.(type) {
		case []any:
			var formattedValues []string
			for _, item := range v {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", item))
			}
			return fmt.Sprintf("(%s)", strings.Join(formattedValues, ",")), nil
		default:
			return fmt.Sprintf("('%v')", value), nil
		}
	case types.OperationIsNull, types.OperationIsNotNull:
		return "", nil
	default:
		return fmt.Sprintf("'%v'", value), nil
	}
}

// BuildConditionString builds a SQL condition string for a given column, operation, and value.
func BuildConditionString(column string, operation types.Operation, value any) (string, error) {
	if _, ok := types.AvailableOperations[operation]; !ok {
		return "", types.InvalidOperationError{Operation: operation}
	}

	formattedValue, err := FormatValueForOperation(operation, value)
	if err != nil {
		return "", err
	}

	if operation == types.OperationIsNull || operation == types.OperationIsNotNull {
		return fmt.Sprintf("%s %s", column, types.OperationMapping[operation]), nil
	}

	return fmt.Sprintf("%s %s %s", column, types.OperationMapping[operation], formattedValue), nil
}

// BuildConditionalLogicSQL constructs a SQL conditional logic string from a given data map.
func BuildConditionalLogicSQL(data map[string]any) (string, error) {
	conditionsInterface, ok := data["conditions"]
	if !ok {
		return "", errors.New("missing 'conditions' key in data")
	}

	conditions, ok := conditionsInterface.([]any)
	if !ok {
		return "", errors.New("'conditions' should be a list")
	}

	var conditionStrings []string

	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]any)
		if !ok {
			return "", errors.New("each condition should be a map")
		}

		column, ok := conditionMap["column"].(string)
		if !ok {
			return "", errors.New("each condition must have a 'column' of type string")
		}

		operationStr, ok := conditionMap["operation"].(string)
		if !ok {
			return "", errors.New("each condition must have an 'operation' of type string")
		}

		operation := types.Operation(operationStr)
		value := conditionMap["value"]

		conditionStr, err := BuildConditionString(fmt.Sprintf("new.%s", column), operation, value)
		if err != nil {
			return "", err
		}

		conditionStrings = append(conditionStrings, conditionStr)
	}

	return strings.Join(conditionStrings, " AND "), nil
}

func CreateInsertFunctionString(listenerName, functionName string, fields []string, orgID uuid.UUID, conditionalLogic map[string]any) string {
	fieldsString := CreateInsertFieldString(fields, 63) // PostgreSQL max identifier length
	whereClause, _ := BuildConditionalLogicSQL(conditionalLogic)

	if whereClause == "" {
		whereClause = "TRUE"
	}

	return fmt.Sprintf(`
	CREATE OR REPLACE FUNCTION %s()
	RETURNS trigger
	LANGUAGE 'plpgsql'
	AS $BODY$
	BEGIN
		IF TG_OP = 'INSERT' AND NEW.organization_id = '%s' AND (%s) THEN
			PERFORM pg_notify('%s',
				json_build_object(
					%s
				)::text);
		END IF;
		RETURN NULL;
	END
	$BODY$;
	`, functionName, orgID, whereClause, listenerName, fieldsString)
}

func CreateInsertFunction(ctx context.Context, tx *ent.Tx, listenerName, functionName string, fields []string, orgID uuid.UUID, conditionalLogic map[string]any) error {
	query := CreateInsertFunctionString(listenerName, functionName, fields, orgID, conditionalLogic)
	_, err := tx.QueryContext(ctx, query)
	return err
}
