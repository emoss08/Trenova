package domainvalidation

import (
	"errors"
	"time"
)

var ErrTimezoneMustBeString = errors.New("timezone must be a string")

func ValidateTimezone(value any) error {
	tz, ok := value.(string)
	if !ok {
		return ErrTimezoneMustBeString
	}

	if tz == "" || tz == "auto" {
		return nil
	}

	_, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}

	return nil
}
