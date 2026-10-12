package forms

import (
	"fmt"
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

const (
	IncludeExternalIDs     = "externalIds"
	maxListIDs             = 100
	MaxStreamFilterEntries = 50
)

type TemplateListParams struct {
	IDs   []string
	After string
}

func (p TemplateListParams) Validate() error {
	if len(p.IDs) > maxListIDs {
		return ErrTemplateIDsTooMany
	}
	return nil
}

func (p TemplateListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetString(values, "after", p.After)
	return values
}

type SubmissionListParams struct {
	IDs     []string
	Include []string
}

func (p SubmissionListParams) Validate() error {
	if len(p.IDs) == 0 {
		return ErrSubmissionIDsRequired
	}
	if len(p.IDs) > maxListIDs {
		return ErrSubmissionIDsTooMany
	}
	return validateInclude(p.Include)
}

func (p SubmissionListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetStringsCSV(values, "include", p.Include)
	return values
}

type SubmissionStreamParams struct {
	StartTime              *time.Time
	EndTime                *time.Time
	FormTemplateIDs        []string
	UserIDs                []string
	DriverIDs              []string
	Include                []string
	AssignedToRouteStopIDs []string
	After                  string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p SubmissionStreamParams) Validate() error {
	if p.StartTime == nil {
		return ErrStreamStartTimeRequired
	}
	if p.EndTime != nil && p.EndTime.Before(*p.StartTime) {
		return ErrStreamTimeRangeInvalid
	}
	for _, filter := range [][]string{
		p.FormTemplateIDs,
		p.UserIDs,
		p.DriverIDs,
		p.AssignedToRouteStopIDs,
	} {
		if len(filter) > MaxStreamFilterEntries {
			return ErrStreamFilterTooMany
		}
	}
	return validateInclude(p.Include)
}

func validateInclude(include []string) error {
	for _, value := range include {
		if value != IncludeExternalIDs {
			return fmt.Errorf("%w: %q", ErrIncludeInvalid, value)
		}
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p SubmissionStreamParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetStringsCSV(values, "formTemplateIds", p.FormTemplateIDs)
	httpx.SetStringsCSV(values, "userIds", p.UserIDs)
	httpx.SetStringsCSV(values, "driverIds", p.DriverIDs)
	httpx.SetStringsCSV(values, "include", p.Include)
	httpx.SetStringsCSV(values, "assignedToRouteStopIds", p.AssignedToRouteStopIDs)
	httpx.SetString(values, "after", p.After)
	return values
}
