package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundOperator struct {
	changes  map[pulid.ID]*accountingsync.AccountingInboundChange
	previews map[pulid.ID]*accountingsyncports.AccountingInboundApplyPreview

	applied      *accountingsyncports.DecideAccountingInboundChangeRequest
	appliedActor *permissionports.RequestActor
	ignored      *accountingsyncports.DecideAccountingInboundChangeRequest
}

func (f *fakeInboundOperator) Get(
	_ context.Context,
	req *accountingsyncports.GetAccountingInboundChangeRequest,
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
	req *accountingsyncports.GetAccountingInboundChangeRequest,
) (*accountingsyncports.AccountingInboundApplyPreview, error) {
	preview, ok := f.previews[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
	}
	return preview, nil
}

func (f *fakeInboundOperator) Apply(
	_ context.Context,
	req *accountingsyncports.DecideAccountingInboundChangeRequest,
	actor *permissionports.RequestActor,
) (*accountingsync.AccountingInboundChange, error) {
	f.applied = req
	f.appliedActor = actor
	return f.changes[req.ID], nil
}

func (f *fakeInboundOperator) Ignore(
	_ context.Context,
	req *accountingsyncports.DecideAccountingInboundChangeRequest,
	_ *permissionports.RequestActor,
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
	preview *accountingsyncports.AccountingInboundApplyPreview,
) *fakeInboundOperator {
	preview.Change = change
	return &fakeInboundOperator{
		changes:  map[pulid.ID]*accountingsync.AccountingInboundChange{change.ID: change},
		previews: map[pulid.ID]*accountingsyncports.AccountingInboundApplyPreview{change.ID: preview},
	}
}

func TestApplyAccountingInboundChange_AppliesAsTheApprover(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &accountingsyncports.AccountingInboundApplyPreview{
		CanApply:       true,
		PaidAt:         1_790_208_000,
		CashMinor:      145_025,
		UnappliedMinor: 5_000,
		Lines: []accountingsyncports.AccountingInboundPostingLine{{
			ObjectType:   accountingsync.SyncObjectInvoice,
			ObjectID:     pulid.MustNew("inv_"),
			ObjectNumber: "INV-1001",
			AmountMinor:  145_025,
			OpenMinor:    145_025,
		}},
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	require.NoError(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
	assert.Nil(t, operator.applied, "validating posts nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.applied)
	assert.Equal(t, change.ID, operator.applied.ID)
	assert.Equal(t, params.Actor.UserID, operator.appliedActor.UserID)

	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, agentports.ToolTarget{Resource: permission.ResourceAccountingSync, ID: change.ID}, target)
}

func TestApplyAccountingInboundChange_RefusesWhatNoLongerMatches(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &accountingsyncports.AccountingInboundApplyPreview{
		Blocker: "Pays 1500.25 USD on INV-1001, which has 0.00 USD open in Trenova.",
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{"inboundChangeId": change.ID.String()})

	err := tool.(agentports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be applied")
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, operator.applied)
}

func TestIgnoreAccountingInboundChange_IgnoresWithTheNote(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(change, &accountingsyncports.AccountingInboundApplyPreview{})
	tool := newIgnoreAccountingInboundChangeTool(operator)
	params := syncParams(map[string]any{
		"inboundChangeId": change.ID.String(),
		"note":            "  Keyed in Trenova\n by the AR team.  ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.ignored)
	assert.Equal(t, "Keyed in Trenova by the AR team.", operator.ignored.Note)
}

func TestIgnoreAccountingInboundChange_NeedsANoteAndAnOpenPayment(t *testing.T) {
	t.Parallel()

	applied := proposedPayment(accountingsync.InboundStatusApplied)
	open := proposedPayment(accountingsync.InboundStatusProposed)
	operator := newInboundOperator(applied, &accountingsyncports.AccountingInboundApplyPreview{})
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

	apply := newApplyAccountingInboundChangeTool(&fakeInboundOperator{}).(agentports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, apply.MaxTier, "applying posts money")
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, apply.Egress)

	ignore := newIgnoreAccountingInboundChangeTool(&fakeInboundOperator{}).(agentports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, ignore.MaxTier)
}
