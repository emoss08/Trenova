package utils

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v2"
)

// ParseBodyAndValidate parses the request body, validates it, and returns a ProblemDetail response if there are validation errors.
func ParseBodyAndValidate(c *fiber.Ctx, data any) error {
	if err := c.BodyParser(data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	if err := validation.Validate(data); err != nil {
		var invalidParams []types.InvalidParam

		// Check if the error is a validation error
		var validationErr validation.Errors
		if errors.As(err, &validationErr) {
			invalidParams = processValidationErrors("", validationErr)
		}

		problemDetail := &types.ProblemDetail{
			Type:          "invalid",
			Title:         "Invalid Request",
			Status:        fiber.StatusBadRequest,
			Detail:        "Validation error",
			Instance:      fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
			InvalidParams: invalidParams,
		}

		return problemDetail
	}

	return nil
}

// processValidationErrors recursively processes validation errors and builds detailed field names
func processValidationErrors(prefix string, err error) []types.InvalidParam {
	var invalidParams []types.InvalidParam

	var validationErrors validation.Errors
	if errors.As(err, &validationErrors) {
		for field, fieldErr := range validationErrors {
			fullFieldName := joinFieldNames(prefix, field)
			invalidParams = append(invalidParams, processFieldError(fullFieldName, fieldErr)...)
		}
		return invalidParams
	}

	// If it's not a validation.Errors, treat it as a single error
	return []types.InvalidParam{{
		Name:   prefix,
		Reason: err.Error(),
	}}
}

// processFieldError handles different types of field errors
func processFieldError(fieldName string, err error) []types.InvalidParam {
	var validationErrors validation.Errors
	if errors.As(err, &validationErrors) {
		return processValidationErrors(fieldName, validationErrors)
	}

	// Check if it's a slice of errors
	if reflect.TypeOf(err).Kind() == reflect.Slice {
		return processSliceError(fieldName, err)
	}

	// It's a single error
	return []types.InvalidParam{{
		Name:   fieldName,
		Reason: err.Error(),
	}}
}

// processSliceError handles errors for slice fields
func processSliceError(fieldName string, err error) []types.InvalidParam {
	var invalidParams []types.InvalidParam

	sliceValue := reflect.ValueOf(err)
	for i := 0; i < sliceValue.Len(); i++ {
		sliceErr := sliceValue.Index(i).Interface()
		sliceFieldName := fmt.Sprintf("%s.%d", fieldName, i)
		invalidParams = append(invalidParams, processFieldError(sliceFieldName, sliceErr.(error))...)
	}

	return invalidParams
}

// joinFieldNames joins the prefix and field name, handling empty prefix
func joinFieldNames(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

func CreateServiceError(c *fiber.Ctx, err error) error {
	var dbValidationErr *validator.DBValidationError
	var businessLogicErr *validator.BusinessLogicError
	var multiErr validator.MultiValidationError

	switch {
	case errors.As(err, &dbValidationErr):
		return handleDBValidationError(c, dbValidationErr)
	case errors.As(err, &businessLogicErr):
		return handleBusinessLogicError(c, businessLogicErr)
	case errors.As(err, &multiErr):
		return handleMultiError(c, multiErr)
	default:
		return handleGenericError(c, err)
	}
}

func handleMultiError(c *fiber.Ctx, errs validator.MultiValidationError) error {
	var invalidParams []types.InvalidParam
	for _, err := range errs.Errors {
		invalidParams = append(invalidParams, types.InvalidParam{
			Name:   err.Field,
			Reason: err.Message,
		})
	}

	return createProblemDetail(c, "invalid", "Invalid Request", fiber.StatusBadRequest, "Multiple validation errors", invalidParams)
}

func handleDBValidationError(c *fiber.Ctx, err *validator.DBValidationError) error {
	invalidParam := types.InvalidParam{
		Name:   err.Field,
		Reason: err.Message,
	}

	return createProblemDetail(c, "invalid", "Invalid Request", fiber.StatusBadRequest, "Database validation error", []types.InvalidParam{invalidParam})
}

func handleBusinessLogicError(c *fiber.Ctx, err *validator.BusinessLogicError) error {
	// Business logic errors might not always have a specific field associated,
	// but we can still use InvalidParam to provide more context
	invalidParam := types.InvalidParam{
		Name:   "businessLogic",
		Reason: err.Message,
	}

	return createProblemDetail(c, "business_logic_error", "Business Logic Error", fiber.StatusUnprocessableEntity, "Business logic error", []types.InvalidParam{invalidParam})
}

func handleGenericError(c *fiber.Ctx, err error) error {
	invalidParam := types.InvalidParam{
		Name:   "genericError",
		Reason: err.Error(),
	}

	return createProblemDetail(c, "internal_server_error",
		"Internal Server Error",
		fiber.StatusInternalServerError,
		"An unexpected error occurred",
		[]types.InvalidParam{invalidParam})
}

func createProblemDetail(c *fiber.Ctx, errType, title string, status int, detail string, invalidParams []types.InvalidParam) error {
	return &types.ProblemDetail{
		Type:          errType,
		Title:         title,
		Status:        status,
		Detail:        detail,
		Instance:      fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
		InvalidParams: invalidParams,
	}
}
