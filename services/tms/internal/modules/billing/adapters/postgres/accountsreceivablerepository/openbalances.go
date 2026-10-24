package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const customerOpenBalancesSQL = `
		SELECT
			inv.customer_id,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS open_minor,
			SUM(CASE WHEN inv.due_date < $1 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			MIN(inv.currency_code) AS currency_code
		FROM invoices inv
		WHERE inv.organization_id = $2
		  AND inv.business_unit_id = $3
		  AND inv.customer_id = ANY($4::TEXT[])
		  AND ` + receivablesql.OpenInvoice + `
		GROUP BY inv.customer_id`

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

			org, bu := tenantArgs(req.TenantInfo)
			balances, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[kernelports.CustomerOpenBalance],
				customerOpenBalancesSQL,
				timeutils.NowUnix(),
				org,
				bu,
				sliceutils.Strings(req.CustomerIDs),
			)
			if err != nil {
				r.l.Error("failed to list customer open balances", zap.Error(err))
				return nil, fmt.Errorf("list customer open balances: %w", err)
			}

			return balances, nil
		},
	)
}
