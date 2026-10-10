package compliance

import (
	"fmt"
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
)

const maxClocksLimit = 512

type HOSClocksParams struct {
	TagIDs       []string
	ParentTagIDs []string
	DriverIDs    []string
	After        string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSClocksParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > maxClocksLimit) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSClocksParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetString(values, "after", p.After)
	httpx.SetInt(values, "limit", p.Limit)
	return values
}

type HOSDailyLogsParams struct {
	DriverIDs              []string
	StartDate              string
	EndDate                string
	TagIDs                 []string
	ParentTagIDs           []string
	DriverActivationStatus string
	After                  string
	Expand                 []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSDailyLogsParams) Validate() error {
	start, err := parseDate(p.StartDate)
	if err != nil {
		return err
	}
	end, err := parseDate(p.EndDate)
	if err != nil {
		return err
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return ErrDateRangeInvalid
	}
	if p.DriverActivationStatus != "" &&
		!samsaraspec.GetHosDailyLogsParamsDriverActivationStatus(p.DriverActivationStatus).Valid() {
		return fmt.Errorf("%w: %q", ErrDriverActivationStatusInvalid, p.DriverActivationStatus)
	}
	for _, expand := range p.Expand {
		if !samsaraspec.GetHosDailyLogsParamsExpand(expand).Valid() {
			return fmt.Errorf("%w: %q", ErrExpandInvalid, expand)
		}
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, ErrDateFormatInvalid
	}
	return parsed, nil
}

func validateTimeRange(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return ErrTimeRangeInvalid
	}
	return nil
}

func validateRequiredTimeRange(start, end *time.Time) error {
	if start == nil || end == nil {
		return ErrTimeRangeRequired
	}
	return validateTimeRange(start, end)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSDailyLogsParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetString(values, "startDate", p.StartDate)
	httpx.SetString(values, "endDate", p.EndDate)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetString(values, "driverActivationStatus", p.DriverActivationStatus)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "expand", p.Expand)
	return values
}

type HOSViolationsParams struct {
	DriverIDs    []string
	StartTime    *time.Time
	EndTime      *time.Time
	TagIDs       []string
	ParentTagIDs []string
	Types        []string
	After        string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSViolationsParams) Validate() error {
	return validateTimeRange(p.StartTime, p.EndTime)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSViolationsParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetString(values, "after", p.After)
	return values
}

type HOSLogsParams struct {
	TagIDs       []string
	ParentTagIDs []string
	DriverIDs    []string
	StartTime    *time.Time
	EndTime      *time.Time
	After        string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSLogsParams) Validate() error {
	return validateTimeRange(p.StartTime, p.EndTime)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSLogsParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetString(values, "after", p.After)
	return values
}

type DriverTachographParams struct {
	After        string
	StartTime    *time.Time
	EndTime      *time.Time
	DriverIDs    []string
	ParentTagIDs []string
	TagIDs       []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p DriverTachographParams) Validate() error {
	return validateRequiredTimeRange(p.StartTime, p.EndTime)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p DriverTachographParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	return values
}

type VehicleTachographParams struct {
	After        string
	StartTime    *time.Time
	EndTime      *time.Time
	VehicleIDs   []string
	ParentTagIDs []string
	TagIDs       []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p VehicleTachographParams) Validate() error {
	return validateRequiredTimeRange(p.StartTime, p.EndTime)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p VehicleTachographParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	return values
}
