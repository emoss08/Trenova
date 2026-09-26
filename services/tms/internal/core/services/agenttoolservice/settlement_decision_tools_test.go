package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/internal/testutil/providertest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPerformedDuringPreview = errors.New("a preview performed a settlement action")

type fakeSettlementBook[E any] struct {
	plan      func(req *settlementshared.ActionRequest) *settlementshared.ActionPlan[*E]
	planned   *settlementshared.ActionRequest
	performed *settlementshared.ActionRequest
	actor     *serviceports.RequestActor
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
	actor *serviceports.RequestActor,
) (*E, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	f.performed = req
	f.actor = actor

	return f.plan(req).After, nil
}

func buildTool(t *testing.T, provider any) serviceports.AgentTool {
	t.Helper()

	tool, ok := providertest.Build(t, provider, nil).(serviceports.AgentTool)
	require.Truef(t, ok, "%T does not build a tool", provider)

	return tool
}

func pendingDriverSettlement() *driversettlement.Settlement {
	entity := &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		SettlementNumber: "DS-2041",
		Status:           driversettlement.StatusApproved,
		CurrencyCode:     "USD",
		Version:          4,
		Worker:           &worker.Worker{FirstName: "Ana", LastName: "Ruiz"},
		Lines: []*driversettlement.SettlementLine{
			{
				ID:          pulid.MustNew("dstll_"),
				Category:    driversettlement.LineCategoryEarning,
				Description: "Linehaul",
				AmountMinor: 180000,
			},
			{
				ID:          pulid.MustNew("dstll_"),
				Category:    driversettlement.LineCategoryDeduction,
				Description: "Insurance",
				AmountMinor: -6000,
			},
		},
	}
	entity.SyncTotals()

	return entity
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

func settlementTool(t *testing.T, provider any, book any) serviceports.AgentTool {
	t.Helper()

	switch build := provider.(type) {
	case func(*driversettlementservice.Service) serviceports.AgentTool:
		tool := build(nil).(*settlementDecisionTool[driversettlement.Settlement])
		tool.book = book.(settlementBook[driversettlement.Settlement])

		return tool
	case func(*carriersettlementservice.Service) serviceports.AgentTool:
		tool := build(nil).(*settlementDecisionTool[carriersettlement.CarrierSettlement])
		tool.book = book.(settlementBook[carriersettlement.CarrierSettlement])

		return tool
	default:
		require.FailNow(t, "not a settlement decision provider")

		return nil
	}
}

func TestSettlementDecisions_MoneyAndFinalOnesAreAPersonsDecision(t *testing.T) {
	t.Parallel()

	personOnly := map[string]permission.Operation{
		"approve_driver_settlement":         permission.OpApprove,
		"post_driver_settlement":            permission.OpApprove,
		"record_driver_settlement_payment":  permission.OpUpdate,
		"void_driver_settlement":            permission.OpCancel,
		"approve_carrier_settlement":        permission.OpApprove,
		"post_carrier_settlement":           permission.OpApprove,
		"record_carrier_settlement_payment": permission.OpUpdate,
		"void_carrier_settlement":           permission.OpCancel,
	}
	inside := map[string]permission.Operation{
		"submit_driver_settlement":             permission.OpSubmit,
		"reject_driver_settlement":             permission.OpReject,
		"recalculate_driver_settlement":        permission.OpUpdate,
		"add_driver_settlement_adjustment":     permission.OpUpdate,
		"remove_driver_settlement_adjustment":  permission.OpUpdate,
		"submit_carrier_settlement":            permission.OpSubmit,
		"reject_carrier_settlement":            permission.OpReject,
		"recalculate_carrier_settlement":       permission.OpUpdate,
		"add_carrier_settlement_adjustment":    permission.OpUpdate,
		"remove_carrier_settlement_adjustment": permission.OpUpdate,
	}

	seen := map[string]bool{}
	for _, provider := range settlementDecisionProviders() {
		tool := buildTool(t, provider)
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

	driverPost := buildTool(t, providePostDriverSettlementTool).Policy()
	assert.True(t, driverPost.HasEgress(agent.EgressDriverVisible))
	require.NotNil(t, driverPost.Classify)
	assert.Equal(t, agent.EgressMoney, driverPost.Classify(serviceports.ToolExecuteParams{}).Egress)
	assert.Equal(t, permission.ResourceCarrierSettlement,
		buildTool(t, provideVoidCarrierSettlementTool).Policy().Resource)
	assert.NotNil(t, buildTool(t, provideRejectDriverSettlementTool).Policy().TaintHold)
}

func TestApproveDriverSettlement_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	entity.Status = driversettlement.StatusPendingApproval
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideApproveDriverSettlementTool, book)
	params := executeParams(map[string]any{paramSettlementID: entity.ID.String()})

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrSettlementNeedsAPerson)
	assert.Nil(t, book.performed)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, book.performed)
	assert.Equal(t, settlementshared.ActionApprove, book.performed.Action)
	assert.Equal(t, entity.ID, book.performed.SettlementID)
	assert.Equal(t, params.OrganizationID, book.performed.TenantInfo.OrgID)
	assert.Equal(t, params.Actor, book.actor)

	agentParams := executeParams(map[string]any{paramSettlementID: entity.ID.String()})
	agentParams.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	agentParams.ProposalID = pulid.MustNew("ap_")
	require.ErrorIs(t, tool.Execute(t.Context(), agentParams), ErrAgentCannotApprove)
}

func TestSubmitDriverSettlement_RunsWithoutAProposal(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideSubmitDriverSettlementTool, book)
	params := executeParams(map[string]any{paramSettlementID: entity.ID.String()})

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, settlementshared.ActionSubmit, book.performed.Action)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{
		Resource: permission.ResourceDriverSettlement,
		ID:       entity.ID,
	}, target)
}

func TestPostDriverSettlement_PreviewShowsTheStatusMoneyAndJournal(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
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
	tool := settlementTool(t, providePostDriverSettlementTool, book)
	params := executeParams(map[string]any{paramSettlementID: entity.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary,
		"Would post to the general ledger driver settlement DS-2041 for Ana Ruiz (now Approved).")
	assert.Contains(t, preview.Summary, "A posted journal entry would be booked.")
	assert.Contains(t, preview.Summary, "driver portal")
	require.Len(t, preview.Changes, 2)

	settlement := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceDriverSettlement, settlement.Resource)
	assert.Equal(t, "Posted", fieldByPath(t, settlement, "status").After)
	require.NotNil(t, settlement.Money)
	assert.True(t, settlement.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("1740")))

	entry := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceJournalEntry, entry.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, entry.Operation)
	require.NotNil(t, entry.Money)
	assert.True(t, entry.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("1800")))
}

func TestSettlementDecision_ARefusalIsAWouldFail(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{
		plan:   driverPlanner(entity, nil),
		locked: true,
	}
	tool := settlementTool(t, provideApproveDriverSettlementTool, book)
	params := executeParams(map[string]any{paramSettlementID: entity.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)

	err = tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestRejectDriverSettlement_NeedsABoundedReason(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	entity.Status = driversettlement.StatusPendingApproval
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideRejectDriverSettlementTool, book)

	missing := executeParams(map[string]any{paramSettlementID: entity.ID.String()})
	require.Error(t, tool.Execute(t.Context(), missing))

	long := executeParams(map[string]any{
		paramSettlementID:     entity.ID.String(),
		paramSettlementReason: string(make([]byte, maxSettlementReason+1)),
	})
	require.Error(t, tool.Execute(t.Context(), long))
	assert.Nil(t, book.performed)

	params := executeParams(map[string]any{
		paramSettlementID:     entity.ID.String(),
		paramSettlementReason: "  Detention line is doubled  ",
	})
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "Draft", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Detention line is doubled", book.performed.Reason)
}

func TestMarkSettlementPaid_TakesOnlyAKnownPaymentMethod(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideMarkDriverSettlementPaidTool, book)
	params := executeParams(map[string]any{
		paramSettlementID:     entity.ID.String(),
		paramPaymentMethod:    "Bitcoin",
		paramPaymentReference: "CHK-1",
	})
	params.ProposalID = pulid.MustNew("ap_")

	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ACH, Check, InstantPay, Other")

	params.Params[paramPaymentMethod] = "Check"
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
	carrierParams := executeParams(map[string]any{
		paramSettlementID:  pulid.MustNew("carstl_").String(),
		paramPaymentMethod: "InstantPay",
	})
	carrierParams.ProposalID = pulid.MustNew("ap_")
	require.Error(t, carrierTool.Execute(t.Context(), carrierParams))
}

func TestAddDriverSettlementAdjustment_PreviewsTheNewNet(t *testing.T) {
	t.Parallel()

	entity := pendingDriverSettlement()
	entity.Status = driversettlement.StatusDraft
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideAddDriverSettlementAdjustmentTool, book)
	payCode := pulid.MustNew("payc_")

	zero := executeParams(map[string]any{
		paramSettlementID:          entity.ID.String(),
		paramAdjustmentDescription: "Nothing",
		paramAdjustmentAmount:      "0.00",
	})
	require.Error(t, tool.Execute(t.Context(), zero))

	badCode := executeParams(map[string]any{
		paramSettlementID:          entity.ID.String(),
		paramAdjustmentDescription: "Fuel advance repaid",
		paramAdjustmentAmount:      "-50",
		paramPayCodeID:             "not-an-id",
	})
	require.Error(t, tool.Execute(t.Context(), badCode))

	params := executeParams(map[string]any{
		paramSettlementID:          entity.ID.String(),
		paramAdjustmentDescription: "Fuel advance repaid",
		paramAdjustmentAmount:      "-50.00",
		paramPayCodeID:             payCode.String(),
	})
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Money)
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.RequireFromString("-50")))
	assert.Equal(t, true, fieldByPath(t, change, fieldHasExceptions).After)

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

	require.Error(t, tool.Execute(t.Context(),
		executeParams(map[string]any{paramSettlementID: settlementID.String()})))

	lineID := pulid.MustNew("carstll_")
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramSettlementID: settlementID.String(),
		paramLineID:       lineID.String(),
	})))
	assert.Equal(t, lineID, book.performed.LineID)
	assert.Equal(t, settlementshared.ActionRemoveAdjustment, book.performed.Action)
}
