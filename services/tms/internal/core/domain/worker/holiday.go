package worker

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var ErrInvalidHolidayKind = errors.New("invalid holiday kind")

type HolidayKind string

const (
	HolidayKindHoliday  = HolidayKind("Holiday")
	HolidayKindBlackout = HolidayKind("Blackout")
)

func (k HolidayKind) String() string { return string(k) }

func (k HolidayKind) IsValid() bool {
	switch k {
	case HolidayKindHoliday, HolidayKindBlackout:
		return true
	default:
		return false
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*OrgHoliday)(nil)
	_ validationframework.TenantedEntity = (*OrgHoliday)(nil)
)

// OrgHoliday is one calendar date the organisation treats specially. The date
// is stored as midnight UTC of the calendar day so it compares the same way
// regardless of the organisation's timezone.
type OrgHoliday struct {
	bun.BaseModel `bun:"table:org_holidays,alias:ohol" json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string      `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	HolidayDate    int64       `json:"holidayDate"    bun:"holiday_date,type:BIGINT,notnull"`
	Kind           HolidayKind `json:"kind"           bun:"kind,type:org_holiday_kind_enum,notnull,default:'Holiday'"`
	RecursAnnually bool        `json:"recursAnnually" bun:"recurs_annually,type:BOOLEAN,notnull"`
	Description    string      `json:"description"    bun:"description,type:TEXT,nullzero"`
	Version        int64       `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (h *OrgHoliday) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(h,
		validation.Field(&h.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&h.HolidayDate,
			validation.Required.Error("Date is required"),
			validation.Min(int64(1)).Error("Date must be a valid date"),
		),
		validation.Field(&h.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[HolidayKind]("kind must be Holiday or Blackout"),
		),
	))
}

// DayKey is the calendar day this holiday falls on, in the UTC date used for
// storage, or the month/day only when it recurs every year.
func (h *OrgHoliday) DayKey() string {
	t := time.Unix(h.HolidayDate, 0).UTC()
	if h.RecursAnnually {
		return t.Format("01-02")
	}
	return t.Format("2006-01-02")
}

func (h *OrgHoliday) GetID() pulid.ID { return h.ID }

func (h *OrgHoliday) GetCreatedAt() int64 { return h.CreatedAt }

func (h *OrgHoliday) GetOrganizationID() pulid.ID { return h.OrganizationID }

func (h *OrgHoliday) GetBusinessUnitID() pulid.ID { return h.BusinessUnitID }

func (h *OrgHoliday) GetTableName() string { return "org_holidays" }

func (h *OrgHoliday) GetResourceType() string { return "org_holiday" }

func (h *OrgHoliday) GetResourceID() string { return h.ID.String() }

func (h *OrgHoliday) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if h.ID.IsNil() {
			h.ID = pulid.MustNew("ohol_")
		}
		h.CreatedAt = now
		h.UpdatedAt = now
	case *bun.UpdateQuery:
		h.UpdatedAt = now
	}

	return nil
}

// HolidayCalendar answers "is this local calendar day a holiday / a blackout"
// for the day walk in ComputePTODays and the blackout check on requests.
type HolidayCalendar struct {
	holidays  map[string]*OrgHoliday
	blackouts map[string]*OrgHoliday
	recurring map[string]*OrgHoliday
	recurBlk  map[string]*OrgHoliday
}

func NewHolidayCalendar(rows []*OrgHoliday) *HolidayCalendar {
	cal := &HolidayCalendar{
		holidays:  make(map[string]*OrgHoliday, len(rows)),
		blackouts: make(map[string]*OrgHoliday),
		recurring: make(map[string]*OrgHoliday),
		recurBlk:  make(map[string]*OrgHoliday),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		key := row.DayKey()
		switch {
		case row.Kind == HolidayKindBlackout && row.RecursAnnually:
			cal.recurBlk[key] = row
		case row.Kind == HolidayKindBlackout:
			cal.blackouts[key] = row
		case row.RecursAnnually:
			cal.recurring[key] = row
		default:
			cal.holidays[key] = row
		}
	}
	return cal
}

func (c *HolidayCalendar) IsEmpty() bool {
	return c == nil || (len(c.holidays) == 0 && len(c.blackouts) == 0 &&
		len(c.recurring) == 0 && len(c.recurBlk) == 0)
}

// HolidayOn reports whether the local day is an observed holiday.
func (c *HolidayCalendar) HolidayOn(day time.Time) bool {
	if c == nil {
		return false
	}
	if _, ok := c.holidays[day.Format("2006-01-02")]; ok {
		return true
	}
	_, ok := c.recurring[day.Format("01-02")]
	return ok
}

// BlackoutOn returns the blackout that covers the local day, if any.
func (c *HolidayCalendar) BlackoutOn(day time.Time) *OrgHoliday {
	if c == nil {
		return nil
	}
	if row, ok := c.blackouts[day.Format("2006-01-02")]; ok {
		return row
	}
	if row, ok := c.recurBlk[day.Format("01-02")]; ok {
		return row
	}
	return nil
}

// BlackoutsBetween lists the blackout dates a request range touches, in day
// order, so the validation message can name them.
func (c *HolidayCalendar) BlackoutsBetween(start, end int64, loc *time.Location) []*OrgHoliday {
	if c == nil || loc == nil {
		if loc == nil {
			loc = time.UTC
		}
	}
	first := localDate(start, loc)
	last := localDate(end, loc)
	out := make([]*OrgHoliday, 0, 2)
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		if row := c.BlackoutOn(d); row != nil {
			out = append(out, row)
		}
	}
	return out
}
