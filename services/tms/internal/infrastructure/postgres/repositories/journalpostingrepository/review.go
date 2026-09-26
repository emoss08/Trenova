package journalpostingrepository

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func NewReview(p Params) repositories.JournalReviewRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-review-repository")}
}

func (r *repository) LockEntry(
	ctx context.Context,
	req repositories.LockJournalEntryRequest,
) (*journalentry.JournalEntry, error) {
	entry := new(journalentry.JournalEntry)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entry).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.JournalEntryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.JournalEntryColumns.ID.Eq(), req.EntryID)
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "JournalEntry")
	}

	lines := make([]*journalentry.JournalEntryLine, 0, 2)
	lineCols := buncolgen.JournalEntryLineColumns
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&lines).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.JournalEntryLineScopeTenant(sq, req.TenantInfo).
				Where(lineCols.JournalEntryID.Eq(), req.EntryID)
		}).
		Order(lineCols.LineNumber.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read journal entry lines: %w", err)
	}
	entry.Lines = lines

	return entry, nil
}

func (r *repository) ApproveEntry(
	ctx context.Context,
	params *repositories.ApproveJournalEntryParams,
) error {
	cols := buncolgen.JournalEntryColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*journalentry.JournalEntry)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.JournalEntryScopeTenantUpdate(uq, params.TenantInfo).
				Where(cols.ID.Eq(), params.EntryID).
				Where(cols.Version.Eq(), params.Version).
				Where(cols.Status.Eq(), journalentry.StatusPending)
		}).
		Set(cols.Status.Set(), journalentry.StatusApproved).
		Set(cols.IsApproved.Set(), true).
		Set(cols.ApprovedByID.Set(), params.ApprovedByID).
		Set(cols.ApprovedAt.Set(), params.ApprovedAt).
		Set(cols.UpdatedByID.Set(), params.ApprovedByID).
		Set(cols.UpdatedAt.Set(), params.ApprovedAt).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("approve journal entry: %w", err)
	}
	if err = requireOneRow(result, "approved"); err != nil {
		return err
	}

	return r.moveBatchAndSources(ctx, &stateChange{
		tenantInfo: params.TenantInfo,
		entryID:    params.EntryID,
		batchID:    params.BatchID,
		status:     string(journalentry.StatusApproved),
		actorID:    params.ApprovedByID,
		changedAt:  params.ApprovedAt,
	})
}

func (r *repository) PostEntry(
	ctx context.Context,
	params *repositories.PostJournalEntryParams,
) error {
	cols := buncolgen.JournalEntryColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*journalentry.JournalEntry)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.JournalEntryScopeTenantUpdate(uq, params.TenantInfo).
				Where(cols.ID.Eq(), params.EntryID).
				Where(cols.Version.Eq(), params.Version).
				Where(cols.Status.Eq(), journalentry.StatusApproved).
				Where(cols.IsPosted.IsFalse())
		}).
		Set(cols.Status.Set(), journalentry.StatusPosted).
		Set(cols.IsPosted.Set(), true).
		Set(cols.PostedByID.Set(), params.PostedByID).
		Set(cols.PostedAt.Set(), params.PostedAt).
		Set(cols.FiscalYearID.Set(), params.FiscalYearID).
		Set(cols.FiscalPeriodID.Set(), params.FiscalPeriodID).
		Set(cols.AccountingDate.Set(), params.AccountingDate).
		Set(cols.UpdatedByID.Set(), params.PostedByID).
		Set(cols.UpdatedAt.Set(), params.PostedAt).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("post journal entry: %w", err)
	}
	if err = requireOneRow(result, "posted"); err != nil {
		return err
	}

	if err = r.moveBatchAndSources(ctx, &stateChange{
		tenantInfo: params.TenantInfo,
		entryID:    params.EntryID,
		batchID:    params.BatchID,
		status:     string(journalentry.StatusPosted),
		actorID:    params.PostedByID,
		changedAt:  params.PostedAt,
		postedInto: &postedInto{
			fiscalYearID:   params.FiscalYearID,
			fiscalPeriodID: params.FiscalPeriodID,
			accountingDate: params.AccountingDate,
		},
	}); err != nil {
		return err
	}

	return r.applyBalances(ctx, balanceTarget{
		organizationID: params.TenantInfo.OrgID.String(),
		businessUnitID: params.TenantInfo.BuID.String(),
		fiscalYearID:   params.FiscalYearID.String(),
		fiscalPeriodID: params.FiscalPeriodID.String(),
		entryID:        params.EntryID.String(),
	}, params.Lines)
}

type postedInto struct {
	fiscalYearID   pulid.ID
	fiscalPeriodID pulid.ID
	accountingDate int64
}

type stateChange struct {
	tenantInfo pagination.TenantInfo
	entryID    pulid.ID
	batchID    pulid.ID
	status     string
	actorID    pulid.ID
	changedAt  int64
	postedInto *postedInto
}

func (r *repository) moveBatchAndSources(ctx context.Context, change *stateChange) error {
	if change.batchID.IsNotNil() {
		batchCols := buncolgen.JournalBatchColumns
		update := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*journalentry.JournalBatch)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.JournalBatchScopeTenantUpdate(uq, change.tenantInfo).
					Where(batchCols.ID.Eq(), change.batchID)
			}).
			Set(batchCols.Status.Set(), change.status).
			Set(batchCols.UpdatedByID.Set(), change.actorID).
			Set(batchCols.UpdatedAt.Set(), change.changedAt)
		if change.postedInto != nil {
			update = update.
				Set(batchCols.PostedAt.Set(), change.changedAt).
				Set(batchCols.PostedByID.Set(), change.actorID).
				Set(batchCols.FiscalYearID.Set(), change.postedInto.fiscalYearID).
				Set(batchCols.FiscalPeriodID.Set(), change.postedInto.fiscalPeriodID).
				Set(batchCols.AccountingDate.Set(), change.postedInto.accountingDate)
		}
		if _, err := update.Exec(ctx); err != nil {
			return fmt.Errorf("update journal batch: %w", err)
		}
	}

	sourceCols := buncolgen.SourceColumns
	if _, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*journalsource.Source)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SourceScopeTenantUpdate(uq, change.tenantInfo).
				Where(sourceCols.JournalEntryID.Eq(), change.entryID)
		}).
		Set(sourceCols.Status.Set(), change.status).
		Exec(ctx); err != nil {
		return fmt.Errorf("update journal sources: %w", err)
	}
	return nil
}

func requireOneRow(result sql.Result, verb string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return errortypes.NewConflictError(
			"The journal entry changed before it could be " + verb + "; reload it and try again",
		)
	}
	return nil
}

const reviewAccountingDateSort = "accountingDate"

var reviewStatuses = []journalentry.Status{
	journalentry.StatusPending,
	journalentry.StatusApproved,
}

func (r *repository) applyReviewFilters(
	q *bun.SelectQuery,
	req *repositories.ListJournalReviewRequest,
) *bun.SelectQuery {
	cols := buncolgen.JournalEntryColumns
	if req.Filter != nil {
		q = q.Apply(buncolgen.JournalEntryApplyTenant(req.Filter.TenantInfo))
	}
	statuses := reviewStatuses
	if len(req.Statuses) > 0 {
		statuses = make([]journalentry.Status, 0, len(req.Statuses))
		for _, status := range req.Statuses {
			if slices.Contains(reviewStatuses, status) {
				statuses = append(statuses, status)
			}
		}
		if len(statuses) == 0 {
			statuses = reviewStatuses
		}
	}
	return q.
		Where(cols.Status.In(), bun.List(statuses)).
		Where(cols.IsPosted.IsFalse())
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListJournalReviewRequest,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: reviewAccountingDateSort, Direction: dbtype.SortDirectionAsc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*journalentry.JournalEntry)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.JournalEntryTable.Alias,
					req.Filter,
					(*journalentry.JournalEntry)(nil),
				)
				return r.applyReviewFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count journal entries awaiting review: %w", err)
		}
		totalCount = &total
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
					ColumnExpr(buncolgen.JournalEntryTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.JournalEntryTable.Alias,
					req.Filter,
					req.Cursor,
					(*journalentry.JournalEntry)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyReviewFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list journal entries awaiting review: %w", err)
	}
	return result, nil
}

func (r *repository) Summarize(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.JournalReviewSummary, error) {
	cols := buncolgen.JournalEntryColumns
	summary := new(repositories.JournalReviewSummary)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*journalentry.JournalEntry)(nil)).
		ColumnExpr(buncolgen.CountFilter("awaiting_approval", cols.Status.Eq()), journalentry.StatusPending).
		ColumnExpr(buncolgen.CountFilter("ready_to_post", cols.Status.Eq()), journalentry.StatusApproved).
		ColumnExpr(buncolgen.Min(cols.AccountingDate, "oldest_accounting_date")).
		Apply(buncolgen.JournalEntryApplyTenant(tenantInfo)).
		Where(cols.Status.In(), bun.List(reviewStatuses)).
		Where(cols.IsPosted.IsFalse()).
		Scan(ctx, summary); err != nil {
		return nil, fmt.Errorf("summarize journal entries awaiting review: %w", err)
	}
	return summary, nil
}
