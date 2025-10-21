package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var (
	ErrInvalidTimezone   = errors.New("invalid timezone. Please provide a valid timezone")
	ErrInvalidPostalCode = errors.New(
		"invalid postal code. Please provide a valid US or Canadian postal code",
	)
	ErrInvalidEmail = errors.New(
		"invalid email address. Please provide a valid email address",
	)
	ErrInvalidStringSlice = errors.New(
		"value must be a slice of strings",
	)
	ErrInvalidVin         = errors.New("invalid VIN. Please provide a valid VIN")
	ErrInvalidTemperature = errors.New(
		"invalid temperature. Please provide a valid temperature",
	)
	ErrInvalidStringOrCommaSeparated = errors.New(
		"invalid string format. Please provide a valid string or comma-separated values",
	)
)

var (
	temperatureMax    int16 = 150
	temperatureMin    int16 = -100
	caPostalCodeRegex       = regexp.MustCompile(`^[A-Z]\d[A-Z][ -]?\d[A-Z]\d$`)
	vinRegex                = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	usPostalCodeRegex       = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

type Validatable interface {
	Validate(multiErr *errortypes.MultiError)
	GetTableName() string
}

func ValidateTimezone(value any) error {
	tz, _ := value.(string)
	if tz == "" || tz == "auto" {
		return nil
	}

	_, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}

	return nil
}

func ValidateStringSlice(v any) error {
	_, ok := v.([]string)
	if !ok {
		return ErrInvalidStringSlice
	}

	return nil
}

func ValidatePostalCode(value any) error {
	pc, _ := value.(string)
	if pc == "" {
		return nil // Skip empty postal code validation here as Required rule will catch it
	}

	if !usPostalCodeRegex.MatchString(pc) && !caPostalCodeRegex.MatchString(pc) {
		return ErrInvalidPostalCode
	}

	return nil
}

func ValidateVin(value any) error {
	vin, _ := value.(string)
	if vin == "" {
		return nil // Skip empty VIN validation here as Required rule will catch it
	}

	if !vinRegex.MatchString(vin) {
		return ErrInvalidVin
	}

	return nil
}

func ValidateCommaSeparatedEmails(value any) error {
	emails, _ := value.(string)
	if emails == "" {
		return nil
	}

	for email := range strings.SplitSeq(emails, ",") {
		trimmedEmail := strings.TrimSpace(email)
		if trimmedEmail == "" {
			continue
		}

		if !govalidator.IsEmail(trimmedEmail) {
			return ErrInvalidEmail
		}
	}

	return nil
}

func ValidateTemperature(value any) error {
	temperature, _ := value.(int16)
	if temperature == 0 {
		return nil
	}

	if temperature > temperatureMax || temperature < temperatureMin {
		return ErrInvalidTemperature
	}
	return nil
}

func ValidateStringOrCommaSeparated(value any) error {
	str, ok := value.(string)
	if !ok {
		return ErrInvalidStringOrCommaSeparated
	}

	if str == "" {
		return nil // Allow empty strings (Required rule will catch if needed)
	}

	for part := range strings.SplitSeq(str, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return ErrInvalidStringOrCommaSeparated
		}
	}

	return nil
}

func ValidateTemperaturePointer(value any) error {
	temperature, _ := value.(*int16)
	if temperature == nil {
		return nil
	}

	if *temperature == 0 {
		return nil
	}

	return ValidateTemperature(*temperature)
}
