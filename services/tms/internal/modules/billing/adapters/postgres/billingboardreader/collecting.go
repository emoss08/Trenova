package billingboardreader

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

type Collecting struct {
	analytics customerledgerports.ARAnalyticsRepository
}

func NewCollecting(p AnalyticsParams) *Collecting {
	return &Collecting{analytics: p.Analytics}
}

func (*Collecting) Metric() string { return billingboard.MetricCollecting }

func (r *Collecting) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	earlier := earlierThan(req.Now, req.Period, req.Location)
	now, err := r.totalsAt(ctx, req, req.Now)
	if err != nil {
		return nil, err
	}
	then, err := r.totalsAt(ctx, req, earlier)
	if err != nil {
		return nil, err
	}

	return collectingRead(ctx, &collectingFacts{
		now:      now,
		then:     then,
		nowAt:    req.Now,
		thenAt:   earlier,
		location: req.Location,
	}), nil
}

func (r *Collecting) totalsAt(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
	at int64,
) (*customerledgerports.ARCollectionTotals, error) {
	totals, err := r.analytics.GetCollectionTotals(
		ctx,
		customerledgerports.GetARCollectionMetricsRequest{
			TenantInfo: req.Tenant,
			AsOfDate:   at,
			PeriodDays: customerledger.CollectionWindowDays,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read the collection totals: %w", err)
	}
	if totals == nil {
		return &customerledgerports.ARCollectionTotals{}, nil
	}

	return totals, nil
}

type collectingFacts struct {
	now, then     *customerledgerports.ARCollectionTotals
	nowAt, thenAt int64
	location      *time.Location
}

func daysToPay(totals *customerledgerports.ARCollectionTotals) *boardcatalog.Value {
	if totals.ApplicationCount == 0 {
		return nil
	}
	value := boardcatalog.Number(decimal.NewFromFloat(totals.AvgDaysToPay).Round(1))

	return &value
}

func collectedShare(totals *customerledgerports.ARCollectionTotals) *boardcatalog.Value {
	if totals.BeginningOpenMinor+totals.CreditSalesMinor-totals.EndingCurrentMinor <= 0 {
		return nil
	}
	value := boardcatalog.Percent(decimal.NewFromFloat(totals.CollectedShare()).Round(3))

	return &value
}

func collectingRead(ctx context.Context, facts *collectingFacts) *boardcatalog.MetricRead {
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Collecting"),
		Source: i18n.T(ctx,
			"{0, plural, one {# payment applied} other {# payments applied}} in the 91 days to {1}",
			facts.now.ApplicationCount, dayText(ctx, facts.nowAt, facts.location, facts.nowAt)),
		Figures: map[string]*boardcatalog.Value{
			"daysToPay":       daysToPay(facts.now),
			"collectedShare":  collectedShare(facts.now),
			"collectedAmount": moneyOf(facts.now.CollectedMinor),
		},
		Changes: map[string]boardcatalog.FigureShift{},
	}
	if was := daysToPay(facts.then); was != nil && read.Figures["daysToPay"] != nil {
		shift := boardcatalog.FigureShift{Was: was, WasAt: facts.thenAt}
		if read.Figures["daysToPay"].Number.GreaterThan(was.Number) {
			shift.Tone = boardcatalog.ToneWarn
		}
		read.Changes["daysToPay"] = shift
	}
	if was := collectedShare(facts.then); was != nil && read.Figures["collectedShare"] != nil {
		shift := boardcatalog.FigureShift{Was: was, WasAt: facts.thenAt}
		if read.Figures["collectedShare"].Number.LessThan(was.Number) {
			shift.Tone = boardcatalog.ToneWarn
		}
		read.Changes["collectedShare"] = shift
	}
	read.Changes["collectedAmount"] = boardcatalog.FigureShift{
		Was:   moneyOf(facts.then.CollectedMinor),
		WasAt: facts.thenAt,
	}

	return read
}
