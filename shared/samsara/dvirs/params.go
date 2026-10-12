package dvirs

import (
	"fmt"
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

const (
	maxStreamLimit  = 200
	maxHistoryLimit = 512
)

var safetyStatuses = map[string]struct{}{
	"safe":     {},
	"unsafe":   {},
	"resolved": {},
}

type StreamParams struct {
	StartTime          *time.Time
	EndTime            *time.Time
	SafetyStatuses     []string
	After              string
	Limit              int
	IncludeExternalIDs bool
}

func (p StreamParams) Validate() error {
	if p.StartTime == nil {
		return ErrStartTimeRequired
	}
	if p.EndTime != nil && p.EndTime.Before(*p.StartTime) {
		return ErrTimeRangeInvalid
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > maxStreamLimit) {
		return ErrStreamLimitInvalid
	}
	for _, status := range p.SafetyStatuses {
		if _, ok := safetyStatuses[status]; !ok {
			return fmt.Errorf("%w: %q", ErrSafetyStatusInvalid, status)
		}
	}
	return nil
}

func (p StreamParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	for _, status := range p.SafetyStatuses {
		values.Add("safetyStatus", status)
	}
	httpx.SetString(values, "after", p.After)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetBool(values, "includeExternalIds", p.IncludeExternalIDs)
	return values
}

type GetParams struct {
	IncludeExternalIDs bool
}

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
	if p.EndTime.Before(*p.StartTime) {
		return ErrTimeRangeInvalid
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > maxHistoryLimit) {
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
