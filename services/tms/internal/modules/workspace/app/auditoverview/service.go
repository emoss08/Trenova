package auditoverview

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/auditreviewservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	MaxDays       = 90
	HourlyMaxDays = 7
	ActorLimit    = 5
	ReviewLimit   = 3
)

type Params struct {
	fx.In

	Repo auditports.OverviewRepository
	Now  timeutils.Clock
}

type Service struct {
	repo auditports.OverviewRepository
	now  timeutils.Clock
}

func New(p Params) *Service {
	return &Service{repo: p.Repo, now: p.Now}
}

type Request struct {
	TenantInfo     pagination.TenantInfo
	Days           int
	Access         auditports.EntryAccess
	IncludeReviews bool
}

type Unit string

const (
	UnitHour = Unit("Hour")
	UnitDay  = Unit("Day")
)

type Overview struct {
	Days            int
	From            int64
	To              int64
	Timezone        string
	Unit            Unit
	Totals          auditports.OverviewFigures
	PreviousEntries int64
	Activity        []auditports.ActivityBucket
	Actors          auditports.ActorCounts
	TopActors       []auditports.ActorStat
	OtherActors     int
	OtherEntries    int64
	ReviewsIncluded bool
	OpenReviews     []auditreviewservice.View
	OpenReviewCount int
	Summary         []Segment
}

type window struct {
	loc      *time.Location
	zone     string
	unit     Unit
	current  auditports.OverviewWindow
	previous auditports.OverviewWindow
	until    time.Time
}

func (s *Service) window(ctx context.Context, req *Request) (*window, error) {
	if req.Days < 1 || req.Days > MaxDays {
		return nil, errortypes.NewValidationError(
			"days",
			errortypes.ErrInvalid,
			"The audit overview looks back between a day and 90 days",
		)
	}

	zone, err := s.repo.OrganizationTimezone(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	loc := auditreviewservice.Location(zone)
	if loc == time.UTC {
		zone = "UTC"
	}

	now := time.Unix(s.now(), 0).In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	first := today.AddDate(0, 0, -(req.Days - 1))
	end := today.AddDate(0, 0, 1)
	previousFirst := first.AddDate(0, 0, -req.Days)

	unit := UnitDay
	if req.Days <= HourlyMaxDays {
		unit = UnitHour
	}

	return &window{
		loc:      loc,
		zone:     zone,
		unit:     unit,
		current:  auditports.OverviewWindow{From: first.Unix(), To: end.Unix() - 1},
		previous: auditports.OverviewWindow{From: previousFirst.Unix(), To: first.Unix() - 1},
		until:    now,
	}, nil
}

func (s *Service) Build(ctx context.Context, req *Request) (*Overview, error) {
	w, err := s.window(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &Overview{
		Days:            req.Days,
		From:            w.current.From,
		To:              w.current.To,
		Timezone:        w.zone,
		Unit:            w.unit,
		ReviewsIncluded: req.IncludeReviews,
	}
	if !req.Access.AllResources && len(req.Access.Resources) == 0 {
		result.Activity = everyBucket(nil, w)
		result.Summary = summarize(
			&narration{locale: i18n.FromContext(ctx), days: req.Days, result: result},
		)
		return result, nil
	}

	rollup, err := s.repo.Overview(ctx, &auditports.OverviewRequest{
		TenantInfo:     req.TenantInfo,
		Access:         req.Access,
		Current:        w.current,
		Previous:       w.previous,
		Hourly:         w.unit == UnitHour,
		ActorLimit:     ActorLimit,
		IncludeReviews: req.IncludeReviews,
		ReviewLimit:    ReviewLimit,
	})
	if err != nil {
		return nil, err
	}

	locale := i18n.FromContext(ctx)
	result.Totals = rollup.Totals
	result.PreviousEntries = rollup.PreviousEntries
	result.Activity = everyBucket(rollup.Activity, w)
	result.Actors = rollup.Actors
	result.TopActors = rollup.TopActors
	result.OtherActors = max(rollup.Actors.Total()-len(rollup.TopActors), 0)
	result.OtherEntries = rollup.Totals.Entries
	for _, actor := range rollup.TopActors {
		result.OtherEntries -= actor.Entries
	}
	result.OtherEntries = max(result.OtherEntries, 0)
	result.OpenReviews = auditreviewservice.Present(locale, w.loc, rollup.OpenReviews)
	result.OpenReviewCount = rollup.OpenReviewCount
	result.Summary = summarize(&narration{locale: locale, days: req.Days, result: result})

	return result, nil
}

func everyBucket(rows []auditports.ActivityBucket, w *window) []auditports.ActivityBucket {
	byStart := make(map[int64]auditports.ActivityBucket, len(rows))
	for _, row := range rows {
		byStart[row.At.Unix()] = row
	}

	start := time.Unix(w.current.From, 0).In(w.loc)
	end := time.Unix(w.current.To, 0).In(w.loc)
	if w.until.Before(end) {
		end = w.until
	}

	out := make([]auditports.ActivityBucket, 0, len(rows))
	for at := start; !at.After(end); at = next(at, w.unit) {
		bucket, ok := byStart[at.Unix()]
		if !ok {
			bucket = auditports.ActivityBucket{}
		}
		bucket.At = at
		out = append(out, bucket)
	}

	return out
}

func next(at time.Time, unit Unit) time.Time {
	if unit == UnitHour {
		return at.Add(time.Hour)
	}
	return at.AddDate(0, 0, 1)
}
