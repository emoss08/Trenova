package glbalancerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.GLBalanceRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.gl-balance-repository")}
}

func (r *repository) ListTrialBalanceByPeriod(
	ctx context.Context,
	req repositories.ListTrialBalanceByPeriodRequest,
) ([]*repositories.GLPeriodAccountBalance, error) {
	balances := make([]*repositories.GLPeriodAccountBalance, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("gl_account_balances_by_period AS gb").
		ColumnExpr("gb.organization_id").
		ColumnExpr("gb.business_unit_id").
		ColumnExpr("gb.gl_account_id").
		ColumnExpr("gb.fiscal_year_id").
		ColumnExpr("gb.fiscal_period_id").
		ColumnExpr("gla.account_code").
		ColumnExpr("gla.name AS account_name").
		ColumnExpr("at.category AS account_category").
		ColumnExpr("gb.period_debit_minor").
		ColumnExpr("gb.period_credit_minor").
		ColumnExpr("gb.net_change_minor").
		Join("JOIN gl_accounts AS gla ON gla.id = gb.gl_account_id AND gla.organization_id = gb.organization_id AND gla.business_unit_id = gb.business_unit_id").
		Join("JOIN account_types AS at ON at.id = gla.account_type_id AND at.organization_id = gla.organization_id AND at.business_unit_id = gla.business_unit_id").
		Where("gb.organization_id = ?", req.TenantInfo.OrgID).
		Where("gb.business_unit_id = ?", req.TenantInfo.BuID).
		Where("gb.fiscal_period_id = ?", req.FiscalPeriodID).
		OrderExpr("gla.account_code ASC").
		Scan(ctx, &balances)
	if err != nil {
		return nil, err
	}

	return balances, nil
}

func (r *repository) ListYearToDateBalances(
	ctx context.Context,
	req repositories.ListYearToDateBalancesRequest,
) ([]*repositories.GLPeriodAccountBalance, error) {
	balances := make([]*repositories.GLPeriodAccountBalance, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("gl_account_balances_by_period AS gb").
		ColumnExpr("gb.organization_id").
		ColumnExpr("gb.business_unit_id").
		ColumnExpr("gb.gl_account_id").
		ColumnExpr("gb.fiscal_year_id").
		ColumnExpr("gla.account_code").
		ColumnExpr("gla.name AS account_name").
		ColumnExpr("at.category AS account_category").
		ColumnExpr("SUM(gb.period_debit_minor) AS period_debit_minor").
		ColumnExpr("SUM(gb.period_credit_minor) AS period_credit_minor").
		ColumnExpr("SUM(gb.net_change_minor) AS net_change_minor").
		Join("JOIN gl_accounts AS gla ON gla.id = gb.gl_account_id AND gla.organization_id = gb.organization_id AND gla.business_unit_id = gb.business_unit_id").
		Join("JOIN account_types AS at ON at.id = gla.account_type_id AND at.organization_id = gla.organization_id AND at.business_unit_id = gla.business_unit_id").
		Where("gb.organization_id = ?", req.TenantInfo.OrgID).
		Where("gb.business_unit_id = ?", req.TenantInfo.BuID).
		Where("gb.fiscal_year_id = ?", req.FiscalYearID).
		GroupExpr("gb.organization_id, gb.business_unit_id, gb.gl_account_id, gb.fiscal_year_id, gla.account_code, gla.name, at.category").
		OrderExpr("gla.account_code ASC").
		Scan(ctx, &balances)
	if err != nil {
		return nil, err
	}

	return balances, nil
}

// ListCumulativeBalancesThroughPeriod sums each account across every period of
// the target period's fiscal year up to and including it. Ordering is by period
// number rather than by date so that the year-end adjusting period — which
// shares the final operating period's dates — only counts when the caller asks
// for the adjusting period itself.
func (r *repository) ListCumulativeBalancesThroughPeriod(
	ctx context.Context,
	req repositories.ListCumulativeBalancesThroughPeriodRequest,
) ([]*repositories.GLPeriodAccountBalance, error) {
	balances := make([]*repositories.GLPeriodAccountBalance, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("gl_account_balances_by_period AS gb").
		ColumnExpr("gb.organization_id").
		ColumnExpr("gb.business_unit_id").
		ColumnExpr("gb.gl_account_id").
		ColumnExpr("gb.fiscal_year_id").
		ColumnExpr("? AS fiscal_period_id", req.FiscalPeriodID).
		ColumnExpr("gla.account_code").
		ColumnExpr("gla.name AS account_name").
		ColumnExpr("at.category AS account_category").
		ColumnExpr("SUM(gb.period_debit_minor) AS period_debit_minor").
		ColumnExpr("SUM(gb.period_credit_minor) AS period_credit_minor").
		ColumnExpr("SUM(gb.net_change_minor) AS net_change_minor").
		Join("JOIN fiscal_periods AS fp ON fp.id = gb.fiscal_period_id AND fp.organization_id = gb.organization_id AND fp.business_unit_id = gb.business_unit_id").
		Join("JOIN fiscal_periods AS target ON target.fiscal_year_id = fp.fiscal_year_id AND target.organization_id = fp.organization_id AND target.business_unit_id = fp.business_unit_id").
		Join("JOIN gl_accounts AS gla ON gla.id = gb.gl_account_id AND gla.organization_id = gb.organization_id AND gla.business_unit_id = gb.business_unit_id").
		Join("JOIN account_types AS at ON at.id = gla.account_type_id AND at.organization_id = gla.organization_id AND at.business_unit_id = gla.business_unit_id").
		Where("gb.organization_id = ?", req.TenantInfo.OrgID).
		Where("gb.business_unit_id = ?", req.TenantInfo.BuID).
		Where("target.id = ?", req.FiscalPeriodID).
		Where("fp.period_number <= target.period_number").
		GroupExpr("gb.organization_id, gb.business_unit_id, gb.gl_account_id, gb.fiscal_year_id, gla.account_code, gla.name, at.category").
		OrderExpr("gla.account_code ASC").
		Scan(ctx, &balances)
	if err != nil {
		return nil, err
	}

	return balances, nil
}
