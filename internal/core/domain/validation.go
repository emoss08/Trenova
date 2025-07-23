// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/rotisserie/eris"
)

var (
	temperatureMax    int16 = 150
	temperatureMin    int16 = -100
	caPostalCodeRegex       = regexp.MustCompile(`^[A-Z]\d[A-Z][ -]?\d[A-Z]\d$`)
	vinRegex                = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	usPostalCodeRegex       = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

func ValidateTimezone(value any) error {
	tz, _ := value.(string)
	if tz == "" || tz == "auto" {
		return nil // Skip empty timezone validation here as Required rule will catch it
	}
	_, err := time.LoadLocation(tz)
	if err != nil {
		return eris.New("Invalid timezone. Please provide a valid timezone")
	}
	return nil
}

func ValidatePostalCode(value any) error {
	pc, _ := value.(string)
	if pc == "" {
		return nil // Skip empty postal code validation here as Required rule will catch it
	}

	if !usPostalCodeRegex.MatchString(pc) && !caPostalCodeRegex.MatchString(pc) {
		return eris.New("Invalid postal code. Please provide a valid US or Canadian postal code")
	}

	return nil
}

func ValidateVin(value any) error {
	vin, _ := value.(string)
	if vin == "" {
		return nil // Skip empty VIN validation here as Required rule will catch it
	}

	if !vinRegex.MatchString(vin) {
		return eris.New("Invalid VIN. Please provide a valid VIN")
	}

	return nil
}

func ValidateTemperature(value any) error {
	temperature, _ := value.(int16)
	if temperature == 0 {
		return nil
	}

	if temperature > temperatureMax || temperature < temperatureMin {
		return eris.New("Invalid temperature. Please provide a valid temperature")
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

func ValidateCommaSeparatedEmails(value any) error {
	emails, _ := value.(string)
	if emails == "" {
		return nil
	}

	// Use range with SplitSeq
	for email := range strings.SplitSeq(emails, ",") {
		trimmedEmail := strings.TrimSpace(email)
		if trimmedEmail == "" {
			continue
		}

		if !govalidator.IsEmail(trimmedEmail) {
			return eris.New("Invalid email address. Please provide a valid email address")
		}
	}

	return nil
}

func ValidateStringSlice(v any) error {
	_, ok := v.([]string)
	if !ok {
		return eris.New("value must be a slice of strings")
	}

	return nil
}
