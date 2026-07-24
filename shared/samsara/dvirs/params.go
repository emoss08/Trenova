package dvirs

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type StreamParams struct {
	StartTime          *time.Time
	EndTime            *time.Time
	SafetyStatuses     []string
	After              string
	Limit              int
	IncludeExternalIDs bool
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StreamParams) Validate() error {
	if p.StartTime == nil {
		return ErrStartTimeRequired
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 200) {
		return ErrStreamLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StreamParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "safetyStatus", p.SafetyStatuses)
	httpx.SetString(values, "after", p.After)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetBool(values, "includeExternalIds", p.IncludeExternalIDs)
	return values
}

type GetParams struct {
	IncludeExternalIDs bool
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p GetParams) Query() url.Values {
	values := url.Values{}
	httpx.SetBool(values, "includeExternalIds", p.IncludeExternalIDs)
	return values
}

type HistoryParams struct {
	StartTime    *time.Time
	EndTime      *time.Time
	TagIDs       []string
	ParentTagIDs []string
	After        string
	Limit        int
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HistoryParams) Validate() error {
	if p.StartTime == nil {
		return ErrStartTimeRequired
	}
	if p.EndTime == nil {
		return ErrEndTimeRequired
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrHistoryLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p HistoryParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetString(values, "after", p.After)
	httpx.SetInt(values, "limit", p.Limit)
	return values
}
