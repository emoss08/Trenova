package journalentryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/journalentrycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) journalentryports.JournalEntryRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-entry-repository")}
}

func (r *repository) List(
	ctx context.Context,
	req *journalentryports.ListJournalEntriesRequest,
) (*pagination.ListResult[*journalentry.JournalEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*journalentry.JournalEntry], error) {
			limit := req.Filter.Pagination.SafeLimit()
			entries := make([]*journalentry.JournalEntry, 0, limit)
			cols := journalentrycol.JournalEntryColumns
			query := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entries).
				Apply(journalentrycol.JournalEntryApplyTenant(req.Filter.TenantInfo)).
				Order(cols.AccountingDate.OrderDesc()).
				Order(cols.EntryNumber.OrderDesc()).
				Limit(int64(limit)).
				Offset(int64(req.Filter.Pagination.SafeOffset()))

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

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*journalentry.JournalEntry)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return querybuilder.ApplyFiltersWithoutSort(
							sq,
							journalentrycol.JournalEntryTable.Alias,
							req.Filter,
							(*journalentry.JournalEntry)(nil),
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count journal entries: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*journalentry.JournalEntry]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(items *[]*journalentry.JournalEntry) *bun.SelectQuery {
						return dba.NewSelect().
							Model(items).
							ColumnExpr(journalentrycol.JournalEntryTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							sq,
							journalentrycol.JournalEntryTable.Alias,
							req.Filter,
							req.Cursor,
							(*journalentry.JournalEntry)(nil),
						)
					},
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
		cols := journalentrycol.JournalEntryColumns
		entry := new(journalentry.JournalEntry)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entry).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Relation(journalentrycol.JournalEntryRelations.Lines, func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order(journalentrycol.JournalEntryLineColumns.LineNumber.OrderAsc())
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "JournalEntry")
		}

		return entry, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req journalentryports.GetJournalEntriesByIDsRequest,
) ([]*journalentry.JournalEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*journalentry.JournalEntry, error) {
		cols := journalentrycol.JournalEntryColumns
		entries := make([]*journalentry.JournalEntry, 0, len(req.IDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entries).
			Apply(journalentrycol.JournalEntryApplyTenant(req.TenantInfo)).
			Where(cols.ID.In(), bun.List(req.IDs)).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("get journal entries by ids: %w", err)
		}

		return entries, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *journalentryports.JournalEntrySelectOptionsRequest,
) (*pagination.ListResult[*journalentry.JournalEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*journalentry.JournalEntry], error) {
			cols := journalentrycol.JournalEntryColumns
			return dbhelper.SelectOptions[*journalentry.JournalEntry](
				ctx,
				r.db.DBForContext(ctx),
				req.SelectQueryRequest,
				&dbhelper.SelectOptionsConfig{
					ColumnRefs: []buncolgen.Column{
						cols.ID,
						cols.OrganizationID,
						cols.BusinessUnitID,
						cols.FiscalPeriodID,
						cols.EntryNumber,
						cols.EntryType,
						cols.Status,
						cols.AccountingDate,
						cols.CreatedAt,
					},
					OrgColumnRef:     &cols.OrganizationID,
					BuColumnRef:      &cols.BusinessUnitID,
					SearchColumnRefs: []buncolgen.Column{cols.EntryNumber},
					EntityName:       "JournalEntry",
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						if req.ReversibleOnly {
							q = q.Where(cols.Status.Eq(), journalentry.StatusPosted).
								Where(cols.IsPosted.IsTrue()).
								Where(cols.IsReversal.IsFalse()).
								Where(cols.ReversedByID.IsNull())
						}
						return q.Order(cols.AccountingDate.OrderDesc(), cols.ID.OrderDesc())
					},
				},
			)
		},
	)
}

//nolint:gocritic // repository contract uses value request types
func (r *repository) MarkReversed(
	ctx context.Context,
	req journalentryports.MarkJournalEntryReversedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := journalentrycol.JournalEntryColumns
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
	})
}
