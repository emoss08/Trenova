package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

			now := timeutils.NowUnix()
			balances := make([]*kernelports.CustomerOpenBalance, 0, len(req.CustomerIDs))
			err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			inv.customer_id,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS open_minor,
			SUM(CASE WHEN inv.due_date < ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			MIN(inv.currency_code) AS currency_code
		FROM invoices inv
		WHERE inv.organization_id = ?
		  AND inv.business_unit_id = ?
		  AND inv.customer_id IN (?)
		  AND inv.status = 'Posted'
		  AND inv.bill_type IN ('Invoice', 'DebitMemo')
		  AND inv.total_amount_minor > inv.applied_amount_minor
		GROUP BY inv.customer_id
	`, now, req.TenantInfo.OrgID, req.TenantInfo.BuID, bun.List(req.CustomerIDs)).Scan(ctx, &balances)
			if err != nil {
				r.l.Error("failed to list customer open balances", zap.Error(err))
				return nil, fmt.Errorf("list customer open balances: %w", err)
			}

			return balances, nil
		},
	)
}
