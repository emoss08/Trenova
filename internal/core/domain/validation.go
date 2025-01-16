package domain

import (
	"time"

	"github.com/rotisserie/eris"
)

func ValidateTimezone(value interface{}) error {
	tz, _ := value.(string)
	if tz == "" {
		return nil // Skip empty timezone validation here as Required rule will catch it
	}
	_, err := time.LoadLocation(tz)
	if err != nil {
		return eris.New("Invalid timezone. Please provide a valid timezone")
	}
	return nil
}
