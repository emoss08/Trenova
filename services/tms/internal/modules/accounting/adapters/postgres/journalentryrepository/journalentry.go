package journalentryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var journalEntryErrors = dberror.Mapping{Entity: "JournalEntry"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) journalentryports.JournalEntryRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-entry-repository")}
}

func listFilters(req *journalentryports.ListJournalEntriesRequest) []keel.Pred {
	je := journalentrydb.JournalEntries
	query := req.Filter.Query

	return []keel.Pred{
		keel.When(query != "", keel.Or(
			je.EntryNumber.ContainsFold(query),
			je.Description.ContainsFold(query),
			je.ReferenceID.ContainsFold(query),
			je.ReferenceType.ContainsFold(query),
		)),
		keel.When(!req.FiscalYearID.IsNil(), je.FiscalYearID.Eq(req.FiscalYearID)),
		keel.When(!req.FiscalPeriodID.IsNil(), je.FiscalPeriodID.Eq(req.FiscalPeriodID)),
		keel.When(req.ReferenceType != "", je.ReferenceType.Eq(req.ReferenceType)),
		keel.When(req.Status != "", je.Status.Eq(journalentry.Status(req.Status))),
		keel.When(req.AccountingDateStart > 0, je.AccountingDate.Gte(req.AccountingDateStart)),
		keel.When(req.AccountingDateEnd > 0, je.AccountingDate.Lte(req.AccountingDateEnd)),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *journalentryports.ListJournalEntriesRequest,
) (*pagination.ListResult[*journalentry.JournalEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*journalentry.JournalEntry], error) {
			je := journalentrydb.JournalEntries
			q := je.Where(listFilters(req)...)
			counted := q.Clone()
			rows := q.Order(je.AccountingDate.Desc(), je.EntryNumber.Desc()).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())

			var entries []*journalentry.JournalEntry
			var total int64
			err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &entries),
				keel.CountInto(counted, &total),
			)
			if err != nil {
				return nil, fmt.Errorf("list journal entries: %w", err)
			}

			return &pagination.ListResult[*journalentry.JournalEntry]{
				Items: entries,
				Total: int(total),
			}, nil
		},
	)
}

const newestAccountingDate = "accountingDate"

func (r *repository) ListConnection(
	ctx context.Context,
	req *journalentryports.ListJournalEntryConnectionRequest,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
			if len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: newestAccountingDate, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				journalentrydb.JournalEntries.Query(),
				&listquery.Request[journalentry.JournalEntry]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list journal entries: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req journalentryports.GetJournalEntryByIDRequest,
) (*journalentry.JournalEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*journalentry.JournalEntry, error) {
		je := journalentrydb.JournalEntries
		lines := je.Lines()
		entry, err := je.Where(je.ID.Eq(req.ID)).
			Load(lines.Order(lines.LineNumber.Asc())).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, journalEntryErrors.Map(err, req.ID.String())
		}

		return entry, nil
	})
}

//
//nolint:gocritic // repository contract uses value request types
func (r *repository) MarkReversed(
	ctx context.Context,
	req journalentryports.MarkJournalEntryReversedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		je := journalentrydb.JournalEntries
		_, err := je.Where(je.ID.Eq(req.OriginalEntryID), je.ReversedByID.IsNull()).
			Update(
				ctx,
				r.db.Keel(ctx),
				je.Status.To(journalentry.StatusReversed),
				je.ReversedByID.To(req.ReversalEntryID),
				je.ReversalDate.To(req.ReversalDate),
				je.ReversalReason.To(req.ReversalReason),
				je.UpdatedByID.To(req.UpdatedByID),
			)
		return err
	})
}
