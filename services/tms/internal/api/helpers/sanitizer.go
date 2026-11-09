package helpers

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/go-playground/validator/v10"
)

const genericServerError = "An unexpected error occurred. Please try again later."

type Sanitizer struct {
	debug bool
}

func NewSanitizer(debug bool) *Sanitizer {
	return &Sanitizer{debug: debug}
}

func (s *Sanitizer) SanitizeMessage(err error, problemType ProblemType) Message {
	if problemType.IsInternal() && !s.debug {
		return Message{Text: genericServerError}
	}
	return MessageOf(err)
}

func MessageOf(err error) Message {
	if multiErr, ok := errortypes.RefusalOf(err); ok && len(multiErr.Errors) > 0 {
		parts := make([]Message, 0, len(multiErr.Errors))
		seen := make(map[string]struct{}, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			if _, dup := seen[fieldErr.Message]; dup {
				continue
			}
			seen[fieldErr.Message] = struct{}{}
			parts = append(parts, Message{Text: fieldErr.Message, Args: fieldErr.Args})
		}
		return Message{Parts: parts}
	}

	var businessErr *errortypes.BusinessError
	if errors.As(err, &businessErr) {
		return Message{
			Text:    businessErr.Message,
			Args:    businessErr.Args,
			Details: businessErr.Details,
		}
	}

	var localizable errortypes.Localizable
	if errors.As(err, &localizable) {
		text, args := localizable.LocalizedMessage()
		return Message{Text: text, Args: args}
	}

	return Message{Text: err.Error()}
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
	if parameterized, ok := errors.AsType[errortypes.Parameterized](err); ok {
		return parameterized.Params()
	}

	var businessErr *errortypes.BusinessError
	if errors.As(err, &businessErr) && len(businessErr.Params) > 0 {
		return businessErr.Params
	}
	return nil
}

func (s *Sanitizer) ExtractAdvisories(err error) []Advisory {
	multiErr, ok := errortypes.RefusalOf(err)
	if !ok {
		return nil
	}

	advisories := multiErr.AllAdvisories()
	if len(advisories) == 0 {
		return nil
	}

	result := make([]Advisory, 0, len(advisories))
	for _, advisory := range advisories {
		result = append(result, Advisory{
			Path:     advisory.Path,
			Code:     string(advisory.Code),
			Message:  advisory.Message,
			Args:     advisory.Args,
			Severity: advisory.Severity,
			RuleKey:  advisory.RuleKey,
		})
	}

	return result
}

func (s *Sanitizer) ExtractConflict(err error) *errortypes.EditConflict {
	conflict, _ := errortypes.EditConflictOf(err)
	return conflict
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
	if multiErr.Internal() != nil {
		return nil, true
	}

	result := make([]ValidationError, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		result = append(result, ValidationError{
			Path:     e.Path,
			Message:  e.Message,
			Args:     e.Args,
			Code:     string(e.Code),
			Location: "body",
			Range:    e.Range,
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
		Path:     validErr.Path,
		Message:  validErr.Message,
		Args:     validErr.Args,
		Code:     string(validErr.Code),
		Location: "body",
		Range:    validErr.Range,
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
			Path:     fieldpath.Of(fieldpath.Key(toLowerFirst(e.Field()))),
			Message:  formatValidatorMessage(e),
			Args:     validatorMessageArgs(e),
			Code:     string(validatorCode(e.Tag())),
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

	return []ValidationError{{
		Message:  businessErr.Message,
		Args:     businessErr.Args,
		Details:  businessErr.Details,
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
		Path:     fieldpath.Parse(rateLimitErr.Field),
		Message:  rateLimitErr.Message,
		Args:     rateLimitErr.Args,
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

func validatorCode(tag string) errortypes.ErrorCode {
	switch tag {
	case "required", "required_if", "required_unless", "required_with", "required_without":
		return errortypes.ErrRequired
	case "len", "min", "max":
		return errortypes.ErrInvalidLength
	case "email", "uuid", "uuid4", "url", "uri", "e164", "datetime", "numeric", "number",
		"alpha", "alphanum", "hexcolor", "ip", "hostname":
		return errortypes.ErrInvalidFormat
	default:
		return errortypes.ErrInvalid
	}
}

type validatorMessage struct {
	Message  string
	WithArgs bool
}

var validatorMessages = map[string]validatorMessage{
	"required":         {Message: "This field is required"},
	"required_if":      {Message: "This field is required"},
	"required_unless":  {Message: "This field is required"},
	"required_with":    {Message: "This field is required"},
	"required_without": {Message: "This field is required"},
	"email":            {Message: "Must be a valid email address"},
	"uuid":             {Message: "Must be a valid UUID"},
	"uuid4":            {Message: "Must be a valid UUID"},
	"min":              {Message: "Must be at least {0}", WithArgs: true},
	"max":              {Message: "Must be no more than {0}", WithArgs: true},
	"len":              {Message: "Must be exactly {0} characters", WithArgs: true},
	"oneof":            {Message: "Must be one of: {0}", WithArgs: true},
}

var invalidValidatorValue = validatorMessage{Message: "This value is not valid"}

func formatValidatorMessage(e validator.FieldError) string {
	if message, ok := validatorMessages[e.Tag()]; ok {
		return message.Message
	}

	return invalidValidatorValue.Message
}

func validatorMessageArgs(e validator.FieldError) []any {
	if message, ok := validatorMessages[e.Tag()]; ok && message.WithArgs {
		return []any{e.Param()}
	}

	return nil
}
