package accountingboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
)

const (
	figureBlocking       = "blocking"
	figurePendingEntries = "pendingEntries"
	figurePendingAmount  = "pendingAmount"
	figureOldestPending  = "oldestPending"
)

type PeriodCloseParams struct {
	fx.In

	DB      *postgres.Connection
	Periods fiscalperiodports.CloseBlockerReader
}

type PeriodClose struct {
	db      *postgres.Connection
	periods fiscalperiodports.CloseBlockerReader
}

func NewPeriodClose(p PeriodCloseParams) *PeriodClose {
	return &PeriodClose{db: p.DB, periods: p.Periods}
}

func (*PeriodClose) Metric() string { return accountingboard.MetricPeriodClose }

type periodHead struct {
	Name string
}

type pendingEntries struct {
	Count         int
	Debit         int64
	OldestCreated *int64
}

type periodFacts struct {
	name    string
	pending pendingEntries
}

func (r *PeriodClose) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	if req.For.IsNil() {
		return nil, nil //nolint:nilnil // the port reads a metric with no period as nothing
	}
	result, err := r.periods.GetCloseBlockers(ctx, fiscalperiodports.GetFiscalPeriodByIDRequest{
		ID:         req.For,
		TenantInfo: req.Tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a period outside the reader's scope reads as nothing
		}
		return nil, err
	}
	facts, err := r.readFacts(ctx, req)
	if err != nil || facts == nil {
		return nil, err
	}

	return closeRead(ctx, req.Now, facts, result), nil
}

func (r *PeriodClose) readFacts(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*periodFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*periodFacts, error) {
		db := r.db.Keel(ctx)
		fp := fiscalperioddb.FiscalPeriods
		heads, err := keel.Into[periodHead](ctx, db, fp.Pick(text(fp.Name)).
			Where(fp.ID.Eq(req.For)).
			Limit(1))
		if err != nil {
			return nil, err
		}
		if len(heads) == 0 {
			return nil, nil
		}

		je := journalentrydb.JournalEntries
		pending, err := keel.IntoOne[pendingEntries](ctx, db, je.Pick(
			keel.Count(),
			keel.Int64Of(je.TotalDebit.Sum()).Coalesce(0),
			je.CreatedAt.Min(),
		).Where(
			je.FiscalPeriodID.Eq(req.For),
			je.Status.In(journalentry.StatusPending, journalentry.StatusApproved),
			je.IsPosted.IsFalse(),
		))
		if err != nil {
			return nil, err
		}

		return &periodFacts{name: strings.TrimSpace(heads[0].Name), pending: pending}, nil
	})
}

func closeRead(
	ctx context.Context,
	now int64,
	facts *periodFacts,
	result *fiscalclose.Result,
) *boardcatalog.MetricRead {
	checks := make([]boardcatalog.Check, 0, len(result.Blockers)+len(result.Cleared))
	for _, blocker := range result.Blockers {
		if blocker == nil {
			continue
		}
		checks = append(checks, blockedCheck(ctx, blocker))
	}
	blocked := len(checks)
	for _, cleared := range result.Cleared {
		if check, ok := clearedCheck(ctx, cleared); ok {
			checks = append(checks, check)
		}
	}

	oldest := boardcatalog.Duration(0)
	if facts.pending.OldestCreated != nil {
		oldest = boardcatalog.Duration(max(now-*facts.pending.OldestCreated, 0))
	}
	blocking := boardcatalog.Count(int64(blocked))
	pending := boardcatalog.Count(int64(facts.pending.Count))
	amount := boardcatalog.Money(facts.pending.Debit, "")

	return &boardcatalog.MetricRead{
		Title:  i18n.T(ctx, "What holds {0}", facts.name),
		Source: i18n.T(ctx, "{0} and its close checks", facts.name),
		Figures: map[string]*boardcatalog.Value{
			figureBlocking:       &blocking,
			figurePendingEntries: &pending,
			figurePendingAmount:  &amount,
			figureOldestPending:  &oldest,
		},
		Checks: checks,
	}
}

func blockedCheck(ctx context.Context, blocker *fiscalclose.Blocker) boardcatalog.Check {
	check := boardcatalog.Check{
		Title:  blockedTitle(ctx, blocker.Check),
		Tone:   boardcatalog.ToneBad,
		Detail: blocker.Message,
	}
	if check.Title == "" {
		check.Title, check.Detail = blocker.Message, ""
	}
	if blocker.Count != nil {
		count := boardcatalog.Count(int64(*blocker.Count))
		check.Count = &count
	}

	return check
}

func clearedCheck(ctx context.Context, cleared fiscalclose.Check) (boardcatalog.Check, bool) {
	title := clearedTitle(ctx, cleared)
	if title == "" {
		return boardcatalog.Check{}, false
	}
	check := boardcatalog.Check{Title: title, Tone: boardcatalog.ToneOK}
	if cleared.Counted() {
		zero := boardcatalog.Count(0)
		check.Count = &zero
	}

	return check, true
}

func blockedTitle(ctx context.Context, check fiscalclose.Check) string {
	switch check {
	case fiscalclose.CheckPeriodStatus:
		return i18n.T(ctx, "The period is not open or locked")
	case fiscalclose.CheckEarlierPeriods:
		return i18n.T(ctx, "An earlier period is not closed")
	case fiscalclose.CheckCloseControl:
		return i18n.T(ctx, "Accounting controls stop closing by hand")
	case fiscalclose.CheckManualJournals:
		return i18n.T(ctx, "Manual journal requests waiting")
	case fiscalclose.CheckJournalReversals:
		return i18n.T(ctx, "Journal reversals waiting")
	case fiscalclose.CheckUnpostedSources:
		return i18n.T(ctx, "Accounting sources unposted")
	case fiscalclose.CheckReconciliation:
		return i18n.T(ctx, "Posted invoices that do not reconcile")
	default:
		return ""
	}
}

func clearedTitle(ctx context.Context, check fiscalclose.Check) string {
	switch check {
	case fiscalclose.CheckPeriodStatus:
		return i18n.T(ctx, "The period is open or locked")
	case fiscalclose.CheckEarlierPeriods:
		return i18n.T(ctx, "Earlier periods are closed")
	case fiscalclose.CheckCloseControl:
		return i18n.T(ctx, "Accounting controls allow closing by hand")
	case fiscalclose.CheckManualJournals:
		return i18n.T(ctx, "No manual journal requests waiting")
	case fiscalclose.CheckJournalReversals:
		return i18n.T(ctx, "No journal reversals waiting")
	case fiscalclose.CheckUnpostedSources:
		return i18n.T(ctx, "Every accounting source posted")
	case fiscalclose.CheckReconciliation:
		return i18n.T(ctx, "Posted invoices reconcile")
	default:
		return ""
	}
}
