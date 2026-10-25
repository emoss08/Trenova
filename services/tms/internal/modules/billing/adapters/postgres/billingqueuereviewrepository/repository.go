package billingqueuereviewrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	issueErrors = dberror.Mapping{Entity: "Billing queue issue"}
	eventErrors = dberror.Mapping{Entity: "Billing queue event"}
	itemErrors  = dberror.Mapping{Entity: "Billing queue item"}
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

func New(p Params) billingqueueports.BillingQueueReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-review-repository"),
	}
}

func (r *repository) ListIssues(
	ctx context.Context,
	_ pagination.TenantInfo,
	itemID pulid.ID,
) ([]*billingqueue.Issue, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*billingqueue.Issue, error) {
		iss := billingqueuedb.Issues
		issues, err := iss.Where(iss.ItemID.Eq(itemID)).
			Order(iss.CreatedAt.Asc(), iss.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if issues == nil {
			return []*billingqueue.Issue{}, nil
		}

		return issues, nil
	})
}

func (r *repository) SyncIssues(
	ctx context.Context,
	req *billingqueueports.SyncBillingQueueIssuesRequest,
) (*billingqueueports.SyncBillingQueueIssuesResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueueports.SyncBillingQueueIssuesResult, error) {
			existing, err := r.ListIssues(ctx, req.TenantInfo, req.ItemID)
			if err != nil {
				return nil, err
			}
			byKey := make(map[string]*billingqueue.Issue, len(existing))
			for _, issue := range existing {
				byKey[issue.FindingKey()] = issue
			}

			result := &billingqueueports.SyncBillingQueueIssuesResult{}
			found := make(map[string]struct{}, len(req.Findings))
			for _, finding := range req.Findings {
				key := finding.FindingKey()
				found[key] = struct{}{}
				if err = r.landFinding(ctx, req, byKey[key], finding, result); err != nil {
					return nil, err
				}
			}

			now := timeutils.NowUnix()
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
		},
	)
}

func (r *repository) landFinding(
	ctx context.Context,
	req *billingqueueports.SyncBillingQueueIssuesRequest,
	current, finding *billingqueue.Issue,
	result *billingqueueports.SyncBillingQueueIssuesResult,
) error {
	switch {
	case current == nil:
		finding.ItemID = req.ItemID
		iss := billingqueuedb.Issues
		n, err := iss.CreateOrSkip(ctx, r.db.Keel(ctx), []*billingqueue.Issue{finding},
			iss.OrganizationID, iss.BusinessUnitID, iss.ItemID, iss.Code, iss.SubjectKey)
		if err != nil {
			return issueErrors.Map(err, finding.ID.String())
		}
		if n > 0 {
			result.Raised = append(result.Raised, finding)
		}
	case current.ResolutionKey != nil && *current.ResolutionKey == billingqueue.ResolutionCleared:
		current.ResolutionKey = nil
		current.ResolutionText = ""
		current.ResolvedAt = nil
		current.ResolvedByID = nil
		refreshFinding(current, finding)
		if _, err := r.UpdateIssue(ctx, current); err != nil {
			return err
		}
		result.Reopened = append(result.Reopened, current)
	case current.IsOpen() && findingChanged(current, finding):
		refreshFinding(current, finding)
		if _, err := r.UpdateIssue(ctx, current); err != nil {
			return err
		}
	}

	return nil
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
	req *billingqueueports.GetBillingQueueIssueRequest,
) (*billingqueue.Issue, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.Issue, error) {
		iss := billingqueuedb.Issues
		issue, err := iss.Where(iss.ItemID.Eq(req.ItemID), iss.ID.Eq(req.IssueID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, issueErrors.Map(err, req.IssueID.String())
		}

		return issue, nil
	})
}

func (r *repository) UpdateIssue(
	ctx context.Context,
	issue *billingqueue.Issue,
) (*billingqueue.Issue, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingqueue.Issue, error) {
		iss := billingqueuedb.Issues
		err := iss.Save(ctx, r.db.Keel(ctx), issue,
			iss.CheckKey,
			iss.SubjectKey,
			iss.Summary,
			iss.Reasoning,
			iss.FlaggedChargeID,
			iss.Options,
			iss.ResolutionKey,
			iss.ResolutionText,
			iss.EffectSnapshot,
			iss.ResolvedByID,
			iss.ResolvedAt,
			iss.RequestedAt,
			iss.RequestedByID,
		)
		if err != nil {
			return nil, issueErrors.Map(err, issue.ID.String())
		}

		return issue, nil
	})
}

func (r *repository) CountOpenIssues(
	ctx context.Context,
	_ pagination.TenantInfo,
	itemID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		iss := billingqueuedb.Issues
		n, err := iss.Where(iss.ItemID.Eq(itemID), iss.ResolutionKey.IsNull()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) CreateEvents(ctx context.Context, events ...*billingqueue.ItemEvent) error {
	if len(events) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := billingqueuedb.ItemEvents.CreateMany(ctx, r.db.Keel(ctx), events)
		return eventErrors.Map(err, "")
	})
}

func (r *repository) ListEvents(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueEventsRequest,
) (*billingqueueports.BillingQueueEventPage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueueports.BillingQueueEventPage, error) {
			limit := req.Limit
			if limit <= 0 || limit > 100 {
				limit = 30
			}

			e := billingqueuedb.ItemEvents
			var (
				total  int64
				events []*billingqueue.ItemEvent
			)
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.CountInto(e.Where(e.ItemID.Eq(req.ItemID)), &total),
				keel.ListInto(e.Where(
					e.ItemID.Eq(req.ItemID),
					keel.When(
						req.BeforeAt > 0,
						keel.DynRow(e.At, e.ID).Lt(keel.DynRow(req.BeforeAt, req.BeforeID.String())),
					),
				).Order(e.At.Desc(), e.ID.Desc()).Limit(limit+1), &events),
			)
			if err != nil {
				return nil, err
			}

			page := &billingqueueports.BillingQueueEventPage{Total: int(total)}
			if len(events) > limit {
				page.HasMore = true
				events = events[:limit]
			}
			if events == nil {
				events = []*billingqueue.ItemEvent{}
			}
			page.Items = events

			return page, nil
		},
	)
}

func (r *repository) GetNeighbors(
	ctx context.Context,
	req *billingqueueports.GetBillingQueueNeighborsRequest,
) (*billingqueueports.BillingQueueNeighbors, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueueports.BillingQueueNeighbors, error) {
			db := r.db.Keel(ctx)
			bq := billingqueuedb.BillingQueueItems
			anchor, err := bq.Pick(bq.CreatedAt).Where(bq.ID.Eq(req.ItemID)).One(ctx, db)
			if err != nil {
				return nil, itemErrors.Map(err, req.ItemID.String())
			}

			n := neighborQueries(req, anchor.CreatedAt)
			var (
				total, before int64
				member        bool
				prev, next    []pulid.ID
			)
			err = keel.Batch(ctx, db,
				keel.CountInto(n.base(), &total),
				keel.CountInto(n.base().Where(n.after), &before),
				keel.ExistsInto(n.base().Where(bq.ID.Eq(req.ItemID)), &member),
				keel.IntoSlice(n.base().Pick(bq.ID).Where(n.after).
					Order(bq.CreatedAt.Asc(), bq.ID.Asc()).Limit(1), &prev),
				keel.IntoSlice(n.base().Pick(bq.ID).Where(n.beforeAnchor).
					Order(bq.CreatedAt.Desc(), bq.ID.Desc()).Limit(1), &next),
			)
			if err != nil {
				return nil, err
			}

			out := &billingqueueports.BillingQueueNeighbors{Total: int(total)}
			if member {
				out.Position = int(before + 1)
			}
			if len(prev) > 0 {
				out.PrevID = &prev[0]
			}
			if len(next) > 0 {
				out.NextID = &next[0]
			}

			return out, nil
		},
	)
}

type neighbors struct {
	base         func() *keel.Query[billingqueue.BillingQueueItem]
	after        keel.Pred
	beforeAnchor keel.Pred
}

func neighborQueries(
	req *billingqueueports.GetBillingQueueNeighborsRequest,
	createdAt int64,
) neighbors {
	bq := billingqueuedb.BillingQueueItems
	id := req.ItemID.String()

	return neighbors{
		base: func() *keel.Query[billingqueue.BillingQueueItem] {
			return listquery.Filter(
				bq.Where(keel.When(!req.IncludePosted, bq.Status.Ne(billingqueue.StatusPosted))),
				req.Filter,
			)
		},
		after:        keel.DynRow(bq.CreatedAt, bq.ID).Gt(keel.DynRow(createdAt, id)),
		beforeAnchor: keel.DynRow(bq.CreatedAt, bq.ID).Lt(keel.DynRow(createdAt, id)),
	}
}

type duplicateRow struct {
	ID     pulid.ID
	Number string
	Status string
}

func (r *repository) FindDuplicates(
	ctx context.Context,
	req *billingqueueports.FindBillingQueueDuplicatesRequest,
) ([]*billingqueue.DuplicateRef, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*billingqueue.DuplicateRef, error) {
		if req.ShipmentID.IsNil() {
			return nil, nil
		}

		var items, invoices []duplicateRow
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(duplicateItems(req), &items),
			keel.IntoSlice(duplicateInvoices(req), &invoices),
		)
		if err != nil {
			return nil, err
		}

		out := make([]*billingqueue.DuplicateRef, 0, len(items)+len(invoices))
		for _, item := range items {
			out = append(out, &billingqueue.DuplicateRef{
				Kind: "item", ID: item.ID, Number: item.Number, Status: item.Status,
			})
		}
		for _, inv := range invoices {
			out = append(out, &billingqueue.DuplicateRef{
				Kind: "invoice", ID: inv.ID, Number: inv.Number, Status: inv.Status,
			})
		}

		return out, nil
	})
}

func duplicateItems(
	req *billingqueueports.FindBillingQueueDuplicatesRequest,
) *keel.Query[billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems
	return bq.Pick(bq.ID, bq.Number, bq.Status).
		Where(
			bq.ShipmentID.Eq(req.ShipmentID),
			bq.BillToCustomerID.Eq(req.BillToCustomerID),
			bq.BillType.Eq(req.BillType),
			bq.ID.Ne(req.ItemID),
			bq.Status.Ne(billingqueue.StatusCanceled),
			bq.IsAdjustmentOrigin.IsFalse(),
			bq.SourceInvoiceID.IsNull(),
		).
		Order(bq.CreatedAt.Asc()).
		Limit(5)
}

func duplicateInvoices(
	req *billingqueueports.FindBillingQueueDuplicatesRequest,
) *keel.Query[invoice.Invoice] {
	inv := invoicedb.Invoices
	lines := inv.Lines()
	return inv.Pick(inv.ID, inv.Number, inv.Status).
		Where(
			inv.CustomerID.Eq(req.BillToCustomerID),
			inv.BillType.Eq(req.BillType),
			inv.Status.Ne(invoice.StatusVoided),
			inv.IsAdjustmentArtifact.IsFalse(),
			inv.BillingQueueItemID.Ne(req.ItemID),
			keel.Or(inv.ShipmentID.Eq(req.ShipmentID), lines.Any(lines.ShipmentID.Eq(req.ShipmentID))),
			keel.When(req.InvoiceID.IsNotNil(), inv.ID.Ne(req.InvoiceID)),
		).
		Order(inv.CreatedAt.Asc()).
		Limit(5)
}

type summaryRow struct {
	ID                   pulid.ID
	Number               string
	Status               billingqueue.Status
	HoldReasonCode       *billingqueue.HoldReasonCode
	AssignedBillerID     *pulid.ID
	AllocatedTotalAmount decimal.Decimal
	OpenChecks           int
}

func (r *repository) ListSummaries(
	ctx context.Context,
	req *billingqueueports.ListBillingQueueSummariesRequest,
) ([]*billingqueueports.BillingQueueItemSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.BillingQueueItemSummary, error) {
			out := make([]*billingqueueports.BillingQueueItemSummary, 0, len(req.ItemIDs))
			if len(req.ItemIDs) == 0 {
				return out, nil
			}

			rows, err := keel.Into[summaryRow](ctx, r.db.Keel(ctx), itemSummaries(req.ItemIDs))
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				needs := row.OpenChecks
				if row.AssignedBillerID == nil || row.AssignedBillerID.IsNil() {
					needs++
				}
				out = append(out, &billingqueueports.BillingQueueItemSummary{
					ID:                   row.ID,
					Number:               row.Number,
					Status:               row.Status,
					HoldReasonCode:       row.HoldReasonCode,
					AssignedBillerID:     row.AssignedBillerID,
					AllocatedTotalAmount: row.AllocatedTotalAmount,
					OpenChecks:           row.OpenChecks,
					NeedsCount:           needs,
					Ready:                billingqueue.IsReviewable(row.Status) && needs == 0,
				})
			}

			return out, nil
		},
	)
}

func itemSummaries(itemIDs []pulid.ID) *keel.Query[billingqueue.BillingQueueItem] {
	bq := billingqueuedb.BillingQueueItems
	iss := billingqueuedb.Issues
	open := keel.Scalar(iss.Pick(iss.CheckKey.CountDistinct()).Where(
		iss.ItemID.EqCol(bq.ID),
		iss.ResolutionKey.IsNull(),
	))

	return bq.Pick(
		bq.ID,
		bq.Number,
		bq.Status,
		bq.HoldReasonCode,
		bq.AssignedBillerID,
		bq.AllocatedTotalAmount,
		open,
	).Where(bq.ID.In(itemIDs...)).Limit(len(itemIDs))
}
