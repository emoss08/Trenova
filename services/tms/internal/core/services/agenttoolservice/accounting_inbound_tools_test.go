package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundOperator struct {
	changes  map[pulid.ID]*accountingsync.AccountingInboundChange
	previews map[pulid.ID]*serviceports.AccountingInboundApplyPreview

	applied      *serviceports.DecideAccountingInboundChangeRequest
	appliedActor *serviceports.RequestActor
	ignored      *serviceports.DecideAccountingInboundChangeRequest
}

func (f *fakeInboundOperator) Get(
	_ context.Context,
	req *serviceports.GetAccountingInboundChangeRequest,
) (*accountingsync.AccountingInboundChange, error) {
	change, ok := f.changes[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
	}
	out := *change
	return &out, nil
}

func (f *fakeInboundOperator) PreviewApply(
	_ context.Context,
	req *serviceports.GetAccountingInboundChangeRequest,
) (*serviceports.AccountingInboundApplyPreview, error) {
	preview, ok := f.previews[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
	}
	return preview, nil
}

func (f *fakeInboundOperator) Apply(
	_ context.Context,
	req *serviceports.DecideAccountingInboundChangeRequest,
	actor *serviceports.RequestActor,
) (*accountingsync.AccountingInboundChange, error) {
	f.applied = req
	f.appliedActor = actor
	return f.changes[req.ID], nil
}

func (f *fakeInboundOperator) Ignore(
	_ context.Context,
	req *serviceports.DecideAccountingInboundChangeRequest,
	_ *serviceports.RequestActor,
) (*accountingsync.AccountingInboundChange, error) {
	f.ignored = req
	return f.changes[req.ID], nil
}

func proposedPayment(status accountingsync.InboundChangeStatus) *accountingsync.AccountingInboundChange {
	return &accountingsync.AccountingInboundChange{
		ID:             pulid.MustNew("acctic_"),
		Kind:           accountingsync.InboundCustomerPayment,
		Status:         status,
		Reason:         accountingsync.InboundReasonPolicyPropose,
		ExternalNumber: "10442",
		PartyName:      "Acme Foods",
		AmountMinor:    150_025,
		CurrencyCode:   "USD",
	}
}

func newInboundOperator(
	change *accountingsync.AccountingInboundChange,
	preview *serviceports.AccountingInboundApplyPreview,
) *fakeInboundOperator {
	preview.Change = change
	return &fakeInboundOperator{
		changes:  map[pulid.ID]*accountingsync.AccountingInboundChange{change.ID: change},
		previews: map[pulid.ID]*serviceports.AccountingInboundApplyPreview{change.ID: preview},
	}
}

func TestApplyAccountingInboundChange_PreviewsWhatItPostsAndAppliesAsTheApprover(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{
		CanApply:       true,
		PaidAt:         1_790_208_000,
		CashMinor:      145_025,
		UnappliedMinor: 5_000,
		Lines: []serviceports.AccountingInboundPostingLine{{
			ObjectType:   accountingsync.SyncObjectInvoice,
			ObjectID:     pulid.MustNew("inv_"),
			ObjectNumber: "INV-1001",
			AmountMinor:  145_025,
			OpenMinor:    145_025,
		}},
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t,
		"Would post Payment 10442 from Acme Foods for 1500.25 USD on 2026-09-24, "+
			"leaving 50.00 USD as unapplied cash",
		sim.Summary,
	)
	assert.Equal(t, []agent.FieldChange{
		{Field: "status", From: "Proposed", To: "Applied"},
		{Field: "INV-1001", From: "open 1450.25 USD", To: "pays 1450.25 USD"},
	}, sim.Changes)
	assert.Nil(t, operator.applied, "a preview posts nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.applied)
	assert.Equal(t, change.ID, operator.applied.ID)
	assert.Equal(t, params.Actor.UserID, operator.appliedActor.UserID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceAccountingSync, ID: change.ID}, target)
}

func TestApplyAccountingInboundChange_RefusesWhatNoLongerMatches(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{
		Blocker: "Pays 1500.25 USD on INV-1001, which has 0.00 USD open in Trenova.",
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be applied")
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, operator.applied)
}

func TestIgnoreAccountingInboundChange_IgnoresWithTheNote(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &serviceports.AccountingInboundApplyPreview{})
	tool := newIgnoreAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{
		"inboundChangeId": change.ID.String(),
		"note":            "  Keyed in Trenova\n by the AR team.  ",
	})

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, []agent.FieldChange{{Field: "status", From: "Proposed", To: "Ignored"}}, sim.Changes)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.ignored)
	assert.Equal(t, "Keyed in Trenova by the AR team.", operator.ignored.Note)
}

func TestIgnoreAccountingInboundChange_NeedsANoteAndAnOpenPayment(t *testing.T) {
	t.Parallel()

	applied := proposedPayment(accountingsync.InboundStatusApplied)
	open := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(applied, &serviceports.AccountingInboundApplyPreview{})
	operator.changes[open.ID] = open
	tool := newIgnoreAccountingInboundChangeTool(operator)

	for _, params := range []map[string]any{
		{"inboundChangeId": open.ID.String()},
		{"inboundChangeId": open.ID.String(), "note": "   "},
		{"inboundChangeId": "nope", "note": "duplicate"},
		{"inboundChangeId": applied.ID.String(), "note": "duplicate"},
	} {
		require.Error(t, tool.Execute(t.Context(), syncParams(params)), params)
	}
	assert.Nil(t, operator.ignored)
}

func TestAccountingInboundTools_AskAPersonFirst(t *testing.T) {
	t.Parallel()

	apply := newApplyAccountingInboundChangeTool(&fakeInboundOperator{}).(serviceports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, apply.MaxTier, "applying posts money")
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, apply.Egress)

	ignore := newIgnoreAccountingInboundChangeTool(&fakeInboundOperator{}).(serviceports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, ignore.MaxTier)
}
