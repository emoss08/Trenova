package validation

import (
	"backend/models"
	"reflect"
	"regexp"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	uni      *ut.UniversalTranslator
)

func init() {
	en := en.New()
	uni = ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	validate = validator.New()

	// Register the custom ZIP code validator
	validate.RegisterValidation("usazipcode", validateUSAZipCode)

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "-" {
			return ""
		}
		return name
	})

	validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "`{0}` is required and cannot be empty", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})
}

// Custom validation function for USA ZIP codes
func validateUSAZipCode(fl validator.FieldLevel) bool {
	zipCode := fl.Field().String()
	regex := regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	return regex.MatchString(zipCode)
}

func ValidateStruct(data interface{}) ([]models.FieldError, error) {
	trans, _ := uni.GetTranslator("en")

	if err := validate.Struct(data); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var fieldErrors []models.FieldError
			for _, ve := range validationErrors {
				fieldErrors = append(fieldErrors, models.FieldError{
					Code:   "validationError",
					Detail: ve.Translate(trans),
					Attr:   ve.Field(),
				})
			}
			return fieldErrors, nil
		}
		// If the error is not a ValidationErrors type, return the original error
		return nil, err
	}
	return nil, nil
}
