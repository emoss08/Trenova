package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *repository) attachHeadlines(
	ctx context.Context,
	db *keel.DB,
	items []*billingqueue.BillingQueueItem,
) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	issues, err := openIssues(ids).All(ctx, db)
	if err != nil {
		return err
	}

	byItem := make(map[pulid.ID][]*billingqueue.Issue, len(items))
	for _, issue := range issues {
		byItem[issue.ItemID] = append(byItem[issue.ItemID], issue)
	}
	for _, item := range items {
		item.Headline = headlineOf(byItem[item.ID])
	}

	return nil
}

func openIssues(itemIDs []pulid.ID) *keel.Query[*billingqueue.Issue] {
	iss := billingqueuedb.Issues

	return iss.Pick(iss.ItemID, iss.CheckKey, iss.Code, iss.Summary, iss.RequestedAt, iss.CreatedAt).
		Where(
			iss.ItemID.In(itemIDs...),
			iss.ResolutionKey.IsNull(),
			iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
		).
		Order(iss.ItemID.Asc(), iss.CreatedAt.Asc(), iss.ID.Asc())
}

func headlineOf(issues []*billingqueue.Issue) *billingqueue.Headline {
	if len(issues) == 0 {
		return nil
	}

	lead := issues[0]
	for _, issue := range issues {
		if !waits(issue) {
			lead = issue
			break
		}
	}

	return &billingqueue.Headline{
		Code:     lead.Code,
		CheckKey: lead.CheckKey,
		Summary:  lead.Summary,
		Open:     len(issues),
		Waiting:  waits(lead),
	}
}

func waits(issue *billingqueue.Issue) bool {
	return issue.Code.WaitsOnOthers() || issue.RequestedAt != nil
}
