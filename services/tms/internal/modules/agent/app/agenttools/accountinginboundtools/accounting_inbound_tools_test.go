package accountinginboundtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundOperator struct {
	changes  map[pulid.ID]*accountinginbound.AccountingInboundChange
	previews map[pulid.ID]*accountinginboundports.AccountingInboundApplyPreview

	applied      *accountinginboundports.DecideAccountingInboundChangeRequest
	appliedActor *permissionports.RequestActor
	ignored      *accountinginboundports.DecideAccountingInboundChangeRequest
}

func (f *fakeInboundOperator) Get(
	_ context.Context,
	req *accountinginboundports.GetAccountingInboundChangeRequest,
) (*accountinginbound.AccountingInboundChange, error) {
	change, ok := f.changes[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
	}
	out := *change
	return &out, nil
}

func (f *fakeInboundOperator) PreviewApply(
	_ context.Context,
	req *accountinginboundports.GetAccountingInboundChangeRequest,
) (*accountinginboundports.AccountingInboundApplyPreview, error) {
	preview, ok := f.previews[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting inbound change not found")
	}
	return preview, nil
}

func (f *fakeInboundOperator) Apply(
	_ context.Context,
	req *accountinginboundports.DecideAccountingInboundChangeRequest,
	actor *permissionports.RequestActor,
) (*accountinginbound.AccountingInboundChange, error) {
	f.applied = req
	f.appliedActor = actor
	return f.changes[req.ID], nil
}

func (f *fakeInboundOperator) Ignore(
	_ context.Context,
	req *accountinginboundports.DecideAccountingInboundChangeRequest,
	_ *permissionports.RequestActor,
) (*accountinginbound.AccountingInboundChange, error) {
	f.ignored = req
	return f.changes[req.ID], nil
}

func proposedPayment(
	status accountinginbound.InboundChangeStatus,
) *accountinginbound.AccountingInboundChange {
	return &accountinginbound.AccountingInboundChange{
		ID:             pulid.MustNew("acctic_"),
		Kind:           accountinginbound.InboundCustomerPayment,
		Status:         status,
		Reason:         accountinginbound.InboundReasonPolicyPropose,
		ExternalNumber: "10442",
		PartyName:      "Acme Foods",
		AmountMinor:    150_025,
		CurrencyCode:   "USD",
	}
}

func newInboundOperator(
	change *accountinginbound.AccountingInboundChange,
	preview *accountinginboundports.AccountingInboundApplyPreview,
) *fakeInboundOperator {
	preview.Change = change
	return &fakeInboundOperator{
		changes: map[pulid.ID]*accountinginbound.AccountingInboundChange{change.ID: change},
		previews: map[pulid.ID]*accountinginboundports.AccountingInboundApplyPreview{
			change.ID: preview,
		},
	}
}

func TestApplyAccountingInboundChange_AppliesAsTheApprover(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountinginbound.InboundStatusProposed)
	operator := newInboundOperator(change, &accountinginboundports.AccountingInboundApplyPreview{
		CanApply:       true,
		PaidAt:         1_790_208_000,
		CashMinor:      145_025,
		UnappliedMinor: 5_000,
		Lines: []accountinginboundports.AccountingInboundPostingLine{{
			ObjectType:   accountingsync.SyncObjectInvoice,
			ObjectID:     pulid.MustNew("inv_"),
			ObjectNumber: "INV-1001",
			AmountMinor:  145_025,
			OpenMinor:    145_025,
		}},
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := agenttoolstest.PersonCallParams(
		map[string]any{"inboundChangeId": change.ID.String()},
	)

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	assert.Nil(t, operator.applied, "validating posts nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.applied)
	assert.Equal(t, change.ID, operator.applied.ID)
	assert.Equal(t, params.Actor.UserID, operator.appliedActor.UserID)

	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(
		t,
		agenttoolports.ToolTarget{Resource: permission.ResourceAccountingSync, ID: change.ID},
		target,
	)
}

func TestApplyAccountingInboundChange_RefusesWhatNoLongerMatches(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountinginbound.InboundStatusProposed)
	operator := newInboundOperator(change, &accountinginboundports.AccountingInboundApplyPreview{
		Blocker: "Pays 1500.25 USD on INV-1001, which has 0.00 USD open in Trenova.",
	})
	tool := newApplyAccountingInboundChangeTool(operator)
	params := agenttoolstest.PersonCallParams(
		map[string]any{"inboundChangeId": change.ID.String()},
	)

	err := tool.(agenttoolports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be applied")
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, operator.applied)
}

func TestIgnoreAccountingInboundChange_IgnoresWithTheNote(t *testing.T) {
	t.Parallel()

	change := proposedPayment(accountinginbound.InboundStatusProposed)
	operator := newInboundOperator(change, &accountinginboundports.AccountingInboundApplyPreview{})
	tool := newIgnoreAccountingInboundChangeTool(operator)
	params := agenttoolstest.PersonCallParams(map[string]any{
		"inboundChangeId": change.ID.String(),
		"note":            "  Keyed in Trenova\n by the AR team.  ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.ignored)
	assert.Equal(t, "Keyed in Trenova by the AR team.", operator.ignored.Note)
}

func TestIgnoreAccountingInboundChange_NeedsANoteAndAnOpenPayment(t *testing.T) {
	t.Parallel()

	applied := proposedPayment(accountinginbound.InboundStatusApplied)
	open := proposedPayment(accountinginbound.InboundStatusProposed)
	operator := newInboundOperator(applied, &accountinginboundports.AccountingInboundApplyPreview{})
	operator.changes[open.ID] = open
	tool := newIgnoreAccountingInboundChangeTool(operator)

	for _, params := range []map[string]any{
		{"inboundChangeId": open.ID.String()},
		{"inboundChangeId": open.ID.String(), "note": "   "},
		{"inboundChangeId": "nope", "note": "duplicate"},
		{"inboundChangeId": applied.ID.String(), "note": "duplicate"},
	} {
		require.Error(
			t,
			tool.Execute(t.Context(), agenttoolstest.PersonCallParams(params)),
			params,
		)
	}
	assert.Nil(t, operator.ignored)
}

func TestAccountingInboundTools_AskAPersonFirst(t *testing.T) {
	t.Parallel()

	apply := newApplyAccountingInboundChangeTool(&fakeInboundOperator{}).(agenttoolports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, apply.MaxTier, "applying posts money")
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, apply.Egress)

	ignore := newIgnoreAccountingInboundChangeTool(&fakeInboundOperator{}).(agenttoolports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, ignore.MaxTier)
}
