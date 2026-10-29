package carriertools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMatches struct {
	match     *carriersettlement.InvoiceMatch
	performed *carriersettlementservice.MatchDecisionRequest
}

func (f *fakeMatches) PlanMatchDecision(
	_ context.Context,
	req *carriersettlementservice.MatchDecisionRequest,
) (*carriersettlementservice.MatchDecisionPlan, error) {
	after := *f.match
	plan := &carriersettlementservice.MatchDecisionPlan{Before: f.match, After: &after}
	const now = int64(1_790_000_000)
	userID := req.TenantInfo.UserID
	switch req.Decision {
	case carriersettlementservice.MatchDecisionAccept:
		plan.Refusal = carriersettlementservice.PlanAcceptMatch(&after, req.Note, userID, now)
	case carriersettlementservice.MatchDecisionAcceptWithVariance:
		plan.Refusal = carriersettlementservice.PlanAcceptWithVariance(
			&after,
			&carriersettlementservice.VarianceResolution{
				Note:       req.Note,
				UserID:     userID,
				ResolvedAt: now,
			},
		)
		if plan.Refusal == nil {
			plan.Adjustment = carriersettlementservice.VarianceAdjustmentEvent(
				req.TenantInfo, f.match, now)
		}
	case carriersettlementservice.MatchDecisionReject:
		plan.Refusal = carriersettlementservice.PlanRejectMatch(&after, req.Note, userID, now)
	}

	return plan, nil
}

func (f *fakeMatches) PerformMatchDecision(
	_ context.Context,
	req *carriersettlementservice.MatchDecisionRequest,
	_ *permissionports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	f.performed = req

	return f.match, nil
}

func varianceMatch() *carriersettlement.InvoiceMatch {
	invoiceID := pulid.MustNew("edici_")

	return &carriersettlement.InvoiceMatch{
		ID:                  pulid.MustNew("cim_"),
		CarrierID:           pulid.MustNew("car_"),
		EDICarrierInvoiceID: &invoiceID,
		InvoiceNumber:       "INV-8812",
		Status:              carriersettlement.InvoiceMatchStatusVariance,
		InvoiceTotalMinor:   165000,
		ExpectedTotalMinor:  150000,
		VarianceMinor:       15000,
		CurrencyCode:        "USD",
		Version:             1,
	}
}

func matchTool(t *testing.T, provider any, matches *fakeMatches) *CarrierInvoiceMatchTool {
	t.Helper()

	tool, ok := agenttoolstest.BuildTool(t, provider).(*CarrierInvoiceMatchTool)
	require.True(t, ok)
	tool.matches = matches

	return tool
}

func TestAcceptCarrierInvoiceMatchWithVariance_AccruesTheDifference(t *testing.T) {
	t.Parallel()

	matches := &fakeMatches{match: varianceMatch()}
	tool := matchTool(t, ProvideAcceptCarrierInvoiceMatchWithVarianceTool, matches)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)

	target, ok := tool.Target(map[string]any{paramMatchID: matches.match.ID.String()})
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCarrierInvoiceMatch, target.Resource)

	require.Error(t, tool.Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{paramMatchID: matches.match.ID.String()})))

	raw := map[string]any{
		paramMatchID:   matches.match.ID.String(),
		paramMatchNote: "Two hours of agreed detention at the consignee",
	}
	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary,
		"Would accept with its variance Carrier invoice INV-8812 for 1650.00 USD against "+
			"1500.00 USD expected (now Variance).")
	assert.Contains(t, preview.Summary, "The EDI invoice is marked reconciled.")
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		"Resolved",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			fieldMatchStatus,
		).After,
	)
	adjustment := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agentpreview.PreviewOperationCreate, adjustment.Operation)
	require.NotNil(t, adjustment.Money)
	assert.True(t, adjustment.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(150)))

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)),
		settlementpreview.ErrSettlementNeedsAPerson,
	)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	assert.Equal(t, carriersettlementservice.MatchDecisionAcceptWithVariance,
		matches.performed.Decision)
	assert.Equal(t, "Two hours of agreed detention at the consignee", matches.performed.Note)
}

func TestAcceptCarrierInvoiceMatch_RefusesAVariance(t *testing.T) {
	t.Parallel()

	matches := &fakeMatches{match: varianceMatch()}
	tool := matchTool(t, ProvideAcceptCarrierInvoiceMatchTool, matches)
	raw := map[string]any{paramMatchID: matches.match.ID.String()}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)

	matches.match.Status = carriersettlement.InvoiceMatchStatusMatched
	matches.match.VarianceMinor = 0
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "cleared for the carrier's settlement at the expected cost")
	assert.Len(t, preview.Changes, 1)
}

func TestRejectCarrierInvoiceMatch_RunsInsideAndHoldsOnOutsideText(t *testing.T) {
	t.Parallel()

	matches := &fakeMatches{match: varianceMatch()}
	tool := matchTool(t, ProvideRejectCarrierInvoiceMatchTool, matches)
	policy := tool.Policy()
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, permission.OpReject, policy.Operation)
	require.NotNil(t, policy.TaintHold)

	raw := map[string]any{
		paramMatchID:   matches.match.ID.String(),
		paramMatchNote: "Billed for load PRO-1 we never tendered to this carrier",
	}
	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Equal(
		t,
		"Rejected",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			fieldMatchStatus,
		).After,
	)
	assert.NotContains(t, preview.Summary, "reconciled")

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)))
	assert.Equal(t, carriersettlementservice.MatchDecisionReject, matches.performed.Decision)
}

type fakeMatchCreator struct {
	plan    *carriersettlementservice.CreateMatchPlan
	created *carriersettlementservice.CreateMatchRequest
	planned *carriersettlementservice.CreateMatchRequest
}

func (f *fakeMatchCreator) PlanCreateMatch(
	_ context.Context,
	req *carriersettlementservice.CreateMatchRequest,
) (*carriersettlementservice.CreateMatchPlan, error) {
	f.planned = req

	return f.plan, nil
}

func (f *fakeMatchCreator) CreateMatch(
	_ context.Context,
	req *carriersettlementservice.CreateMatchRequest,
	_ *permissionports.RequestActor,
) (*carriersettlement.InvoiceMatch, error) {
	f.created = req

	return f.plan.Match, nil
}

func TestCreateCarrierInvoiceMatch_MatchesAnEDIInvoiceOnly(t *testing.T) {
	t.Parallel()

	match := varianceMatch()
	creator := &fakeMatchCreator{plan: &carriersettlementservice.CreateMatchPlan{
		Match:   match,
		Invoice: &edicarrierinvoice.CarrierInvoice{},
	}}
	tool := &CreateCarrierInvoiceMatchTool{matches: creator}
	shipmentID := pulid.MustNew("shp_")
	raw := map[string]any{
		paramEDIInvoiceID:             match.EDICarrierInvoiceID.String(),
		changepreview.FieldShipmentID: shipmentID.String(),
		paramMatchProNumber:           "  PRO-7781  ",
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Equal(t, *match.EDICarrierInvoiceID, *creator.planned.EDICarrierInvoiceID)
	assert.Equal(t, shipmentID, creator.planned.ShipmentID)
	assert.Equal(t, "PRO-7781", creator.planned.ProNumber)
	assert.Contains(t, preview.Summary, "against 1500.00 USD expected: Variance.")
	assert.Equal(
		t,
		agentpreview.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)))
	require.NotNil(t, creator.created)

	creator.plan.Refusal = errortypes.NewBusinessError("The invoice is already matched")
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

type fakeInvoiceLinker struct {
	refusal error
	linked  pulid.ID
}

func (f *fakeInvoiceLinker) PlanLinkInvoice(
	_ context.Context,
	_ pagination.TenantInfo,
	invoiceID, carrierID pulid.ID,
) (*carriersettlementservice.LinkInvoicePlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	before := &edicarrierinvoice.CarrierInvoice{ID: invoiceID, InvoiceNumber: "INV-8812"}
	after := *before
	after.CarrierID = carrierID

	return &carriersettlementservice.LinkInvoicePlan{
		Carrier: &carrier.Carrier{ID: carrierID, Name: "Blue Ridge Haulers"},
		Before:  before,
		After:   &after,
	}, nil
}

func (f *fakeInvoiceLinker) LinkInvoiceToCarrier(
	_ context.Context,
	_ pagination.TenantInfo,
	_, carrierID pulid.ID,
	_ *permissionports.RequestActor,
) (*edicarrierinvoice.CarrierInvoice, error) {
	f.linked = carrierID

	return &edicarrierinvoice.CarrierInvoice{}, nil
}

func TestLinkEDICarrierInvoice_NamesTheCarrier(t *testing.T) {
	t.Parallel()

	linker := &fakeInvoiceLinker{}
	tool := &LinkEDICarrierInvoiceTool{invoices: linker}
	carrierID := pulid.MustNew("car_")
	raw := map[string]any{
		paramEDIInvoiceID:         pulid.MustNew("edici_").String(),
		toolparams.ParamCarrierID: carrierID.String(),
	}

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary,
		"Would link EDI carrier invoice INV-8812 to Blue Ridge Haulers so it can be matched.")
	field := agenttoolstest.FieldByPath(
		t,
		agenttoolstest.PreviewChange(t, preview, 0),
		toolparams.ParamCarrierID,
	)
	require.NotNil(t, field.AfterRef)
	assert.Equal(t, permission.ResourceCarrier, field.AfterRef.Resource)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)))
	assert.Equal(t, carrierID, linker.linked)

	linker.refusal = errortypes.NewNotFoundError("Carrier not found")
	preview, err = tool.Preview(t.Context(), agenttoolstest.ExecuteParams(raw))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(raw)))
}
