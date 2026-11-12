package billingqueueprojection

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	openIssueAlias       = "prj_open"
	actionableIssueAlias = "prj_act"
)

func Refresh(ctx context.Context, db *keel.DB, itemIDs ...pulid.ID) error {
	ids := distinct(itemIDs)
	if len(ids) == 0 {
		return nil
	}

	bq := billingqueuedb.BillingQueueItems
	lane := laneExpr(ctx, bq)
	if _, err := tenantscope.Scoped(ctx, bq.Where(
		bq.ID.In(ids...),
		bq.Lane.Text().IsDistinctFromExpr(lane),
	)).Update(
		ctx,
		db,
		bq.Lane.ToExpr(lane),
		bq.Version.ToExpr(bq.Version),
		bq.UpdatedAt.ToExpr(bq.UpdatedAt),
	); err != nil {
		return fmt.Errorf("refresh billing queue lanes: %w", err)
	}

	return nil
}

func MarkChecked(ctx context.Context, db *keel.DB, at int64, itemIDs ...pulid.ID) error {
	ids := distinct(itemIDs)
	if len(ids) == 0 {
		return nil
	}

	bq := billingqueuedb.BillingQueueItems
	if _, err := tenantscope.Scoped(ctx, bq.Where(bq.ID.In(ids...))).Update(
		ctx,
		db,
		bq.CheckedAt.To(at),
		bq.Version.ToExpr(bq.Version),
		bq.UpdatedAt.ToExpr(bq.UpdatedAt),
	); err != nil {
		return fmt.Errorf("mark billing queue items checked: %w", err)
	}

	return Refresh(ctx, db, ids...)
}

func laneExpr(ctx context.Context, bq *billingqueuedb.BillingQueueItemsTable) keel.StringExpr {
	return keel.CaseString().
		When(bq.Status.In(billingqueue.ClosedStatuses()...), lane(billingqueue.LaneClosed)).
		When(bq.Status.Eq(billingqueue.StatusApproved), lane(billingqueue.LaneApproved)).
		When(bq.Status.In(billingqueue.HeldStatuses()...), lane(billingqueue.LaneHeld)).
		When(bq.CheckedAt.IsNull(), lane(billingqueue.LaneChecking)).
		When(
			keel.Or(bq.AssignedBillerID.IsNull(), HasActionableIssue(ctx, bq)),
			lane(billingqueue.LaneNeedsYou),
		).
		When(keel.Exists(openIssues(ctx, bq)), lane(billingqueue.LaneWaiting)).
		Else(lane(billingqueue.LaneReady))
}

func HasActionableIssue(ctx context.Context, bq *billingqueuedb.BillingQueueItemsTable) keel.Pred {
	return keel.Exists(actionableIssues(ctx, bq))
}

func openIssues(
	ctx context.Context,
	bq *billingqueuedb.BillingQueueItemsTable,
) keel.Query[*billingqueue.Issue] {
	iss := billingqueuedb.Issues.As(openIssueAlias)

	return tenantscope.Scoped(ctx, iss.Pick(iss.ID).Where(
		iss.ItemID.EqCol(bq.ID),
		iss.OrganizationID.EqCol(bq.OrganizationID),
		iss.BusinessUnitID.EqCol(bq.BusinessUnitID),
		iss.ResolutionKey.IsNull(),
		iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
	))
}

func actionableIssues(
	ctx context.Context,
	bq *billingqueuedb.BillingQueueItemsTable,
) keel.Query[*billingqueue.Issue] {
	iss := billingqueuedb.Issues.As(actionableIssueAlias)

	return tenantscope.Scoped(ctx, iss.Pick(iss.ID).Where(
		iss.ItemID.EqCol(bq.ID),
		iss.OrganizationID.EqCol(bq.OrganizationID),
		iss.BusinessUnitID.EqCol(bq.BusinessUnitID),
		iss.ResolutionKey.IsNull(),
		iss.RequestedAt.IsNull(),
		iss.Code.NotIn(billingqueue.WaitingIssueCodes()...),
		iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
	))
}

// Unclosed leaves out the items in the closed lane, posted or canceled, unless
// they are asked for.
func Unclosed(includeClosed bool) keel.Pred {
	bq := billingqueuedb.BillingQueueItems

	return keel.When(!includeClosed, bq.Lane.Ne(billingqueue.LaneClosed))
}

func lane(l billingqueue.Lane) keel.StringExpr {
	return keel.Text(string(l))
}

func distinct(ids []pulid.ID) []pulid.ID {
	return sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(ids), pulid.ID.IsNil))
}
