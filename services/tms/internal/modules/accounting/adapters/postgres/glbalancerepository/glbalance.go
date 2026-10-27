package glbalancerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glbalanceports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
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

func New(p Params) glbalanceports.GLBalanceRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.gl-balance-repository")}
}

type yearBalance struct {
	OrganizationID    pulid.ID
	BusinessUnitID    pulid.ID
	GLAccountID       pulid.ID
	FiscalYearID      pulid.ID
	AccountCode       string
	AccountName       string
	AccountCategory   accounttype.Category
	PeriodDebitMinor  int64
	PeriodCreditMinor int64
	NetChangeMinor    int64
}

func (b *yearBalance) balance() *glbalanceports.GLPeriodAccountBalance {
	return &glbalanceports.GLPeriodAccountBalance{
		OrganizationID:    b.OrganizationID,
		BusinessUnitID:    b.BusinessUnitID,
		GLAccountID:       b.GLAccountID,
		FiscalYearID:      b.FiscalYearID,
		AccountCode:       b.AccountCode,
		AccountName:       b.AccountName,
		AccountCategory:   b.AccountCategory,
		PeriodDebitMinor:  b.PeriodDebitMinor,
		PeriodCreditMinor: b.PeriodCreditMinor,
		NetChangeMinor:    b.NetChangeMinor,
	}
}

func balancesWithAccounts() *keel.Query[*glbalance.GLAccountPeriodBalance] {
	gb := glbalancedb.GLAccountPeriodBalances
	gla := glaccountdb.GLAccounts
	at := accounttypedb.AccountTypes

	return gb.Join(gla, gla.ID.EqCol(gb.GLAccountID)).
		Join(at, at.ID.EqCol(gla.AccountTypeID)).
		Order(gla.AccountCode.Asc())
}

func (r *repository) ListTrialBalanceByPeriod(
	ctx context.Context,
	req glbalanceports.ListTrialBalanceByPeriodRequest,
) ([]*glbalanceports.GLPeriodAccountBalance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*glbalanceports.GLPeriodAccountBalance, error) {
			gb := glbalancedb.GLAccountPeriodBalances
			gla := glaccountdb.GLAccounts
			at := accounttypedb.AccountTypes

			rows, err := keel.Into[glbalanceports.GLPeriodAccountBalance](
				ctx,
				r.db.Keel(ctx),
				balancesWithAccounts().
					Pick(
						gb.OrganizationID,
						gb.BusinessUnitID,
						gb.GLAccountID,
						gb.FiscalYearID,
						gb.FiscalPeriodID,
						gla.AccountCode,
						gla.Name,
						at.Category,
						gb.PeriodDebitMinor,
						gb.PeriodCreditMinor,
						gb.NetChangeMinor,
					).
					Where(gb.FiscalPeriodID.Eq(req.FiscalPeriodID)),
			)
			if err != nil {
				return nil, err
			}

			return sliceutils.Pointers(rows), nil
		},
	)
}

func (r *repository) ListYearToDateBalances(
	ctx context.Context,
	req glbalanceports.ListYearToDateBalancesRequest,
) ([]*glbalanceports.GLPeriodAccountBalance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*glbalanceports.GLPeriodAccountBalance, error) {
			gb := glbalancedb.GLAccountPeriodBalances
			gla := glaccountdb.GLAccounts
			at := accounttypedb.AccountTypes

			rows, err := keel.Into[yearBalance](
				ctx,
				r.db.Keel(ctx),
				balancesWithAccounts().
					Pick(
						gb.OrganizationID,
						gb.BusinessUnitID,
						gb.GLAccountID,
						gb.FiscalYearID,
						gla.AccountCode,
						gla.Name,
						at.Category,
						gb.PeriodDebitMinor.Sum(),
						gb.PeriodCreditMinor.Sum(),
						gb.NetChangeMinor.Sum(),
					).
					Where(gb.FiscalYearID.Eq(req.FiscalYearID)).
					Group(
						gb.OrganizationID,
						gb.BusinessUnitID,
						gb.GLAccountID,
						gb.FiscalYearID,
						gla.AccountCode,
						gla.Name,
						at.Category,
					),
			)
			if err != nil {
				return nil, err
			}

			balances := make([]*glbalanceports.GLPeriodAccountBalance, 0, len(rows))
			for i := range rows {
				balances = append(balances, rows[i].balance())
			}

			return balances, nil
		},
	)
}

func (r *repository) ListCumulativeBalancesThroughPeriod(
	ctx context.Context,
	req glbalanceports.ListCumulativeBalancesThroughPeriodRequest,
) ([]*glbalanceports.GLPeriodAccountBalance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*glbalanceports.GLPeriodAccountBalance, error) {
			gb := glbalancedb.GLAccountPeriodBalances
			gla := glaccountdb.GLAccounts
			at := accounttypedb.AccountTypes
			fp := fiscalperioddb.FiscalPeriods
			target := fp.As("target")

			rows, err := keel.Into[glbalanceports.GLPeriodAccountBalance](
				ctx,
				r.db.Keel(ctx),
				balancesWithAccounts().
					Join(fp, fp.ID.EqCol(gb.FiscalPeriodID)).
					Join(target, target.FiscalYearID.EqCol(fp.FiscalYearID)).
					Pick(
						gb.OrganizationID,
						gb.BusinessUnitID,
						gb.GLAccountID,
						gb.FiscalYearID,
						target.ID,
						gla.AccountCode,
						gla.Name,
						at.Category,
						gb.PeriodDebitMinor.Sum(),
						gb.PeriodCreditMinor.Sum(),
						gb.NetChangeMinor.Sum(),
					).
					Where(
						target.ID.Eq(req.FiscalPeriodID),
						keel.Int64Of(fp.PeriodNumber).LteExpr(target.PeriodNumber),
					).
					Group(
						gb.OrganizationID,
						gb.BusinessUnitID,
						gb.GLAccountID,
						gb.FiscalYearID,
						target.ID,
						gla.AccountCode,
						gla.Name,
						at.Category,
					),
			)
			if err != nil {
				return nil, err
			}

			return sliceutils.Pointers(rows), nil
		},
	)
}
