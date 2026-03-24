package routes

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	After     string
	Include   []string
}

func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	if p.StartTime != nil && p.EndTime != nil && p.EndTime.Before(*p.StartTime) {
		return ErrListLimitInvalid
	}
	return nil
}

func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "include", p.Include)
	return values
}
