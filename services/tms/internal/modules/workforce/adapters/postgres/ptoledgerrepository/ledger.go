package ptoledgerrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto/ptodb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var balanceErrors = dberror.Mapping{Entity: "WorkerPTOBalance"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.PTOLedgerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pto-ledger-repository"),
	}
}

func sumOrZero(e keel.Expr) keel.NumericExpr {
	return keel.NumericOf(e).Coalesce(0)
}

func balanceQuery(key *workerports.PTOBalanceKey) *keel.Query[*pto.WorkerPTOBalance] {
	b := ptodb.WorkerPTOBalances

	return b.Where(b.WorkerID.Eq(key.WorkerID), b.PTOType.Eq(key.PTOType))
}

func (r *repository) EnsureBalance(ctx context.Context, key *workerports.PTOBalanceKey) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		b := ptodb.WorkerPTOBalances
		_, err := b.CreateOrSkip(ctx, r.db.Keel(ctx), []*pto.WorkerPTOBalance{{
			OrganizationID: key.TenantInfo.OrgID,
			BusinessUnitID: key.TenantInfo.BuID,
			WorkerID:       key.WorkerID,
			PTOType:        key.PTOType,
		}}, b.OrganizationID, b.BusinessUnitID, b.WorkerID, b.PTOType)
		if err != nil {
			r.l.Error("failed to ensure PTO balance row", zap.Error(err))
			return balanceErrors.Map(err, "")
		}

		return nil
	})
}

func (r *repository) LockBalance(
	ctx context.Context,
	key *workerports.PTOBalanceKey,
) (*pto.WorkerPTOBalance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pto.WorkerPTOBalance, error) {
		entity, err := balanceQuery(key).Lock(keel.ForUpdate).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, balanceErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) GetBalance(
	ctx context.Context,
	key *workerports.PTOBalanceKey,
) (*pto.WorkerPTOBalance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pto.WorkerPTOBalance, error) {
		entity, err := balanceQuery(key).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, balanceErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) ListBalances(
	ctx context.Context,
	req *workerports.ListPTOBalancesRequest,
) ([]*pto.WorkerPTOBalance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*pto.WorkerPTOBalance, error) {
		b := ptodb.WorkerPTOBalances
		entities, err := b.Where(b.WorkerID.Eq(req.WorkerID)).
			Order(b.PTOType.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list PTO balances", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListOrgBalances(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*pto.WorkerPTOBalance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*pto.WorkerPTOBalance, error) {
		b := ptodb.WorkerPTOBalances
		entities, err := b.Query().
			Load(b.Worker()).
			Order(b.WorkerID.Asc(), b.PTOType.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list organisation PTO balances", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) UpdateBalance(
	ctx context.Context,
	entity *pto.WorkerPTOBalance,
) (*pto.WorkerPTOBalance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pto.WorkerPTOBalance, error) {
		if err := ptodb.WorkerPTOBalances.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update PTO balance", zap.Error(err))
			return nil, balanceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) InsertEntry(
	ctx context.Context,
	entry *pto.WorkerPTOLedgerEntry,
) (*pto.WorkerPTOLedgerEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pto.WorkerPTOLedgerEntry, error) {
		if err := ptodb.WorkerPTOLedgerEntries.Create(ctx, r.db.Keel(ctx), entry); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, workerports.ErrDuplicatePTOLedgerEntry
			}
			r.l.Error("failed to insert PTO ledger entry", zap.Error(err))
			return nil, err
		}

		return entry, nil
	})
}

func entryFilter(req *workerports.ListPTOLedgerRequest) []keel.Pred {
	e := ptodb.WorkerPTOLedgerEntries

	return []keel.Pred{
		keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
		keel.When(req.PTOType != "", e.PTOType.Eq(worker.PTOType(req.PTOType))),
		keel.When(
			req.EntryType != "",
			e.EntryType.Eq(pto.PTOLedgerEntryType(req.EntryType)),
		),
		keel.When(req.EffectiveFrom > 0, e.EffectiveAt.Gte(req.EffectiveFrom)),
		keel.When(req.EffectiveTo > 0, e.EffectiveAt.Lte(req.EffectiveTo)),
	}
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *workerports.ListPTOLedgerRequest,
) (*pagination.CursorListResult[*pto.WorkerPTOLedgerEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*pto.WorkerPTOLedgerEntry], error) {
			e := ptodb.WorkerPTOLedgerEntries
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), e.Where(entryFilter(req)...),
				&listquery.Request[pto.WorkerPTOLedgerEntry]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*pto.WorkerPTOLedgerEntry],
					) *keel.Query[*pto.WorkerPTOLedgerEntry] {
						return q.Load(listquery.When(req.IncludeWorker, e.Worker())...)
					},
				})
			if err != nil {
				r.l.Error("failed to list PTO ledger entries", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

type sumRow struct {
	Total decimal.Decimal
	Count int64
}

func (r *repository) SumEntries(
	ctx context.Context,
	key *workerports.PTOBalanceKey,
) (decimal.Decimal, int64, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (decimal.Decimal, int64, error) {
		e := ptodb.WorkerPTOLedgerEntries
		row, err := keel.IntoOne[sumRow](ctx, r.db.Keel(ctx), e.
			Pick(sumOrZero(e.AmountDays.Sum()), keel.Count()).
			Where(e.WorkerID.Eq(key.WorkerID), e.PTOType.Eq(key.PTOType)))
		if err != nil {
			r.l.Error("failed to sum PTO ledger entries", zap.Error(err))
			return decimal.Zero, 0, err
		}

		return row.Total, row.Count, nil
	})
}

func (r *repository) PendingDays(
	ctx context.Context,
	req *workerports.PendingPTODaysRequest,
) (decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (decimal.Decimal, error) {
		p := workerdb.WorkerPTOs
		total, err := keel.IntoOne[decimal.Decimal](ctx, r.db.Keel(ctx), p.
			Pick(sumOrZero(p.Days.Sum())).
			Where(
				p.WorkerID.Eq(req.WorkerID),
				p.Type.Eq(req.PTOType),
				p.Status.Eq(worker.PTOStatusRequested),
				keel.When(req.ExcludeID.IsNotNil(), p.ID.Ne(req.ExcludeID)),
			))
		if err != nil {
			r.l.Error("failed to sum pending PTO days", zap.Error(err))
			return decimal.Zero, err
		}

		return total, nil
	})
}

func (r *repository) HasEntry(
	ctx context.Context,
	req *workerports.HasPTOLedgerEntryRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		e := ptodb.WorkerPTOLedgerEntries
		exists, err := e.Where(e.SourcePTOID.Eq(req.SourcePTOID), e.EntryType.Eq(req.EntryType)).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check PTO ledger entry", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

type summaryRow struct {
	WorkersTracked int64
	TotalBalance   decimal.Decimal
	AccruedYTD     decimal.Decimal
	UsedYTD        decimal.Decimal
}

func (r *repository) Summary(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*workerports.PTOBalanceSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerports.PTOBalanceSummary, error) {
		db := r.db.Keel(ctx)
		b := ptodb.WorkerPTOBalances
		balances, err := keel.IntoOne[summaryRow](ctx, db, b.Pick(
			b.WorkerID.CountDistinct(),
			sumOrZero(b.BalanceDays.Sum()),
			sumOrZero(b.AccruedYTDDays.Sum()),
			sumOrZero(b.UsedYTDDays.Sum()),
		))
		if err != nil {
			r.l.Error("failed to summarise PTO balances", zap.Error(err))
			return nil, err
		}

		p := workerdb.WorkerPTOs
		pending, err := keel.IntoOne[decimal.Decimal](ctx, db, p.
			Pick(sumOrZero(p.Days.Sum())).
			Where(p.Status.Eq(worker.PTOStatusRequested)))
		if err != nil {
			r.l.Error("failed to summarise pending PTO", zap.Error(err))
			return nil, err
		}

		w := workerdb.Workers
		a := ptodb.WorkerPTOPolicyAssignments
		unassigned, err := w.Where(
			w.Status.Eq(domaintypes.StatusActive),
			keel.NotExists(a.Where(a.WorkerID.EqCol(w.ID), a.EffectiveTo.IsNull())),
		).Count(ctx, db)
		if err != nil {
			r.l.Error("failed to count unassigned workers", zap.Error(err))
			return nil, err
		}

		return &workerports.PTOBalanceSummary{
			WorkersTracked:    int(balances.WorkersTracked),
			WorkersUnassigned: int(unassigned),
			TotalBalanceDays:  balances.TotalBalance,
			TotalPendingDays:  pending,
			AccruedYTDDays:    balances.AccruedYTD,
			UsedYTDDays:       balances.UsedYTD,
		}, nil
	})
}
