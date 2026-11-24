package settlementtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHolds struct {
	settlement *driversettlement.Settlement
	locked     bool
	changed    *driversettlementservice.HoldRequest
}

func (f *fakeHolds) PlanHoldChange(
	_ context.Context,
	req *driversettlementservice.HoldRequest,
) (*driversettlementservice.ActionPlan, error) {
	plan := &driversettlementservice.ActionPlan{
		Before: f.settlement,
		After:  driversettlementservice.CloneSettlement(f.settlement),
	}
	plan.Refusal = driversettlementservice.PlanHold(plan.After, req, pulid.MustNew("usr_"), 1)

	return plan, nil
}

func (f *fakeHolds) ChangeHold(
	_ context.Context,
	req *driversettlementservice.HoldRequest,
	_ *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	if f.locked {
		return nil, agenttoolstest.ErrPerformedDuringPreview
	}
	f.changed = req

	return f.settlement, nil
}

func approvedSettlement() *driversettlement.Settlement {
	return &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		SettlementNumber: "DS-2041",
		Status:           driversettlement.StatusApproved,
		CurrencyCode:     "USD",
		Version:          3,
		Worker:           &worker.Worker{FirstName: "Deborah", LastName: "Mendez"},
	}
}

func TestTheHoldToolIsAPersonsDecisionOnTheHoldPermission(t *testing.T) {
	t.Parallel()

	tool := &SettlementHoldTool{settlements: &fakeHolds{}}
	policy := tool.Policy()
	assert.Equal(t, HoldDriverSettlementToolName, tool.Name())
	assert.Equal(t, permission.ResourceDriverSettlement, policy.Resource)
	assert.Equal(t, permission.OpHold, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.True(t, policy.Reversible)
}

func TestHoldingASettlementPreviewsTheHoldAndRunsOnlyFromApproval(t *testing.T) {
	t.Parallel()

	holds := &fakeHolds{settlement: approvedSettlement(), locked: true}
	tool := &SettlementHoldTool{settlements: holds}
	args := map[string]any{
		settlementpreview.ParamSettlementID: holds.settlement.ID.String(),
		toolparams.ParamReason:              string(driversettlement.HoldReasonRateDispute),
		paramHoldNote:                       "Waiting on the broker's corrected rate",
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(args))
	require.NoError(t, err)
	assert.Equal(t,
		"Would hold driver settlement DS-2041 for Deborah Mendez (now Approved) out of the pay "+
			"run for a rate dispute.",
		preview.Summary)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "RateDispute",
		agenttoolstest.FieldByPath(t, change, fieldHoldReason).After)
	assert.Equal(t, "Waiting on the broker's corrected rate",
		agenttoolstest.FieldByPath(t, change, fieldHoldNote).After)

	holds.locked = false
	require.ErrorIs(t,
		tool.Execute(t.Context(), agenttoolstest.ExecuteParams(args)),
		settlementpreview.ErrSettlementNeedsAPerson,
		"only a person's approval holds a driver's settlement")
	require.Nil(t, holds.changed)

	require.NoError(t, tool.Execute(t.Context(),
		agenttoolstest.Approved(agenttoolstest.ExecuteParams(args))))
	require.NotNil(t, holds.changed)
	assert.Equal(t, driversettlement.HoldReasonRateDispute, holds.changed.Reason)
	assert.False(t, holds.changed.Release)
}

func TestTheHoldToolNeedsAReasonFromTheSetToHold(t *testing.T) {
	t.Parallel()

	holds := &fakeHolds{settlement: approvedSettlement(), locked: true}
	tool := &SettlementHoldTool{settlements: holds}
	id := holds.settlement.ID.String()

	_, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		settlementpreview.ParamSettlementID: id,
	}))
	require.Error(t, err)
	_, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		settlementpreview.ParamSettlementID: id,
		toolparams.ParamReason:              "Vacation",
	}))
	require.Error(t, err)

	other := agenttoolstest.ExecuteParams(map[string]any{
		settlementpreview.ParamSettlementID: id,
		toolparams.ParamReason:              string(driversettlement.HoldReasonOther),
	})
	preview, err := tool.Preview(t.Context(), other)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), other))
}

func TestReleasingAHoldPreviewsTheReleaseAndAFreeSettlementWouldFail(t *testing.T) {
	t.Parallel()

	settlement := approvedSettlement()
	holds := &fakeHolds{settlement: settlement, locked: true}
	tool := &SettlementHoldTool{settlements: holds}
	args := map[string]any{
		settlementpreview.ParamSettlementID: settlement.ID.String(),
		paramRelease:                        true,
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(args))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)

	require.NoError(t, settlement.PlaceHold(&driversettlement.HoldInput{
		Reason: driversettlement.HoldReasonMissingPaperwork, UserID: pulid.MustNew("usr_"), At: 1,
	}))
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(args))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would release the hold on driver settlement DS-2041")
	assert.Nil(t, agenttoolstest.FieldByPath(
		t, agenttoolstest.PreviewChange(t, preview, 0), fieldHoldReason).After)

	holds.locked = false
	require.NoError(t, tool.Execute(t.Context(),
		agenttoolstest.Approved(agenttoolstest.ExecuteParams(args))))
	assert.True(t, holds.changed.Release)
}

func TestAPostedSettlementCannotBeHeldByTheTool(t *testing.T) {
	t.Parallel()

	settlement := approvedSettlement()
	settlement.Status = driversettlement.StatusPosted
	tool := &SettlementHoldTool{settlements: &fakeHolds{settlement: settlement, locked: true}}
	params := agenttoolstest.ExecuteParams(map[string]any{
		settlementpreview.ParamSettlementID: settlement.ID.String(),
		toolparams.ParamReason:              string(driversettlement.HoldReasonNegativeNet),
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), params))
}
