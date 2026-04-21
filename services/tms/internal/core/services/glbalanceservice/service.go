package glbalanceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	repositoryports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositoryports.GLBalanceRepository
}

type Service struct {
	l    *zap.Logger
	repo repositoryports.GLBalanceRepository
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.gl-balance"), repo: p.Repo}
}

func (s *Service) ListTrialBalanceByPeriod(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) ([]*repositoryports.GLPeriodAccountBalance, error) {
	return s.repo.ListTrialBalanceByPeriod(ctx, repositoryports.ListTrialBalanceByPeriodRequest{
		TenantInfo:     tenantInfo,
		FiscalPeriodID: fiscalPeriodID,
	})
}

func (s *Service) GetIncomeStatement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) (*serviceports.GLIncomeStatement, error) {
	balances, err := s.ListTrialBalanceByPeriod(ctx, tenantInfo, fiscalPeriodID)
	if err != nil {
		return nil, err
	}

	revenue := newSection("revenue", "Revenue")
	costOfRevenue := newSection("cost_of_revenue", "Cost Of Revenue")
	operatingExpense := newSection("operating_expense", "Operating Expense")

	for _, balance := range balances {
		line := toStatementLine(balance, statementAmount(balance))
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

	return &serviceports.GLIncomeStatement{
		FiscalPeriodID:   fiscalPeriodID,
		Revenue:          revenue,
		CostOfRevenue:    costOfRevenue,
		OperatingExpense: operatingExpense,
		GrossProfitMinor: grossProfit,
		NetIncomeMinor:   netIncome,
	}, nil
}

func (s *Service) GetBalanceSheet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fiscalPeriodID pulid.ID,
) (*serviceports.GLBalanceSheet, error) {
	balances, err := s.ListTrialBalanceByPeriod(ctx, tenantInfo, fiscalPeriodID)
	if err != nil {
		return nil, err
	}

	assets := newSection("assets", "Assets")
	liabilities := newSection("liabilities", "Liabilities")
	equity := newSection("equity", "Equity")
	var currentPeriodNetIncome int64

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
			currentPeriodNetIncome += line.AmountMinor
		case accounttype.CategoryCostOfRevenue, accounttype.CategoryExpense:
			currentPeriodNetIncome -= line.AmountMinor
		}
	}

	return &serviceports.GLBalanceSheet{
		FiscalPeriodID:              fiscalPeriodID,
		Assets:                      assets,
		Liabilities:                 liabilities,
		Equity:                      equity,
		CurrentPeriodNetIncomeMinor: currentPeriodNetIncome,
		TotalAssetsMinor:            assets.TotalMinor,
		TotalLiabilitiesMinor:       liabilities.TotalMinor,
		TotalEquityMinor:            equity.TotalMinor + currentPeriodNetIncome,
	}, nil
}

func newSection(key, label string) *serviceports.GLStatementSection {
	return &serviceports.GLStatementSection{
		Key:   key,
		Label: label,
		Lines: make([]*serviceports.GLStatementLine, 0),
	}
}

func toStatementLine(
	balance *repositoryports.GLPeriodAccountBalance,
	amountMinor int64,
) *serviceports.GLStatementLine {
	return &serviceports.GLStatementLine{
		GLAccountID:     balance.GLAccountID,
		AccountCode:     balance.AccountCode,
		AccountName:     balance.AccountName,
		AccountCategory: balance.AccountCategory,
		AmountMinor:     amountMinor,
	}
}

func statementAmount(balance *repositoryports.GLPeriodAccountBalance) int64 {
	switch balance.AccountCategory {
	case accounttype.CategoryAsset, accounttype.CategoryExpense, accounttype.CategoryCostOfRevenue:
		return balance.PeriodDebitMinor - balance.PeriodCreditMinor
	case accounttype.CategoryLiability, accounttype.CategoryEquity, accounttype.CategoryRevenue:
		return balance.PeriodCreditMinor - balance.PeriodDebitMinor
	default:
		return balance.NetChangeMinor
	}
}
