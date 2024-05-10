package util

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

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

// IsValidationError checks if the provided error is a ValidationError using errors.As for proper error unwrapping.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var e *ValidationError
	return errors.As(err, &e)
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

	if err = registerCustomValidations(validate); err != nil {
		return nil, err
	}

	return &Validator{validate: validate, trans: trans}, nil
}

func (v *Validator) Validate(payload any) error {
	err := v.validate.Struct(payload)
	if err == nil {
		return nil
	}

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

func registerCustomValidations(v *validator.Validate) error {
	err := v.RegisterValidation("commaSeparatedEmails", validateCommaSeparatedEmails)
	if err != nil {
		return err
	}
	phoneNumErr := v.RegisterValidation("phoneNum", validatePhoneNumber)
	if phoneNumErr != nil {
		return phoneNumErr
	}
	timezoneErr := v.RegisterValidation("timezone", validateTimezone)
	if timezoneErr != nil {
		return timezoneErr
	}

	return nil
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

func validateTimezone(fl validator.FieldLevel) bool {
	_, err := time.LoadLocation(fl.Field().String())
	return err == nil
}

// preCompile the regular expression to avoid repeated compilation.
var fieldErrorRegex = regexp.MustCompile(`field "(.+?)\.(.+?)": (.+)`)

// CreateDBErrorResponse formats a database error into a structured response.
func CreateDBErrorResponse(err error) types.ValidationErrorResponse {
	var details []types.ValidationErrorDetail

	switch {
	// Handle validation errors from ent
	case ent.IsValidationError(err):
		// Regex to extract field information
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
	// Handle custom ValidationError

	case IsValidationError(err):
		var vErr *ValidationError
		if errors.As(err, &vErr) {
			return vErr.Response
		}
		detail := "Failed to cast to ValidationError despite passing IsValidationError check"
		details = append(details, types.ValidationErrorDetail{
			Code:   "typeAssertionError",
			Detail: detail,
			Attr:   "internalError",
		})

	// Handle other generic errors
	default:
		detail := "An unexpected error occurred"
		if err != nil {
			detail = err.Error()
		}
		details = append(details, types.ValidationErrorDetail{
			Code:   "unknownError",
			Detail: detail,
			Attr:   "general",
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
