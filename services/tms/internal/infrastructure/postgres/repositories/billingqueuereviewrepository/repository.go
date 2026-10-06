package billingqueuereviewrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.BillingQueueReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-review-repository"),
	}
}

func tenantWhere(alias string, ti pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("?.organization_id = ?", bun.Ident(alias), ti.OrgID).
			Where("?.business_unit_id = ?", bun.Ident(alias), ti.BuID)
	}
}

func (r *repository) ListIssues(
	ctx context.Context,
	ti pagination.TenantInfo,
	itemID pulid.ID,
) ([]*billingqueue.Issue, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*billingqueue.Issue, error) {
		issues := make([]*billingqueue.Issue, 0, 4)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(&issues).
			Apply(tenantWhere("bqis", ti)).
			Where("bqis.item_id = ?", itemID).
			Order("bqis.created_at ASC", "bqis.id ASC").
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list billing queue issues: %w", err)
		}

		return issues, nil
	})
}

// SyncIssues lands the checks' findings on the stored issues inside one
// transaction. A finding is inserted with ON CONFLICT DO NOTHING so two reads
// racing to raise the same issue both end up looking at the one row.
func (r *repository) SyncIssues(
	ctx context.Context,
	req *repositories.SyncBillingQueueIssuesRequest,
) (*repositories.SyncBillingQueueIssuesResult, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*repositories.SyncBillingQueueIssuesResult, error) {
		existing, err := r.ListIssues(ctx, req.TenantInfo, req.ItemID)
		if err != nil {
			return nil, err
		}
		byKey := make(map[string]*billingqueue.Issue, len(existing))
		for _, issue := range existing {
			byKey[issue.FindingKey()] = issue
		}

		result := &repositories.SyncBillingQueueIssuesResult{}
		db := r.db.DBForContext(ctx)
		now := timeutils.NowUnix()
		found := make(map[string]struct{}, len(req.Findings))

		for _, finding := range req.Findings {
			key := finding.FindingKey()
			found[key] = struct{}{}
			current, ok := byKey[key]
			switch {
			case !ok:
				finding.OrganizationID = req.TenantInfo.OrgID
				finding.BusinessUnitID = req.TenantInfo.BuID
				finding.ItemID = req.ItemID
				res, insErr := db.NewInsert().
					Model(finding).
					On("CONFLICT (organization_id, business_unit_id, item_id, code, subject_key) DO NOTHING").
					Exec(ctx)
				if insErr != nil {
					return nil, fmt.Errorf("raise billing queue issue: %w", insErr)
				}
				if n, _ := res.RowsAffected(); n > 0 {
					result.Raised = append(result.Raised, finding)
				}
			case current.ResolutionKey != nil && *current.ResolutionKey == billingqueue.ResolutionCleared:
				current.ResolutionKey = nil
				current.ResolutionText = ""
				current.ResolvedAt = nil
				current.ResolvedByID = nil
				refreshFinding(current, finding)
				if _, err = r.UpdateIssue(ctx, current); err != nil {
					return nil, err
				}
				result.Reopened = append(result.Reopened, current)
			case current.IsOpen() && findingChanged(current, finding):
				refreshFinding(current, finding)
				if _, err = r.UpdateIssue(ctx, current); err != nil {
					return nil, err
				}
			}
		}

		for _, issue := range existing {
			if !issue.IsOpen() || issue.Source != billingqueue.IssueSourceDeterministic {
				continue
			}
			if _, ok := found[issue.FindingKey()]; ok {
				continue
			}
			cleared := billingqueue.ResolutionCleared
			issue.ResolutionKey = &cleared
			issue.ResolutionText = "Cleared"
			issue.ResolvedAt = &now
			issue.ResolvedByID = nil
			if _, err = r.UpdateIssue(ctx, issue); err != nil {
				return nil, err
			}
			result.Cleared = append(result.Cleared, issue)
		}

		result.Issues, err = r.ListIssues(ctx, req.TenantInfo, req.ItemID)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

func refreshFinding(current, finding *billingqueue.Issue) {
	current.Summary = finding.Summary
	current.Reasoning = finding.Reasoning
	current.Options = finding.Options
	current.FlaggedChargeID = finding.FlaggedChargeID
	current.CheckKey = finding.CheckKey
}

func findingChanged(current, finding *billingqueue.Issue) bool {
	if current.Summary != finding.Summary || current.Reasoning != finding.Reasoning ||
		len(current.Options) != len(finding.Options) {
		return true
	}
	for i := range current.Options {
		if current.Options[i].Label != finding.Options[i].Label ||
			current.Options[i].Key != finding.Options[i].Key {
			return true
		}
	}

	return false
}

func (r *repository) GetIssue(
	ctx context.Context,
	req *repositories.GetBillingQueueIssueRequest,
) (*billingqueue.Issue, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.Issue, error) {
		issue := new(billingqueue.Issue)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(issue).
			Apply(tenantWhere("bqis", req.TenantInfo)).
			Where("bqis.item_id = ?", req.ItemID).
			Where("bqis.id = ?", req.IssueID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Billing queue issue")
		}

		return issue, nil
	})
}

func (r *repository) UpdateIssue(
	ctx context.Context,
	issue *billingqueue.Issue,
) (*billingqueue.Issue, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.Issue, error) {
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Model(issue).
			Column(
				"check_key", "subject_key", "summary", "reasoning", "flagged_charge_id", "options",
				"resolution_key", "resolution_text", "effect_snapshot", "resolved_by_id",
				"resolved_at", "requested_at", "requested_by_id", "updated_at",
			).
			Set("version = version + 1").
			Where("id = ?", issue.ID).
			Where("organization_id = ?", issue.OrganizationID).
			Where("business_unit_id = ?", issue.BusinessUnitID).
			Where("version = ?", issue.Version).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update billing queue issue: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, "Billing queue issue", issue.ID.String()); err != nil {
			return nil, err
		}
		issue.Version++

		return issue, nil
	})
}

func (r *repository) CountOpenIssues(
	ctx context.Context,
	ti pagination.TenantInfo,
	itemID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		return r.db.DBForContext(ctx).NewSelect().
			Model((*billingqueue.Issue)(nil)).
			Apply(tenantWhere("bqis", ti)).
			Where("bqis.item_id = ?", itemID).
			Where("bqis.resolution_key IS NULL").
			Count(ctx)
	})
}

func (r *repository) CreateEvents(ctx context.Context, events ...*billingqueue.ItemEvent) error {
	if len(events) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&events).Exec(ctx); err != nil {
			return fmt.Errorf("record billing queue activity: %w", err)
		}

		return nil
	})
}

func (r *repository) ListEvents(
	ctx context.Context,
	req *repositories.ListBillingQueueEventsRequest,
) (*repositories.BillingQueueEventPage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.BillingQueueEventPage, error) {
		limit := req.Limit
		if limit <= 0 || limit > 100 {
			limit = 30
		}
		db := r.db.DBForContext(ctx)

		total, err := db.NewSelect().
			Model((*billingqueue.ItemEvent)(nil)).
			Apply(tenantWhere("bqe", req.TenantInfo)).
			Where("bqe.item_id = ?", req.ItemID).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count billing queue activity: %w", err)
		}

		events := make([]*billingqueue.ItemEvent, 0, limit+1)
		q := db.NewSelect().
			Model(&events).
			Apply(tenantWhere("bqe", req.TenantInfo)).
			Where("bqe.item_id = ?", req.ItemID)
		if req.BeforeAt > 0 {
			q = q.Where("(bqe.at, bqe.id) < (?, ?)", req.BeforeAt, req.BeforeID)
		}
		if err = q.Order("bqe.at DESC", "bqe.id DESC").Limit(limit + 1).Scan(ctx); err != nil {
			return nil, fmt.Errorf("list billing queue activity: %w", err)
		}

		page := &repositories.BillingQueueEventPage{Total: total}
		if len(events) > limit {
			page.HasMore = true
			events = events[:limit]
		}
		page.Items = events

		return page, nil
	})
}

// GetNeighbors walks the queue in the list's own order, newest first. The
// filter is applied without its sort, because the keyset is the order: a
// row comes before the item when it was queued later, or at the same moment
// with a greater id.
func (r *repository) GetNeighbors(
	ctx context.Context,
	req *repositories.GetBillingQueueNeighborsRequest,
) (*repositories.BillingQueueNeighbors, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.BillingQueueNeighbors, error) {
		db := r.db.DBForContext(ctx)
		ti := req.Filter.TenantInfo

		var anchor struct {
			CreatedAt int64 `bun:"created_at"`
		}
		if err := db.NewSelect().
			TableExpr("billing_queue_items AS bqi").
			Column("bqi.created_at").
			Apply(tenantWhere("bqi", ti)).
			Where("bqi.id = ?", req.ItemID).
			Scan(ctx, &anchor); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Billing queue item")
		}

		base := func() *bun.SelectQuery {
			q := querybuilder.ApplyFiltersWithoutSort(
				db.NewSelect().Model((*billingqueue.BillingQueueItem)(nil)),
				"bqi",
				req.Filter,
				(*billingqueue.BillingQueueItem)(nil),
			)
			if !req.IncludePosted {
				q = q.Where("bqi.status <> ?", billingqueue.StatusPosted)
			}
			return q
		}

		total, err := base().Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count billing queue: %w", err)
		}
		before, err := base().
			Where("(bqi.created_at, bqi.id) > (?, ?)", anchor.CreatedAt, req.ItemID).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count billing queue before item: %w", err)
		}
		member, err := base().Where("bqi.id = ?", req.ItemID).Exists(ctx)
		if err != nil {
			return nil, fmt.Errorf("find item in billing queue: %w", err)
		}

		out := &repositories.BillingQueueNeighbors{Total: total}
		if member {
			out.Position = before + 1
		}

		var prev, next []pulid.ID
		if err = base().
			Column("bqi.id").
			Where("(bqi.created_at, bqi.id) > (?, ?)", anchor.CreatedAt, req.ItemID).
			OrderExpr("bqi.created_at ASC, bqi.id ASC").
			Limit(1).
			Scan(ctx, &prev); err != nil {
			return nil, fmt.Errorf("find previous billing queue item: %w", err)
		}
		if err = base().
			Column("bqi.id").
			Where("(bqi.created_at, bqi.id) < (?, ?)", anchor.CreatedAt, req.ItemID).
			OrderExpr("bqi.created_at DESC, bqi.id DESC").
			Limit(1).
			Scan(ctx, &next); err != nil {
			return nil, fmt.Errorf("find next billing queue item: %w", err)
		}
		if len(prev) > 0 {
			out.PrevID = &prev[0]
		}
		if len(next) > 0 {
			out.NextID = &next[0]
		}

		return out, nil
	})
}

// FindDuplicates lists the other bills for the same freight and payer: queue
// items that are not canceled, and invoices that are not voided, whether the
// invoice bills the shipment on its own or carries it as one of its lines.
// Corrections are left out; they bill the same shipment on purpose.
func (r *repository) FindDuplicates(
	ctx context.Context,
	req *repositories.FindBillingQueueDuplicatesRequest,
) ([]*billingqueue.DuplicateRef, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*billingqueue.DuplicateRef, error) {
		if req.ShipmentID.IsNil() {
			return nil, nil
		}
		db := r.db.DBForContext(ctx)
		out := make([]*billingqueue.DuplicateRef, 0)

		var items []struct {
			ID     pulid.ID `bun:"id"`
			Number string   `bun:"number"`
			Status string   `bun:"status"`
		}
		if err := db.NewSelect().
			TableExpr("billing_queue_items AS bqi").
			Column("bqi.id", "bqi.number", "bqi.status").
			Apply(tenantWhere("bqi", req.TenantInfo)).
			Where("bqi.shipment_id = ?", req.ShipmentID).
			Where("bqi.bill_to_customer_id = ?", req.BillToCustomerID).
			Where("bqi.bill_type = ?", req.BillType).
			Where("bqi.id <> ?", req.ItemID).
			Where("bqi.status <> ?", billingqueue.StatusCanceled).
			Where("bqi.is_adjustment_origin = FALSE").
			Where("bqi.source_invoice_id IS NULL").
			Order("bqi.created_at ASC").
			Limit(5).
			Scan(ctx, &items); err != nil {
			return nil, fmt.Errorf("find duplicate billing queue items: %w", err)
		}
		for _, item := range items {
			out = append(out, &billingqueue.DuplicateRef{
				Kind: "item", ID: item.ID, Number: item.Number, Status: item.Status,
			})
		}

		var invoices []struct {
			ID     pulid.ID `bun:"id"`
			Number string   `bun:"number"`
			Status string   `bun:"status"`
		}
		q := db.NewSelect().
			TableExpr("invoices AS inv").
			Column("inv.id", "inv.number", "inv.status").
			Apply(tenantWhere("inv", req.TenantInfo)).
			Where("inv.customer_id = ?", req.BillToCustomerID).
			Where("inv.bill_type = ?", req.BillType).
			Where("inv.status <> 'Voided'").
			Where("inv.is_adjustment_artifact = FALSE").
			Where("inv.billing_queue_item_id <> ?", req.ItemID).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("inv.shipment_id = ?", req.ShipmentID).
					WhereOr(`EXISTS (SELECT 1 FROM invoice_lines AS invl
						WHERE invl.invoice_id = inv.id
						AND invl.organization_id = inv.organization_id
						AND invl.business_unit_id = inv.business_unit_id
						AND invl.shipment_id = ?)`, req.ShipmentID)
			})
		if req.InvoiceID.IsNotNil() {
			q = q.Where("inv.id <> ?", req.InvoiceID)
		}
		if err := q.Order("inv.created_at ASC").Limit(5).Scan(ctx, &invoices); err != nil {
			return nil, fmt.Errorf("find duplicate invoices: %w", err)
		}
		for _, inv := range invoices {
			out = append(out, &billingqueue.DuplicateRef{
				Kind: "invoice", ID: inv.ID, Number: inv.Number, Status: inv.Status,
			})
		}

		return out, nil
	})
}

// ListSummaries reads the rows' live state in one query: each item with how
// many of its checks wait on an open issue.
func (r *repository) ListSummaries(
	ctx context.Context,
	req *repositories.ListBillingQueueSummariesRequest,
) ([]*repositories.BillingQueueItemSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*repositories.BillingQueueItemSummary, error) {
		out := make([]*repositories.BillingQueueItemSummary, 0, len(req.ItemIDs))
		if len(req.ItemIDs) == 0 {
			return out, nil
		}
		db := r.db.DBForContext(ctx)
		open := db.NewSelect().
			TableExpr("billing_queue_issues AS bqis").
			ColumnExpr("COUNT(DISTINCT bqis.check_key)").
			Where("bqis.item_id = bqi.id").
			Where("bqis.organization_id = bqi.organization_id").
			Where("bqis.business_unit_id = bqi.business_unit_id").
			Where("bqis.resolution_key IS NULL")

		if err := db.NewSelect().
			TableExpr("billing_queue_items AS bqi").
			Column(
				"bqi.id", "bqi.number", "bqi.status", "bqi.hold_reason_code",
				"bqi.assigned_biller_id", "bqi.allocated_total_amount",
			).
			ColumnExpr("(?) AS open_checks", open).
			Apply(tenantWhere("bqi", req.TenantInfo)).
			Where("bqi.id IN (?)", bun.In(req.ItemIDs)).
			Scan(ctx, &out); err != nil {
			return nil, fmt.Errorf("list billing queue summaries: %w", err)
		}

		for _, summary := range out {
			needs := summary.OpenChecks
			if summary.AssignedBillerID == nil || summary.AssignedBillerID.IsNil() {
				needs++
			}
			summary.NeedsCount = needs
			summary.Ready = billingqueue.IsReviewable(summary.Status) && needs == 0
		}

		return out, nil
	})
}
