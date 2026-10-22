package ratetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var texasStateID = pulid.MustNew("us_")

func texasStates() *agenttoolstest.FakeStates {
	return &agenttoolstest.FakeStates{State: &usstate.UsState{
		ID:           texasStateID,
		Abbreviation: "TX",
		Name:         "Texas",
	}}
}

type fakeAgreements struct {
	guard     *agenttoolstest.WriteGuard
	stored    *rateagreement.RateAgreement
	created   *rateagreement.RateAgreement
	updated   *rateagreement.RateAgreement
	copied    *rateagreementservice.DuplicateRateAgreementRequest
	reviewed  []rateagreementservice.Review
	amended   *rateagreementports.AmendRateAgreementRulesRequest
	increased *rateagreementservice.RateIncreaseRequest
}

func literal(t *testing.T, text string) tarifftypes.Inputs {
	t.Helper()
	inputs, err := tarifftypes.InputsFromLiterals(map[string]string{"rate": text})
	require.NoError(t, err)
	return inputs
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
		Inputs:               tarifftypes.Inputs{"rate": rateValue(rate)},
		LaneKey:              "ST:" + texasStateID.String() + ">ANY",
	}

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
	*rateagreementports.GetRateAgreementByIDRequest,
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
	if err := f.guard.Write(); err != nil {
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
	_ *permissionports.RequestActor,
) (*rateagreementservice.UpdatePlan, error) {
	return &rateagreementservice.UpdatePlan{Before: f.stored, After: entity}, nil
}

func (f *fakeAgreements) Update(
	_ context.Context,
	entity *rateagreement.RateAgreement,
	_ *permissionports.RequestActor,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.Write(); err != nil {
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
	if err := f.guard.Write(); err != nil {
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
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reviewed = append(f.reviewed, review)

	return f.stored, nil
}

func (f *fakeAgreements) PlanAmendRules(
	_ context.Context,
	req *rateagreementports.AmendRateAgreementRulesRequest,
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
	req *rateagreementports.AmendRateAgreementRulesRequest,
	_ pulid.ID,
) (*rateagreement.RateAgreement, error) {
	if err := f.guard.Write(); err != nil {
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
		SkippedNoMoney: 2,
		Lines: []*rateagreementservice.RateIncreaseLine{{
			AgreementCode: "ACME-2026",
			Label:         "Texas out",
			Before:        tarifftypes.Inputs{"rate": rateValue("2.35")},
			After:         tarifftypes.Inputs{"rate": rateValue("2.43")},
			Changed:       []string{"rate"},
		}},
	}, nil
}

func (f *fakeAgreements) ApplyRateIncrease(
	ctx context.Context,
	req *rateagreementservice.RateIncreaseRequest,
	_ pulid.ID,
) (*rateagreementservice.RateIncreasePlan, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.increased = req

	return f.PlanApplyRateIncrease(ctx, req)
}

func rateValue(amount string) rules.RawJSON {
	return rules.RawJSON(`{"$q":"` + amount + `","u":"$/mi"}`)
}

func agreementDraftParams(customerID pulid.ID) map[string]any {
	return map[string]any{
		paramPartyType:            "Customer",
		common.ParamCustomerID:    customerID.String(),
		paramAgreementCode:        "ACME-2027",
		paramAgreementName:        "Acme dry van 2027",
		common.ParamEffectiveFrom: "2027-01-01",
		paramInputs:               map[string]any{"minimumCharge": "$150"},
		paramLanes: []any{map[string]any{
			paramLaneLabel:         "Texas out",
			paramOriginScopeType:   "State",
			paramOriginValue:       "tx",
			paramFormulaTemplateID: pulid.MustNew("ft_").String(),
			paramInputs:            map[string]any{"rate": "$2.35/mi"},
		}},
	}
}

func TestDraftRateAgreement_DraftsLanesItCanReadAndPricesNothing(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{guard: &agenttoolstest.WriteGuard{}}
	tool := NewDraftRateAgreementTool(agreements, texasStates())
	customerID := pulid.MustNew("cus_")
	params := agenttoolstest.ExecuteParams(agreementDraftParams(customerID))

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		agreements.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would draft the customer rate agreement ACME-2027 "+
		"with 1 lane")
	assert.Contains(t, preview.Summary, "it prices nothing until it is approved")

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "drafted", result.Action)
	require.Len(t, agreements.created.Rules, 1)
	lane := agreements.created.Rules[0]
	assert.Equal(t, texasStateID.String(), lane.OriginScopeValue)
	assert.Equal(t, rategeo.ScopeTypeAny, lane.DestinationScopeType)
	assert.Equal(t, agreements.created.EffectiveFrom, lane.EffectiveFrom)
	assert.Equal(t, "$150", tarifftypes.DisplayValue(agreements.created.Inputs["minimumCharge"]))
	assert.True(t, literal(t, "$2.35/mi").Equal(lane.Inputs))
	assert.Equal(t, customerID, *agreements.created.CustomerID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceRateAgreement, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
}

func TestDraftRateAgreement_RefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	tool := NewDraftRateAgreementTool(&fakeAgreements{}, texasStates())
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
			p[common.ParamEffectiveFrom] = "January 2027"
		},
		"an input that is not a Tariff value": func(p map[string]any) {
			p[paramInputs] = map[string]any{"minimumCharge": "a lot"}
		},
		"a lane input that is not text": func(p map[string]any) {
			p[paramLanes].([]any)[0].(map[string]any)[paramInputs] = map[string]any{"rate": 2.35}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := agreementDraftParams(pulid.MustNew("cus_"))
			change(params)
			require.Error(t, tool.(agentports.ToolValidator).Validate(
				t.Context(),
				agenttoolstest.ExecuteParams(params),
			))
		})
	}
}

func TestReviseRateAgreementDraft_KeepsNamedLanesAndRevisesOnlyDrafts(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &agenttoolstest.WriteGuard{},
		stored: storedAgreement(rateagreement.StatusDraft),
	}
	tool := NewReviseRateAgreementDraftTool(agreements, texasStates())
	kept := agreements.stored.Rules[0]
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramLanes: []any{map[string]any{
			paramRuleID: kept.ID.String(),
			paramInputs: map[string]any{"rate": "$2.50/mi", "minimumCharge": "$300"},
		}},
	})

	agenttoolstest.PreviewWithoutWrites(t, agreements.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	_, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, agreements.updated.Rules, 1)
	assert.Equal(t, kept.ID, agreements.updated.Rules[0].ID)
	assert.Equal(t, kept.OriginScopeValue, agreements.updated.Rules[0].OriginScopeValue)
	revised := agreements.updated.Rules[0].Inputs
	assert.Equal(t, "$2.5/mi", tarifftypes.DisplayValue(revised["rate"]))
	assert.Equal(t, "$300", tarifftypes.DisplayValue(revised["minimumCharge"]))
	assert.True(t, literal(t, "$2.35/mi").Equal(kept.Inputs))
	assert.Equal(t, agreements.stored.Version, agreements.updated.Version)

	unknown := agenttoolstest.ExecuteParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramLanes:           []any{map[string]any{paramRuleID: pulid.MustNew("rar_").String()}},
	})
	require.ErrorContains(t, tool.(agentports.ToolValidator).Validate(t.Context(), unknown),
		"is not a lane of this agreement")

	active := NewReviseRateAgreementDraftTool(&fakeAgreements{
		stored: storedAgreement(rateagreement.StatusActive),
	}, texasStates())
	require.ErrorIs(t, active.(agentports.ToolValidator).Validate(t.Context(), params),
		errOnlyDraftsRevise)
}

func TestDuplicateRateAgreement_CopiesIntoADraft(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &agenttoolstest.WriteGuard{},
		stored: storedAgreement(rateagreement.StatusActive),
	}
	tool := NewDuplicateRateAgreementTool(agreements)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRateAgreementID: agreements.stored.ID.String(),
		paramAgreementCode:   "ACME-2027",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		agreements.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(t, "Would copy the rate agreement ACME-2026 into the draft ACME-2027 with 1 lane.",
		preview.Summary)

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "duplicated", result.Action)
	assert.Equal(t, "ACME-2027", agreements.copied.Code)
	assert.Equal(t, permission.OpDuplicate, tool.Policy().Operation)
}

func reviewTool(t *testing.T, agreements *fakeAgreements, name string) agentports.AgentTool {
	t.Helper()

	for _, step := range AgreementReviewSteps() {
		if step.name == name {
			return NewAgreementReviewTool(agreements, step)
		}
	}
	require.Failf(t, "review step missing", "no %s step", name)

	return nil
}

func TestAgreementReviewTools_OnlyAPersonTurnsPricingOnOrOff(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &agenttoolstest.WriteGuard{},
		stored: storedAgreement(rateagreement.StatusInReview),
	}
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramRateAgreementID: agreements.stored.ID.String()},
	)

	approve := reviewTool(t, agreements, "rate_approve_agreement")
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		agreements.guard,
		func() (*agent.ToolPreview, error) {
			return approve.(agentports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would approve the rate agreement ACME-2026")
	assert.Equal(
		t,
		"Active",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)
	require.ErrorIs(t, approve.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	assert.Empty(t, agreements.reviewed)

	_, err := approve.(agentports.ToolPreviewer).Preview(
		t.Context(),
		agenttoolstest.AgentParamsFor(params.Params),
	)
	require.ErrorIs(t, err, common.ErrAgentCannotApprove)

	approved := params
	approved.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, approve.Execute(t.Context(), approved))
	assert.Equal(t, []rateagreementservice.Review{rateagreementservice.ReviewApprove},
		agreements.reviewed)

	for _, name := range []string{
		"rate_approve_agreement",
		"rate_suspend_agreement",
		"rate_resume_agreement",
		"rate_archive_agreement",
	} {
		policy := reviewTool(t, agreements, name).Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, name)
	}
	for _, name := range []string{"rate_submit_agreement", "rate_reject_agreement"} {
		policy := reviewTool(t, agreements, name).Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress, name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, name)
	}

	for _, name := range []string{"rate_reject_agreement", "rate_suspend_agreement"} {
		require.Error(t, reviewTool(t, agreements, name).(agentports.ToolValidator).Validate(
			t.Context(), params,
		), name)
	}

	submit := reviewTool(t, agreements, "rate_submit_agreement")
	require.NoError(t, submit.Execute(t.Context(), params))
	assert.Equal(t, rateagreementservice.ReviewSubmit, agreements.reviewed[1])
}

func TestAmendRateAgreementRules_ShowsTheRatesBeforeAndAfter(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{
		guard:  &agenttoolstest.WriteGuard{},
		stored: storedAgreement(rateagreement.StatusActive),
	}
	tool := NewAmendRateAgreementRulesTool(agreements, texasStates())
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramRateAgreementID:      agreements.stored.ID.String(),
		common.ParamEffectiveFrom: "2026-11-01",
		paramSupersededRuleIDs:    []any{agreements.stored.Rules[0].ID.String()},
		paramLanes: []any{map[string]any{
			paramLaneLabel:         "Texas out",
			paramOriginScopeType:   "State",
			paramOriginValue:       "TX",
			paramFormulaTemplateID: pulid.MustNew("ft_").String(),
			paramInputs:            map[string]any{"rate": "$2.49/mi"},
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		agreements.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "close out 1 lane and add 1 lane")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agreements.stored.ID, change.EntityID)
	assert.Nil(t, change.Money, "negotiated rates are confidential and never carried")
	assert.NotContains(t, preview.Summary, "2.49")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
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

	agreements := &fakeAgreements{guard: &agenttoolstest.WriteGuard{}}
	tool := NewApplyRateIncreaseTool(agreements)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamEffectiveFrom: "2027-01-01",
		paramPartyType:            "Customer",
		paramPercentChange:        "3.5",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		agreements.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary,
		"Would move the money inputs of 1 lane on 1 rate agreement by 3.5% from ")
	assert.Contains(t, preview.Summary, ": ACME-2026 Texas out.")
	assert.NotContains(t, preview.Summary, "2.43", "negotiated rates are confidential")
	assert.Contains(t, preview.Summary, "2 lanes with no money inputs are left as they are")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, rateagreement.PartyTypeCustomer, agreements.increased.PartyType)
	assert.Equal(t, "3.5", agreements.increased.Adjustment.PercentChange.Decimal.String())

	params.Params[paramPartyType] = "Shipper"
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), params))
}
