package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/go-playground/validator/v10"
)

const genericServerError = "An unexpected error occurred. Please try again later."

type Sanitizer struct {
	debug bool
}

func NewSanitizer(debug bool) *Sanitizer {
	return &Sanitizer{debug: debug}
}

func (s *Sanitizer) SanitizeMessage(err error, problemType ProblemType) string {
	if problemType.IsInternal() && !s.debug {
		return genericServerError
	}
	return err.Error()
}

func (s *Sanitizer) ExtractErrors(err error) []ValidationError {
	extractors := []func(error) ([]ValidationError, bool){
		s.extractMultiError,
		s.extractValidationError,
		s.extractValidatorErrors,
		s.extractBusinessError,
		s.extractRateLimitError,
	}

	for _, extract := range extractors {
		if valErrs, ok := extract(err); ok {
			return valErrs
		}
	}
	return nil
}

func (s *Sanitizer) ExtractParams(err error) map[string]string {
	var businessErr *errortypes.BusinessError
	if errors.As(err, &businessErr) && len(businessErr.Params) > 0 {
		return businessErr.Params
	}
	return nil
}

func (s *Sanitizer) ExtractUsageStats(err error) any {
	var conflictErr *errortypes.ConflictError
	if errors.As(err, &conflictErr) {
		return conflictErr.UsageStats
	}
	return nil
}

func (s *Sanitizer) extractMultiError(err error) ([]ValidationError, bool) {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
		return nil, false
	}

	result := make([]ValidationError, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		result = append(result, ValidationError{
			Field:    e.Field,
			Message:  e.Message,
			Code:     string(e.Code),
			Location: "body",
		})
	}
	return result, true
}

func (s *Sanitizer) extractValidationError(err error) ([]ValidationError, bool) {
	var validErr *errortypes.Error
	if !errors.As(err, &validErr) {
		return nil, false
	}

	return []ValidationError{{
		Field:    validErr.Field,
		Message:  validErr.Message,
		Code:     string(validErr.Code),
		Location: "body",
	}}, true
}

func (s *Sanitizer) extractValidatorErrors(err error) ([]ValidationError, bool) {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return nil, false
	}

	result := make([]ValidationError, 0, len(validationErrs))
	for _, e := range validationErrs {
		result = append(result, ValidationError{
			Field:    toLowerFirst(e.Field()),
			Message:  formatValidatorMessage(e),
			Code:     strings.ToUpper(e.Tag()),
			Location: "body",
		})
	}
	return result, true
}

func (s *Sanitizer) extractBusinessError(err error) ([]ValidationError, bool) {
	var businessErr *errortypes.BusinessError
	if !errors.As(err, &businessErr) {
		return nil, false
	}

	message := businessErr.Message
	if businessErr.Details != "" {
		message = fmt.Sprintf("%s: %s", businessErr.Message, businessErr.Details)
	}

	return []ValidationError{{
		Field:    "business",
		Message:  message,
		Code:     string(businessErr.Code),
		Location: "business",
	}}, true
}

func (s *Sanitizer) extractRateLimitError(err error) ([]ValidationError, bool) {
	var rateLimitErr *errortypes.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		return nil, false
	}

	return []ValidationError{{
		Field:    rateLimitErr.Field,
		Message:  rateLimitErr.Message,
		Code:     string(rateLimitErr.Code),
		Location: "rate-limit",
	}}, true
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func formatValidatorMessage(e validator.FieldError) string {
	messages := map[string]string{
		"required": "This field is required",
		"email":    "Must be a valid email address",
		"min":      "Value is below the minimum allowed",
		"max":      "Value exceeds the maximum allowed",
		"uuid":     "Must be a valid UUID",
	}

	if msg, ok := messages[e.Tag()]; ok {
		return msg
	}

	switch e.Tag() {
	case "len":
		return "Value must be exactly " + e.Param() + " characters"
	case "oneof":
		return "Must be one of: " + e.Param()
	default:
		return "Validation failed on '" + e.Tag() + "' constraint"
	}
}
