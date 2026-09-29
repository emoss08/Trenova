package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var texasStateID = pulid.MustNew("us_")

func texasStates() *fakeStates {
	return &fakeStates{state: &usstate.UsState{
		ID:           texasStateID,
		Abbreviation: "TX",
		Name:         "Texas",
	}}
}

type fakeAgreements struct {
	guard     *writeGuard
	stored    *rateagreement.RateAgreement
	created   *rateagreement.RateAgreement
	updated   *rateagreement.RateAgreement
	copied    *rateagreementservice.DuplicateRateAgreementRequest
	reviewed  []rateagreementservice.Review
	amended   *repositories.AmendRateAgreementRulesRequest
	increased *rateagreementservice.RateIncreaseRequest
}

func agreementRule(label string, rate string) *rateagreement.RateAgreementRule {
	rule := &rateagreement.RateAgreementRule{
		ID:                   pulid.MustNew("rar_"),
		Label:                label,
		Status:               rateagreement.RuleStatusActive,
		OriginScopeType:      rategeo.ScopeTypeState,
		OriginScopeValue:     texasStateID.String(),
		DestinationScopeType: rategeo.ScopeTypeAny,
		Direction:            rateagreement.DirectionDirectional,
		Rate:                 decimal.NewNullDecimal(decimal.RequireFromString(rate)),
	}
	rule.ApplyLaneKey()

	return rule
}

func storedAgreement(status rateagreement.Status) *rateagreement.RateAgreement {
	customerID := pulid.MustNew("cus_")

	return &rateagreement.RateAgreement{
		ID:            pulid.MustNew("rag_"),
		Code:          "ACME-2026",
		Name:          "Acme dry van",
		PartyType:     rateagreement.PartyTypeCustomer,
		CustomerID:    &customerID,
		AgreementType: rateagreement.AgreementTypeContract,
		Status:        status,
		EffectiveFrom: 1_767_225_600,
		Currency:      "USD",
		Rules:         []*rateagreement.RateAgreementRule{agreementRule("Texas out", "2.35")},
		Version:       7,
	}
}

func (f *fakeAgreements) GetByID(
	context.Context,
	*repositories.GetRateAgreementByIDRequest,
) (*rateagreement.RateAgreement, error) {
	copied := *f.stored

	return &copied, nil
}

func (f *fakeAgreements) PlanCreate(
	_ context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreement.RateAgreement, error) {
	planned := *entity
	planned.Status = rateagreement.StatusDraft

	return &planned, nil
}

func (f *fakeAgreements) Create(
	_ context.Context,
	entity *rateagreement.RateAgreement,
	_ pulid.ID,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = entity
	saved := *entity
	saved.ID = pulid.MustNew("rag_")

	return &saved, nil
}

func (f *fakeAgreements) PlanUpdate(
	_ context.Context,
	entity *rateagreement.RateAgreement,
) (*rateagreementservice.UpdatePlan, error) {
	return &rateagreementservice.UpdatePlan{Before: f.stored, After: entity}, nil
}

func (f *fakeAgreements) Update(
	_ context.Context,
	entity *rateagreement.RateAgreement,
	_ pulid.ID,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeAgreements) PlanDuplicate(
	_ context.Context,
	req *rateagreementservice.DuplicateRateAgreementRequest,
) (*rateagreementservice.DuplicatePlan, error) {
	copied := *f.stored
	copied.ID = pulid.Nil
	copied.Code = req.Code
	copied.Status = rateagreement.StatusDraft

	return &rateagreementservice.DuplicatePlan{Original: f.stored, Copy: &copied}, nil
}

func (f *fakeAgreements) Duplicate(
	_ context.Context,
	req *rateagreementservice.DuplicateRateAgreementRequest,
	_ pulid.ID,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.copied = req

	return &rateagreement.RateAgreement{ID: pulid.MustNew("rag_"), Code: req.Code}, nil
}

func (f *fakeAgreements) PlanReview(
	_ context.Context,
	review rateagreementservice.Review,
	req *rateagreementservice.ApprovalActionRequest,
) (*rateagreementservice.AgreementChange, error) {
	after := *f.stored
	after.Status = rateagreement.StatusActive
	after.ReviewComment = req.Comment

	return &rateagreementservice.AgreementChange{Before: f.stored, After: &after}, nil
}

func (f *fakeAgreements) Review(
	_ context.Context,
	review rateagreementservice.Review,
	_ *rateagreementservice.ApprovalActionRequest,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reviewed = append(f.reviewed, review)

	return f.stored, nil
}

func (f *fakeAgreements) PlanAmendRules(
	_ context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
) (*rateagreementservice.AmendmentPlan, error) {
	return &rateagreementservice.AmendmentPlan{
		Agreement:     f.stored,
		EffectiveFrom: req.EffectiveFrom,
		Superseded:    f.stored.Rules,
		Rules:         req.Rules,
	}, nil
}

func (f *fakeAgreements) AmendRules(
	_ context.Context,
	req *repositories.AmendRateAgreementRulesRequest,
	_ pulid.ID,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.amended = req

	return f.stored, nil
}

func (f *fakeAgreements) PlanApplyRateIncrease(
	_ context.Context,
	req *rateagreementservice.RateIncreaseRequest,
) (*rateagreementservice.RateIncreasePlan, error) {
	return &rateagreementservice.RateIncreasePlan{
		EffectiveFrom:  req.EffectiveFrom,
		AgreementCount: 1,
		SkippedNoRate:  2,
		Lines: []*rateagreementservice.RateIncreaseLine{{
			AgreementCode: "ACME-2026",
			Label:         "Texas out",
			Before:        decimal.RequireFromString("2.35"),
			After:         decimal.RequireFromString("2.43"),
		}},
	}, nil
}

func (f *fakeAgreements) ApplyRateIncrease(
	ctx context.Context,
	req *rateagreementservice.RateIncreaseRequest,
	_ pulid.ID,
) (*rateagreementservice.RateIncreasePlan, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.increased = req

	return f.PlanApplyRateIncrease(ctx, req)
}

func agreementDraftParams(customerID pulid.ID) map[string]any {
	return map[string]any{
		paramPartyType:        "Customer",
		paramCustomerID:       customerID.String(),
		paramAgreementCode:    "ACME-2027",
		paramAgreementName:    "Acme dry van 2027",
		paramEffectiveFrom:    "2027-01-01",
		paramDefaultMinCharge: "150.00",
		paramLanes: []any{map[string]any{
			paramLaneLabel:         "Texas out",
			paramOriginScopeType:   "State",
			paramOriginValue:       "tx",
			paramFormulaTemplateID: pulid.MustNew("ft_").String(),
			paramLaneRate:          "2.35",
		}},
	}
}

func TestDraftRateAgreement_DraftsLanesItCanReadAndPricesNothing(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{guard: &writeGuard{}}
	tool := newDraftRateAgreementTool(agreements, texasStates())
	customerID := pulid.MustNew("cus_")
	params := executeParams(agreementDraftParams(customerID))

	preview := previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would draft the customer rate agreement ACME-2027 "+
		"with 1 lane")
	assert.Contains(t, preview.Summary, "it prices nothing until it is approved")

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "drafted", result.Action)
	require.Len(t, agreements.created.Rules, 1)
	lane := agreements.created.Rules[0]
	assert.Equal(t, texasStateID.String(), lane.OriginScopeValue)
	assert.Equal(t, rategeo.ScopeTypeAny, lane.DestinationScopeType)
	assert.Equal(t, agreements.created.EffectiveFrom, lane.EffectiveFrom)
	assert.Equal(t, "150", agreements.created.DefaultMinCharge.Decimal.String())
	assert.Equal(t, customerID, *agreements.created.CustomerID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceRateAgreement, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
}

func TestDraftRateAgreement_RefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	tool := newDraftRateAgreementTool(&fakeAgreements{}, texasStates())
	for name, change := range map[string]func(map[string]any){
		"a state it does not know": func(p map[string]any) {
			p[paramLanes].([]any)[0].(map[string]any)[paramOriginValue] = "ZZ"
		},
		"a radius lane": func(p map[string]any) {
			p[paramLanes].([]any)[0].(map[string]any)[paramOriginScopeType] = "Radius"
		},
		"a city lane with no city": func(p map[string]any) {
			p[paramLanes].([]any)[0].(map[string]any)[paramOriginScopeType] = "CityState"
		},
		"a carrier agreement with no carrier": func(p map[string]any) {
			p[paramPartyType] = "Carrier"
		},
		"a day that is not a day": func(p map[string]any) {
			p[paramEffectiveFrom] = "January 2027"
		},
		"a negative minimum": func(p map[string]any) {
			p[paramDefaultMinCharge] = "-1"
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := agreementDraftParams(pulid.MustNew("cus_"))
			change(params)
			require.Error(t, tool.(serviceports.ToolValidator).Validate(
				t.Context(),
				executeParams(params),
			))
		})
	}
}

func TestReviseRateAgreementDraft_KeepsNamedLanesAndRevisesOnlyDrafts(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &writeGuard{},
		stored: storedAgreement(rateagreement.StatusDraft),
	}
	tool := newReviseRateAgreementDraftTool(agreements, texasStates())
	kept := agreements.stored.Rules[0]
	params := executeParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramLanes: []any{map[string]any{
			paramRuleID:   kept.ID.String(),
			paramLaneRate: "2.50",
		}},
	})

	previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, agreements.updated.Rules, 1)
	assert.Equal(t, kept.ID, agreements.updated.Rules[0].ID)
	assert.Equal(t, kept.OriginScopeValue, agreements.updated.Rules[0].OriginScopeValue)
	assert.Equal(t, "2.5", agreements.updated.Rules[0].Rate.Decimal.String())
	assert.Equal(t, "2.35", kept.Rate.Decimal.String())
	assert.Equal(t, agreements.stored.Version, agreements.updated.Version)

	unknown := executeParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramLanes:           []any{map[string]any{paramRuleID: pulid.MustNew("rar_").String()}},
	})
	require.ErrorContains(t, tool.(serviceports.ToolValidator).Validate(t.Context(), unknown),
		"is not a lane of this agreement")

	active := newReviseRateAgreementDraftTool(&fakeAgreements{
		stored: storedAgreement(rateagreement.StatusActive),
	}, texasStates())
	require.ErrorIs(t, active.(serviceports.ToolValidator).Validate(t.Context(), params),
		errOnlyDraftsRevise)
}

func TestDuplicateRateAgreement_CopiesIntoADraft(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &writeGuard{},
		stored: storedAgreement(rateagreement.StatusActive),
	}
	tool := newDuplicateRateAgreementTool(agreements)
	params := executeParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramAgreementCode:   "ACME-2027",
	})

	preview := previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would copy the rate agreement ACME-2026 into the draft ACME-2027 with 1 lane.",
		preview.Summary)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "duplicated", result.Action)
	assert.Equal(t, "ACME-2027", agreements.copied.Code)
	assert.Equal(t, permission.OpDuplicate, tool.Policy().Operation)
}

func reviewTool(t *testing.T, agreements *fakeAgreements, name string) serviceports.AgentTool {
	t.Helper()

	for _, step := range agreementReviewSteps() {
		if step.name == name {
			return newAgreementReviewTool(agreements, step)
		}
	}
	require.Failf(t, "review step missing", "no %s step", name)

	return nil
}

func TestAgreementReviewTools_OnlyAPersonTurnsPricingOnOrOff(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &writeGuard{},
		stored: storedAgreement(rateagreement.StatusInReview),
	}
	params := executeParams(map[string]any{paramRateAgreementID: agreements.stored.ID.String()})

	approve := reviewTool(t, agreements, "approve_rate_agreement")
	preview := previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return approve.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would approve the rate agreement ACME-2026")
	assert.Equal(t, "Active", fieldByPath(t, previewChange(t, preview, 0), "status").After)
	require.ErrorIs(t, approve.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	assert.Empty(t, agreements.reviewed)

	_, err := approve.(serviceports.ToolPreviewer).Preview(
		t.Context(),
		agentParamsFor(params.Params),
	)
	require.ErrorIs(t, err, ErrAgentCannotApprove)

	approved := params
	approved.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, approve.Execute(t.Context(), approved))
	assert.Equal(t, []rateagreementservice.Review{rateagreementservice.ReviewApprove},
		agreements.reviewed)

	for _, name := range []string{
		"approve_rate_agreement",
		"suspend_rate_agreement",
		"resume_rate_agreement",
		"archive_rate_agreement",
	} {
		policy := reviewTool(t, agreements, name).Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, name)
	}
	for _, name := range []string{"submit_rate_agreement", "reject_rate_agreement"} {
		policy := reviewTool(t, agreements, name).Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress, name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, name)
	}

	for _, name := range []string{"reject_rate_agreement", "suspend_rate_agreement"} {
		require.Error(t, reviewTool(t, agreements, name).(serviceports.ToolValidator).Validate(
			t.Context(), params,
		), name)
	}

	submit := reviewTool(t, agreements, "submit_rate_agreement")
	require.NoError(t, submit.Execute(t.Context(), params))
	assert.Equal(t, rateagreementservice.ReviewSubmit, agreements.reviewed[1])
}

func TestAmendRateAgreementRules_ShowsTheRatesBeforeAndAfter(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &writeGuard{},
		stored: storedAgreement(rateagreement.StatusActive),
	}
	tool := newAmendRateAgreementRulesTool(agreements, texasStates())
	params := executeParams(map[string]any{
		paramRateAgreementID:   agreements.stored.ID.String(),
		paramEffectiveFrom:     "2026-11-01",
		paramSupersededRuleIDs: []any{agreements.stored.Rules[0].ID.String()},
		paramLanes: []any{map[string]any{
			paramLaneLabel:         "Texas out",
			paramOriginScopeType:   "State",
			paramOriginValue:       "TX",
			paramFormulaTemplateID: pulid.MustNew("ft_").String(),
			paramLaneRate:          "2.49",
		}},
	})

	preview := previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "close out 1 lane and add 1 lane")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agreements.stored.ID, change.EntityID)
	assert.Nil(t, change.Money, "negotiated rates are confidential and never carried")
	assert.NotContains(t, preview.Summary, "2.49")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, agreements.amended.Rules, 1)
	rule := agreements.amended.Rules[0]
	assert.Equal(t, agreements.stored.ID, rule.RateAgreementID)
	assert.Equal(t, agreements.amended.EffectiveFrom, rule.EffectiveFrom)
	assert.Equal(t, []pulid.ID{agreements.stored.Rules[0].ID}, agreements.amended.SupersededIDs)
}

func TestApplyRateIncrease_ProposesTheLaneMovesItWouldMake(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{guard: &writeGuard{}}
	tool := newApplyRateIncreaseTool(agreements)
	params := executeParams(map[string]any{
		paramEffectiveFrom: "2027-01-01",
		paramPartyType:     "Customer",
		paramPercentChange: "3.5",
	})

	preview := previewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary,
		"Would move 1 lane rate on 1 rate agreement by 3.5% from ")
	assert.Contains(t, preview.Summary, ": ACME-2026 Texas out.")
	assert.NotContains(t, preview.Summary, "2.43", "negotiated rates are confidential")
	assert.Contains(t, preview.Summary, "2 lanes priced by a rate matrix are left as they are")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, rateagreement.PartyTypeCustomer, agreements.increased.PartyType)
	assert.Equal(t, "3.5", agreements.increased.Adjustment.PercentChange.Decimal.String())
	assert.False(t, agreements.increased.Adjustment.FlatChange.Valid)

	params.Params[paramPartyType] = "Shipper"
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
}
