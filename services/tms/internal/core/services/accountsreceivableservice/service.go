package accountsreceivableservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.AccountsReceivableRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.AccountsReceivableRepository
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.accounts-receivable"), repo: p.Repo}
}

func (s *Service) ListCustomerLedger(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) ([]*accountsreceivable.LedgerEntry, error) {
	return s.repo.ListCustomerLedger(
		ctx,
		repositories.ListCustomerLedgerRequest{TenantInfo: tenantInfo, CustomerID: customerID},
	)
}

func (s *Service) ListOpenItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	asOfDate int64,
) ([]*accountsreceivable.OpenItem, error) {
	if asOfDate == 0 {
		asOfDate = timeutils.NowUnix()
	}

	return s.repo.ListOpenItems(
		ctx,
		repositories.ListAROpenItemsRequest{
			TenantInfo: tenantInfo,
			CustomerID: customerID,
			AsOfDate:   asOfDate,
		},
	)
}

func (s *Service) GetCustomerStatement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	startDate, asOfDate int64,
) (*accountsreceivable.CustomerStatement, error) {
	if asOfDate == 0 {
		asOfDate = timeutils.NowUnix()
	}

	customerName, err := s.repo.GetCustomerName(
		ctx,
		repositories.GetARCustomerNameRequest{TenantInfo: tenantInfo, CustomerID: customerID},
	)
	if err != nil {
		return nil, err
	}

	ledger, err := s.repo.ListCustomerLedger(
		ctx,
		repositories.ListCustomerLedgerRequest{TenantInfo: tenantInfo, CustomerID: customerID},
	)
	if err != nil {
		return nil, err
	}

	openItems, err := s.repo.ListOpenItems(
		ctx,
		repositories.ListAROpenItemsRequest{
			TenantInfo: tenantInfo,
			CustomerID: customerID,
			AsOfDate:   asOfDate,
		},
	)
	if err != nil {
		return nil, err
	}

	aging, err := s.repo.GetCustomerAging(
		ctx,
		repositories.GetARCustomerAgingRequest{
			TenantInfo: tenantInfo,
			CustomerID: customerID,
			AsOfDate:   asOfDate,
		},
	)
	if err != nil {
		return nil, err
	}

	statement := &accountsreceivable.CustomerStatement{
		CustomerID:    customerID,
		CustomerName:  customerName,
		StatementDate: asOfDate,
		StartDate:     startDate,
		OpenItems:     openItems,
	}
	if aging != nil {
		statement.Aging = aging.Buckets
	}

	runningBalance := int64(0)
	transactions := make([]*accountsreceivable.StatementTransaction, 0, len(ledger))
	for _, entry := range ledger {
		if entry == nil || entry.TransactionDate > asOfDate {
			continue
		}
		if startDate > 0 && entry.TransactionDate < startDate {
			statement.OpeningBalanceMinor += entry.AmountMinor
			runningBalance += entry.AmountMinor
			continue
		}

		txn := &accountsreceivable.StatementTransaction{
			TransactionDate: entry.TransactionDate,
			EventType:       entry.EventType,
			DocumentNumber:  entry.DocumentNumber,
			SourceObjectID:  entry.SourceObjectID,
			AmountMinor:     entry.AmountMinor,
		}
		if entry.AmountMinor >= 0 {
			txn.ChargeMinor = entry.AmountMinor
			statement.TotalChargesMinor += entry.AmountMinor
		} else {
			txn.PaymentMinor = -entry.AmountMinor
			statement.TotalPaymentsMinor += -entry.AmountMinor
		}
		runningBalance += entry.AmountMinor
		txn.RunningBalanceMinor = runningBalance
		transactions = append(transactions, txn)
	}

	statement.Transactions = transactions
	statement.EndingBalanceMinor = runningBalance
	return statement, nil
}

func (s *Service) GetAgingSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	asOfDate int64,
) (*accountsreceivable.AgingSummary, error) {
	if asOfDate == 0 {
		asOfDate = timeutils.NowUnix()
	}
	rows, err := s.repo.ListARAging(
		ctx,
		repositories.ListARAgingRequest{TenantInfo: tenantInfo, AsOfDate: asOfDate},
	)
	if err != nil {
		return nil, err
	}
	summary := &accountsreceivable.AgingSummary{AsOfDate: asOfDate, Rows: rows}
	for _, row := range rows {
		summary.Totals.CurrentMinor += row.Buckets.CurrentMinor
		summary.Totals.Days1To30Minor += row.Buckets.Days1To30Minor
		summary.Totals.Days31To60Minor += row.Buckets.Days31To60Minor
		summary.Totals.Days61To90Minor += row.Buckets.Days61To90Minor
		summary.Totals.DaysOver90Minor += row.Buckets.DaysOver90Minor
		summary.Totals.TotalOpenMinor += row.Buckets.TotalOpenMinor
	}
	return summary, nil
}
