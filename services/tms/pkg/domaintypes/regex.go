package domaintypes

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidVin        = errors.New("invalid VIN. Please provide a valid VIN")
	ErrInvalidPostalCode = errors.New(
		"invalid postal code. Please provide a valid US or Canadian postal code",
	)
	VinRegex          = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	UsPostalCodeRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)

	// IdentifierPattern matches the codes that get referenced by name from
	// inside a stored expression — rate table keys, rate matrix codes. They end
	// up as bare identifiers in a formula, so they have to look like one.
	IdentifierPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
)

func ValidateVin(value any) error {
	vin, _ := value.(string)
	if vin == "" {
		return nil // skip empty VIN validation here as Required rule will catch it
	}

	if !VinRegex.MatchString(vin) {
		return ErrInvalidVin
	}

	return nil
}

func ValidatePostalCode(value any) error {
	pc, _ := value.(string)
	if pc == "" {
		return nil // skip empty postal code validation here as Required rule will catch it
	}

	if !UsPostalCodeRegex.MatchString(pc) {
		return ErrInvalidPostalCode
	}

	return nil
}
