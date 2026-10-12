package customerledgerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
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

func New(p Params) customerledgerports.CustomerLedgerProjectionRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-ledger-repository")}
}

type entryRecord struct {
	bun.BaseModel `bun:"table:customer_ledger_entries"`

	ID               string `bun:"id,pk"`
	OrganizationID   string `bun:"organization_id,pk"`
	BusinessUnitID   string `bun:"business_unit_id,pk"`
	CustomerID       string `bun:"customer_id"`
	SourceObjectType string `bun:"source_object_type"`
	SourceObjectID   string `bun:"source_object_id"`
	SourceEventType  string `bun:"source_event_type"`
	RelatedInvoiceID string `bun:"related_invoice_id,nullzero"`
	DocumentNumber   string `bun:"document_number,nullzero"`
	TransactionDate  int64  `bun:"transaction_date"`
	LineNumber       int    `bun:"line_number"`
	AmountMinor      int64  `bun:"amount_minor"`
	CreatedByID      string `bun:"created_by_id"`
}

func (r *repository) AppendEntries(
	ctx context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entries) == 0 {
			return nil
		}
		records := make([]*entryRecord, 0, len(entries))
		for _, entry := range entries {
			if entry == nil {
				continue
			}
			records = append(
				records,
				&entryRecord{
					ID:               entry.ID.String(),
					OrganizationID:   entry.OrganizationID.String(),
					BusinessUnitID:   entry.BusinessUnitID.String(),
					CustomerID:       entry.CustomerID.String(),
					SourceObjectType: entry.SourceObjectType,
					SourceObjectID:   entry.SourceObjectID,
					SourceEventType:  entry.SourceEventType,
					RelatedInvoiceID: entry.RelatedInvoiceID.String(),
					DocumentNumber:   entry.DocumentNumber,
					TransactionDate:  entry.TransactionDate,
					LineNumber:       entry.LineNumber,
					AmountMinor:      entry.AmountMinor,
					CreatedByID:      entry.CreatedByID.String(),
				},
			)
		}
		if len(records) == 0 {
			return nil
		}
		_, err := r.db.DBForContext(ctx).NewInsert().Model(&records).Exec(ctx)
		return err
	})
}

// SumBalanceAsOf totals every subledger row up to the date. This is the figure a
// year-end close compares against the AR control account in the general ledger:
// the GL carries one reconciliation total, the detail behind it lives here.
func (r *repository) SumBalanceAsOf(
	ctx context.Context,
	req customerledgerports.SumCustomerLedgerBalanceRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		var total int64
		err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("customer_ledger_entries AS cle").
			ColumnExpr("COALESCE(SUM(cle.amount_minor), 0)").
			Where("cle.organization_id = ?", req.TenantInfo.OrgID).
			Where("cle.business_unit_id = ?", req.TenantInfo.BuID).
			Where("cle.transaction_date <= ?", req.AsOfDate).
			Scan(ctx, &total)
		if err != nil {
			r.l.Error("failed to sum customer ledger balance", zap.Error(err))
			return 0, err
		}

		return total, nil
	})
}

// SumCustomerBalances totals each named customer's subledger. A customer with
// no entries is absent from the result, which reads as a zero balance.
func (r *repository) SumCustomerBalances(
	ctx context.Context,
	req customerledgerports.SumCustomerBalancesRequest,
) (map[pulid.ID]int64, error) {
	balances := make(map[pulid.ID]int64, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return balances, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int64, error) {
		type row struct {
			CustomerID pulid.ID `bun:"customer_id"`
			Balance    int64    `bun:"balance"`
		}
		rows := make([]row, 0, len(req.CustomerIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("customer_ledger_entries AS cle").
			ColumnExpr("cle.customer_id").
			ColumnExpr("COALESCE(SUM(cle.amount_minor), 0) AS balance").
			Where("cle.organization_id = ?", req.TenantInfo.OrgID).
			Where("cle.business_unit_id = ?", req.TenantInfo.BuID).
			Where("cle.customer_id IN (?)", bun.In(req.CustomerIDs)).
			GroupExpr("cle.customer_id").
			Scan(ctx, &rows)
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
