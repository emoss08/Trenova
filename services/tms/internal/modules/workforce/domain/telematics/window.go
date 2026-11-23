package telematics

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	MaxProviderWindowDays = 31
	providerDayLayout     = "2006-01-02"
)

func RequireProviderWindow(startAt, endAt int64) error {
	if startAt <= 0 || endAt <= startAt {
		return errortypes.NewValidationError(
			"startTime",
			errortypes.ErrInvalid,
			"startTime must be before endTime",
		)
	}
	if endAt-startAt > MaxProviderWindowDays*timeutils.SecondsPerDay {
		return windowTooWide("startTime")
	}
	return nil
}

func RequireProviderDays(startDate, endDate string) error {
	start, startErr := time.Parse(providerDayLayout, startDate)
	end, endErr := time.Parse(providerDayLayout, endDate)
	if startErr != nil || endErr != nil {
		return errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"Dates must use the YYYY-MM-DD format",
		)
	}
	if end.Before(start) {
		return errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"startDate must not be after endDate",
		)
	}
	if end.Sub(start) >= MaxProviderWindowDays*24*time.Hour {
		return windowTooWide("startDate")
	}
	return nil
}

func windowTooWide(field string) error {
	return errortypes.NewValidationError(
		field,
		errortypes.ErrInvalid,
		fmt.Sprintf("Ask for at most %d days at a time", MaxProviderWindowDays),
	)
}
