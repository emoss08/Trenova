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
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var ErrInvalidMeasurementMethod = errors.New("invalid leave measurement method")

// LeaveMeasurementMethod is how the twelve-month period is measured. An
// employer picks one of the four in 29 CFR 825.200(b) and must apply it to
// every employee alike, which is why it is one organisation-wide setting and
// not a choice made per case.
type LeaveMeasurementMethod string

const (
	// MeasureCalendarYear is the calendar year, January to December.
	MeasureCalendarYear = LeaveMeasurementMethod("CalendarYear")
	// MeasureHireAnniversary is a fixed twelve months starting on the
	// employee's own hire anniversary.
	MeasureHireAnniversary = LeaveMeasurementMethod("HireAnniversary")
	// MeasureForwardFromFirstUse runs twelve months forward from the first day
	// of leave taken.
	MeasureForwardFromFirstUse = LeaveMeasurementMethod("ForwardFromFirstUse")
	// MeasureRollingBackward looks back twelve months from each day of leave.
	// It is the only method that cannot be stacked to take 24 weeks in a row,
	// which is why most employers choose it.
	MeasureRollingBackward = LeaveMeasurementMethod("RollingBackward")
)

func (m LeaveMeasurementMethod) String() string { return string(m) }

func (m LeaveMeasurementMethod) IsValid() bool {
	switch m {
	case MeasureCalendarYear, MeasureHireAnniversary, MeasureForwardFromFirstUse,
		MeasureRollingBackward:
		return true
	default:
		return false
	}
}

func (m LeaveMeasurementMethod) Label() string {
	switch m {
	case MeasureCalendarYear:
		return "Calendar year"
	case MeasureHireAnniversary:
		return "Twelve months from the hire anniversary"
	case MeasureForwardFromFirstUse:
		return "Twelve months forward from first use"
	case MeasureRollingBackward:
		return "Rolling twelve months looking back"
	default:
		return string(m)
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*LeaveControl)(nil)
	_ validationframework.TenantedEntity = (*LeaveControl)(nil)
)

// LeaveControl is the organisation's leave settings. There is exactly one row
// per organisation because the measurement method has to be applied
// consistently (29 CFR 825.200(e)).
type LeaveControl struct {
	bun.BaseModel `bun:"table:leave_controls,alias:lctl" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	MeasurementMethod LeaveMeasurementMethod `json:"measurementMethod" bun:"measurement_method,type:leave_measurement_method_enum,notnull,default:'RollingBackward'"`

	EntitlementWeeks       decimal.Decimal `json:"entitlementWeeks"       bun:"entitlement_weeks,type:NUMERIC(5,2),notnull"`
	MilitaryCaregiverWeeks decimal.Decimal `json:"militaryCaregiverWeeks" bun:"military_caregiver_weeks,type:NUMERIC(5,2),notnull"`
	// WorkweekHours is what a week of entitlement is worth. Intermittent leave
	// is taken in hours (29 CFR 825.205), so weeks only become a usable balance
	// once there is a week's length to convert them with.
	WorkweekHours decimal.Decimal `json:"workweekHours" bun:"workweek_hours,type:NUMERIC(5,2),notnull"`

	EligibilityMonths    int32 `json:"eligibilityMonths"    bun:"eligibility_months,type:INTEGER,notnull"`
	EligibilityHours     int32 `json:"eligibilityHours"     bun:"eligibility_hours,type:INTEGER,notnull"`
	CertificationDueDays int32 `json:"certificationDueDays" bun:"certification_due_days,type:INTEGER,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// DefaultLeaveControl is what an organisation gets before anybody configures
// one: the statutory figures.
func DefaultLeaveControl() *LeaveControl {
	return &LeaveControl{
		MeasurementMethod:      MeasureRollingBackward,
		EntitlementWeeks:       decimal.NewFromInt(12),
		MilitaryCaregiverWeeks: decimal.NewFromInt(26),
		WorkweekHours:          decimal.NewFromInt(40),
		EligibilityMonths:      12,
		EligibilityHours:       1250,
		CertificationDueDays:   15,
	}
}

func (c *LeaveControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.MeasurementMethod,
			validation.Required.Error("Measurement method is required"),
			domainvalidation.ValidEnum[LeaveMeasurementMethod](
				"Measurement method is not valid",
			),
		),
	))

	if !c.EntitlementWeeks.IsPositive() {
		multiErr.Add(
			"entitlementWeeks",
			errortypes.ErrInvalid,
			"The entitlement must be more than zero weeks",
		)
	}
	if !c.MilitaryCaregiverWeeks.IsPositive() {
		multiErr.Add(
			"militaryCaregiverWeeks",
			errortypes.ErrInvalid,
			"The military caregiver entitlement must be more than zero weeks",
		)
	}
	if !c.WorkweekHours.IsPositive() {
		multiErr.Add(
			"workweekHours",
			errortypes.ErrInvalid,
			"A workweek must be more than zero hours",
		)
	}
	if c.CertificationDueDays <= 0 {
		multiErr.Add(
			"certificationDueDays",
			errortypes.ErrInvalid,
			"Certification must be given at least a day",
		)
	}
	// The statute is a floor, not a target: an employer may be more generous
	// but cannot offer less than twelve weeks and call it FMLA.
	if c.EntitlementWeeks.LessThan(decimal.NewFromInt(12)) {
		multiErr.Add(
			"entitlementWeeks",
			errortypes.ErrInvalid,
			"FMLA entitles an eligible employee to at least twelve weeks (29 CFR 825.200(a))",
		)
	}
}

// EntitlementHours is a full entitlement expressed in hours, which is the unit
// a balance is actually drawn down in.
func (c *LeaveControl) EntitlementHours(militaryCaregiver bool) decimal.Decimal {
	weeks := c.EntitlementWeeks
	if militaryCaregiver {
		weeks = c.MilitaryCaregiverWeeks
	}
	return weeks.Mul(c.WorkweekHours)
}

func (c *LeaveControl) GetID() pulid.ID { return c.ID }

func (c *LeaveControl) GetCreatedAt() int64 { return c.CreatedAt }

func (c *LeaveControl) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *LeaveControl) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *LeaveControl) GetTableName() string { return "leave_controls" }

func (c *LeaveControl) GetResourceType() string { return "leave_control" }

func (c *LeaveControl) GetResourceID() string { return c.ID.String() }

func (c *LeaveControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("lctl_")
		}
		if c.MeasurementMethod == "" {
			c.MeasurementMethod = MeasureRollingBackward
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

// LeaveWindow is the twelve-month period a balance is measured over.
type LeaveWindow struct {
	From    int64
	Through int64
}

// Contains reports whether a day falls in the window. The window is
// half-open — a day on Through belongs to the next period — except for the
// rolling method, where Through is the day being asked about and has to count.
func (w LeaveWindow) Contains(at int64, inclusiveEnd bool) bool {
	if at < w.From {
		return false
	}
	if inclusiveEnd {
		return at <= w.Through
	}
	return at < w.Through
}

// EntitlementWindow works out the twelve months a balance is measured over.
//
// firstUseAt matters only to the forward method, where the clock starts on the
// first day of leave taken rather than on a date in the calendar. Zero means no
// leave has been taken yet, and the period has therefore not started.
func EntitlementWindow(
	method LeaveMeasurementMethod,
	asOf int64,
	hireDate int64,
	firstUseAt int64,
) LeaveWindow {
	at := time.Unix(asOf, 0).UTC()

	switch method {
	case MeasureCalendarYear:
		start := time.Date(at.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		return LeaveWindow{From: start.Unix(), Through: start.AddDate(1, 0, 0).Unix()}

	case MeasureHireAnniversary:
		return anniversaryWindow(at, hireDate)

	case MeasureForwardFromFirstUse:
		if firstUseAt <= 0 {
			// Nothing has been taken, so the period starts the moment it is.
			return LeaveWindow{From: asOf, Through: timeutils.AddMonthsUTC(asOf, 12)}
		}
		return forwardWindow(asOf, firstUseAt)

	default:
		// Rolling backward: the twelve months preceding the day asked about.
		return LeaveWindow{From: timeutils.AddMonthsUTC(asOf, -12), Through: asOf}
	}
}

// anniversaryWindow finds the twelve months beginning on the most recent hire
// anniversary. A worker with no hire date on file falls back to the calendar
// year rather than to the epoch, which would report a window fifty years wide.
func anniversaryWindow(at time.Time, hireDate int64) LeaveWindow {
	if hireDate <= 0 {
		start := time.Date(at.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		return LeaveWindow{From: start.Unix(), Through: start.AddDate(1, 0, 0).Unix()}
	}

	hired := time.Unix(hireDate, 0).UTC()
	start := anniversaryIn(at.Year(), hired)
	if start.After(at) {
		start = anniversaryIn(at.Year()-1, hired)
	}
	return LeaveWindow{From: start.Unix(), Through: anniversaryIn(
		start.Year()+1,
		hired,
	).Unix()}
}

// anniversaryIn is the hire anniversary falling in a given year. A leap-day
// hire has none in three years out of four; the day is clamped to the end of
// the month rather than allowed to spill into March, which would move the
// anniversary a day every time the leap year came round.
func anniversaryIn(year int, hired time.Time) time.Time {
	lastDay := time.Date(year, hired.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := hired.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, hired.Month(), day, 0, 0, 0, 0, time.UTC)
}

// forwardWindow steps twelve-month periods forward from the first use until it
// finds the one the day asked about falls in. Somebody who took leave three
// years ago is in their fourth period, not still in their first.
func forwardWindow(asOf int64, firstUseAt int64) LeaveWindow {
	from := firstUseAt
	for {
		through := timeutils.AddMonthsUTC(from, 12)
		if asOf < through {
			return LeaveWindow{From: from, Through: through}
		}
		from = through
	}
}

// LeaveEntitlement is a worker's FMLA standing: what they are entitled to, what
// they have used inside the measurement window, and what is left.
type LeaveEntitlement struct {
	Method         LeaveMeasurementMethod
	Window         LeaveWindow
	TotalHours     decimal.Decimal
	UsedHours      decimal.Decimal
	RemainingHours decimal.Decimal
	TotalWeeks     decimal.Decimal
	UsedWeeks      decimal.Decimal
	RemainingWeeks decimal.Decimal
	// Exhausted is true once nothing is left. It is reported rather than
	// inferred from RemainingHours because a worker who has used more than the
	// entitlement — which happens when leave is designated after the fact —
	// still has a remaining balance of zero, not a negative one.
	Exhausted bool
	// EligibleOnTenure says whether the worker has been employed long enough.
	// The hours-worked half of the test cannot be answered from tenure alone,
	// so it is reported separately and only where the office has recorded it.
	EligibleOnTenure  bool
	MonthsEmployed    int
	OpenCaseCount     int
	MilitaryCaregiver bool
}

// LeaveEntitlementInput is the evidence a balance is derived from.
type LeaveEntitlementInput struct {
	Control  *LeaveControl
	HireDate int64
	Cases    []*WorkerLeaveCase
	Entries  []*WorkerLeaveEntry
	AsOf     int64
}

// BuildLeaveEntitlement measures the balance. Only entries that count against
// the entitlement are drawn down: leave the employer chose not to designate is
// still recorded, because the decision not to count it is worth keeping.
func BuildLeaveEntitlement(in LeaveEntitlementInput) LeaveEntitlement {
	control := in.Control
	if control == nil {
		control = DefaultLeaveControl()
	}
	asOf := in.AsOf
	if asOf <= 0 {
		asOf = timeutils.NowUnix()
	}

	militaryCaregiver, openCases := scanCases(in.Cases)
	window := EntitlementWindow(
		control.MeasurementMethod,
		asOf,
		in.HireDate,
		firstUse(in.Entries),
	)

	// The rolling method asks "how much was used in the twelve months up to
	// today", so today itself counts. Every other method is a period with an
	// end that belongs to the next one.
	inclusiveEnd := control.MeasurementMethod == MeasureRollingBackward

	used := decimal.Zero
	for _, entry := range in.Entries {
		if entry == nil || !entry.CountsAgainstEntitlement {
			continue
		}
		if !window.Contains(entry.UsedOn, inclusiveEnd) {
			continue
		}
		used = used.Add(entry.Hours)
	}

	total := control.EntitlementHours(militaryCaregiver)
	remaining := total.Sub(used)
	if remaining.IsNegative() {
		remaining = decimal.Zero
	}

	months := monthsBetween(in.HireDate, asOf)

	entitlement := LeaveEntitlement{
		Method:            control.MeasurementMethod,
		Window:            window,
		TotalHours:        total,
		UsedHours:         used,
		RemainingHours:    remaining,
		Exhausted:         remaining.IsZero(),
		EligibleOnTenure:  in.HireDate > 0 && months >= int(control.EligibilityMonths),
		MonthsEmployed:    months,
		OpenCaseCount:     openCases,
		MilitaryCaregiver: militaryCaregiver,
	}

	if control.WorkweekHours.IsPositive() {
		entitlement.TotalWeeks = total.Div(control.WorkweekHours).Round(2)
		entitlement.UsedWeeks = used.Div(control.WorkweekHours).Round(2)
		entitlement.RemainingWeeks = remaining.Div(control.WorkweekHours).Round(2)
	}

	return entitlement
}

// scanCases reports whether any open case is a military caregiver case — which
// raises the entitlement for the period — and how many cases are open.
func scanCases(cases []*WorkerLeaveCase) (militaryCaregiver bool, open int) {
	for _, leaveCase := range cases {
		if leaveCase == nil {
			continue
		}
		if leaveCase.IsOpen() {
			open++
			if leaveCase.MilitaryCaregiver {
				militaryCaregiver = true
			}
		}
	}
	return militaryCaregiver, open
}

func firstUse(entries []*WorkerLeaveEntry) int64 {
	var earliest int64
	for _, entry := range entries {
		if entry == nil || !entry.CountsAgainstEntitlement {
			continue
		}
		if earliest == 0 || entry.UsedOn < earliest {
			earliest = entry.UsedOn
		}
	}
	return earliest
}

func monthsBetween(from int64, to int64) int {
	if from <= 0 || to <= from {
		return 0
	}
	start := time.Unix(from, 0).UTC()
	end := time.Unix(to, 0).UTC()
	months := (end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month())
	if end.Day() < start.Day() {
		months--
	}
	if months < 0 {
		return 0
	}
	return months
}
