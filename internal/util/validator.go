package util

import (
	"errors"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/nyaruka/phonenumbers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

const SplitStrNum = 2

type ValidationError struct {
	Response types.ValidationErrorResponse
}

func (e *ValidationError) Error() string {
	errBytes, _ := sonic.Marshal(e.Response)
	return string(errBytes)
}

// newValidationError creates a new ValidationError with the specified message and code.
func NewValidationError(message, code, attr string) *ValidationError {
	return &ValidationError{
		Response: types.ValidationErrorResponse{
			Type: "validationError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   code,
					Detail: message,
					Attr:   attr,
				},
			},
		},
	}
}

// IsValidationError checks if the provided error is a ValidationError.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

func NewValidator() (*Validator, error) {
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")
	validate := validator.New()
	err := enTranslations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, err
	}

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", SplitStrNum)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	registerCustomValidations(validate)

	return &Validator{validate: validate, trans: trans}, nil
}

func (v *Validator) Validate(payload any) error {
	err := v.validate.Struct(payload)
	if err == nil {
		return nil
	}

	log.Printf("Validation error: %v", err)

	var valError []types.ValidationErrorDetail
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}
	for _, ve := range validationErrors {
		detail := ve.Translate(v.trans)
		// Uppercase the first letter of the error message
		if len(detail) > 0 {
			detail = strings.ToUpper(detail[:1]) + detail[1:]
		}

		fieldName := ve.Field()
		valError = append(valError, types.ValidationErrorDetail{
			Code:   "invalid",
			Detail: detail,
			Attr:   fieldName,
		})
	}

	return &ValidationError{
		Response: types.ValidationErrorResponse{
			Type:   "validationError",
			Errors: valError,
		},
	}
}

func registerCustomValidations(v *validator.Validate) {
	err := v.RegisterValidation("commaSeparatedEmails", validateCommaSeparatedEmails)
	if err != nil {
		return
	}
	phoneNumErr := v.RegisterValidation("phoneNum", validatePhoneNumber)
	if phoneNumErr != nil {
		return
	}
}

func validateCommaSeparatedEmails(fl validator.FieldLevel) bool {
	emailsStr := fl.Field().String()
	emails := strings.Split(emailsStr, ",")

	for _, email := range emails {
		trimmedEmail := strings.TrimSpace(email)
		if trimmedEmail == "" || !govalidator.IsEmail(trimmedEmail) {
			return false
		}
	}
	return true
}

func validatePhoneNumber(fl validator.FieldLevel) bool {
	num, err := phonenumbers.Parse(fl.Field().String(), "")
	if err != nil {
		return false
	}
	return phonenumbers.IsValidNumber(num)
}

// preCompile the regular expression to avoid repeated compilation.
var fieldErrorRegex = regexp.MustCompile(`field "(.+?)\.(.+?)": (.+)`)

// CreateDBErrorResponse formats a database error into a structured response.
func CreateDBErrorResponse(err error) types.ValidationErrorResponse {
	var details []types.ValidationErrorDetail

	switch {
	case ent.IsValidationError(err):
		matches := fieldErrorRegex.FindStringSubmatch(err.Error())
		if len(matches) >= 4 {
			field := toCamelCase(matches[2]) // Apply camel casing to the field name.
			detail := matches[3]
			details = append(details, types.ValidationErrorDetail{
				Code:   "validationError",
				Detail: detail,
				Attr:   field,
			})
		}
	case IsValidationError(err):
		if validationError, ok := err.(*ValidationError); ok {
			return validationError.Response
		}
	default:
		detail := "An unexpected error occurred"
		if err != nil {
			detail = err.Error()
		}
		details = append(details, types.ValidationErrorDetail{
			Code:   "databaseError",
			Detail: detail,
			Attr:   "databaseError",
		})
	}

	return types.ValidationErrorResponse{
		Type:   "databaseError",
		Errors: details,
	}
}

// toCamelCase converts strings to camel case (handling special cases for your field naming convention.
func toCamelCase(s string) string {
	caser := cases.Title(language.AmericanEnglish)

	parts := strings.Split(s, "_")
	for i, part := range parts {
		if i > 0 { // Convert to title case only if it's not the first part
			parts[i] = caser.String(part)
		}
	}

	return strings.Join(parts, "")
}
