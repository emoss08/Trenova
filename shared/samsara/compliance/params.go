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
