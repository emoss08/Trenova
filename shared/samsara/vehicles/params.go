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
