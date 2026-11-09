package factoringoverviewservice

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	MovedWindowDays = 30
	PaceSchedules   = 20
	AttentionLimit  = 20
	defaultCurrency = "USD"
)

type Params struct {
	fx.In

	Repo      factoringports.OverviewRepository
	Customers factoringports.FactorCustomerRepository
	Now       timeutils.Clock
}

type Service struct {
	repo      factoringports.OverviewRepository
	customers factoringports.FactorCustomerRepository
	now       timeutils.Clock
}

func New(p Params) *Service {
	return &Service{repo: p.Repo, customers: p.Customers, now: p.Now}
}

type Request struct {
	TenantInfo         pagination.TenantInfo
	FactoringCompanyID pulid.ID
	AmountsIncluded    bool
}

type PacePoint struct {
	SubmissionID pulid.ID
	Number       string
	SentAt       int64
	FundedAt     int64
	BusinessDays int
	Late         bool
}

type Delivery struct {
	Configured    bool
	Method        factoringsubmission.TransportMethod
	LastTestedAt  *int64
	LastTestError string
}

func (d Delivery) Failing() bool {
	return d.Configured && d.LastTestError != ""
}

type Setup struct {
	Agreement bool
	Delivery  bool
	Customers bool
	Notices   bool
}

func (s Setup) Done() int {
	done := 0
	for _, step := range []bool{s.Agreement, s.Delivery, s.Customers, s.Notices} {
		if step {
			done++
		}
	}

	return done
}

type Overview struct {
	FactoringCompanyID pulid.ID
	FactorName         string
	CurrencyCode       string
	AmountsIncluded    bool
	MovedWindowDays    int
	Ready              factoringports.VerdictTotals
	ReadyLate          factoringports.VerdictTotals
	Submitted          factoringports.VerdictTotals
	Funded             factoringports.VerdictTotals
	ReserveHeld        factoringports.VerdictTotals
	Released           factoringports.VerdictTotals
	ChargebacksDue     factoringports.VerdictTotals
	ChargedBack        factoringports.VerdictTotals
	OldestFundedOn     *int64
	ChargebackDeadline *int64
	Awaiting           *factoringports.AwaitingSubmission
	AwaitingCount      int
	Pace               []PacePoint
	PaceMedianDays     *int
	PaceLate           int
	Agreement          *factoring.Agreement
	Delivery           Delivery
	Notices            factoringports.NoticeCounts
	Attention          []Attention
	Setup              Setup
	Summary            []Segment
}

func (s *Service) Build(ctx context.Context, req *Request) (*Overview, error) {
	now := s.now()
	read, err := s.repo.Get(ctx, &factoringports.GetOverviewRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: req.FactoringCompanyID,
		At:                 now,
		ReadyBefore:        now - factoringreadiness.ReadyUnsubmittedSeconds,
		ChargebackBefore:   now + int64(factoring.ChargebackWarningDays)*timeutils.SecondsPerDay,
		MovedSince:         now - MovedWindowDays*timeutils.SecondsPerDay,
		PaceLimit:          PaceSchedules,
		AttentionLimit:     AttentionLimit,
	})
	if err != nil {
		return nil, err
	}

	loc, _ := timeutils.ResolveZone(read.Factor.Timezone)
	summary := read.Summary
	out := &Overview{
		FactoringCompanyID: req.FactoringCompanyID,
		FactorName:         read.Factor.Name,
		CurrencyCode:       currencyOf(read),
		AmountsIncluded:    req.AmountsIncluded,
		MovedWindowDays:    MovedWindowDays,
		Ready:              summary.Ready,
		ReadyLate:          summary.ReadyUnsubmitted,
		Submitted:          summary.Submitted,
		Funded:             summary.Funded,
		ReserveHeld:        summary.ReserveHeld,
		Released:           read.Released,
		ChargebacksDue:     summary.ChargebacksDue,
		ChargedBack:        read.ChargedBack,
		OldestFundedOn:     summary.OldestFundedOn,
		AwaitingCount:      read.AwaitingCount,
		Agreement:          read.Agreement,
		Delivery:           deliveryOf(read.Transport),
		Notices:            read.Notices,
	}
	if read.SoonestChargeback != nil {
		deadline := read.SoonestChargeback.DeadlineAt
		out.ChargebackDeadline = &deadline
	}
	if len(read.Awaiting) > 0 {
		out.Awaiting = read.Awaiting[0]
	}
	out.Pace, out.PaceMedianDays, out.PaceLate = pace(read, loc)
	out.Setup = Setup{
		Agreement: read.Agreement != nil,
		Delivery:  out.Delivery.Configured && !out.Delivery.Failing(),
		Customers: read.Notices.Assigned > 0,
		Notices: read.Notices.Assigned > 0 && read.Notices.NeedsNotice == 0 &&
			read.Notices.Awaiting == 0,
	}
	out.Attention = attention(read, out)
	if !req.AmountsIncluded {
		withholdAmounts(out)
	}
	out.Summary = summarize(&narration{
		locale: i18n.FromContext(ctx),
		loc:    loc,
		read:   read,
		result: out,
	})

	return out, nil
}

func currencyOf(read *factoringports.OverviewRead) string {
	for _, sub := range read.Awaiting {
		if sub.CurrencyCode != "" {
			return sub.CurrencyCode
		}
	}
	for _, rem := range read.Remittances {
		if rem.CurrencyCode != "" {
			return rem.CurrencyCode
		}
	}
	for _, sub := range read.Failed {
		if sub.CurrencyCode != "" {
			return sub.CurrencyCode
		}
	}

	return defaultCurrency
}

func deliveryOf(t *factoringsubmission.Transport) Delivery {
	if t == nil {
		return Delivery{}
	}

	return Delivery{
		Configured:    true,
		Method:        t.Method,
		LastTestedAt:  t.LastTestedAt,
		LastTestError: t.LastTestError,
	}
}

func pace(
	read *factoringports.OverviewRead,
	loc *time.Location,
) (points []PacePoint, median *int, late int) {
	agreed := -1
	if read.Agreement != nil {
		agreed = read.Agreement.FundingDays
	}

	points = make([]PacePoint, 0, len(read.Funded))
	for i := len(read.Funded) - 1; i >= 0; i-- {
		sub := read.Funded[i]
		days := timeutils.BusinessDaysBetween(sub.SentAt, sub.FundedAt, loc)
		point := PacePoint{
			SubmissionID: sub.SubmissionID,
			Number:       sub.Number,
			SentAt:       sub.SentAt,
			FundedAt:     sub.FundedAt,
			BusinessDays: days,
			Late:         agreed >= 0 && days > agreed,
		}
		if point.Late {
			late++
		}
		points = append(points, point)
	}
	if len(points) == 0 {
		return points, nil, 0
	}

	days := make([]int, len(points))
	for i, point := range points {
		days[i] = point.BusinessDays
	}
	slices.Sort(days)
	middle := days[len(days)/2]

	return points, &middle, late
}

func withholdAmounts(o *Overview) {
	for _, totals := range []*factoringports.VerdictTotals{
		&o.Ready, &o.ReadyLate, &o.Submitted, &o.Funded, &o.ReserveHeld,
		&o.Released, &o.ChargebacksDue, &o.ChargedBack,
	} {
		totals.AmountMinor = 0
	}
	if o.Awaiting != nil {
		awaiting := *o.Awaiting
		awaiting.FaceMinor = 0
		o.Awaiting = &awaiting
	}
	for i := range o.Attention {
		o.Attention[i].AmountMinor = 0
	}
}
