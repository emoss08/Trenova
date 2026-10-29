package billingqueuereviewrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	runErrors     = dberror.Mapping{Entity: "Approval run"}
	runItemErrors = dberror.Mapping{Entity: "Approval run item"}
)

func (r *repository) CreateApprovalRun(
	ctx context.Context,
	req *billingqueueports.CreateApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingapprovalrun.ApprovalRun, error) {
			db := r.db.Keel(ctx)
			run := req.Run
			if err := billingqueuedb.ApprovalRuns.Create(ctx, db, run); err != nil {
				return nil, runErrors.Map(err, run.ID.String())
			}

			items := make([]*billingapprovalrun.ApprovalRunItem, 0, len(req.ItemIDs))
			for i, itemID := range req.ItemIDs {
				items = append(items, &billingapprovalrun.ApprovalRunItem{
					OrganizationID: run.OrganizationID,
					BusinessUnitID: run.BusinessUnitID,
					RunID:          run.ID,
					ItemID:         itemID,
					Sequence:       i,
					Status:         billingapprovalrun.ApprovalItemPending,
				})
			}
			if len(items) > 0 {
				if err := billingqueuedb.ApprovalRunItems.CreateMany(ctx, db, items); err != nil {
					return nil, runItemErrors.Map(err, run.ID.String())
				}
			}
			run.Items = items

			return run, nil
		},
	)
}

func approvalRunWithItems() *keel.Query[*billingapprovalrun.ApprovalRun] {
	run := billingqueuedb.ApprovalRuns
	items := run.Items()
	return run.Load(items.Order(items.Sequence.Asc()))
}

func (r *repository) GetApprovalRun(
	ctx context.Context,
	req *billingqueueports.GetApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingapprovalrun.ApprovalRun, error) {
		run := billingqueuedb.ApprovalRuns
		q := run.Query()
		if req.IncludeItems {
			q = approvalRunWithItems()
		}

		entity, err := q.Where(run.ID.Eq(req.RunID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, runErrors.Map(err, req.RunID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetApprovalRunByKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	key string,
) (*billingapprovalrun.ApprovalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingapprovalrun.ApprovalRun, error) {
		entity, err := approvalRunWithItems().
			Where(billingqueuedb.ApprovalRuns.IdempotencyKey.Eq(key)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, runErrors.Map(err, "")
		}

		return entity, nil
	})
}

func scheduledRun(runID pulid.ID) *keel.Query[*billingapprovalrun.ApprovalRun] {
	run := billingqueuedb.ApprovalRuns
	return run.Where(
		run.ID.Eq(runID),
		run.Status.Eq(billingapprovalrun.ApprovalRunScheduled),
		run.CancelRequestedAt.IsNull(),
	)
}

func (r *repository) MarkApprovalRunRunning(
	ctx context.Context,
	req *billingqueueports.MarkApprovalRunRunningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		run := billingqueuedb.ApprovalRuns
		n, err := scheduledRun(req.RunID).Update(ctx, r.db.Keel(ctx),
			run.Status.To(billingapprovalrun.ApprovalRunRunning),
			run.StartedAt.To(timeutils.NowUnix()),
			run.TemporalWorkflowID.To(req.TemporalWorkflowID),
			run.TemporalRunID.To(req.TemporalRunID),
		)
		if err != nil {
			return false, runErrors.Map(err, req.RunID.String())
		}

		return n > 0, nil
	})
}

func (r *repository) RequestApprovalRunUndo(
	ctx context.Context,
	req *billingqueueports.RequestApprovalRunUndoRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		run := billingqueuedb.ApprovalRuns
		n, err := scheduledRun(req.RunID).Update(ctx, r.db.Keel(ctx),
			run.CancelRequestedAt.To(timeutils.NowUnix()),
			run.CancelRequestedByID.ToExpr(keel.Val(req.RequestedByID)),
		)
		if err != nil {
			return false, runErrors.Map(err, req.RunID.String())
		}

		return n > 0, nil
	})
}

func (r *repository) SetApprovalRunWorkflow(
	ctx context.Context,
	req *billingqueueports.MarkApprovalRunRunningRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		run := billingqueuedb.ApprovalRuns
		_, err := run.Where(run.ID.Eq(req.RunID)).Update(ctx, r.db.Keel(ctx),
			run.TemporalWorkflowID.To(req.TemporalWorkflowID),
			run.TemporalRunID.To(req.TemporalRunID),
			run.Version.ToExpr(run.Version),
			run.UpdatedAt.ToExpr(run.UpdatedAt),
		)

		return runErrors.Map(err, req.RunID.String())
	})
}

func (r *repository) ListPendingApprovalRunItems(
	ctx context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) ([]*billingapprovalrun.ApprovalRunItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingapprovalrun.ApprovalRunItem, error) {
			item := billingqueuedb.ApprovalRunItems
			items, err := item.Where(
				item.RunID.Eq(runID),
				item.Status.Eq(billingapprovalrun.ApprovalItemPending),
			).Order(item.Sequence.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			if items == nil {
				return []*billingapprovalrun.ApprovalRunItem{}, nil
			}

			return items, nil
		},
	)
}

func (r *repository) RecordApprovalRunItem(
	ctx context.Context,
	req *billingqueueports.RecordApprovalRunItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		entry := req.Item
		item := billingqueuedb.ApprovalRunItems
		_, err := item.Where(
			item.RunID.Eq(entry.RunID),
			item.ItemID.Eq(entry.ItemID),
			item.Status.Eq(billingapprovalrun.ApprovalItemPending),
		).Update(ctx, r.db.Keel(ctx),
			item.Status.To(entry.Status),
			item.FailureCode.ToExpr(keel.Val(nullable(string(entry.FailureCode)))),
			item.ErrorMessage.ToExpr(keel.Val(nullable(entry.ErrorMessage))),
			item.InvoiceID.ToExpr(keel.Val(entry.InvoiceID)),
			item.InvoiceNumber.ToExpr(keel.Val(nullable(entry.InvoiceNumber))),
			item.ProcessedAt.To(timeutils.NowUnix()),
		)

		return runItemErrors.Map(err, entry.ItemID.String())
	})
}

type runCounts struct {
	Approved int
	Failed   int
	Skipped  int
}

func (r *repository) FinalizeApprovalRun(
	ctx context.Context,
	req *billingqueueports.FinalizeApprovalRunRequest,
) (*billingapprovalrun.ApprovalRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingapprovalrun.ApprovalRun, error) {
			db := r.db.Keel(ctx)
			now := timeutils.NowUnix()
			leftover := req.LeftoverCode
			if leftover == "" {
				leftover = billingapprovalrun.ApprovalFailureUnexpected
			}

			item := billingqueuedb.ApprovalRunItems
			if _, err := item.Where(
				item.RunID.Eq(req.RunID),
				item.Status.Eq(billingapprovalrun.ApprovalItemPending),
			).Update(ctx, db,
				item.Status.To(billingapprovalrun.ApprovalItemSkipped),
				item.FailureCode.To(leftover),
				item.ProcessedAt.To(now),
			); err != nil {
				return nil, runItemErrors.Map(err, req.RunID.String())
			}

			counts, err := keel.IntoOne[runCounts](ctx, db, item.
				Pick(
					keel.Count().Filter(item.Status.Eq(billingapprovalrun.ApprovalItemApproved)),
					keel.Count().Filter(item.Status.Eq(billingapprovalrun.ApprovalItemFailed)),
					keel.Count().Filter(item.Status.Eq(billingapprovalrun.ApprovalItemSkipped)),
				).
				Where(item.RunID.Eq(req.RunID)))
			if err != nil {
				return nil, err
			}

			run := billingqueuedb.ApprovalRuns
			if _, err = run.Where(run.ID.Eq(req.RunID)).Update(ctx, db,
				run.Status.To(req.Status),
				run.FailureMessage.ToExpr(keel.Val(nullable(req.FailureMessage))),
				run.CompletedAt.To(now),
				run.ApprovedCount.To(int64(counts.Approved)),
				run.FailedCount.To(int64(counts.Failed)),
				run.SkippedCount.To(int64(counts.Skipped)),
			); err != nil {
				return nil, runErrors.Map(err, req.RunID.String())
			}

			return r.GetApprovalRun(ctx, &billingqueueports.GetApprovalRunRequest{
				TenantInfo:   req.TenantInfo,
				RunID:        req.RunID,
				IncludeItems: true,
			})
		},
	)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}

	return s
}
