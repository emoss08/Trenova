package billingboardreader

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
)

type UnappliedReceipts struct{ db *postgres.Connection }

func NewUnappliedReceipts(p Params) *UnappliedReceipts { return &UnappliedReceipts{db: p.DB} }

func (*UnappliedReceipts) Metric() string { return billingboard.MetricUnappliedReceipts }

type unappliedRow struct {
	Count       int64
	AmountMinor int64
	Oldest      *int64
}

type standingCount struct {
	Standing string
	Count    int64
}

func (r *UnappliedReceipts) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*boardcatalog.MetricRead, error) {
		br := bankreceiptdb.BankReceipts
		c, joined, shown := shownCandidates()
		best := keel.CTE("best", []string{"bank_receipt_id", "standing"}, joined.
			Join(br, br.ID.EqCol(c.BankReceiptID)).
			DistinctOn(c.BankReceiptID).
			Pick(c.BankReceiptID, text(c.Standing)).
			Where(br.Status.Eq(bankreceipt.StatusException), shown).
			Order(c.BankReceiptID.Asc(), c.Rank.Asc()))
		var (
			row      unappliedRow
			standing []standingCount
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(br.
				Pick(
					keel.Count(),
					keel.NumericOf(br.AmountMinor.Sum()).Coalesce(0),
					br.ReceiptDate.Min(),
				).
				Where(br.Status.Eq(bankreceipt.StatusException)), &row),
			keel.IntoSlice(keel.RowsOf(best.TableRef()).With(best).
				Group(best.Col("standing")).
				Pick(best.Col("standing"), keel.Count()), &standing),
		)
		if err != nil {
			return nil, err
		}
		read := unappliedRead(ctx, &row, req.Now, req.Location)
		addStandings(read, row.Count, standing)

		return read, nil
	})
}

func addStandings(read *boardcatalog.MetricRead, total int64, rows []standingCount) {
	counts := map[bankreceipt.Standing]int64{}
	matched := int64(0)
	for _, row := range rows {
		counts[bankreceipt.Standing(row.Standing)] = row.Count
		matched += row.Count
	}
	figure := func(name string, n int64) {
		value := boardcatalog.Count(n)
		read.Figures[name] = &value
	}
	figure(billingboard.FigureReceiptsExact, counts[bankreceipt.StandingExact])
	figure(billingboard.FigureReceiptsSameAmount, counts[bankreceipt.StandingSameAmount])
	figure(billingboard.FigureReceiptsShort, counts[bankreceipt.StandingShort])
	figure(billingboard.FigureReceiptsOver, counts[bankreceipt.StandingOver])
	figure(billingboard.FigureReceiptsWithoutCandidate, max(total-matched, 0))
}

func unappliedRead(
	ctx context.Context,
	row *unappliedRow,
	now int64,
	loc *time.Location,
) *boardcatalog.MetricRead {
	count := boardcatalog.Count(row.Count)
	amount := boardcatalog.Money(row.AmountMinor, "")
	read := &boardcatalog.MetricRead{
		Title: i18n.T(ctx, "Bank receipts not applied"),
		Source: i18n.T(ctx,
			"{0, plural, one {# bank receipt} other {# bank receipts}} in Exception", row.Count),
		Figures: map[string]*boardcatalog.Value{
			billingboard.FigureReceiptsToApply: &count,
			billingboard.FigureAmountToApply:   &amount,
			billingboard.FigureOldestToApply:   nil,
		},
	}
	if row.Oldest != nil && row.Count > 0 {
		waited := boardcatalog.Days(wholeDaysBetween(*row.Oldest, now, loc))
		read.Figures[billingboard.FigureOldestToApply] = &waited
	}

	return read
}

func wholeDaysBetween(from, to int64, loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	day := func(at int64) time.Time {
		moment := time.Unix(at, 0).In(loc)

		return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)
	}

	return max(int64(day(to).Sub(day(from))/(24*time.Hour)), 0)
}
