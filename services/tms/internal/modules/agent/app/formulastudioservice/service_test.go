package formulastudioservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCompleter struct {
	reply    any
	err      error
	requests []*agentports.StructuredCompletionRequest
}

func (s *stubCompleter) CompleteStructured(
	_ context.Context,
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return nil, s.err
	}

	text, err := sonic.MarshalString(s.reply)
	if err != nil {
		return nil, err
	}

	return &agentports.StructuredCompletionResult{Text: text}, nil
}

type stubProvisioner struct {
	definition *agentdefinition.Definition
}

func (s *stubProvisioner) EnsureSystem(
	_ context.Context,
	req agentdefinitionports.EnsureSystemAgentRequest,
) (*agentdefinition.Definition, error) {
	if req.SystemKey != agentdefinition.SystemKeyFormulaAssistant {
		return nil, errors.New("unexpected system agent")
	}
	return s.definition, nil
}

type stubAccess struct {
	allowed bool
}

func (s stubAccess) MayUseAgent(
	context.Context,
	*permissionports.RequestActor,
	*agentdefinition.Definition,
) (bool, error) {
	return s.allowed, nil
}

type stubEngine struct {
	evaluation *formulatemplateservice.TestExpressionResponse
	trace      *formulatemplateports.ScenarioTrace
	priced     []*formulatemplateservice.PriceScenarioRequest
	tested     []*formulatemplateservice.TestExpressionRequest
}

func (s *stubEngine) GetByID(
	context.Context,
	formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return &formulatemplate.FormulaTemplate{
		Name:       "Per mile",
		Expression: "totalDistance * rate",
	}, nil
}

func (s *stubEngine) TestExpression(
	_ context.Context,
	req *formulatemplateservice.TestExpressionRequest,
) (*formulatemplateservice.TestExpressionResponse, error) {
	s.tested = append(s.tested, req)
	return s.evaluation, nil
}

func (s *stubEngine) PriceScenario(
	_ context.Context,
	req *formulatemplateservice.PriceScenarioRequest,
) *formulatemplateservice.ScenarioPrice {
	s.priced = append(s.priced, req)
	return &formulatemplateservice.ScenarioPrice{Amount: "412.5", Valid: true}
}

func (s *stubEngine) ScenarioTrace(
	context.Context,
	*formulatemplateservice.ScenarioTraceRequest,
) (*formulatemplateports.ScenarioTrace, error) {
	return s.trace, nil
}

func (s *stubEngine) ReviewDiff(
	context.Context,
	*formulatemplateservice.ReviewDiffRequest,
) (*formulatemplateservice.ReviewDiffResponse, error) {
	return &formulatemplateservice.ReviewDiffResponse{
		CurrentExpression: "totalDistance * rate",
	}, nil
}

func (s *stubEngine) Backtest(
	context.Context,
	*formulatemplateservice.BacktestRequest,
) (*formulatemplateservice.BacktestResponse, error) {
	return &formulatemplateservice.BacktestResponse{}, nil
}

type fixture struct {
	svc        *Service
	completer  *stubCompleter
	engine     *stubEngine
	definition *agentdefinition.Definition
	tenant     pagination.TenantInfo
	actor      *permissionports.RequestActor
}

func newFixture(t *testing.T, reply any) *fixture {
	t.Helper()

	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	definition := &agentdefinition.Definition{
		ID:                  pulid.MustNew("agd_"),
		Name:                "Nova",
		Enabled:             true,
		Version:             7,
		PreferredProviderID: pulid.MustNew("aip_"),
	}
	completer := &stubCompleter{reply: reply}
	engine := &stubEngine{
		evaluation: &formulatemplateservice.TestExpressionResponse{
			Valid:  true,
			Result: new(decimal.NewFromInt(250)),
			Statements: []formulatemplatetypes.Statement{
				{
					Index: 0,
					Kind:  formulatemplatetypes.StatementLet,
					Name:  "base",
					Code:  "let base = totalDistance * rate",
					Value: rules.RawJSON(`"200"`),
				},
				{
					Index: 1,
					Kind:  formulatemplatetypes.StatementLet,
					Name:  "fuel",
					Code:  "let fuel = base * 0.25",
					Value: rules.RawJSON(`"50"`),
				},
				{
					Index: 2,
					Kind:  formulatemplatetypes.StatementResult,
					Code:  "base + fuel",
					Value: rules.RawJSON(`"250"`),
				},
			},
		},
	}

	return &fixture{
		svc: &Service{
			l:            zap.NewNop(),
			engine:       engine,
			access:       stubAccess{allowed: true},
			systemAgents: &stubProvisioner{definition: definition},
			completion:   completer,
		},
		completer:  completer,
		engine:     engine,
		definition: definition,
		tenant:     tenant,
		actor:      permissionports.UserActor(tenant),
	}
}

func draft() agentports.FormulaStudioDraft {
	return agentports.FormulaStudioDraft{
		Name:     "Per mile",
		SchemaID: "shipment",
		Content: formulatemplate.ScenarioContent{
			Expression: "let base = totalDistance * rate; let fuel = base * 0.25; base + fuel",
			VariableDefinitions: []*formulatypes.VariableDefinition{
				{Name: "rate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 2.0},
				{
					Name:        "fuelPercent",
					Type:        formulatypes.VariableValueTypeNumber,
					Description: "Fuel",
				},
				{Name: "stops", Type: formulatypes.VariableValueTypeNumber},
			},
		},
	}
}

func TestExplainFormula_BoundsTheModelAndFillsEveryStatement(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"items": []map[string]any{
		{"statementIndex": 0, "says": "  `base`   is the linehaul.  "},
		{"statementIndex": 0, "says": "a second answer for the same line"},
		{"statementIndex": 9, "says": "out of range"},
		{"statementIndex": 2, "says": strings.Repeat("long ", 200)},
	}})

	resp, err := f.svc.ExplainFormula(t.Context(), &agentports.ExplainFormulaRequest{
		TenantInfo:   f.tenant,
		Draft:        draft(),
		SampleValues: map[string]any{"totalDistance": 100.0},
	}, f.actor)
	require.NoError(t, err)

	require.Len(t, resp.Items, 3)
	assert.Equal(t, "`base` is the linehaul.", resp.Items[0].Says)
	assert.Equal(t, "`fuel` comes to 50 on this load.", resp.Items[1].Says)
	assert.LessOrEqual(t, len([]rune(resp.Items[2].Says)), maxSaysRunes+1)
	for index, item := range resp.Items {
		assert.Equal(t, index, item.StatementIndex)
	}
	assert.Equal(t, "250", *resp.BilledAmount)
	assert.Len(t, resp.Statements, 3)

	require.Len(t, f.engine.tested, 1)
	assert.True(t, f.engine.tested[0].TraceStatements)
	assert.Equal(t, 2.0, f.engine.tested[0].Variables["rate"], "defaults come first")
	assert.Equal(t, 100.0, f.engine.tested[0].Variables["totalDistance"])

	require.Len(t, f.completer.requests, 1)
	request := f.completer.requests[0]
	assert.Equal(t, f.definition.PreferredProviderID, request.PreferredProviderID)
	assert.Equal(t, f.definition.ID, request.Attribution.AgentDefinitionID)
	assert.Equal(t, f.tenant.UserID, request.Attribution.UserID)
	assert.Equal(t, aiusage.FeatureFormulaExplain, request.Attribution.Feature)
	for _, section := range request.Context.Sections {
		assert.False(t, section.Trusted, "page content is untrusted: %s", section.Title)
	}
}

func TestExplainFormula_WithoutAProviderIsABusinessError(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.completer.err = agentports.ErrNoProviderConfigured

	_, err := f.svc.ExplainFormula(t.Context(), &agentports.ExplainFormulaRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, "No AI provider is configured for the formula assistant", business.Error())
}

func TestAssistant_RefusesWhenTheAgentIsWithheldOrOff(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.svc.access = stubAccess{allowed: false}
	_, err := f.svc.DescribeInputs(t.Context(), &agentports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	assert.True(t, errortypes.IsAuthorizationError(err), "%v", err)

	f = newFixture(t, nil)
	f.definition.Enabled = false
	_, err = f.svc.DescribeInputs(t.Context(), &agentports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, f.completer.requests)
}

func TestDescribeInputs_OnlyBlankOnesAndBounded(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"descriptions": []map[string]any{
		{"name": "rate", "description": "The rate per loaded mile"},
		{"name": "fuelPercent", "description": "already described"},
		{"name": "invented", "description": "not an input"},
		{"name": "stops", "description": strings.Repeat("x", 300)},
	}})

	resp, err := f.svc.DescribeInputs(t.Context(), &agentports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.NoError(t, err)

	assert.Equal(t, "The rate per loaded mile", resp.Descriptions["rate"])
	assert.NotContains(t, resp.Descriptions, "fuelPercent")
	assert.NotContains(t, resp.Descriptions, "invented")
	assert.LessOrEqual(t, len([]rune(resp.Descriptions["stops"])), maxInputDescriptionRunes)
	assert.Equal(t, aiusage.FeatureFormulaGenerate, f.completer.requests[0].Attribution.Feature)
}

func TestDescribeInputs_NothingBlankAsksNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	described := draft()
	described.Content.VariableDefinitions = described.Content.VariableDefinitions[1:2]

	resp, err := f.svc.DescribeInputs(t.Context(), &agentports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      described,
	}, f.actor)
	require.NoError(t, err)
	assert.Empty(t, resp.Descriptions)
	assert.Empty(t, f.completer.requests)
}

func TestSuggestScenarios_ValidatesAndPricesWithTheEngine(t *testing.T) {
	t.Parallel()

	scenario := func(name string, values ...map[string]any) map[string]any {
		return map[string]any{"name": name, "description": "Checks " + name, "variables": values}
	}
	f := newFixture(t, map[string]any{"scenarios": []map[string]any{
		scenario("Short haul", map[string]any{"name": "totalDistance", "value": 50}),
		scenario("existing one", map[string]any{"name": "totalDistance", "value": 1}),
		scenario("Bad path", map[string]any{"name": "1nope", "value": 3}),
		scenario("No values"),
		scenario("Long haul", map[string]any{"name": "totalDistance", "value": 2000}),
		scenario("Short haul", map[string]any{"name": "totalDistance", "value": 60}),
		scenario("Team", map[string]any{"name": "serviceType.code", "value": "TEAM"}),
		scenario("Four", map[string]any{"name": "rate", "value": 4}),
		scenario("Five", map[string]any{"name": "rate", "value": 5}),
		scenario("Six", map[string]any{"name": "rate", "value": 6}),
	}})

	resp, err := f.svc.SuggestScenarios(t.Context(), &agentports.SuggestFormulaScenariosRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		Existing:   []agentports.ExistingFormulaScenario{{Name: "Existing One"}},
	}, f.actor)
	require.NoError(t, err)

	names := make([]string, 0, len(resp.Scenarios))
	for _, priced := range resp.Scenarios {
		names = append(names, priced.Name)
		assert.Equal(t, "412.5", priced.Amount, "every amount comes from the engine")
		assert.True(t, priced.Valid)
	}
	assert.Equal(t, []string{"Short haul", "Long haul", "Team", "Four", "Five"}, names)
	assert.Equal(t, map[string]any{"totalDistance": float64(50)}, resp.Scenarios[0].Variables)
	require.Len(t, f.engine.priced, 5)
	assert.Equal(t, "shipment", f.engine.priced[0].SchemaID)
}

func TestExplainScenario_UsesTheModelOnlyWhenItAnswersInShape(t *testing.T) {
	t.Parallel()

	trace := &formulatemplateports.ScenarioTrace{
		Expected:        "250",
		BaselineLabel:   "v3",
		FallbackVerdict: formulatemplateports.ScenarioVerdictIntended,
		FallbackWhy:     "`rate`'s default changed.",
		Rows: []formulatemplateports.ScenarioTraceRow{
			{Kind: formulatemplateports.ScenarioRowInput, Name: "rate"},
		},
	}

	f := newFixture(t, map[string]any{"why": "Something.", "verdict": "maybe"})
	f.engine.trace = trace
	resp, err := f.svc.ExplainScenario(t.Context(), &agentports.ExplainFormulaScenarioRequest{
		TenantInfo: f.tenant,
	}, f.actor)
	require.NoError(t, err)
	assert.False(t, resp.Generated)
	assert.Equal(t, formulatemplateports.ScenarioVerdictIntended, resp.Verdict)
	assert.Equal(t, "`rate`'s default changed.", resp.Why)
	assert.Equal(t, "v3", resp.BaselineLabel)
	require.Len(t, resp.Rows, 1)

	f = newFixture(t, map[string]any{
		"why":     "The `rate` default rose from 2.5 to 3. That moves `base` to $300.00. Extra.",
		"verdict": "bug",
	})
	f.engine.trace = trace
	resp, err = f.svc.ExplainScenario(t.Context(), &agentports.ExplainFormulaScenarioRequest{
		TenantInfo: f.tenant,
	}, f.actor)
	require.NoError(t, err)
	assert.True(t, resp.Generated)
	assert.Equal(t, formulatemplateports.ScenarioVerdictBug, resp.Verdict)
	assert.Equal(
		t,
		"The `rate` default rose from 2.5 to 3. That moves `base` to $300.00.",
		resp.Why,
	)
}

func TestExplainScenario_FallsBackWithoutAProvider(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.svc.completion = nil
	f.engine.trace = &formulatemplateports.ScenarioTrace{
		Expected:        "250",
		FallbackVerdict: formulatemplateports.ScenarioVerdictBug,
		FallbackWhy:     "The formula fails on this load now: boom",
	}

	resp, err := f.svc.ExplainScenario(t.Context(), &agentports.ExplainFormulaScenarioRequest{
		TenantInfo: f.tenant,
	}, f.actor)
	require.NoError(t, err)
	assert.False(t, resp.Generated)
	assert.Equal(t, formulatemplateports.ScenarioVerdictBug, resp.Verdict)
	assert.NotNil(t, resp.Rows, "rows are an empty list, never null")
}

func TestDraftReviewNote_IsBoundedAndNeedsAProvider(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"note": strings.Repeat("word ", 1200)})
	resp, err := f.svc.DraftReviewNote(t.Context(), &agentports.DraftFormulaReviewNoteRequest{
		TenantInfo:      f.tenant,
		IncludeBacktest: true,
	}, f.actor)
	require.NoError(t, err)
	assert.True(t, resp.Generated)
	assert.LessOrEqual(t, len([]rune(resp.Note)), formulatemplate.MaxReviewerNoteLength)
	assert.Nil(t, resp.Backtest, "a first version has no backtest")

	f = newFixture(t, nil)
	f.svc.completion = nil
	_, err = f.svc.DraftReviewNote(t.Context(), &agentports.DraftFormulaReviewNoteRequest{
		TenantInfo: f.tenant,
	}, f.actor)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), "No AI provider")
}
