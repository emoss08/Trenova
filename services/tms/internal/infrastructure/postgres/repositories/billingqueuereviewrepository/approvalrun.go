package billingqueuereviewrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// CreateApprovalRun writes the run and one Pending row per item together, so
// the run's report is complete before the worker picks it up.
func (r *repository) CreateApprovalRun(
	ctx context.Context,
	req *repositories.CreateApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.ApprovalRun, error) {
		db := r.db.DBForContext(ctx)
		run := req.Run
		if _, err := db.NewInsert().Model(run).Exec(ctx); err != nil {
			return nil, err
		}

		items := make([]*billingqueue.ApprovalRunItem, 0, len(req.ItemIDs))
		for i, itemID := range req.ItemIDs {
			items = append(items, &billingqueue.ApprovalRunItem{
				OrganizationID: run.OrganizationID,
				BusinessUnitID: run.BusinessUnitID,
				RunID:          run.ID,
				ItemID:         itemID,
				Sequence:       i,
				Status:         billingqueue.ApprovalItemPending,
			})
		}
		if len(items) > 0 {
			if _, err := db.NewInsert().Model(&items).Exec(ctx); err != nil {
				return nil, fmt.Errorf("seed approval run items: %w", err)
			}
		}
		run.Items = items

		return run, nil
	})
}

func (r *repository) GetApprovalRun(
	ctx context.Context,
	req *repositories.GetApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.ApprovalRun, error) {
		run := new(billingqueue.ApprovalRun)
		q := r.db.DBForContext(ctx).NewSelect().
			Model(run).
			Apply(tenantWhere("bqar", req.TenantInfo)).
			Where("bqar.id = ?", req.RunID)
		if req.IncludeItems {
			q = q.Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("bqari.sequence ASC")
			})
		}
		if err := q.Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Approval run")
		}

		return run, nil
	})
}

func (r *repository) GetApprovalRunByKey(
	ctx context.Context,
	ti pagination.TenantInfo,
	key string,
) (*billingqueue.ApprovalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.ApprovalRun, error) {
		run := new(billingqueue.ApprovalRun)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(run).
			Apply(tenantWhere("bqar", ti)).
			Where("bqar.idempotency_key = ?", key).
			Relation("Items", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("bqari.sequence ASC")
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Approval run")
		}

		return run, nil
	})
}

// MarkApprovalRunRunning and RequestApprovalRunUndo race for the same row, and
// each only wins while the run is still Scheduled with no undo on it. That
// single conditional update is the whole guarantee that an undo either stops
// every write or none.
func (r *repository) MarkApprovalRunRunning(
	ctx context.Context,
	req *repositories.MarkApprovalRunRunningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		now := timeutils.NowUnix()
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Table("billingqueue_approval_runs").
			Set("status = ?", billingqueue.ApprovalRunRunning).
			Set("started_at = ?", now).
			Set("temporal_workflow_id = ?", req.TemporalWorkflowID).
			Set("temporal_run_id = ?", req.TemporalRunID).
			Set("updated_at = ?", now).
			Set("version = version + 1").
			Where("id = ?", req.RunID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Where("status = ?", billingqueue.ApprovalRunScheduled).
			Where("cancel_requested_at IS NULL").
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("start approval run: %w", err)
		}
		n, _ := res.RowsAffected()

		return n > 0, nil
	})
}

func (r *repository) RequestApprovalRunUndo(
	ctx context.Context,
	req *repositories.RequestApprovalRunUndoRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		now := timeutils.NowUnix()
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Table("billingqueue_approval_runs").
			Set("cancel_requested_at = ?", now).
			Set("cancel_requested_by_id = ?", req.RequestedByID).
			Set("updated_at = ?", now).
			Set("version = version + 1").
			Where("id = ?", req.RunID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Where("status = ?", billingqueue.ApprovalRunScheduled).
			Where("cancel_requested_at IS NULL").
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("undo approval run: %w", err)
		}
		n, _ := res.RowsAffected()

		return n > 0, nil
	})
}

func (r *repository) SetApprovalRunWorkflow(
	ctx context.Context,
	req *repositories.MarkApprovalRunRunningRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := r.db.DBForContext(ctx).NewUpdate().
			Table("billingqueue_approval_runs").
			Set("temporal_workflow_id = ?", req.TemporalWorkflowID).
			Set("temporal_run_id = ?", req.TemporalRunID).
			Where("id = ?", req.RunID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Exec(ctx)

		return err
	})
}

func (r *repository) ListPendingApprovalRunItems(
	ctx context.Context,
	ti pagination.TenantInfo,
	runID pulid.ID,
) ([]*billingqueue.ApprovalRunItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*billingqueue.ApprovalRunItem, error) {
		items := make([]*billingqueue.ApprovalRunItem, 0)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&items).
			Apply(tenantWhere("bqari", ti)).
			Where("bqari.run_id = ?", runID).
			Where("bqari.status = ?", billingqueue.ApprovalItemPending).
			Order("bqari.sequence ASC").
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list pending approval run items: %w", err)
		}

		return items, nil
	})
}

func (r *repository) RecordApprovalRunItem(
	ctx context.Context,
	req *repositories.RecordApprovalRunItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		item := req.Item
		now := timeutils.NowUnix()
		_, err := r.db.DBForContext(ctx).NewUpdate().
			Table("billingqueue_approval_run_items").
			Set("status = ?", item.Status).
			Set("failure_code = ?", nullable(string(item.FailureCode))).
			Set("error_message = ?", nullable(item.ErrorMessage)).
			Set("invoice_id = ?", item.InvoiceID).
			Set("invoice_number = ?", nullable(item.InvoiceNumber)).
			Set("processed_at = ?", now).
			Set("updated_at = ?", now).
			Where("run_id = ?", item.RunID).
			Where("item_id = ?", item.ItemID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Where("status = ?", billingqueue.ApprovalItemPending).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("record approval run item: %w", err)
		}

		return nil
	})
}

// FinalizeApprovalRun closes a run: whatever is still Pending is marked
// Skipped with the leftover code, and the counters are taken from the item
// rows rather than carried, so a retried activity cannot count twice.
func (r *repository) FinalizeApprovalRun(
	ctx context.Context,
	req *repositories.FinalizeApprovalRunRequest,
) (*billingqueue.ApprovalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.ApprovalRun, error) {
		db := r.db.DBForContext(ctx)
		now := timeutils.NowUnix()
		leftover := req.LeftoverCode
		if leftover == "" {
			leftover = billingqueue.ApprovalFailureUnexpected
		}

		if _, err := db.NewUpdate().
			Table("billingqueue_approval_run_items").
			Set("status = ?", billingqueue.ApprovalItemSkipped).
			Set("failure_code = ?", leftover).
			Set("processed_at = ?", now).
			Set("updated_at = ?", now).
			Where("run_id = ?", req.RunID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Where("status = ?", billingqueue.ApprovalItemPending).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("skip leftover approval run items: %w", err)
		}

		var counts struct {
			Approved int `bun:"approved"`
			Failed   int `bun:"failed"`
			Skipped  int `bun:"skipped"`
		}
		if err := db.NewSelect().
			TableExpr("billingqueue_approval_run_items AS bqari").
			ColumnExpr("COUNT(*) FILTER (WHERE bqari.status = 'Approved') AS approved").
			ColumnExpr("COUNT(*) FILTER (WHERE bqari.status = 'Failed') AS failed").
			ColumnExpr("COUNT(*) FILTER (WHERE bqari.status = 'Skipped') AS skipped").
			Apply(tenantWhere("bqari", req.TenantInfo)).
			Where("bqari.run_id = ?", req.RunID).
			Scan(ctx, &counts); err != nil {
			return nil, fmt.Errorf("count approval run items: %w", err)
		}

		if _, err := db.NewUpdate().
			Table("billingqueue_approval_runs").
			Set("status = ?", req.Status).
			Set("failure_message = ?", nullable(req.FailureMessage)).
			Set("completed_at = ?", now).
			Set("updated_at = ?", now).
			Set("version = version + 1").
			Set("approved_count = ?", counts.Approved).
			Set("failed_count = ?", counts.Failed).
			Set("skipped_count = ?", counts.Skipped).
			Where("id = ?", req.RunID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("finalize approval run: %w", err)
		}

		return r.GetApprovalRun(ctx, &repositories.GetApprovalRunRequest{
			TenantInfo:   req.TenantInfo,
			RunID:        req.RunID,
			IncludeItems: true,
		})
	})
}

func nullable(s string) any {
	if s == "" {
		return nil
	}

	return s
}
