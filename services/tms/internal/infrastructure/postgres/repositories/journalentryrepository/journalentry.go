package journalentryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListJournalEntriesRequest,
) (*pagination.ListResult[*journalentry.JournalEntry], error) {
	limit := req.Filter.Pagination.SafeLimit()
	entries := make([]*journalentry.JournalEntry, 0, limit)
	cols := buncolgen.JournalEntryColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entries).
		Apply(buncolgen.JournalEntryApplyTenant(req.Filter.TenantInfo)).
		Order(cols.AccountingDate.OrderDesc()).
		Order(cols.EntryNumber.OrderDesc()).
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		like := "%" + req.Filter.Query + "%"
		query = query.Where(
			"("+cols.EntryNumber.ILike()+" OR "+cols.Description.ILike()+
				" OR "+cols.ReferenceID.ILike()+" OR "+cols.ReferenceType.ILike()+")",
			like,
			like,
			like,
			like,
		)
	}
	if !req.FiscalYearID.IsNil() {
		query = query.Where(cols.FiscalYearID.Eq(), req.FiscalYearID)
	}
	if !req.FiscalPeriodID.IsNil() {
		query = query.Where(cols.FiscalPeriodID.Eq(), req.FiscalPeriodID)
	}
	if req.ReferenceType != "" {
		query = query.Where(cols.ReferenceType.Eq(), req.ReferenceType)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.AccountingDateStart > 0 {
		query = query.Where(cols.AccountingDate.Gte(), req.AccountingDateStart)
	}
	if req.AccountingDateEnd > 0 {
		query = query.Where(cols.AccountingDate.Lte(), req.AccountingDateEnd)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	return &pagination.ListResult[*journalentry.JournalEntry]{Items: entries, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetJournalEntryByIDRequest,
) (*journalentry.JournalEntry, error) {
	cols := buncolgen.JournalEntryColumns
	entry := new(journalentry.JournalEntry)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entry).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Relation(buncolgen.JournalEntryRelations.Lines, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(buncolgen.JournalEntryLineColumns.LineNumber.OrderAsc())
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "JournalEntry")
	}

	return entry, nil
}

//
//nolint:gocritic // repository contract uses value request types
func (r *repository) MarkReversed(
	ctx context.Context,
	req repositories.MarkJournalEntryReversedRequest,
) error {
	cols := buncolgen.JournalEntryColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*journalentry.JournalEntry)(nil)).
		Set(cols.Status.Set(), journalentry.StatusReversed).
		Set(cols.ReversedByID.Set(), req.ReversalEntryID).
		Set(cols.ReversalDate.Set(), req.ReversalDate).
		Set(cols.ReversalReason.Set(), req.ReversalReason).
		Set(cols.UpdatedByID.Set(), req.UpdatedByID).
		Set(cols.Version.Inc(1)).
		Where(cols.ID.Eq(), req.OriginalEntryID).
		Where(cols.OrganizationID.Eq(), req.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), req.BusinessUnitID).
		Where(cols.ReversedByID.IsNull()).
		Exec(ctx)
	return err
}
