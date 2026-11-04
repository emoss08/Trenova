// Package detentionintelligence answers what an organization needs to lower detention: where the
// hours go, who caused the wait, when trucks wait, how much of the money is recovered, whether
// it is getting better, and what to do next.
package detentionintelligence

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const (
	// FacilityLimit bounds the facilities and customers one read returns. The totals cover
	// every stop whatever the limit.
	FacilityLimit = 100
	// MaxDays is the longest window a read covers.
	MaxDays     = 366
	defaultZone = "UTC"
	dayLayout   = "2006-01-02"

	daysPerWeek    = 7
	secondsPerWeek = int64(daysPerWeek * 24 * 60 * 60)
)

type Params struct {
	fx.In

	Repo detentionports.DetentionIntelligenceRepository
	Now  timeutils.Clock
}

type Service struct {
	repo detentionports.DetentionIntelligenceRepository
	now  timeutils.Clock
}

func New(p Params) *Service {
	return &Service{repo: p.Repo, now: p.Now}
}

// window is a read's days in the organization's time zone: the window and the one of the same
// length before it.
type window struct {
	loc      *time.Location
	zone     string
	days     int
	through  string
	current  detentionports.IntelligenceWindow
	previous detentionports.IntelligenceWindow
}

// daysWindow is the days [first, last], local midnights, with the whole months inside them.
func daysWindow(first, last time.Time) detentionports.IntelligenceWindow {
	loc := first.Location()
	after := last.AddDate(0, 0, 1)
	months := first
	if first.Day() != 1 {
		months = time.Date(first.Year(), first.Month()+1, 1, 0, 0, 0, 0, loc)
	}
	end := time.Date(after.Year(), after.Month(), 1, 0, 0, 0, 0, loc)
	if !end.After(months) {
		months, end = after, after
	}

	return detentionports.IntelligenceWindow{
		From:       first.Unix(),
		To:         after.Unix() - 1,
		MonthsFrom: months.Unix(),
		MonthsEnd:  end.Unix(),
	}
}

func (s *Service) window(ctx context.Context, req *Request) (*window, error) {
	if req.Days < 1 || req.Days > MaxDays {
		return nil, errortypes.NewValidationError(
			"days",
			errortypes.ErrInvalid,
			"Detention intelligence looks back between a day and a year",
		)
	}

	zone, err := s.repo.OrganizationTimezone(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	// Go reads "Local" as the server's own zone, which is no organization's.
	loc, err := time.LoadLocation(zone)
	if zone == "" || zone == "Local" || err != nil {
		zone, loc = defaultZone, time.UTC
	}

	now := time.Unix(s.now(), 0).In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	last := today
	if req.Through != "" {
		last, err = time.ParseInLocation(dayLayout, req.Through, loc)
		if err != nil || last.After(today) {
			return nil, errortypes.NewValidationError(
				"through",
				errortypes.ErrInvalid,
				"The window ends on a day no later than today, written as YYYY-MM-DD",
			)
		}
	}

	first := last.AddDate(0, 0, -(req.Days - 1))
	previousLast := first.AddDate(0, 0, -1)

	return &window{
		loc:      loc,
		zone:     zone,
		days:     req.Days,
		through:  last.Format(dayLayout),
		current:  daysWindow(first, last),
		previous: daysWindow(previousLast.AddDate(0, 0, -(req.Days-1)), previousLast),
	}, nil
}

// Build reads the window and the one before it in one round trip and works out every figure,
// the summary and the next steps from that one read, so they agree.
func (s *Service) Build(ctx context.Context, req *Request) (*Intelligence, error) {
	w, err := s.window(ctx, req)
	if err != nil {
		return nil, err
	}

	rollup, err := s.repo.Intelligence(ctx, &detentionports.IntelligenceRequest{
		TenantInfo: req.TenantInfo,
		Current:    w.current,
		Previous:   w.previous,
		Timezone:   w.zone,
		LocationID: req.LocationID,
		Limit:      FacilityLimit,
	})
	if err != nil {
		return nil, err
	}
	rollup.WithholdFields(req.Visible)

	result := &Intelligence{
		Days:          w.days,
		Through:       w.through,
		From:          w.current.From,
		To:            w.current.To,
		PreviousFrom:  w.previous.From,
		PreviousTo:    w.previous.To,
		Timezone:      w.zone,
		Totals:        rollup.Totals,
		Previous:      rollup.Previous,
		Facilities:    facilities(rollup),
		Customers:     rollup.Customers,
		Waivers:       rollup.Waivers,
		Recovery:      recovery(rollup.Recovery),
		Arrivals:      rollup.Arrivals,
		Splits:        rollup.Splits,
		Weeks:         everyWeek(rollup.Days, w),
		PolicyChanges: policyChanges(rollup),
		Truncated:     len(rollup.Facilities) >= FacilityLimit,
	}

	n := &narration{
		locale:  i18n.FromContext(ctx),
		days:    w.days,
		result:  result,
		margins: rollup.Totals.NetMargin.Valid,
	}
	result.Summary = summarize(n)
	result.NextSteps = nextSteps(n)

	return result, nil
}

// everyWeek adds the window's days up by the local week they fall in, with every week of the
// window present, a quiet one as zeros.
func everyWeek(days []detentionports.DayFigure, w *window) []detentionports.WeekBucket {
	first, last := weekStart(w.current.From, w.loc), weekStart(w.current.To, w.loc)
	byStart := make(map[int64]detentionports.WeekBucket, len(days)/daysPerWeek+2)
	for _, day := range days {
		start := weekStart(day.Day.Unix(), w.loc)
		bucket := byStart[start]
		bucket.StopCount += day.StopCount
		bucket.DetainedCount += day.DetainedCount
		bucket.PastFreeMinutes += day.PastFreeMinutes
		byStart[start] = bucket
	}

	weeks := make([]detentionports.WeekBucket, 0, (last-first)/secondsPerWeek+1)
	for at := first; at <= last; at += secondsPerWeek {
		bucket := byStart[at]
		bucket.WeekStart = at
		weeks = append(weeks, bucket)
	}

	return weeks
}

func weekStart(at int64, loc *time.Location) int64 {
	local := time.Unix(at, 0).In(loc)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	sinceMonday := (int(date.Weekday()) + daysPerWeek - 1) % daysPerWeek

	return date.AddDate(0, 0, -sinceMonday).Unix()
}

func facilities(rollup *detentionports.IntelligenceRollup) []Facility {
	previous := make(map[pulid.ID]int64, len(rollup.PreviousFacilities))
	for _, row := range rollup.PreviousFacilities {
		previous[row.LocationID] = row.PastFreeMinutes
	}

	out := make([]Facility, 0, len(rollup.Facilities))
	for _, row := range rollup.Facilities {
		facility := Facility{FacilityIntelligence: row}
		if minutes, ok := previous[row.LocationID]; ok {
			facility.PreviousPastFreeMinutes = &minutes
		}
		out = append(out, facility)
	}

	return out
}

// recoveryOrder is the order the money falls away in: what the carrier forfeited, what notice
// lost, what someone forgave, what is argued, what is still waiting, and what does not count.
var recoveryOrder = []string{
	detentionports.RecoveryLate,
	detentionports.RecoveryNotice,
	detentionports.RecoveryWaived,
	detentionports.RecoveryDisputed,
	detentionports.RecoveryPending,
	detentionports.RecoveryOther,
}

func recovery(buckets []detentionports.RecoveryBucket) Recovery {
	byBucket := make(map[string]detentionports.RecoveryBucket, len(buckets))
	out := Recovery{}
	for _, bucket := range buckets {
		byBucket[bucket.Bucket] = bucket
		out.Accrued = out.Accrued.Add(bucket.GrossAmount)
	}

	billed := byBucket[detentionports.RecoveryBilled]
	out.Billed = billed.BillableAmount
	out.Invoiced = billed.InvoicedAmount
	out.Collected = billed.CollectedAmount

	out.Steps = make([]RecoveryStep, 0, len(recoveryOrder)+1)
	for _, key := range recoveryOrder {
		bucket := byBucket[key]
		out.Steps = append(out.Steps, RecoveryStep{
			Bucket:          key,
			OccurrenceCount: bucket.OccurrenceCount,
			Amount:          bucket.GrossAmount,
		})
	}
	// What a billed charge lost to caps, minimums and rounding, so the steps add up to accrued.
	out.Steps = append(out.Steps, RecoveryStep{
		Bucket:          detentionports.RecoveryCapped,
		OccurrenceCount: billed.OccurrenceCount,
		Amount:          decimal.Max(billed.GrossAmount.Sub(billed.BillableAmount), decimal.Zero),
	})

	return out
}

// policyChanges turns each facility's policies, ordered by first use, into the moments it moved
// from one to the next.
func policyChanges(rollup *detentionports.IntelligenceRollup) []PolicyChange {
	names := make(map[pulid.ID]string, len(rollup.Facilities))
	for _, facility := range rollup.Facilities {
		names[facility.LocationID] = facility.LocationName
	}

	out := make([]PolicyChange, 0)
	var last pulid.ID
	for idx, use := range rollup.PolicyUses {
		first := idx == 0 || use.LocationID != last
		last = use.LocationID
		if first {
			continue
		}
		out = append(out, PolicyChange{
			LocationID:   use.LocationID,
			LocationName: names[use.LocationID],
			PolicyID:     use.PolicyID,
			PolicyName:   use.PolicyName,
			ChangedAt:    use.FirstClockStart,
		})
	}

	return out
}
