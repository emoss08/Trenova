package assets

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	Type               Type
	After              string
	UpdatedAfterTime   *time.Time
	IncludeExternalIDs bool
	IncludeTags        bool
	TagIDs             []string
	ParentTagIDs       []string
	IDs                []string
	AttributeValueIDs  []string
	Attributes         []string
	Limit              int
}

//nolint:gocritic // value receiver keeps call sites simple for immutable params.
func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver keeps call sites simple for immutable params.
func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "type", string(p.Type))
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "updatedAfterTime", p.UpdatedAfterTime)
	httpx.SetBool(values, "includeExternalIds", p.IncludeExternalIDs)
	httpx.SetBool(values, "includeTags", p.IncludeTags)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetStringsCSV(values, "attributeValueIds", p.AttributeValueIDs)
	httpx.SetInt(values, "limit", p.Limit)
	for _, attr := range p.Attributes {
		if attr != "" {
			values.Add("attributes", attr)
		}
	}
	return values
}

type LocationStreamParams struct {
	After                         string
	Limit                         int
	StartTime                     *time.Time
	EndTime                       *time.Time
	IDs                           []string
	IncludeSpeed                  bool
	IncludeReverseGeo             bool
	IncludeGeofenceLookup         bool
	IncludeHighFrequencyLocations bool
	IncludeExternalIDs            bool
}

func (p LocationStreamParams) Validate() error {
	if p.StartTime == nil {
		return ErrLocationStartTimeRequired
	}
	if p.EndTime != nil && p.EndTime.Before(*p.StartTime) {
		return ErrLocationWindowInvalid
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	if p.IncludeHighFrequencyLocations && p.IncludeGeofenceLookup {
		return ErrHighFrequencyWithGeofence
	}
	return nil
}

func (p LocationStreamParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetBool(values, "includeSpeed", p.IncludeSpeed)
	httpx.SetBool(values, "includeReverseGeo", p.IncludeReverseGeo)
	httpx.SetBool(values, "includeGeofenceLookup", p.IncludeGeofenceLookup)
	httpx.SetBool(values, "includeHighFrequencyLocations", p.IncludeHighFrequencyLocations)
	httpx.SetBool(values, "includeExternalIds", p.IncludeExternalIDs)
	return values
}

type CurrentLocationsParams struct {
	IDs                           []string
	LookbackWindow                time.Duration
	IncludeSpeed                  bool
	IncludeReverseGeo             bool
	IncludeGeofenceLookup         bool
	IncludeHighFrequencyLocations bool
	IncludeExternalIDs            bool
	Limit                         int
}

type HistoricalLocationsParams struct {
	IDs                           []string
	StartTime                     time.Time
	EndTime                       time.Time
	IncludeSpeed                  bool
	IncludeReverseGeo             bool
	IncludeGeofenceLookup         bool
	IncludeHighFrequencyLocations bool
	IncludeExternalIDs            bool
	Limit                         int
}
