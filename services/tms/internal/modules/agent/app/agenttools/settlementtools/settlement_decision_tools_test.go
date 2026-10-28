package settlementtools

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSettlementBook[E any] struct {
	plan      func(req *settlementshared.ActionRequest) *settlementshared.ActionPlan[*E]
	planned   *settlementshared.ActionRequest
	performed *settlementshared.ActionRequest
	actor     *permissionports.RequestActor
	locked    bool
}

func (f *fakeSettlementBook[E]) PlanAction(
	_ context.Context,
	req *settlementshared.ActionRequest,
) (*settlementshared.ActionPlan[*E], error) {
	f.planned = req

	return f.plan(req), nil
}

func (f *fakeSettlementBook[E]) Perform(
	_ context.Context,
	req *settlementshared.ActionRequest,
	actor *permissionports.RequestActor,
) (*E, error) {
	if f.locked {
		return nil, agenttoolstest.ErrPerformedDuringPreview
	}
	f.performed = req
	f.actor = actor

	return f.plan(req).After, nil
}

func driverPlanner(
	entity *driversettlement.Settlement,
	journal *settlementshared.JournalPlan,
) func(*settlementshared.ActionRequest) *settlementshared.ActionPlan[*driversettlement.Settlement] {
	return func(
		req *settlementshared.ActionRequest,
	) *settlementshared.ActionPlan[*driversettlement.Settlement] {
		plan := &settlementshared.ActionPlan[*driversettlement.Settlement]{
			Before: entity,
			After:  driversettlementservice.CloneSettlement(entity),
		}
		const now = int64(1_790_000_000)
		switch req.Action { //nolint:exhaustive // the actions these tests plan
		case settlementshared.ActionSubmit:
			plan.Refusal = driversettlementservice.PlanSubmit(
				plan.After,
				req.TenantInfo.UserID,
				now,
			)
		case settlementshared.ActionApprove:
			plan.Refusal = driversettlementservice.PlanApprove(
				plan.After,
				req.TenantInfo.UserID,
				now,
			)
		case settlementshared.ActionPost:
			plan.Refusal = driversettlementservice.PlanPost(plan.After, req.TenantInfo.UserID, now)
			plan.Journal = journal
		case settlementshared.ActionReject:
			plan.Refusal = driversettlementservice.PlanReject(plan.After, req.Reason)
		case settlementshared.ActionAddAdjustment:
			plan.Refusal = driversettlementservice.PlanAddAdjustment(
				plan.After,
				&driversettlementservice.AdjustmentLineInput{
					Description: req.Adjustment.Description,
					AmountMinor: req.Adjustment.AmountMinor,
					PayCodeID:   req.Adjustment.PayCodeID,
				},
			)
		}

		return plan
	}
}

func settlementTool(t *testing.T, provider any, book any) agenttoolports.AgentTool {
	t.Helper()

	switch build := provider.(type) {
	case func(*driversettlementservice.Service) agenttoolports.AgentTool:
		tool := build(nil).(*SettlementDecisionTool[driversettlement.Settlement])
		tool.book = book.(SettlementBook[driversettlement.Settlement])

		return tool
	case func(*carriersettlementservice.Service) agenttoolports.AgentTool:
		tool := build(nil).(*SettlementDecisionTool[carriersettlement.CarrierSettlement])
		tool.book = book.(SettlementBook[carriersettlement.CarrierSettlement])

		return tool
	default:
		require.FailNow(t, "not a settlement decision provider")

		return nil
	}
}

func TestSettlementDecisions_MoneyAndFinalOnesAreAPersonsDecision(t *testing.T) {
	t.Parallel()

	personOnly := map[string]permission.Operation{
		"settlement_approve_driver":         permission.OpApprove,
		"settlement_post_driver":            permission.OpApprove,
		"settlement_record_driver_payment":  permission.OpUpdate,
		"settlement_void_driver":            permission.OpCancel,
		"settlement_approve_carrier":        permission.OpApprove,
		"settlement_post_carrier":           permission.OpApprove,
		"settlement_record_carrier_payment": permission.OpUpdate,
		"settlement_void_carrier":           permission.OpCancel,
	}
	inside := map[string]permission.Operation{
		"settlement_submit_driver":             permission.OpSubmit,
		"settlement_reject_driver":             permission.OpReject,
		"settlement_recalculate_driver":        permission.OpUpdate,
		"settlement_add_driver_adjustment":     permission.OpUpdate,
		"settlement_remove_driver_adjustment":  permission.OpUpdate,
		"settlement_submit_carrier":            permission.OpSubmit,
		"settlement_reject_carrier":            permission.OpReject,
		"settlement_recalculate_carrier":       permission.OpUpdate,
		"settlement_add_carrier_adjustment":    permission.OpUpdate,
		"settlement_remove_carrier_adjustment": permission.OpUpdate,
	}

	seen := map[string]bool{}
	for _, provider := range slices.Concat(SettlementDecisionProviders(), SettlementBulkProviders()) {
		tool := agenttoolstest.BuildTool(t, provider)
		policy := tool.Policy()
		seen[tool.Name()] = true

		if operation, ok := personOnly[tool.Name()]; ok {
			assert.Equal(t, operation, policy.Operation, tool.Name())
			assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
			assert.False(t, policy.Reversible, tool.Name())
			assert.True(t, policy.HasEgress(agent.EgressMoney), tool.Name())
			continue
		}
		operation, ok := inside[tool.Name()]
		require.Truef(t, ok, "%s is not classified here", tool.Name())
		assert.Equal(t, operation, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
	}
	assert.Len(t, seen, len(personOnly)+len(inside))

	driverPost := agenttoolstest.BuildTool(t, postDriverSingle).Policy()
	assert.True(t, driverPost.HasEgress(agent.EgressDriverVisible))
	require.NotNil(t, driverPost.Classify)
	assert.Equal(
		t,
		agent.EgressMoney,
		driverPost.Classify(agenttoolports.ToolExecuteParams{}).Egress,
	)
	assert.Equal(t, permission.ResourceCarrierSettlement,
		agenttoolstest.BuildTool(t, provideVoidCarrierSettlementTool).Policy().Resource)
	assert.NotNil(
		t,
		agenttoolstest.BuildTool(t, provideRejectDriverSettlementTool).Policy().TaintHold,
	)
}

func TestApproveDriverSettlement_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	entity.Status = driversettlement.StatusPendingApproval
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, approveDriverSingle, book)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrSettlementNeedsAPerson)
	assert.Nil(t, book.performed)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, book.performed)
	assert.Equal(t, settlementshared.ActionApprove, book.performed.Action)
	assert.Equal(t, entity.ID, book.performed.SettlementID)
	assert.Equal(t, params.OrganizationID, book.performed.TenantInfo.OrgID)
	assert.Equal(t, params.Actor, book.actor)

	agentParams := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)
	agentParams.Actor.PrincipalType = permissionports.PrincipalTypeAgent
	agentParams.ProposalID = pulid.MustNew("ap_")
	require.ErrorIs(t, tool.Execute(t.Context(), agentParams), common.ErrAgentCannotApprove)
}

func TestSubmitDriverSettlement_RunsWithoutAProposal(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideSubmitDriverSettlementTool, book)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, settlementshared.ActionSubmit, book.performed.Action)

	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, agenttoolports.ToolTarget{
		Resource: permission.ResourceDriverSettlement,
		ID:       entity.ID,
	}, target)
}

func TestPostDriverSettlement_PreviewShowsTheStatusMoneyAndJournal(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	journal := &settlementshared.JournalPlan{
		AccountingDate: 1_790_000_000,
		FiscalPeriodID: pulid.MustNew("fp_"),
		EntryStatus:    "Posted",
		Description:    "Driver settlement DS-2041",
		Lines: []settlementshared.JournalLine{
			{AccountID: pulid.MustNew("gla_"), DebitMinor: 180000},
			{AccountID: pulid.MustNew("gla_"), CreditMinor: 6000},
			{AccountID: pulid.MustNew("gla_"), CreditMinor: 174000},
		},
	}
	book := &fakeSettlementBook[driversettlement.Settlement]{
		plan:   driverPlanner(entity, journal),
		locked: true,
	}
	tool := settlementTool(t, postDriverSingle, book)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary,
		"Would post to the general ledger driver settlement DS-2041 for Ana Ruiz (now Approved).")
	assert.Contains(t, preview.Summary, "A posted journal entry would be booked.")
	assert.Contains(t, preview.Summary, "driver portal")
	require.Len(t, preview.Changes, 2)

	settlement := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceDriverSettlement, settlement.Resource)
	assert.Equal(t, "Posted", agenttoolstest.FieldByPath(t, settlement, "status").After)
	require.NotNil(t, settlement.Money)
	assert.True(t, settlement.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("1740")))

	entry := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceJournalEntry, entry.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, entry.Operation)
	require.NotNil(t, entry.Money)
	assert.True(t, entry.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("1800")))
}

func TestSettlementDecision_ARefusalIsAWouldFail(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{
		plan:   driverPlanner(entity, nil),
		locked: true,
	}
	tool := settlementTool(t, approveDriverSingle, book)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)

	err = tool.(agenttoolports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestRejectDriverSettlement_NeedsABoundedReason(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	entity.Status = driversettlement.StatusPendingApproval
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideRejectDriverSettlementTool, book)

	missing := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamSettlementID: entity.ID.String()},
	)
	require.Error(t, tool.Execute(t.Context(), missing))

	long := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID: entity.ID.String(),
		paramSettlementReason:    string(make([]byte, maxSettlementReason+1)),
	})
	require.Error(t, tool.Execute(t.Context(), long))
	assert.Nil(t, book.performed)

	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID: entity.ID.String(),
		paramSettlementReason:    "  Detention line is doubled  ",
	})
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(
		t,
		"Draft",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Detention line is doubled", book.performed.Reason)
}

func TestMarkSettlementPaid_TakesOnlyAKnownPaymentMethod(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideMarkDriverSettlementPaidTool, book)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID:     entity.ID.String(),
		common.ParamPaymentMethod:    "Bitcoin",
		common.ParamPaymentReference: "CHK-1",
	})
	params.ProposalID = pulid.MustNew("ap_")

	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ACH, Check, InstantPay, Other")

	params.Params[common.ParamPaymentMethod] = "Check"
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Check", book.performed.PaymentMethod)
	assert.Equal(t, "CHK-1", book.performed.PaymentReference)

	carrierBook := &fakeSettlementBook[carriersettlement.CarrierSettlement]{
		plan: func(*settlementshared.ActionRequest) *settlementshared.ActionPlan[*carriersettlement.CarrierSettlement] {
			return &settlementshared.ActionPlan[*carriersettlement.CarrierSettlement]{
				Before: &carriersettlement.CarrierSettlement{},
				After:  &carriersettlement.CarrierSettlement{},
			}
		},
	}
	carrierTool := settlementTool(t, provideMarkCarrierSettlementPaidTool, carrierBook)
	carrierParams := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID:  pulid.MustNew("carstl_").String(),
		common.ParamPaymentMethod: "InstantPay",
	})
	carrierParams.ProposalID = pulid.MustNew("ap_")
	require.Error(t, carrierTool.Execute(t.Context(), carrierParams))
}

func TestAddDriverSettlementAdjustment_PreviewsTheNewNet(t *testing.T) {
	t.Parallel()

	entity := agenttoolstest.PendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideAddDriverSettlementAdjustmentTool, book)
	payCode := pulid.MustNew("payc_")

	zero := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID:   entity.ID.String(),
		paramAdjustmentDescription: "Nothing",
		paramAdjustmentAmount:      "0.00",
	})
	require.Error(t, tool.Execute(t.Context(), zero))

	badCode := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID:   entity.ID.String(),
		paramAdjustmentDescription: "Fuel advance repaid",
		paramAdjustmentAmount:      "-50",
		common.ParamPayCodeID:      "not-an-id",
	})
	require.Error(t, tool.Execute(t.Context(), badCode))

	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID:   entity.ID.String(),
		paramAdjustmentDescription: "Fuel advance repaid",
		paramAdjustmentAmount:      "-50.00",
		common.ParamPayCodeID:      payCode.String(),
	})
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.RequireFromString("-50")))
	assert.Equal(t, true, agenttoolstest.FieldByPath(t, change, common.FieldHasExceptions).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, book.performed.Adjustment)
	assert.Equal(t, int64(-5000), book.performed.Adjustment.AmountMinor)
	assert.Equal(t, payCode, *book.performed.Adjustment.PayCodeID)
}

func TestRemoveSettlementAdjustment_TakesTheLineFromTheSettlement(t *testing.T) {
	t.Parallel()

	book := &fakeSettlementBook[carriersettlement.CarrierSettlement]{
		plan: func(*settlementshared.ActionRequest) *settlementshared.ActionPlan[*carriersettlement.CarrierSettlement] {
			return &settlementshared.ActionPlan[*carriersettlement.CarrierSettlement]{
				Before: &carriersettlement.CarrierSettlement{},
				After:  &carriersettlement.CarrierSettlement{},
			}
		},
	}
	tool := settlementTool(t, provideRemoveCarrierSettlementAdjustmentTool, book)
	settlementID := pulid.MustNew("carstl_")

	require.Error(t, tool.Execute(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{common.ParamSettlementID: settlementID.String()},
		),
	))

	lineID := pulid.MustNew("carstll_")
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		common.ParamSettlementID: settlementID.String(),
		paramLineID:              lineID.String(),
	})))
	assert.Equal(t, lineID, book.performed.LineID)
	assert.Equal(t, settlementshared.ActionRemoveAdjustment, book.performed.Action)
}

// The single approve and post tools are no longer registered: the merged
// tools run them for each settlement. These build them for the tests that
// exercise one settlement directly.
func approveDriverSingle(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return driverDecisionTool(approveDecision, s)
}

func postDriverSingle(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return driverDecisionTool(postDecision, s)
}

func approveCarrierSingle(s *carriersettlementservice.Service) agenttoolports.AgentTool {
	return carrierDecisionTool(approveDecision, s)
}

func postCarrierSingle(s *carriersettlementservice.Service) agenttoolports.AgentTool {
	return carrierDecisionTool(postDecision, s)
}
