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
		cle := customerledgerdb.CustomerLedgerEntries
		total, err := keel.IntoOne[int64](ctx, r.db.Keel(ctx), cle.
			Pick(keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
			Where(cle.TransactionDate.Lte(req.AsOfDate)))
		if err != nil {
			r.l.Error("failed to sum customer ledger balance", zap.Error(err))
			return 0, err
		}

		return total, nil
	})
}

func nonRecourseFactoredEntry(cle *customerledgerdb.CustomerLedgerEntriesTable) keel.Pred {
	inv := invoicedb.Invoices.As("inv")
	return keel.Exists(inv.Where(
		inv.ID.EqCol(cle.RelatedInvoiceID.Coalesce(
			keel.DynCase().When(cle.SourceObjectType.Eq("Invoice"), cle.SourceObjectID).End(),
		)),
		receivablesql.SoldWithoutRecourse(inv),
	))
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
		cle := customerledgerdb.CustomerLedgerEntries
		rows, err := keel.Into[customerBalance](ctx, r.db.Keel(ctx), cle.
			Pick(cle.CustomerID, keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
			Where(
				cle.CustomerID.In(req.CustomerIDs...),
				keel.When(req.ExcludeNonRecourseFactored, keel.Not(nonRecourseFactoredEntry(cle))),
			).
			Group(cle.CustomerID))
		if err != nil {
			r.l.Error("failed to sum customer balances", zap.Error(err))
			return nil, err
		}
		for _, rec := range rows {
			balances[rec.CustomerID] = rec.Balance
		}
		return balances, nil
	})
}
