package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/nyaruka/phonenumbers"
)

var validate = validator.New()
var english = en.New()
var uni = ut.New(english, english)
var trans, _ = uni.GetTranslator("en")

func init() {
	_ = enTranslations.RegisterDefaultTranslations(validate, trans)

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

type ValidationErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr"`
}

type ValidationErrorResponse struct {
	Type   string                  `json:"type"`
	Errors []ValidationErrorDetail `json:"errors"`
}

// Validate validates the input struct and returns an error interface or nil if the validation passes.
func Validate(payload interface{}) error {
	err := validate.Struct(payload)
	if err != nil {
		var errors []ValidationErrorDetail
		for _, ve := range err.(validator.ValidationErrors) {
			fieldName := ve.Field()
			errors = append(errors, ValidationErrorDetail{
				Code:   "invalid",
				Detail: ve.Translate(trans),
				Attr:   fieldName,
			})
		}
		errorResponse := ValidationErrorResponse{
			Type:   "validationError",
			Errors: errors,
		}

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
