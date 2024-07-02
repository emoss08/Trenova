package validator

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/asaskevich/govalidator"
	"github.com/emoss08/trenova/internal/types"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
	"github.com/nyaruka/phonenumbers"
	"github.com/rs/zerolog/log"
)

type Validator struct {
	validator  *validator.Validate
	translator ut.Translator
}

func New() *Validator {
	english := en.New()
	uni := ut.New(english, english)
	trans, ok := uni.GetTranslator("en")
	if !ok {
		log.Fatal().Err(errors.New("translator not found")).Msg("Failed to get translator")
	}

	validate := validator.New()
	err := enTranslations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register default translations")
	}

	if err = registerCustomValidations(validate); err != nil {
		log.Fatal().Err(err).Msg("Failed to register custom validations")
	}

	return &Validator{
		validator:  validate,
		translator: trans,
	}
}

func (v *Validator) Validate(c *fiber.Ctx, i any) error {
	err := v.validator.Struct(i)
	if err == nil {
		return nil
	}

	var invalidParams []types.InvalidParam
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, err := range validationErrors {
			invalidParam := types.InvalidParam{
				Name:   NamespaceToCamelCase(err.Namespace()),
				Reason: err.Tag(),
			}

			invalidParams = append(invalidParams, invalidParam)
		}
	}

	return &types.ProblemDetail{
		Type:          "https://tools.ietf.org/html/rfc7231#section-6.5.1",
		Title:         "Validation Error",
		Status:        fiber.StatusBadRequest,
		Detail:        "One or more validation errors occurred.",
		Instance:      c.OriginalURL(),
		InvalidParams: invalidParams,
	}
}

// ToCamelCase converts a string to camel case.
func ToCamelCase(s string) string {
	var result strings.Builder
	uppercaseNext := false

	for i, r := range s {
		if i == 0 {
			result.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == '_' || r == '.' || r == '[' || r == ']' {
			result.WriteRune(r)
			uppercaseNext = r == '.' || r == '['
			continue
		}
		if uppercaseNext {
			result.WriteRune(unicode.ToUpper(r))
			uppercaseNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// NamespaceToCamelCase converts a namespace string to camelCase with dot and index
func NamespaceToCamelCase(ns string) string {
	var result strings.Builder
	isUpperNext := false

	// Split the namespace by dot
	segments := strings.Split(ns, ".")

	// Skip the first element which is the root object name
	for i, segment := range segments[1:] {
		if i > 0 {
			result.WriteRune('.')
		}
		isIndex := false
		for j, r := range segment {
			switch {
			case r == '[':
				isIndex = true
				result.WriteRune(r)
			case r == ']':
				isIndex = false
				result.WriteRune(r)
			case isIndex:
				result.WriteRune(r)
			case j == 0 || isUpperNext:
				result.WriteRune(unicode.ToLower(r))
			case r == '.' || r == '[':
				result.WriteRune(r)
				isUpperNext = true
			default:
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

// registerCustomValidations registers custom validation functions.
func registerCustomValidations(v *validator.Validate) error {
	validations := []struct {
		tag       string
		validator validator.Func
	}{
		{"commaSeparatedEmails", validateCommaSeparatedEmails},
		{"phoneNum", validatePhoneNumber},
		{"timezone", validateTimezone},
	}

	for _, val := range validations {
		if err := v.RegisterValidation(val.tag, val.validator); err != nil {
			return err
		}
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
