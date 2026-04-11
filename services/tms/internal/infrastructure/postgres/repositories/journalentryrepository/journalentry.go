package journalentryrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.JournalEntryRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-entry-repository")}
}

type entryRecord struct {
	bun.BaseModel `bun:"table:journal_entries,alias:je"`

	ID             string        `bun:"id,pk"`
	OrganizationID string        `bun:"organization_id,pk"`
	BusinessUnitID string        `bun:"business_unit_id,pk"`
	BatchID        string        `bun:"batch_id"`
	FiscalYearID   string        `bun:"fiscal_year_id"`
	FiscalPeriodID string        `bun:"fiscal_period_id"`
	EntryNumber    string        `bun:"entry_number"`
	EntryType      string        `bun:"entry_type"`
	Status         string        `bun:"status"`
	AccountingDate int64         `bun:"accounting_date"`
	Description    string        `bun:"description"`
	ReferenceType  string        `bun:"reference_type"`
	ReferenceID    string        `bun:"reference_id"`
	TotalDebit     int64         `bun:"total_debit"`
	TotalCredit    int64         `bun:"total_credit"`
	IsPosted       bool          `bun:"is_posted"`
	IsReversal     bool          `bun:"is_reversal"`
	ReversalOfID   string        `bun:"reversal_of_id"`
	ReversedByID   string        `bun:"reversed_by_id"`
	ReversalDate   *int64        `bun:"reversal_date"`
	ReversalReason string        `bun:"reversal_reason"`
	Lines          []*lineRecord `bun:"rel:has-many,join:id=journal_entry_id"`
}

type lineRecord struct {
	bun.BaseModel `bun:"table:journal_entry_lines,alias:jel"`

	ID             string `bun:"id,pk"`
	JournalEntryID string `bun:"journal_entry_id"`
	GLAccountID    string `bun:"gl_account_id"`
	LineNumber     int16  `bun:"line_number"`
	Description    string `bun:"description"`
	DebitAmount    int64  `bun:"debit_amount"`
	CreditAmount   int64  `bun:"credit_amount"`
	NetAmount      int64  `bun:"net_amount"`
	CustomerID     string `bun:"customer_id"`
	LocationID     string `bun:"location_id"`
}

func (r *repository) GetByID(ctx context.Context, req repositories.GetJournalEntryByIDRequest) (*journalentry.Entry, error) {
	rec := new(entryRecord)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(rec).
		Where("je.id = ?", req.ID).
		Where("je.organization_id = ?", req.TenantInfo.OrgID).
		Where("je.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("jel.line_number ASC") }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "JournalEntry")
	}

	entry := &journalentry.Entry{
		ID:             parseID(rec.ID),
		OrganizationID: parseID(rec.OrganizationID),
		BusinessUnitID: parseID(rec.BusinessUnitID),
		BatchID:        parseID(rec.BatchID),
		FiscalYearID:   parseID(rec.FiscalYearID),
		FiscalPeriodID: parseID(rec.FiscalPeriodID),
		EntryNumber:    rec.EntryNumber,
		EntryType:      rec.EntryType,
		Status:         rec.Status,
		AccountingDate: rec.AccountingDate,
		Description:    rec.Description,
		ReferenceType:  rec.ReferenceType,
		ReferenceID:    rec.ReferenceID,
		TotalDebit:     rec.TotalDebit,
		TotalCredit:    rec.TotalCredit,
		IsPosted:       rec.IsPosted,
		IsReversal:     rec.IsReversal,
		ReversalOfID:   parseID(rec.ReversalOfID),
		ReversedByID:   parseID(rec.ReversedByID),
		ReversalDate:   rec.ReversalDate,
		ReversalReason: rec.ReversalReason,
		Lines:          make([]*journalentry.Line, 0, len(rec.Lines)),
	}
	for _, line := range rec.Lines {
		entry.Lines = append(entry.Lines, &journalentry.Line{
			ID:             parseID(line.ID),
			JournalEntryID: parseID(line.JournalEntryID),
			GLAccountID:    parseID(line.GLAccountID),
			LineNumber:     line.LineNumber,
			Description:    line.Description,
			DebitAmount:    line.DebitAmount,
			CreditAmount:   line.CreditAmount,
			NetAmount:      line.NetAmount,
			CustomerID:     parseID(line.CustomerID),
			LocationID:     parseID(line.LocationID),
		})
	}

	return entry, nil
}

func (r *repository) MarkReversed(ctx context.Context, req repositories.MarkJournalEntryReversedRequest) error {
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Table("journal_entries").
		Set("status = ?", "Reversed").
		Set("reversed_by_id = ?", req.ReversalEntryID).
		Set("reversal_date = ?", req.ReversalDate).
		Set("reversal_reason = ?", req.ReversalReason).
		Set("updated_by_id = ?", req.UpdatedByID).
		Set("version = version + 1").
		Where("id = ?", req.OriginalEntryID).
		Where("organization_id = ?", req.OrganizationID).
		Where("business_unit_id = ?", req.BusinessUnitID).
		Where("reversed_by_id IS NULL").
		Exec(ctx)
	return err
}

func parseID(v string) pulid.ID {
	if v == "" {
		return pulid.Nil
	}
	return pulid.ID(v)
}
