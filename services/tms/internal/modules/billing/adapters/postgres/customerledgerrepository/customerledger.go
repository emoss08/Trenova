package customerledgerrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) customerledgerports.CustomerLedgerProjectionRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-ledger-repository")}
}

func (r *repository) AppendEntries(
	ctx context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rows := make([]*customerledger.CustomerLedgerEntry, 0, len(entries))
		for _, entry := range entries {
			if entry != nil {
				rows = append(rows, entry)
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return customerledgerdb.CustomerLedgerEntries.CreateMany(ctx, r.db.Keel(ctx), rows)
	})
}

func (r *repository) SumBalanceAsOf(
	ctx context.Context,
	req customerledgerports.SumCustomerLedgerBalanceRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		parts, err := keel.Into[int64](ctx, r.db.Keel(ctx), balanceAsOfQuery(req.AsOfDate))
		if err != nil {
			r.l.Error("failed to sum customer ledger balance", zap.Error(err))
			return 0, err
		}

		var total int64
		for _, part := range parts {
			total += part
		}
		return total, nil
	})
}

func balanceAsOfQuery(asOf int64) keel.Compound {
	cut := customerledger.MonthStart(asOf + 1)
	return keel.UnionAll(closedMonthsBeforeQuery(cut), entriesBetweenQuery(cut, asOf))
}

func closedMonthsBeforeQuery(cut int64) keel.Query[*customerledger.MonthlyBalance] {
	mb := customerledgerdb.MonthlyBalances
	return mb.
		Pick(keel.NumericOf(mb.AmountMinor.Sum()).Coalesce(0)).
		Where(mb.MonthStart.Lt(cut))
}

func entriesBetweenQuery(
	from, through int64,
) keel.Query[*customerledger.CustomerLedgerEntry] {
	cle := customerledgerdb.CustomerLedgerEntries
	return cle.
		Pick(keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
		Where(cle.TransactionDate.Gte(from), cle.TransactionDate.Lte(through))
}

type customerBalance struct {
	CustomerID pulid.ID
	Balance    int64
}

func (r *repository) SumCustomerBalances(
	ctx context.Context,
	req customerledgerports.SumCustomerBalancesRequest,
) (map[pulid.ID]int64, error) {
	balances := make(map[pulid.ID]int64, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return balances, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int64, error) {
		totals, err := keel.Into[customerBalance](
			ctx,
			r.db.Keel(ctx),
			customerTotalsQuery(req.CustomerIDs),
		)
		if err != nil {
			r.l.Error("failed to sum customer balances", zap.Error(err))
			return nil, err
		}
		for _, rec := range totals {
			balances[rec.CustomerID] = rec.Balance
		}
		if !req.ExcludeNonRecourseFactored {
			return balances, nil
		}

		sold, err := keel.Into[customerBalance](
			ctx,
			r.db.Keel(ctx),
			nonRecourseFactoredQuery(req.CustomerIDs),
		)
		if err != nil {
			r.l.Error("failed to sum non-recourse factored balances", zap.Error(err))
			return nil, err
		}
		for _, rec := range sold {
			balances[rec.CustomerID] -= rec.Balance
		}
		return balances, nil
	})
}

func customerTotalsQuery(
	customerIDs []pulid.ID,
) keel.Query[*customerledger.MonthlyBalance] {
	mb := customerledgerdb.MonthlyBalances
	return mb.
		Pick(mb.CustomerID, keel.NumericOf(mb.AmountMinor.Sum()).Coalesce(0)).
		Where(mb.CustomerID.In(customerIDs...)).
		Group(mb.CustomerID)
}

func nonRecourseFactoredQuery(
	customerIDs []pulid.ID,
) keel.Query[*customerledger.InvoiceBalance] {
	ib := customerledgerdb.InvoiceBalances
	inv := invoicedb.Invoices.As("inv")
	return ib.
		Pick(ib.CustomerID, keel.NumericOf(ib.AmountMinor.Sum()).Coalesce(0)).
		Where(
			ib.CustomerID.In(customerIDs...),
			ib.AmountMinor.Ne(0),
			keel.Exists(inv.Where(
				inv.ID.EqCol(ib.InvoiceID),
				receivablesql.SoldWithoutRecourse(inv),
			)),
		).
		Group(ib.CustomerID)
}
