package journalpostingrepository

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var journalEntryErrors = dberror.Mapping{Entity: "JournalEntry"}

func NewReview(p Params) journalentryports.JournalReviewRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-review-repository")}
}

func (r *repository) LockEntry(
	ctx context.Context,
	req journalentryports.LockJournalEntryRequest,
) (*journalentry.JournalEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*journalentry.JournalEntry, error) {
		db := r.db.Keel(ctx)
		je := journalentrydb.JournalEntries
		entry, err := je.Where(je.ID.Eq(req.EntryID)).Lock(keel.ForUpdate).One(ctx, db)
		if err != nil {
			return nil, journalEntryErrors.Map(err, req.EntryID.String())
		}

		jel := journalentrydb.JournalEntryLines
		lines, err := jel.Where(jel.JournalEntryID.Eq(req.EntryID)).
			Order(jel.LineNumber.Asc()).
			All(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("read journal entry lines: %w", err)
		}
		entry.Lines = lines

		return entry, nil
	})
}

func (r *repository) ApproveEntry(
	ctx context.Context,
	params *journalentryports.ApproveJournalEntryParams,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		je := journalentrydb.JournalEntries
		affected, err := je.Where(
			je.ID.Eq(params.EntryID),
			je.Version.Eq(params.Version),
			je.Status.Eq(journalentry.StatusPending),
		).Update(
			ctx,
			db,
			je.Status.To(journalentry.StatusApproved),
			je.IsApproved.To(true),
			je.ApprovedByID.To(params.ApprovedByID),
			je.ApprovedAt.To(params.ApprovedAt),
			je.UpdatedByID.To(params.ApprovedByID),
			je.UpdatedAt.To(params.ApprovedAt),
		)
		if err != nil {
			return fmt.Errorf("approve journal entry: %w", err)
		}
		if err = requireOneRow(affected, "approved"); err != nil {
			return err
		}

		return moveBatchAndSources(ctx, db, &stateChange{
			entryID:   params.EntryID,
			batchID:   params.BatchID,
			status:    string(journalentry.StatusApproved),
			actorID:   params.ApprovedByID,
			changedAt: params.ApprovedAt,
		})
	})
}

func (r *repository) PostEntry(
	ctx context.Context,
	params *journalentryports.PostJournalEntryParams,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		je := journalentrydb.JournalEntries
		affected, err := je.Where(
			je.ID.Eq(params.EntryID),
			je.Version.Eq(params.Version),
			je.Status.Eq(journalentry.StatusApproved),
			je.IsPosted.IsFalse(),
		).Update(
			ctx,
			db,
			je.Status.To(journalentry.StatusPosted),
			je.IsPosted.To(true),
			je.PostedByID.To(params.PostedByID),
			je.PostedAt.To(params.PostedAt),
			je.FiscalYearID.To(params.FiscalYearID),
			je.FiscalPeriodID.To(params.FiscalPeriodID),
			je.AccountingDate.To(params.AccountingDate),
			je.UpdatedByID.To(params.PostedByID),
			je.UpdatedAt.To(params.PostedAt),
		)
		if err != nil {
			return fmt.Errorf("post journal entry: %w", err)
		}
		if err = requireOneRow(affected, "posted"); err != nil {
			return err
		}

		if err = moveBatchAndSources(ctx, db, &stateChange{
			entryID:   params.EntryID,
			batchID:   params.BatchID,
			status:    string(journalentry.StatusPosted),
			actorID:   params.PostedByID,
			changedAt: params.PostedAt,
			postedInto: &postedInto{
				fiscalYearID:   params.FiscalYearID,
				fiscalPeriodID: params.FiscalPeriodID,
				accountingDate: params.AccountingDate,
			},
		}); err != nil {
			return err
		}

		return applyBalances(ctx, db, balanceTarget{
			fiscalYearID:   params.FiscalYearID,
			fiscalPeriodID: params.FiscalPeriodID,
			entryID:        params.EntryID,
		}, params.Lines)
	})
}

type postedInto struct {
	fiscalYearID   pulid.ID
	fiscalPeriodID pulid.ID
	accountingDate int64
}

type stateChange struct {
	entryID    pulid.ID
	batchID    pulid.ID
	status     string
	actorID    pulid.ID
	changedAt  int64
	postedInto *postedInto
}

func moveBatchAndSources(ctx context.Context, db *keel.DB, change *stateChange) error {
	if change.batchID.IsNotNil() {
		jb := journalentrydb.JournalBatches
		sets := []keel.Assignment{
			jb.Status.To(change.status),
			jb.UpdatedByID.To(change.actorID),
			jb.UpdatedAt.To(change.changedAt),
		}
		if change.postedInto != nil {
			sets = append(sets,
				jb.PostedAt.To(change.changedAt),
				jb.PostedByID.To(change.actorID),
				jb.FiscalYearID.To(change.postedInto.fiscalYearID),
				jb.FiscalPeriodID.To(change.postedInto.fiscalPeriodID),
				jb.AccountingDate.To(change.postedInto.accountingDate),
			)
		}
		if _, err := jb.Where(jb.ID.Eq(change.batchID)).Update(ctx, db, sets...); err != nil {
			return fmt.Errorf("update journal batch: %w", err)
		}
	}

	js := journalsourcedb.Sources
	if _, err := js.Where(js.JournalEntryID.Eq(change.entryID)).
		Update(ctx, db, js.Status.To(change.status)); err != nil {
		return fmt.Errorf("update journal sources: %w", err)
	}

	return nil
}

func requireOneRow(affected int64, verb string) error {
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

func reviewFilters(req *journalentryports.ListJournalReviewQuery) []keel.Pred {
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

	je := journalentrydb.JournalEntries
	return []keel.Pred{je.Status.In(statuses...), je.IsPosted.IsFalse()}
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *journalentryports.ListJournalReviewQuery,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: reviewAccountingDateSort, Direction: dbtype.SortDirectionAsc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				journalentrydb.JournalEntries.Where(reviewFilters(req)...),
				&listquery.Request[journalentry.JournalEntry]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list journal entries awaiting review: %w", err)
			}
			return result, nil
		},
	)
}

func (r *repository) Summarize(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*journalentryports.JournalReviewSummaryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*journalentryports.JournalReviewSummaryRow, error) {
			je := journalentrydb.JournalEntries
			summary, err := keel.IntoOne[journalentryports.JournalReviewSummaryRow](
				ctx,
				r.db.Keel(ctx),
				je.Pick(
					keel.Count().Filter(je.Status.Eq(journalentry.StatusPending)),
					keel.Count().Filter(je.Status.Eq(journalentry.StatusApproved)),
					je.AccountingDate.Min(),
				).Where(je.Status.In(reviewStatuses...), je.IsPosted.IsFalse()),
			)
			if err != nil {
				return nil, fmt.Errorf("summarize journal entries awaiting review: %w", err)
			}
			return &summary, nil
		},
	)
}
