package compliance

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type HOSClocksParams struct {
	TagIDs       []string
	ParentTagIDs []string
	DriverIDs    []string
	After        string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HOSClocksParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
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
	for _, date := range []string{p.StartDate, p.EndDate} {
		if date == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return ErrDateFormatInvalid
		}
	}
	return nil
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
