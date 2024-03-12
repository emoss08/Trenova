package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"trenova/app/models"

	"github.com/asaskevich/govalidator"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/nyaruka/phonenumbers"
)

type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

const SplitStrNum = 2

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

func (v *Validator) Validate(payload interface{}) error {
	err := v.validate.Struct(payload)
	if err != nil {
		var valError []models.ValidationErrorDetail
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, ve := range validationErrors {
				fieldName := ve.Field()
				valError = append(valError, models.ValidationErrorDetail{
					Code:   "invalid",
					Detail: ve.Translate(v.trans),
					Attr:   fieldName,
				})
			}
			verr := models.ValidationErrorResponse{
				Type:   "validationError",
				Errors: valError,
			}
			return fmt.Errorf("%v", verr)
		}
		return err
	}

	return nil
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

// CreateDBErrorResponse formats a database error into a structured response.
func CreateDBErrorResponse(err error) models.ValidationErrorResponse {
	return models.ValidationErrorResponse{
		Type: "databaseError",
		Errors: []models.ValidationErrorDetail{
			{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "database",
			},
		},
	}
}
