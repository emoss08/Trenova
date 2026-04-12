package customerledgerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.CustomerLedgerProjectionRepository {
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

func (r *repository) AppendEntries(ctx context.Context, entries []*customerledger.Entry) error {
	if len(entries) == 0 {
		return nil
	}
	records := make([]*entryRecord, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		records = append(records, &entryRecord{ID: entry.ID.String(), OrganizationID: entry.OrganizationID.String(), BusinessUnitID: entry.BusinessUnitID.String(), CustomerID: entry.CustomerID.String(), SourceObjectType: entry.SourceObjectType, SourceObjectID: entry.SourceObjectID, SourceEventType: entry.SourceEventType, RelatedInvoiceID: entry.RelatedInvoiceID.String(), DocumentNumber: entry.DocumentNumber, TransactionDate: entry.TransactionDate, LineNumber: entry.LineNumber, AmountMinor: entry.AmountMinor, CreatedByID: entry.CreatedByID.String()})
	}
	if len(records) == 0 {
		return nil
	}
	_, err := r.db.DBForContext(ctx).NewInsert().Model(&records).Exec(ctx)
	return err
}
