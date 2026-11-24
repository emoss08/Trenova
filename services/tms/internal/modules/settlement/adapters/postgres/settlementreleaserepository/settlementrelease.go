package settlementreleaserepository

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const secondsPerDay = 24 * 60 * 60

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) settlementreleaseports.Reader {
	return &repository{db: p.DB}
}

type batchRow struct {
	ID                  pulid.ID
	Name                string
	PeriodStart         int64
	PeriodEnd           int64
	PayDate             int64
	Currency            string
	GrossMinor          int64
	ReimbursementsMinor int64
	NetMinor            int64
}

type splitRow struct {
	Settlements         int64
	GrossMinor          int64
	ReimbursementsMinor int64
	NetMinor            int64
	Currencies          int64
	Currency            string
	Released            int64
	ReleasedNet         int64
	Ready               int64
	ReadyNet            int64
	Held                int64
	HeldNet             int64
	Critical            int64
	CriticalNet         int64
	Disputed            int64
	DisputedNet         int64
}

type settlementRow struct {
	ID                   pulid.ID
	WorkerID             pulid.ID
	BatchID              *pulid.ID
	BatchName            string
	SettlementNumber     string
	Driver               string
	Status               string
	PayDate              int64
	NetPayMinor          int64
	GrossEarningsMinor   int64
	DeductionsMinor      int64
	CarryForwardOutMinor int64
	Currency             string
	Exceptions           []driversettlement.Exception
	HoldReason           string
	HoldNote             string
	DisputeCategory      string
	Block                string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func sum(col keel.Int64Col, filter keel.Pred) keel.Int64Expr {
	return keel.Int64Of(col.Sum().Filter(filter)).Coalesce(0)
}

type release struct {
	held       keel.Pred
	critical   keel.Pred
	disputed   keel.Pred
	unreleased keel.Pred
}

func releaseOf() release {
	s := driversettlementdb.Settlements
	d := driversettlementdb.Disputes

	return release{
		held: s.HoldReason.IsNotNull(),
		critical: keel.Raw(
			`coalesce(? @> '[{"severity":"Critical"}]'::jsonb, false)`, s.Exceptions,
		),
		disputed: keel.Exists(d.Where(
			d.SettlementID.EqCol(s.ID),
			d.Status.In(driversettlement.OpenDisputeStatuses()...),
		)),
		unreleased: s.Status.In(driversettlement.UnreleasedStatuses()...),
	}
}

func (r release) block() keel.StringExpr {
	return keel.CaseString().
		When(keel.And(r.unreleased, r.held),
			keel.Text(driversettlement.ReleaseBlockHeld.String())).
		When(keel.And(r.unreleased, r.critical),
			keel.Text(driversettlement.ReleaseBlockCriticalException.String())).
		When(keel.And(r.unreleased, r.disputed),
			keel.Text(driversettlement.ReleaseBlockDisputeOpen.String())).
		Else(keel.Text(""))
}

func (r release) rank() keel.Int64Expr {
	return keel.CaseInt64().
		When(keel.And(r.unreleased, r.held), keel.Int64(0)).
		When(keel.And(r.unreleased, r.critical), keel.Int64(1)).
		Else(keel.Int64(2))
}

func (r release) blocked() keel.Pred {
	return keel.And(r.unreleased, keel.Or(r.held, r.critical, r.disputed))
}

func (r *repository) recentBatches(ctx context.Context, limit int) ([]batchRow, error) {
	b := driversettlementdb.SettlementBatches

	return keel.Into[batchRow](ctx, r.db.Keel(ctx), b.
		Pick(
			b.ID, b.Name, b.PeriodStart, b.PeriodEnd, b.PayDate, b.CurrencyCode,
			b.TotalGrossMinor, b.TotalReimbursementsMinor, b.TotalNetMinor,
		).
		Where(b.Status.Ne(driversettlement.BatchStatusCanceled)).
		Order(b.PeriodStart.Desc(), b.Name.Asc()).
		Limit(limit))
}

func (r *repository) batches(
	ctx context.Context,
	batchID pulid.ID,
) ([]batchRow, error) {
	if !batchID.IsNil() {
		b := driversettlementdb.SettlementBatches

		return keel.Into[batchRow](ctx, r.db.Keel(ctx), b.
			Pick(
				b.ID, b.Name, b.PeriodStart, b.PeriodEnd, b.PayDate, b.CurrencyCode,
				b.TotalGrossMinor, b.TotalReimbursementsMinor, b.TotalNetMinor,
			).
			Where(b.Status.Ne(driversettlement.BatchStatusCanceled), b.ID.Eq(batchID)).
			Limit(1))
	}
	rows, err := r.recentBatches(ctx, settlementreleaseports.MaxBatches)
	if err != nil {
		return nil, err
	}
	periods := periodsOf(rows)
	if len(periods) == 0 {
		return nil, nil
	}

	return periods[0], nil
}

func periodsOf(rows []batchRow) [][]batchRow {
	periods := make([][]batchRow, 0, len(rows))
	for idx := range rows {
		last := len(periods) - 1
		if last >= 0 && periods[last][0].PeriodStart-rows[idx].PeriodStart < secondsPerDay {
			periods[last] = append(periods[last], rows[idx])

			continue
		}
		periods = append(periods, []batchRow{rows[idx]})
	}

	return periods
}

func batchIDs(rows []batchRow) []pulid.ID {
	ids := make([]pulid.ID, 0, len(rows))
	for idx := range rows {
		ids = append(ids, rows[idx].ID)
	}

	return ids
}

func (r *repository) Batch(
	ctx context.Context,
	req *settlementreleaseports.BatchRequest,
) (*settlementreleaseports.Batch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*settlementreleaseports.Batch, error) {
		rows, err := r.batches(ctx, req.BatchID)
		if err != nil || len(rows) == 0 {
			return nil, err
		}
		split, err := r.split(ctx, batchIDs(rows))
		if err != nil {
			return nil, err
		}

		return batchOf(rows, &split), nil
	})
}

func (r *repository) split(ctx context.Context, ids []pulid.ID) (splitRow, error) {
	s := driversettlementdb.Settlements
	rel := releaseOf()
	live := s.Status.Ne(driversettlement.StatusVoided)
	released := s.Status.In(driversettlement.ReleasedStatuses()...)
	held := keel.And(rel.unreleased, rel.held)
	critical := keel.And(rel.unreleased, keel.Not(rel.held), rel.critical)
	disputed := keel.And(
		rel.unreleased, keel.Not(rel.held), keel.Not(rel.critical), rel.disputed,
	)
	ready := keel.And(
		rel.unreleased, keel.Not(rel.held), keel.Not(rel.critical), keel.Not(rel.disputed),
	)

	return keel.IntoOne[splitRow](ctx, r.db.Keel(ctx), s.
		Pick(
			keel.Count().Filter(live),
			sum(s.GrossEarningsMinor, live),
			sum(s.ReimbursementsMinor, live),
			sum(s.NetPayMinor, live),
			keel.Int64Of(keel.Raw("count(DISTINCT ?) FILTER (WHERE ?)", s.CurrencyCode, live)),
			text(keel.Raw("max(?)", s.CurrencyCode)),
			keel.Count().Filter(released),
			sum(s.NetPayMinor, released),
			keel.Count().Filter(ready),
			sum(s.NetPayMinor, ready),
			keel.Count().Filter(held),
			sum(s.NetPayMinor, held),
			keel.Count().Filter(critical),
			sum(s.NetPayMinor, critical),
			keel.Count().Filter(disputed),
			sum(s.NetPayMinor, disputed),
		).
		Where(s.BatchID.In(ids...)))
}

func batchOf(rows []batchRow, split *splitRow) *settlementreleaseports.Batch {
	first := rows[0]
	names := make([]string, 0, len(rows))
	batch := &settlementreleaseports.Batch{
		Batches:             len(rows),
		PeriodStart:         first.PeriodStart,
		PeriodEnd:           first.PeriodEnd,
		PayDate:             first.PayDate,
		Currency:            split.Currency,
		Currencies:          split.Currencies,
		Settlements:         split.Settlements,
		GrossMinor:          split.GrossMinor,
		ReimbursementsMinor: split.ReimbursementsMinor,
		NetMinor:            split.NetMinor,
		Released: settlementreleaseports.Tally{
			Count:    split.Released,
			NetMinor: split.ReleasedNet,
		},
		Ready: settlementreleaseports.Tally{
			Count:    split.Ready,
			NetMinor: split.ReadyNet,
		},
		Blocked: map[driversettlement.ReleaseBlock]settlementreleaseports.Tally{
			driversettlement.ReleaseBlockHeld: {
				Count: split.Held, NetMinor: split.HeldNet,
			},
			driversettlement.ReleaseBlockCriticalException: {
				Count: split.Critical, NetMinor: split.CriticalNet,
			},
			driversettlement.ReleaseBlockDisputeOpen: {
				Count: split.Disputed, NetMinor: split.DisputedNet,
			},
		},
	}
	for idx := range rows {
		if name := strings.TrimSpace(rows[idx].Name); !slices.Contains(names, name) {
			names = append(names, name)
		}
		batch.PeriodEnd = max(batch.PeriodEnd, rows[idx].PeriodEnd)
		batch.PayDate = max(batch.PayDate, rows[idx].PayDate)
	}
	batch.Name = strings.Join(names, ", ")

	return batch
}

func (r *repository) Weeks(
	ctx context.Context,
	req *settlementreleaseports.WeeksRequest,
) ([]*settlementreleaseports.Week, error) {
	weeks := min(max(req.Weeks, 1), settlementreleaseports.MaxWeeks)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*settlementreleaseports.Week, error) {
		rows, err := r.recentBatches(ctx, weeks*settlementreleaseports.BatchesPerPeriod)
		if err != nil {
			return nil, err
		}
		periods := periodsOf(rows)
		if len(periods) > weeks {
			periods = periods[:weeks]
		}
		out := make([]*settlementreleaseports.Week, 0, len(periods))
		for idx := len(periods) - 1; idx >= 0; idx-- {
			out = append(out, weekOf(periods[idx]))
		}

		return out, nil
	})
}

func weekOf(rows []batchRow) *settlementreleaseports.Week {
	week := &settlementreleaseports.Week{
		PeriodStart: rows[0].PeriodStart,
		PeriodEnd:   rows[0].PeriodEnd,
		PayDate:     rows[0].PayDate,
		Currency:    rows[0].Currency,
	}
	currencies := map[string]struct{}{}
	for idx := range rows {
		row := &rows[idx]
		week.Batches++
		week.PeriodStart = min(week.PeriodStart, row.PeriodStart)
		week.PeriodEnd = max(week.PeriodEnd, row.PeriodEnd)
		week.PayDate = max(week.PayDate, row.PayDate)
		week.GrossMinor += row.GrossMinor
		week.ReimbursementsMinor += row.ReimbursementsMinor
		week.NetMinor += row.NetMinor
		currencies[row.Currency] = struct{}{}
	}
	week.Currencies = int64(len(currencies))

	return week
}

func (r *repository) settlementQuery(disputes bool) keel.Query[*driversettlement.Settlement] {
	s := driversettlementdb.Settlements
	w := workerdb.Workers
	b := driversettlementdb.SettlementBatches
	d := driversettlementdb.Disputes
	rel := releaseOf()
	category := keel.Text("")
	block := keel.CaseString().
		When(keel.And(rel.unreleased, rel.held),
			keel.Text(driversettlement.ReleaseBlockHeld.String())).
		When(keel.And(rel.unreleased, rel.critical),
			keel.Text(driversettlement.ReleaseBlockCriticalException.String())).
		Else(keel.Text(""))
	if disputes {
		category = text(keel.Scalar(d.Pick(d.Category).
			Where(d.SettlementID.EqCol(s.ID),
				d.Status.In(driversettlement.OpenDisputeStatuses()...)).
			Order(d.CreatedAt.Asc()).
			Limit(1)))
		block = rel.block()
	}

	return s.Join(w, w.ID.EqCol(s.WorkerID)).
		LeftJoin(b, b.ID.EqCol(s.BatchID)).
		Pick(
			s.ID, s.WorkerID, s.BatchID, text(b.Name), s.SettlementNumber,
			text(keel.ConcatWS(" ", w.FirstName, w.LastName)), text(s.Status), s.PayDate,
			s.NetPayMinor, s.GrossEarningsMinor, s.DeductionsMinor, s.CarryForwardOutMinor,
			s.CurrencyCode, s.Exceptions, text(s.HoldReason), text(s.HoldNote), category, block,
		)
}

func (r *repository) Settlements(
	ctx context.Context,
	req *settlementreleaseports.SettlementsRequest,
) ([]*settlementreleaseports.Settlement, error) {
	if len(req.IDs) == 0 {
		return nil, nil
	}
	ids := req.IDs[:min(len(req.IDs), settlementreleaseports.MaxSettlements)]

	return dbtx.Read(ctx, r.db,
		func(ctx context.Context) ([]*settlementreleaseports.Settlement, error) {
			s := driversettlementdb.Settlements
			rows, err := keel.Into[settlementRow](ctx, r.db.Keel(ctx),
				r.settlementQuery(req.Disputes).Where(s.ID.In(ids...)))
			if err != nil {
				return nil, err
			}

			return settlementsOf(rows), nil
		})
}

func (r *repository) Blocked(
	ctx context.Context,
	req *settlementreleaseports.BlockedRequest,
) ([]*settlementreleaseports.Settlement, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = settlementreleaseports.DefaultBlocked
	}
	limit = min(limit, settlementreleaseports.MaxBlocked)

	return dbtx.Read(ctx, r.db,
		func(ctx context.Context) ([]*settlementreleaseports.Settlement, error) {
			batches, err := r.batches(ctx, req.BatchID)
			if err != nil || len(batches) == 0 {
				return nil, err
			}
			s := driversettlementdb.Settlements
			rel := releaseOf()
			rows, err := keel.Into[settlementRow](ctx, r.db.Keel(ctx),
				r.settlementQuery(true).
					Where(s.BatchID.In(batchIDs(batches)...), rel.blocked()).
					Order(keel.Asc(rel.rank()), s.NetPayMinor.Desc(), s.SettlementNumber.Asc()).
					Limit(limit))
			if err != nil {
				return nil, err
			}

			return settlementsOf(rows), nil
		})
}

func settlementsOf(rows []settlementRow) []*settlementreleaseports.Settlement {
	out := make([]*settlementreleaseports.Settlement, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		out = append(out, &settlementreleaseports.Settlement{
			ID:                   row.ID,
			WorkerID:             row.WorkerID,
			BatchID:              row.BatchID,
			BatchName:            strings.TrimSpace(row.BatchName),
			SettlementNumber:     row.SettlementNumber,
			Driver:               strings.TrimSpace(row.Driver),
			Status:               driversettlement.Status(row.Status),
			PayDate:              row.PayDate,
			NetPayMinor:          row.NetPayMinor,
			GrossEarningsMinor:   row.GrossEarningsMinor,
			DeductionsMinor:      row.DeductionsMinor,
			CarryForwardOutMinor: row.CarryForwardOutMinor,
			Currency:             row.Currency,
			Exceptions:           row.Exceptions,
			HoldReason:           driversettlement.HoldReason(row.HoldReason),
			HoldNote:             strings.TrimSpace(row.HoldNote),
			DisputeCategory:      driversettlement.DisputeCategory(row.DisputeCategory),
			Block:                driversettlement.ReleaseBlock(row.Block),
		})
	}

	return out
}
