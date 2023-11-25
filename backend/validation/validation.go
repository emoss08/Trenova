package validation

import (
	"backend/models"
	"reflect"
	"regexp"
	"strings"

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
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}

		var path []string
		if name != "" {
			path = append(path, name)
		}

		// Traverse up the field's parent structs
		for t := fld.Type; t.Kind() == reflect.Ptr || t.Kind() == reflect.Struct; {
			if t.Kind() == reflect.Ptr {
				t = t.Elem() // Get the type that the pointer points to
			}

			if t.Kind() != reflect.Struct {
				break
			}

			// Traverse up to parent struct
			if parentField, found := t.FieldByName(fld.Name); found && !parentField.Anonymous {
				if parentJsonTag := parentField.Tag.Get("json"); parentJsonTag != "" {
					parentName := strings.SplitN(parentJsonTag, ",", 2)[0]
					if parentName != "" {
						path = append([]string{parentName}, path...)
					}
				}
			} else {
				break
			}
		}

		return strings.Join(path, ".")
	})

	validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "`{0}` is required and cannot be empty", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	validate.RegisterTranslation("unique", trans, func(ut ut.Translator) error {
		return ut.Add("unique", "This {0} already exists", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("unique", fe.Field())
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
				// Process the namespace to exclude 'CreateUserRequest'
				namespace := ve.Namespace()
				fields := strings.Split(namespace, ".")
				if len(fields) > 1 {
					namespace = strings.Join(fields[1:], ".")
				}

				fieldErrors = append(fieldErrors, models.FieldError{
					Code:   "validationError",
					Detail: ve.Translate(trans),
					Attr:   namespace,
				})
			}
			return fieldErrors, nil
		}
		// If the error is not a ValidationErrors type, return the original error
		return nil, err
	}
	return nil, nil
}
