package billingtools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeQueueItems struct {
	item *billingqueue.BillingQueueItem
}

func (f *fakeQueueItems) GetByID(
	context.Context,
	*billingqueueports.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	copied := *f.item
	return &copied, nil
}

type fakeQueueReview struct {
	guard    agenttoolstest.WriteGuard
	item     *billingqueue.BillingQueueItem
	plan     *billingqueueports.IssueSettlementPlan
	refuse   error
	released []*billingqueueports.BillingQueueItemRequest
	resolved []*billingqueueports.ResolveBillingQueueIssueRequest
	undone   []*billingqueueports.UndoBillingQueueIssueRequest
	actors   []*permissionports.RequestActor
}

func (f *fakeQueueReview) Release(
	_ context.Context,
	req *billingqueueports.BillingQueueItemRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.released = append(f.released, req)
	f.actors = append(f.actors, actor)
	released := *f.item
	released.Status = billingqueue.StatusReadyForReview
	return &released, nil
}

func (f *fakeQueueReview) ResolveIssue(
	_ context.Context,
	req *billingqueueports.ResolveBillingQueueIssueRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.resolved = append(f.resolved, req)
	f.actors = append(f.actors, actor)
	return f.item, nil
}

func (f *fakeQueueReview) UndoIssue(
	_ context.Context,
	req *billingqueueports.UndoBillingQueueIssueRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.undone = append(f.undone, req)
	f.actors = append(f.actors, actor)
	return f.item, nil
}

func (f *fakeQueueReview) PlanResolveIssue(
	context.Context,
	*billingqueueports.ResolveBillingQueueIssueRequest,
) (*billingqueueports.IssueSettlementPlan, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	return f.plan, nil
}

func (f *fakeQueueReview) PlanUndoIssue(
	context.Context,
	*billingqueueports.UndoBillingQueueIssueRequest,
) (*billingqueueports.IssueSettlementPlan, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	return f.plan, nil
}

func reviewedQueueItem(status billingqueue.Status) *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		ID:         pulid.MustNew("bqi_"),
		Number:     "BQ-0042",
		ShipmentID: pulid.MustNew("shp_"),
		Status:     status,
		Version:    4,
		Shipment:   &shipment.Shipment{ProNumber: "PRO-881"},
	}
}

func detentionCharge(amount string) *shipment.AdditionalCharge {
	return &shipment.AdditionalCharge{
		ID:     pulid.MustNew("shpac_"),
		Method: accessorialcharge.MethodFlat,
		Amount: decimal.RequireFromString(amount),
		Unit:   1,
	}
}

func detentionIssue() *billingqueue.Issue {
	return &billingqueue.Issue{
		ID:       pulid.MustNew("bqis_"),
		CheckKey: billingqueue.CheckCharges,
		Code:     billingqueue.IssueDetentionELDMismatch,
		Summary:  "Detention doesn't match the ELD",
	}
}

func issueParams(
	item *billingqueue.BillingQueueItem,
	issue *billingqueue.Issue,
	option string,
) agentports.ToolExecuteParams {
	values := map[string]any{
		common.ParamBillingQueueItemID: item.ID.String(),
		paramIssueID:                   issue.ID.String(),
	}
	if option != "" {
		values[paramOptionKey] = option
	}

	return agenttoolstest.ExecuteParams(values)
}

func TestReleaseBillingQueueItem_PreviewsTheStatusTheHoldFoundThenReleases(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusOnHold)
	before := billingqueue.StatusReadyForReview
	item.StatusBeforeHold = &before
	review := &fakeQueueReview{item: item}
	tool := newReleaseBillingQueueItemTool(&fakeQueueItems{item: item}, review)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamBillingQueueItemID: item.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &review.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := agenttoolstest.PreviewChange(t, preview, 0)
	status := agenttoolstest.FieldByPath(t, change, common.FieldStatus)
	assert.Equal(t, string(billingqueue.StatusOnHold), status.Before)
	assert.Equal(t, string(billingqueue.StatusReadyForReview), status.After)
	assert.Contains(t, preview.Summary, "BQ-0042")
	assert.Empty(t, review.released)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, review.released, 1)
	assert.Equal(t, item.ID, review.released[0].ItemID)
	assert.Same(t, params.Actor, review.actors[0])
}

func TestReleaseBillingQueueItem_RefusesAnItemThatIsNotOnHold(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	review := &fakeQueueReview{item: item}
	tool := newReleaseBillingQueueItemTool(&fakeQueueItems{item: item}, review)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamBillingQueueItemID: item.ID.String(),
	})

	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
}

func TestResolveBillingQueueIssue_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	issue := detentionIssue()
	review := &fakeQueueReview{item: item}
	tool := newResolveBillingQueueIssueTool(review)
	params := issueParams(item, issue, "drop")

	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.True(t, errors.Is(err, common.ErrNeedsAPersonsApproval))
	assert.Empty(t, review.resolved)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	require.Len(t, review.resolved, 1)
	resolved := review.resolved[0]
	assert.Equal(t, item.ID, resolved.ItemID)
	assert.Equal(t, issue.ID, resolved.IssueID)
	assert.Equal(t, "drop", resolved.OptionKey)
}

func TestResolveBillingQueueIssue_AlwaysAProposalAndClassifiedByTheChoice(t *testing.T) {
	t.Parallel()

	policy := newResolveBillingQueueIssueTool(nil).Policy()
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t,
		[]agent.EgressClass{agent.EgressMoney, agent.EgressDriverVisible},
		policy.Egress,
	)
	require.NotNil(t, policy.Classify)

	asks := agenttoolstest.ExecuteParams(
		map[string]any{paramOptionKey: billingqueue.OptionKeyRequest},
	)
	assert.Equal(t, agent.EgressDriverVisible, policy.Classify(asks).Egress)
	drops := agenttoolstest.ExecuteParams(map[string]any{paramOptionKey: "drop"})
	assert.Equal(t, agent.EgressMoney, policy.Classify(drops).Egress)

	undo := newUndoBillingQueueIssueTool(nil).Policy()
	assert.Equal(t, agent.TierPropose, undo.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, undo.Egress)
}

func TestResolveBillingQueueIssue_PreviewShowsTheSettlementAndTheChargeItRemoves(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	issue := detentionIssue()
	charge := detentionCharge("150.00")
	review := &fakeQueueReview{item: item, plan: &billingqueueports.IssueSettlementPlan{
		Item:  item,
		Issue: issue,
		Option: &billingqueue.IssueOption{
			Key:    "drop",
			Label:  "Remove detention",
			Done:   "Detention removed",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectDrop, ChargeID: charge.ID},
		},
		ChargeBefore: charge,
	}}
	tool := newResolveBillingQueueIssueTool(review)

	preview := agenttoolstest.PreviewWithoutWrites(t, &review.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agentports.ToolPreviewer).Preview(
			t.Context(),
			issueParams(item, issue, "drop"),
		)
	})
	require.Len(t, preview.Changes, 2)
	check := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, checkStateSettled, agenttoolstest.FieldByPath(t, check, "state").After)
	removed := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationDelete, removed.Operation)
	assert.Contains(t, preview.Summary, "Remove detention")
	assert.Contains(t, preview.Summary, "removing")
}

func TestResolveBillingQueueIssue_AskingTheDriverPreviewsTheirMessage(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	issue := &billingqueue.Issue{
		ID:       pulid.MustNew("bqis_"),
		CheckKey: billingqueue.CheckPOD,
		Code:     billingqueue.IssuePODUnsigned,
		Summary:  "The POD isn't signed",
	}
	review := &fakeQueueReview{item: item, plan: &billingqueueports.IssueSettlementPlan{
		Item:  item,
		Issue: issue,
		Option: &billingqueue.IssueOption{
			Key:    billingqueue.OptionKeyRequest,
			Label:  "Ask the driver for it",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectRequest},
		},
		AsksDriver:         true,
		DriverName:         "Sam Ortiz",
		DriverMessageTitle: "Signed POD needed for PRO-881",
		DriverMessage:      "Billing needs the receiver's signed POD for PRO-881.",
	}}
	tool := newResolveBillingQueueIssueTool(review)

	preview := agenttoolstest.PreviewWithoutWrites(t, &review.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agentports.ToolPreviewer).Preview(
			t.Context(),
			issueParams(item, issue, billingqueue.OptionKeyRequest),
		)
	})
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationSend, change.Operation)
	require.NotNil(t, change.Message)
	assert.Equal(t, agent.MessageChannelDash, change.Message.Channel)
	assert.Equal(t, "Signed POD needed for PRO-881", change.Message.Subject)
	assert.Contains(t, change.Label, "Sam Ortiz")
	assert.Contains(t, preview.Summary, "stays open")
}

func TestResolveBillingQueueIssue_AChoiceTheCheckDoesNotOfferWouldFail(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	issue := detentionIssue()
	review := &fakeQueueReview{item: item, refuse: errortypes.NewValidationError(
		"optionKey", errortypes.ErrInvalid, "That isn't one of this check's choices",
	)}
	tool := newResolveBillingQueueIssueTool(review)
	params := issueParams(item, issue, "bogus")

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
}

func TestUndoBillingQueueIssue_PreviewPutsTheRemovedChargeBack(t *testing.T) {
	t.Parallel()

	item := reviewedQueueItem(billingqueue.StatusInReview)
	issue := detentionIssue()
	key := "drop"
	issue.ResolutionKey = &key
	issue.ResolutionText = "Detention removed"
	review := &fakeQueueReview{item: item, plan: &billingqueueports.IssueSettlementPlan{
		Item:  item,
		Issue: issue,
		Option: &billingqueue.IssueOption{
			Key:    "drop",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectDrop},
		},
		ChargeAfter: detentionCharge("150.00"),
	}}
	tool := newUndoBillingQueueIssueTool(review)
	params := issueParams(item, issue, "")

	preview := agenttoolstest.PreviewWithoutWrites(t, &review.guard, func() (
		*agent.ToolPreview,
		error,
	) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	check := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, checkStateOpen, agenttoolstest.FieldByPath(t, check, "state").After)
	restored := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, restored.Operation)
	assert.Contains(t, preview.Summary, "Detention removed")

	require.Error(t, tool.Execute(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	require.Len(t, review.undone, 1)
	assert.Equal(t, issue.ID, review.undone[0].IssueID)
}
