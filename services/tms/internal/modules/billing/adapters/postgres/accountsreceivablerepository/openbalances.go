package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func NewOpenBalanceReader(p Params) kernelports.CustomerOpenBalanceReader {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-open-balance-reader")}
}

func (r *repository) ListCustomerOpenBalances(
	ctx context.Context,
	req *kernelports.ListCustomerOpenBalancesRequest,
) ([]*kernelports.CustomerOpenBalance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*kernelports.CustomerOpenBalance, error) {
			if len(req.CustomerIDs) == 0 {
				return nil, nil
			}

			inv := invoicedb.Invoices
			open := receivablesql.OpenMinor(inv)
			overdue := keel.DynCase().When(inv.DueDate.Lt(timeutils.NowUnix()), open).Else(0)
			balances, err := keel.Into[kernelports.CustomerOpenBalance](ctx, r.db.Keel(ctx), inv.
				Pick(inv.CustomerID, open.Sum(), overdue.Sum(), inv.CurrencyCode.Min()).
				Where(inv.CustomerID.In(req.CustomerIDs...), receivablesql.Open(inv)).
				Group(inv.CustomerID))
			if err != nil {
				r.l.Error("failed to list customer open balances", zap.Error(err))
				return nil, fmt.Errorf("list customer open balances: %w", err)
			}

			return sliceutils.Pointers(balances), nil
		},
	)
}
