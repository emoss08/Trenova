package vehicles

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type StatsParams struct {
	After        string
	Time         *time.Time
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsParams) Validate() error {
	if len(p.Types) > 3 {
		return ErrStatsTypesTooMany
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "time", p.Time)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetInt(values, "limit", p.Limit)
	return values
}

type StatsFeedParams struct {
	After        string
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
	Decorations  []string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsFeedParams) Validate() error {
	if len(p.Types) == 0 {
		return ErrStatsTypesRequired
	}
	if len(p.Types) > 3 {
		return ErrStatsTypesTooMany
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsFeedParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetStringsCSV(values, "decorations", p.Decorations)
	httpx.SetInt(values, "limit", p.Limit)
	return values
}

type StatsHistoryParams struct {
	After        string
	StartTime    time.Time
	EndTime      time.Time
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
	Decorations  []string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsHistoryParams) Validate() error {
	if p.StartTime.IsZero() || p.EndTime.IsZero() {
		return ErrStatsTimeRangeRequired
	}
	if len(p.Types) == 0 {
		return ErrStatsTypesRequired
	}
	if len(p.Types) > 3 {
		return ErrStatsTypesTooMany
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsHistoryParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "startTime", &p.StartTime)
	httpx.SetTime(values, "endTime", &p.EndTime)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetStringsCSV(values, "decorations", p.Decorations)
	httpx.SetInt(values, "limit", p.Limit)
	return values
}
