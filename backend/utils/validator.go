package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"
)

var validate = validator.New()

type ValidationErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr"`
}

type ValidationErrorResponse struct {
	Type   string                  `json:"type"`
	Errors []ValidationErrorDetail `json:"errors"`
}

// Validate validates the input struct. Instead of returning a *fiber.Error,
// it returns an error interface or nil if the validation passes.
func Validate(payload interface{}) error {
	err := validate.Struct(payload)

	if err != nil {
		var errors []ValidationErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, ValidationErrorDetail{
				Code:   "invalid",
				Detail: fmt.Sprintf("This field may not be %s.", err.Tag()),
				Attr:   err.Field(),
			})
		}
		errorResponse := ValidationErrorResponse{

			Type:   "validationError",
			Errors: errors,
		}

		// Instead of returning a fiber error, return a standard error
		errMsg, _ := json.Marshal(errorResponse)
		return fmt.Errorf(string(errMsg))
	}

	return nil
}

// FormatDatabaseError formats a database error into a ValidationErrorResponse
func FormatDatabaseError(err error) ValidationErrorResponse {
	return ValidationErrorResponse{
		Type: "databaseError",
		Errors: []ValidationErrorDetail{
			{
				Code:   "invalid",
				Detail: err.Error(),
				Attr:   "all",
			},
		},
	}
}

var _ = validate.RegisterValidation("commaSeparatedEmails", func(fl validator.FieldLevel) bool {
	emailsStr := fl.Field().String()
	emails := strings.Split(emailsStr, ",")

	for _, email := range emails {
		trimmedEmail := strings.TrimSpace(email)
		if trimmedEmail == "" || !govalidator.IsEmail(trimmedEmail) {
			return false
		}
	}

	return true
})

var _ = validate.RegisterValidation("phoneNum", func(fl validator.FieldLevel) bool {
	num, err := phonenumbers.Parse(fl.Field().String(), "")
	if err != nil {
		return false
	}
	return phonenumbers.IsValidNumber(num)
})
