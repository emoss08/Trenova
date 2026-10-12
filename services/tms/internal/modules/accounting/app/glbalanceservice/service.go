package glbalanceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glbalanceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   glbalanceports.GLBalanceRepository
}

type Service struct {
	l    *zap.Logger
	repo glbalanceports.GLBalanceRepository
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.gl-balance"), repo: p.Repo}
}

func (s *Service) ListTrialBalanceByPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) ([]*glbalanceports.GLPeriodAccountBalance, error) {
	return s.repo.ListTrialBalanceByPeriod(ctx, glbalanceports.ListTrialBalanceByPeriodRequest{
		TenantInfo:     tenantInfo,
		FiscalPeriodID: fiscalPeriodID,
	})
}

func (s *Service) GetIncomeStatement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) (*glbalanceports.GLIncomeStatement, error) {
	balances, err := s.ListTrialBalanceByPeriod(ctx, tenantInfo, fiscalPeriodID)
	if err != nil {
		return nil, err
	}

	revenue := newSection("revenue", "Revenue")
	costOfRevenue := newSection("cost_of_revenue", "Cost Of Revenue")
	operatingExpense := newSection("operating_expense", "Operating Expense")

	for _, balance := range balances {
		line := toStatementLine(balance, statementAmount(balance))
		//nolint:exhaustive // only actionable enum states require explicit handling here
		switch balance.AccountCategory {
		case accounttype.CategoryRevenue:
			revenue.Lines = append(revenue.Lines, line)
			revenue.TotalMinor += line.AmountMinor
		case accounttype.CategoryCostOfRevenue:
			costOfRevenue.Lines = append(costOfRevenue.Lines, line)
			costOfRevenue.TotalMinor += line.AmountMinor
		case accounttype.CategoryExpense:
			operatingExpense.Lines = append(operatingExpense.Lines, line)
			operatingExpense.TotalMinor += line.AmountMinor
		}
	}

	grossProfit := revenue.TotalMinor - costOfRevenue.TotalMinor
	netIncome := grossProfit - operatingExpense.TotalMinor

	return &glbalanceports.GLIncomeStatement{
		FiscalPeriodID:   fiscalPeriodID,
		Revenue:          revenue,
		CostOfRevenue:    costOfRevenue,
		OperatingExpense: operatingExpense,
		GrossProfitMinor: grossProfit,
		NetIncomeMinor:   netIncome,
	}, nil
}

// GetBalanceSheet reports the position as at the end of a period, not the
// movement within it: a balance sheet is cumulative by definition. Every period
// of the fiscal year up to and including the requested one is summed, and the
// year's opening entry in period 1 is what carries the prior year in.
func (s *Service) GetBalanceSheet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) (*glbalanceports.GLBalanceSheet, error) {
	balances, err := s.repo.ListCumulativeBalancesThroughPeriod(
		ctx,
		glbalanceports.ListCumulativeBalancesThroughPeriodRequest{
			TenantInfo:     tenantInfo,
			FiscalPeriodID: fiscalPeriodID,
		},
	)
	if err != nil {
		return nil, err
	}

	assets := newSection("assets", "Assets")
	liabilities := newSection("liabilities", "Liabilities")
	equity := newSection("equity", "Equity")
	var currentYearEarnings int64

	for _, balance := range balances {
		line := toStatementLine(balance, statementAmount(balance))
		switch balance.AccountCategory {
		case accounttype.CategoryAsset:
			assets.Lines = append(assets.Lines, line)
			assets.TotalMinor += line.AmountMinor
		case accounttype.CategoryLiability:
			liabilities.Lines = append(liabilities.Lines, line)
			liabilities.TotalMinor += line.AmountMinor
		case accounttype.CategoryEquity:
			equity.Lines = append(equity.Lines, line)
			equity.TotalMinor += line.AmountMinor
		case accounttype.CategoryRevenue:
			currentYearEarnings += line.AmountMinor
		case accounttype.CategoryCostOfRevenue, accounttype.CategoryExpense:
			currentYearEarnings -= line.AmountMinor
		}
	}

	return &glbalanceports.GLBalanceSheet{
		FiscalPeriodID:           fiscalPeriodID,
		Assets:                   assets,
		Liabilities:              liabilities,
		Equity:                   equity,
		CurrentYearEarningsMinor: currentYearEarnings,
		TotalAssetsMinor:         assets.TotalMinor,
		TotalLiabilitiesMinor:    liabilities.TotalMinor,
		TotalEquityMinor:         equity.TotalMinor + currentYearEarnings,
	}, nil
}

func newSection(key, label string) *glbalanceports.GLStatementSection {
	return &glbalanceports.GLStatementSection{
		Key:   key,
		Label: label,
		Lines: make([]*glbalanceports.GLStatementLine, 0),
	}
}

func toStatementLine(
	balance *glbalanceports.GLPeriodAccountBalance,
	amountMinor int64,
) *glbalanceports.GLStatementLine {
	return &glbalanceports.GLStatementLine{
		GLAccountID:     balance.GLAccountID,
		AccountCode:     balance.AccountCode,
		AccountName:     balance.AccountName,
		AccountCategory: balance.AccountCategory,
		AmountMinor:     amountMinor,
	}
}

func statementAmount(balance *glbalanceports.GLPeriodAccountBalance) int64 {
	switch balance.AccountCategory {
	case accounttype.CategoryAsset, accounttype.CategoryExpense, accounttype.CategoryCostOfRevenue:
		return balance.PeriodDebitMinor - balance.PeriodCreditMinor
	case accounttype.CategoryLiability, accounttype.CategoryEquity, accounttype.CategoryRevenue:
		return balance.PeriodCreditMinor - balance.PeriodDebitMinor
	default:
		return balance.NetChangeMinor
	}
}
