package framework

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func Required(fieldName string, value any) *errortypes.Error {
	if value == nil || fmt.Sprintf("%v", value) == "" || fmt.Sprintf("%v", value) == "0" {
		return errortypes.NewValidationError(fieldName, errortypes.ErrRequired,
			fmt.Sprintf("%s is required", fieldName))
	}
	return nil
}

func MinLength(m int) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		str, ok := value.(string)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a string", fieldName))
		}

		if utf8.RuneCountInString(str) < m {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidLength,
				fmt.Sprintf("%s must be at least %d characters", fieldName, m))
		}
		return nil
	}
}

func MaxLength(m int) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		str, ok := value.(string)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a string", fieldName))
		}

		if utf8.RuneCountInString(str) > m {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidLength,
				fmt.Sprintf("%s must be at most %d characters", fieldName, m))
		}
		return nil
	}
}

func LengthBetween(mi, ma int) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		str, ok := value.(string)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a string", fieldName))
		}

		length := utf8.RuneCountInString(str)
		if length < mi || length > ma {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidLength,
				fmt.Sprintf("%s must be between %d and %d characters", fieldName, mi, ma))
		}
		return nil
	}
}

func Email(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil // Use Required validator for mandatory fields
	}

	_, err := mail.ParseAddress(str)
	if err != nil {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
			fmt.Sprintf("%s must be a valid email address", fieldName))
	}
	return nil
}

func URL(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil // Use Required validator for mandatory fields
	}

	u, err := url.Parse(str)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
			fmt.Sprintf("%s must be a valid URL", fieldName))
	}
	return nil
}

func Regex(pattern, message string) func(string, any) *errortypes.Error {
	re := regexp.MustCompile(pattern)
	return func(fieldName string, value any) *errortypes.Error {
		str, ok := value.(string)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a string", fieldName))
		}

		if !re.MatchString(str) {
			if message != "" {
				return errortypes.NewValidationError(
					fieldName,
					errortypes.ErrInvalidFormat,
					message,
				)
			}
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
				fmt.Sprintf("%s does not match required format", fieldName))
		}
		return nil
	}
}

func OneOf[T comparable](allowed ...T) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		val, ok := value.(T)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s has invalid type", fieldName))
		}

		for _, a := range allowed {
			if val == a {
				return nil
			}
		}

		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be one of: %v", fieldName, allowed))
	}
}

func MinValue[T int | int64 | float64](m T) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		val, ok := value.(T)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a number", fieldName))
		}

		if val < m {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be at least %v", fieldName, m))
		}
		return nil
	}
}

func MaxValue[T int | int64 | float64](m T) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		val, ok := value.(T)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a number", fieldName))
		}

		if val > m {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be at most %v", fieldName, m))
		}
		return nil
	}
}

func Between[T int | int64 | float64](mi, ma T) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		val, ok := value.(T)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a number", fieldName))
		}

		if val < mi || val > ma {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be between %v and %v", fieldName, mi, ma))
		}
		return nil
	}
}

func Positive(fieldName string, value any) *errortypes.Error {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be positive", fieldName))
		}
	case int64:
		if v <= 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be positive", fieldName))
		}
	case float64:
		if v <= 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be positive", fieldName))
		}
	default:
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a number", fieldName))
	}
	return nil
}

func NonNegative(fieldName string, value any) *errortypes.Error {
	switch v := value.(type) {
	case int:
		if v < 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be non-negative", fieldName))
		}
	case int64:
		if v < 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be non-negative", fieldName))
		}
	case float64:
		if v < 0 {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be non-negative", fieldName))
		}
	default:
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a number", fieldName))
	}
	return nil
}

func DateAfter(after time.Time) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		date, ok := value.(time.Time)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a date", fieldName))
		}

		if !date.After(after) {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be after %s", fieldName, after.Format("2006-01-02")))
		}
		return nil
	}
}

func DateBefore(before time.Time) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		date, ok := value.(time.Time)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a date", fieldName))
		}

		if !date.Before(before) {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be before %s", fieldName, before.Format("2006-01-02")))
		}
		return nil
	}
}

func DateBetween(start, end time.Time) func(string, any) *errortypes.Error {
	return func(fieldName string, value any) *errortypes.Error {
		date, ok := value.(time.Time)
		if !ok {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be a date", fieldName))
		}

		if date.Before(start) || date.After(end) {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
				fmt.Sprintf("%s must be between %s and %s", fieldName,
					start.Format("2006-01-02"), end.Format("2006-01-02")))
		}
		return nil
	}
}

func Future(fieldName string, value any) *errortypes.Error {
	date, ok := value.(time.Time)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a date", fieldName))
	}

	if !date.After(time.Now()) {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be in the future", fieldName))
	}
	return nil
}

func Past(fieldName string, value any) *errortypes.Error {
	date, ok := value.(time.Time)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a date", fieldName))
	}

	if !date.Before(time.Now()) {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be in the past", fieldName))
	}
	return nil
}

func AlphaNumeric(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil
	}

	for _, r := range str {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
				fmt.Sprintf("%s must contain only alphanumeric characters", fieldName))
		}
	}
	return nil
}

func NoWhitespace(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if strings.ContainsAny(str, " \t\n\r") {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
			fmt.Sprintf("%s must not contain whitespace", fieldName))
	}
	return nil
}

func UUID(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil
	}

	uuidRegex := regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
	)
	if !uuidRegex.MatchString(strings.ToLower(str)) {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
			fmt.Sprintf("%s must be a valid UUID", fieldName))
	}
	return nil
}

func PhoneNumber(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil
	}

	// Remove common formatting characters
	cleaned := strings.ReplaceAll(str, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")

	// Check if it's a valid US phone number (10 or 11 digits)
	if len(cleaned) == 10 || (len(cleaned) == 11 && cleaned[0] == '1') {
		for _, r := range cleaned {
			if r < '0' || r > '9' {
				return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
					fmt.Sprintf("%s must be a valid phone number", fieldName))
			}
		}
		return nil
	}

	return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
		fmt.Sprintf("%s must be a valid phone number", fieldName))
}

func ZipCode(fieldName string, value any) *errortypes.Error {
	str, ok := value.(string)
	if !ok {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", fieldName))
	}

	if str == "" {
		return nil
	}

	zipRegex := regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	if !zipRegex.MatchString(str) {
		return errortypes.NewValidationError(fieldName, errortypes.ErrInvalidFormat,
			fmt.Sprintf("%s must be a valid ZIP code", fieldName))
	}
	return nil
}
