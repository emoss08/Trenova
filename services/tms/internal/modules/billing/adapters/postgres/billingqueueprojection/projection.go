package billingqueueprojection

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/pkg/dbscope"
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
	allTenants := dbscope.IsSystem(ctx)
	lane := laneExpr(bq, allTenants)
	query := bq.Query()
	if allTenants {
		query = query.AllTenants()
	}
	if _, err := query.Where(
		bq.ID.In(ids...),
		bq.Lane.IsDistinctFromExpr(lane),
	).Update(
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
	query := bq.Query()
	if dbscope.IsSystem(ctx) {
		query = query.AllTenants()
	}
	if _, err := query.Where(bq.ID.In(ids...)).Update(
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

func LaneExpr(bq *billingqueuedb.BillingQueueItemsTable) keel.StringExpr {
	return laneExpr(bq, false)
}

func laneExpr(bq *billingqueuedb.BillingQueueItemsTable, allTenants bool) keel.StringExpr {
	return keel.CaseString().
		When(bq.Status.In(billingqueue.ClosedStatuses()...), lane(billingqueue.LaneClosed)).
		When(bq.Status.Eq(billingqueue.StatusApproved), lane(billingqueue.LaneApproved)).
		When(bq.Status.In(billingqueue.HeldStatuses()...), lane(billingqueue.LaneHeld)).
		When(bq.CheckedAt.IsNull(), lane(billingqueue.LaneChecking)).
		When(
			keel.Or(bq.AssignedBillerID.IsNull(), keel.Exists(actionableIssues(bq, allTenants))),
			lane(billingqueue.LaneNeedsYou),
		).
		When(keel.Exists(openIssues(bq, allTenants)), lane(billingqueue.LaneWaiting)).
		Else(lane(billingqueue.LaneReady))
}

func HasOpenIssue(bq *billingqueuedb.BillingQueueItemsTable) keel.Pred {
	return keel.Exists(openIssues(bq, false))
}

func HasActionableIssue(bq *billingqueuedb.BillingQueueItemsTable) keel.Pred {
	return keel.Exists(actionableIssues(bq, false))
}

func openIssues(
	bq *billingqueuedb.BillingQueueItemsTable,
	allTenants bool,
) keel.Query[*billingqueue.Issue] {
	iss := billingqueuedb.Issues.As(openIssueAlias)

	query := iss.Pick(iss.ID)
	if allTenants {
		query = query.AllTenants()
	}

	return query.Where(
		iss.ItemID.EqCol(bq.ID),
		iss.OrganizationID.EqCol(bq.OrganizationID),
		iss.BusinessUnitID.EqCol(bq.BusinessUnitID),
		iss.ResolutionKey.IsNull(),
		iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
	)
}

func actionableIssues(
	bq *billingqueuedb.BillingQueueItemsTable,
	allTenants bool,
) keel.Query[*billingqueue.Issue] {
	iss := billingqueuedb.Issues.As(actionableIssueAlias)

	query := iss.Pick(iss.ID)
	if allTenants {
		query = query.AllTenants()
	}

	return query.Where(
		iss.ItemID.EqCol(bq.ID),
		iss.OrganizationID.EqCol(bq.OrganizationID),
		iss.BusinessUnitID.EqCol(bq.BusinessUnitID),
		iss.ResolutionKey.IsNull(),
		iss.RequestedAt.IsNull(),
		iss.Code.NotIn(billingqueue.WaitingIssueCodes()...),
		iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
	)
}

func lane(l billingqueue.Lane) keel.StringExpr {
	return keel.Text(string(l))
}

func distinct(ids []pulid.ID) []pulid.ID {
	return sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(ids), pulid.ID.IsNil))
}
