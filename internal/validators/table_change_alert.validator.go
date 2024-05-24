package validators

import (
	"context"
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/ent/tablechangealert"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
)

var validDataTypes = []string{
	"string", "int", "float64", "bool", "[]string", "[]int", "[]float64",
}

type ConditionalStructureError struct {
	Message string `json:"message"`
}

func (e *ConditionalStructureError) Error() string {
	return e.Message
}

// ValidateTableChangeAlerts is a function that validates a TableChangeAlertMutation.
//
// Parameters:
//
//	next ent.Mutator: The next mutator in the mutation operation.
//
// The function creates a TableChangeAlertFunc hook that validates the mutation. It first validates the source and then the database action.
// If any validation errors occur, it returns a ValidationErrorResponse with the errors. If no errors occur, it proceeds to the next mutation.
//
// Returns:
//
//	ent.Mutator: A mutator that validates the mutation.
func ValidateTableChangeAlerts(next ent.Mutator) ent.Mutator {
	return hook.TableChangeAlertFunc(func(ctx context.Context, m *ent.TableChangeAlertMutation) (ent.Value, error) {
		var errs []types.ValidationErrorDetail

		// Validate the delivery method
		validateEmailDeliveryMethod(m, &errs)

		if len(errs) > 0 {
			return nil, &types.ValidationErrorResponse{
				Type:   "validationError",
				Errors: errs,
			}
		}

		return next.Mutate(ctx, m)
	})
}

func validateEmailDeliveryMethod(m *ent.TableChangeAlertMutation, validationErrors *[]types.ValidationErrorDetail) {
	deliveryMethod, exists := m.DeliveryMethod()
	_, cexists := m.CustomSubject()
	_, eexists := m.EmailRecipients()

	if !exists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Attr:   "deliveryMethod",
			Detail: "Delivery method is required. Please try again.",
			Code:   "invalid",
		})
	}

	if deliveryMethod != tablechangealert.DeliveryMethodEmail && cexists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Attr:   "customSubject",
			Detail: "Custom subject is only allowed for `Email` delivery method. Please try again.",
			Code:   "invalid",
		})
	}

	if deliveryMethod != tablechangealert.DeliveryMethodEmail && eexists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Attr:   "emailRecipients",
			Detail: "Email recipients are only allowed for `Email` delivery method. Please try again.",
			Code:   "invalid",
		})
	}

	if deliveryMethod == tablechangealert.DeliveryMethodEmail && !eexists {
		*validationErrors = append(*validationErrors, types.ValidationErrorDetail{
			Attr:   "emailRecipients",
			Detail: "Email recipients are required for `Email` delivery method. Please try again.",
			Code:   "invalid",
		})
	}
}

// ValidateConditionalLogic validates the conditional logic provided in the TableChangeAlertMutation.
//
// Parameters:
//
//	data map[string]any: The conditional logic provided in the mutation.
//
// The function validates the conditional logic by checking if all required keys are present and if the data types are valid.
// It also checks if the operation and value are valid for each condition.
//
// Returns:
//
//	error: An error if the conditional logic is invalid.
func ValidateConditionalLogic(data map[string]any) error {
	requiredKeys := []string{"name", "description", "topic", "conditions"}

	// Check if required keys are present
	if err := validateRequiredKeys(data, requiredKeys); err != nil {
		return err
	}

	conditions, ok := data["conditions"].([]any)
	if !ok {
		errMsg := "Conditions should be a list"
		return &ConditionalStructureError{Message: errMsg}
	}

	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]any)
		if !ok {
			errMsg := "Each condition should be a map"
			return &ConditionalStructureError{Message: errMsg}
		}

		conditionRequiredKeys := []string{"id", "column", "operation", "value", "dataType"}

		// Check if all required keys are present in each condition
		if err := validateRequiredKeys(conditionMap, conditionRequiredKeys); err != nil {
			errMsg := fmt.Sprintf("Condition is missing required key: %v", err)
			return &ConditionalStructureError{Message: errMsg}
		}

		// Check if the operation is valid
		if !ContainsOperation(types.AvailableOperations, types.Operation(conditionMap["operation"].(string))) {
			errMsg := fmt.Sprintf("Invalid operation '%s' in condition", conditionMap["operation"])
			return &ConditionalStructureError{Message: errMsg}
		}

		// Check if the data type is valid
		if !isValidDataType(conditionMap["dataType"].(string)) {
			errMsg := fmt.Sprintf("Invalid data type '%s' in condition", conditionMap["dataType"])
			return &ConditionalStructureError{Message: errMsg}
		}

		// Additional checks for specific operations
		if err := validateOperationValue(conditionMap); err != nil {
			return err
		}
	}

	return nil
}

// validateOperationValue checks if the operation and value are valid for the provided condition.
func validateOperationValue(conditionMap map[string]any) error {
	switch conditionMap["operation"] {
	case "IN", "NOT_IN":
		if reflect.TypeOf(conditionMap["value"]).Kind() != reflect.Slice {
			return &ConditionalStructureError{Message: "Operation 'in' expects a list value"}
		}
	case "IS_NULL", "IS_NOT_NULL":
		if conditionMap["value"] != nil {
			return &ConditionalStructureError{Message: "Operation 'isnull or not_isnull' should not have a value"}
		}
	case "CONTAINS", "ICONTAINS":
		if _, ok := conditionMap["value"].(string); !ok {
			return &ConditionalStructureError{Message: "Operation 'contains or icontains' expects a string value"}
		}
	}
	return nil
}

// validateRequiredKeys checks if all required keys are present and not empty in the provided map.
func validateRequiredKeys(data map[string]any, requiredKeys []string) error {
	for _, key := range requiredKeys {
		value, exists := data[key]
		if !exists || value == "" {
			return &ConditionalStructureError{Message: fmt.Sprintf("Conditional Logic is missing required key: '%s'", key)}
		}
	}
	return nil
}

// isValidDataType checks if the provided data type is valid.
func isValidDataType(dataType string) bool {
	return util.ContainsString(validDataTypes, dataType)
}

// validateModelFieldsExist validates that the specified fields exist in the model.
func ValidateModelFieldsExist(data map[string]any, modelFields []string) error {
	conditions, ok := data["conditions"].([]any)
	if !ok {
		errMsg := "Conditions should be a list"
		return &ConditionalStructureError{Message: errMsg}
	}

	excludedFields := []string{"id", "organization_id", "business_unit_id"}

	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]any)
		column, ok := conditionMap["column"].(string)
		if !ok {
			errMsg := "Invalid column type in condition"
			return &ConditionalStructureError{Message: errMsg}
		}

		if !util.ContainsString(modelFields, column) {
			errMsg := fmt.Sprintf("Conditional Field '%s' does not exist", column)
			return &ConditionalStructureError{Message: errMsg}
		}
		if util.ContainsString(excludedFields, column) {
			errMsg := fmt.Sprintf("Conditional Field '%s' is not allowed", column)
			return &ConditionalStructureError{Message: errMsg}
		}
	}

	return nil
}

// ContainsOperation checks whether the given map contains the operation provided.
func ContainsOperation(operations map[types.Operation]struct{}, operation types.Operation) bool {
	_, exists := operations[operation]
	return exists
}
