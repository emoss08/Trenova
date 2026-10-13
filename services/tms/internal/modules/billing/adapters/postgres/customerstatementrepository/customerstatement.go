package customerstatementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/pkg/buncolgen/customerstatementcol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) customerstatementports.Repository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-statement-repository")}
}

// statementCustomersSQL picks the customers whose balance on the period's
// last day is not zero, or who had any transaction inside the period. A
// customer whose account is settled and quiet has nothing to be told.
const statementCustomersSQL = `
	SELECT cle.customer_id
	FROM customer_ledger_entries AS cle
	WHERE cle.organization_id = ?
		AND cle.business_unit_id = ?
		AND cle.transaction_date <= ?
	GROUP BY cle.customer_id
	HAVING SUM(cle.amount_minor) <> 0
		OR MAX(cle.transaction_date) >= ?
	ORDER BY cle.customer_id`

func (r *repository) ListStatementCustomers(
	ctx context.Context,
	req customerstatementports.ListStatementCustomersRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids := make([]pulid.ID, 0, 64)
		if err := r.db.DBForContext(ctx).
			NewRaw(
				statementCustomersSQL,
				req.TenantInfo.OrgID,
				req.TenantInfo.BuID,
				req.Period.End,
				req.Period.Start,
			).
			Scan(ctx, &ids); err != nil {
			return nil, fmt.Errorf("list statement customers: %w", err)
		}
		return ids, nil
	})
}

func (r *repository) ListMonthlySent(
	ctx context.Context,
	req customerstatementports.ListMonthlySentRequest,
) (map[pulid.ID]bool, error) {
	sent := make(map[pulid.ID]bool, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return sent, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]bool, error) {
		cols := customerstatementcol.StatementColumns
		ids := make([]pulid.ID, 0, len(req.CustomerIDs))
		if err := customerstatementcol.StatementScopeTenant(
			r.db.DBForContext(ctx).
				NewSelect().
				Model((*customerstatement.Statement)(nil)).
				Column(cols.CustomerID.Bare()),
			req.TenantInfo,
		).
			Where(cols.Monthly.IsTrue()).
			Where(cols.PeriodEnd.Eq(), req.PeriodEnd).
			Where(cols.CustomerID.In(), bun.In(req.CustomerIDs)).
			Scan(ctx, &ids); err != nil {
			return nil, fmt.Errorf("list monthly statements sent: %w", err)
		}
		for _, id := range ids {
			sent[id] = true
		}
		return sent, nil
	})
}

func (r *repository) Insert(
	ctx context.Context,
	statement *customerstatement.Statement,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		result, err := r.db.DBForContext(ctx).NewInsert().
			Model(statement).
			On("CONFLICT (organization_id, business_unit_id, customer_id, period_end) WHERE monthly DO NOTHING").
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("insert customer statement: %w", err)
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("insert customer statement: %w", err)
		}
		return inserted > 0, nil
	})
}

const sentStatementsSQL = `
	SELECT
		cst.id,
		cst.period_start,
		cst.period_end,
		cst.ending_balance_minor,
		cst.past_due_minor,
		cst.currency_code,
		cst.sent_to,
		cst.monthly,
		COALESCE(usr.name, '') AS sent_by_name,
		cst.sent_at
	FROM customer_statements AS cst
	LEFT JOIN users AS usr ON usr.id = cst.sent_by_id
	WHERE cst.organization_id = ?
		AND cst.business_unit_id = ?
		AND cst.customer_id = ?
	ORDER BY cst.sent_at DESC, cst.id DESC
	LIMIT ?`

func (r *repository) ListSent(
	ctx context.Context,
	req customerstatementports.ListSentRequest,
) ([]*customerstatementports.SentStatement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customerstatementports.SentStatement, error) {
		rows := make([]*customerstatementports.SentStatement, 0, 12)
		if err := r.db.DBForContext(ctx).
			NewRaw(
				sentStatementsSQL,
				req.TenantInfo.OrgID,
				req.TenantInfo.BuID,
				req.CustomerID,
				customerstatementports.MaxHistory,
			).
			Scan(ctx, &rows); err != nil {
			return nil, fmt.Errorf("list sent customer statements: %w", err)
		}
		return rows, nil
	})
}
