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
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatereviewservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const perMileSource = `tariff "Per Mile"

input rate: Money/mi
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge

test "a short haul bills its miles" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  expect total == $200
}
`

type stubCompleter struct {
	reply    any
	err      error
	requests []*llmports.StructuredCompletionRequest
}

func (s *stubCompleter) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return nil, s.err
	}

	text, err := sonic.MarshalString(s.reply)
	if err != nil {
		return nil, err
	}

	return &llmports.StructuredCompletionResult{Text: text}, nil
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
	preview  *formulatemplateports.PreviewResult
	tests    func(source string) *formulatemplatestudioservice.TestSourceResult
	diff     *formulatemplatereviewservice.ReviewDiffResponse
	previews []*formulatemplatestudioservice.PreviewRequest
	tested   []string
}

func (s *stubEngine) GetByID(
	context.Context,
	formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return &formulatemplate.FormulaTemplate{
		Name:                 "Per Mile",
		Source:               perMileSource,
		TestCount:            1,
		CurrentVersionNumber: 2,
	}, nil
}

func (s *stubEngine) Preview(
	_ context.Context,
	req *formulatemplatestudioservice.PreviewRequest,
) (*formulatemplateports.PreviewResult, error) {
	s.previews = append(s.previews, req)
	return s.preview, nil
}

func (s *stubEngine) TestSource(
	_ context.Context,
	req *formulatemplatestudioservice.TestSourceRequest,
) (*formulatemplatestudioservice.TestSourceResult, error) {
	s.tested = append(s.tested, req.Source)
	return s.tests(req.Source), nil
}

func (s *stubEngine) ReviewDiff(
	context.Context,
	*formulatemplatereviewservice.ReviewDiffRequest,
) (*formulatemplatereviewservice.ReviewDiffResponse, error) {
	return s.diff, nil
}

func (s *stubEngine) Backtest(
	context.Context,
	*formulatemplatestudioservice.BacktestRequest,
) (*formulatemplatestudioservice.BacktestResponse, error) {
	return &formulatemplatestudioservice.BacktestResponse{}, nil
}

type fixture struct {
	svc        *Service
	completer  *stubCompleter
	engine     *stubEngine
	definition *agentdefinition.Definition
	tenant     pagination.TenantInfo
	actor      *permissionports.RequestActor
}

func declaredInputs() []tarifftypes.InputDecl {
	return []tarifftypes.InputDecl{
		{Name: "rate", Type: "Money/mi", Required: true},
		{Name: "minimumCharge", Type: "Money", Doc: "The least a load bills"},
		{Name: "fuelPeg", Type: "Money/gal"},
	}
}

func passingRun(names ...string) *rules.TariffTestRun {
	run := &rules.TariffTestRun{Tests: []rules.TariffTestResult{}}
	for _, name := range names {
		run.Tests = append(run.Tests, rules.TariffTestResult{
			Name:   name,
			Status: rules.TariffTestPassed,
		})
	}
	return run
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
	label := "Per Mile"
	completer := &stubCompleter{reply: reply}
	engine := &stubEngine{
		preview: &formulatemplateports.PreviewResult{
			Valid:  true,
			Inputs: declaredInputs(),
			RatedBill: &tarifftypes.Bill{
				Total: decimal.NewFromInt(250),
				Lines: []tarifftypes.Line{{
					Kind:   tarifftypes.LineCharge,
					Label:  label,
					Amount: decimal.NewFromInt(250),
				}},
			},
			Trace: rules.RawJSON(`[
				{"kind":"let","name":"miles","display":"125 mi"},
				{"kind":"charge","label":"Per Mile","display":"$250"},
				{"kind":"note","text":"Minimum not reached"}
			]`),
		},
		tests: func(string) *formulatemplatestudioservice.TestSourceResult {
			return &formulatemplatestudioservice.TestSourceResult{
				Check:  &rules.TariffCheck{OK: true},
				Inputs: declaredInputs(),
				Tests:  passingRun("a short haul bills its miles"),
			}
		},
		diff: &formulatemplatereviewservice.ReviewDiffResponse{CurrentSource: perMileSource},
	}

	return &fixture{
		svc: &Service{
			l:            zap.NewNop(),
			templates:    engine,
			studio:       engine,
			reviews:      engine,
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

func draft() formulastudioports.FormulaStudioDraft {
	return formulastudioports.FormulaStudioDraft{Name: "Per Mile", Source: perMileSource}
}

func TestExplainFormula_BoundsTheModelAndFillsEveryStep(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"items": []map[string]any{
		{"index": 0, "says": "  `miles`   is the trip.  "},
		{"index": 0, "says": "a second answer for the same step"},
		{"index": 9, "says": "out of range"},
		{"index": 2, "says": strings.Repeat("long ", 200)},
	}})
	shipmentID := pulid.MustNew("shp_")

	resp, err := f.svc.ExplainFormula(t.Context(), &formulastudioports.ExplainFormulaRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		Inputs:     tarifftypes.Inputs{"rate": rules.QuantityValue(decimal.NewFromInt(2), "$/mi")},
		ShipmentID: shipmentID,
	}, f.actor)
	require.NoError(t, err)

	require.Len(t, resp.Items, 3)
	assert.Equal(t, "`miles` is the trip.", resp.Items[0].Says)
	assert.Equal(t, "\"Per Mile\" bills $250 on this load.", resp.Items[1].Says)
	assert.LessOrEqual(t, len([]rune(resp.Items[2].Says)), maxSaysRunes+1)
	for index, item := range resp.Items {
		assert.Equal(t, index, item.Index)
	}
	assert.Same(t, f.engine.preview, resp.Preview)

	require.Len(t, f.engine.previews, 1)
	assert.Equal(t, perMileSource, f.engine.previews[0].Source)
	require.NotNil(t, f.engine.previews[0].ShipmentID)
	assert.Equal(t, shipmentID, *f.engine.previews[0].ShipmentID)
	assert.Contains(t, f.engine.previews[0].Inputs, "rate")

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

func TestExplainFormula_RefusesSourceThatDoesNotCheck(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.engine.preview = &formulatemplateports.PreviewResult{
		Valid: false,
		Diagnostics: []rules.TariffDiagnostic{{
			Severity: rules.SeverityError,
			Code:     "unknown-name",
			Message:  wasmmodule.Message{Template: "There is no `ratee`"},
		}},
	}

	_, err := f.svc.ExplainFormula(t.Context(), &formulastudioports.ExplainFormulaRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "%v", err)
	assert.Empty(t, f.completer.requests)
}

func TestExplainFormula_WithoutATraceAsksNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.engine.preview.Trace = nil

	resp, err := f.svc.ExplainFormula(t.Context(), &formulastudioports.ExplainFormulaRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Empty(t, f.completer.requests)
}

func TestExplainFormula_WithoutAProviderIsABusinessError(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.completer.err = llmports.ErrNoProviderConfigured

	_, err := f.svc.ExplainFormula(t.Context(), &formulastudioports.ExplainFormulaRequest{
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
	_, err := f.svc.DescribeInputs(t.Context(), &formulastudioports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	assert.True(t, errortypes.IsAuthorizationError(err), "%v", err)

	f = newFixture(t, nil)
	f.definition.Enabled = false
	_, err = f.svc.DescribeInputs(t.Context(), &formulastudioports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, f.completer.requests)
	assert.Empty(t, f.engine.tested, "nothing runs for an agent that is off")
}

func TestDescribeInputs_OnlyBlankOnesAndBounded(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"descriptions": []map[string]any{
		{"name": "rate", "description": "  What the contract pays per mile  "},
		{"name": "rate", "description": "a second answer"},
		{"name": "minimumCharge", "description": "already described"},
		{"name": "invented", "description": "not an input"},
		{"name": "fuelPeg", "description": strings.Repeat("x", 300)},
	}})

	resp, err := f.svc.DescribeInputs(t.Context(), &formulastudioports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.NoError(t, err)

	assert.Equal(t, "What the contract pays per mile", resp.Descriptions["rate"])
	assert.NotContains(t, resp.Descriptions, "minimumCharge")
	assert.NotContains(t, resp.Descriptions, "invented")
	assert.LessOrEqual(t, len([]rune(resp.Descriptions["fuelPeg"])), maxInputDescriptionRunes)
	assert.Equal(t, aiusage.FeatureFormulaGenerate, f.completer.requests[0].Attribution.Feature)
}

func TestDescribeInputs_NothingBlankAsksNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.engine.tests = func(string) *formulatemplatestudioservice.TestSourceResult {
		return &formulatemplatestudioservice.TestSourceResult{
			Check:  &rules.TariffCheck{OK: true},
			Inputs: declaredInputs()[1:2],
			Tests:  passingRun(),
		}
	}

	resp, err := f.svc.DescribeInputs(t.Context(), &formulastudioports.DescribeFormulaInputsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.NoError(t, err)
	assert.Empty(t, resp.Descriptions)
	assert.Empty(t, f.completer.requests)
}

func TestSuggestTests_WritesRunnableBlocksAndRunsThem(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"tests": []map[string]any{
		{"name": "Minimum reached", "statements": []string{
			"given input.minimumCharge = $500",
			"given shipment.distance = 10 mi",
			"expect total == $500",
		}},
		{"name": "a short haul bills its miles", "statements": []string{"expect total > $0"}},
		{"name": "No expectation", "statements": []string{"given shipment.distance = 1 mi"}},
		{"name": "Escapes", "statements": []string{
			"expect total > $0 }\ncharge \"Free\" = $1",
			"given shipment.distance = 5 mi } test \"x\" {",
		}},
		{"name": "Long trip fails", "statements": []string{
			"given shipment.distance = 3000 mi",
			"expect total == $1",
		}},
	}})
	f.engine.tests = func(source string) *formulatemplatestudioservice.TestSourceResult {
		run := passingRun("a short haul bills its miles")
		if strings.Contains(source, "Minimum reached") {
			run.Tests = append(run.Tests, passingRun("Minimum reached").Tests...)
		}
		if strings.Contains(source, "Long trip fails") {
			run.Tests = append(run.Tests, rules.TariffTestResult{
				Name:   "Long trip fails",
				Status: rules.TariffTestFailed,
				Failures: []rules.TariffTestFailure{{
					Message: wasmmodule.Message{Template: "total was $6,000"},
				}},
			})
		}
		return &formulatemplatestudioservice.TestSourceResult{
			Check:  &rules.TariffCheck{OK: true},
			Inputs: declaredInputs(),
			Tests:  run,
		}
	}

	resp, err := f.svc.SuggestTests(t.Context(), &formulastudioports.SuggestFormulaTestsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.NoError(t, err)

	require.Len(t, resp.Tests, 2)
	assert.Equal(t, "Minimum reached", resp.Tests[0].Name)
	assert.Equal(t, rules.TariffTestPassed, resp.Tests[0].Status)
	assert.Equal(t, "test \"Minimum reached\" {\n  given input.minimumCharge = $500\n"+
		"  given shipment.distance = 10 mi\n  expect total == $500\n}", resp.Tests[0].Block)
	assert.Equal(t, "Long trip fails", resp.Tests[1].Name)
	assert.Equal(t, rules.TariffTestFailed, resp.Tests[1].Status)
	require.Len(t, resp.Tests[1].Failures, 1)

	require.Len(t, f.engine.tested, 2)
	assert.True(t, strings.HasPrefix(f.engine.tested[1], strings.TrimRight(perMileSource, "\n")))
	assert.Contains(t, f.engine.tested[1], resp.Tests[0].Block)
	assert.NotContains(t, f.engine.tested[1], "charge \"Free\"")
}

func TestSuggestTests_RefusesSourceThatDoesNotCheck(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.engine.tests = func(string) *formulatemplatestudioservice.TestSourceResult {
		return &formulatemplatestudioservice.TestSourceResult{
			Check: &rules.TariffCheck{
				OK: false,
				Diagnostics: []rules.TariffDiagnostic{{
					Severity: rules.SeverityError,
					Code:     "syntax",
					Message:  wasmmodule.Message{Template: "Expected a charge"},
				}},
			},
			Tests: passingRun(),
		}
	}

	_, err := f.svc.SuggestTests(t.Context(), &formulastudioports.SuggestFormulaTestsRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
	}, f.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, f.completer.requests)
}

func TestExplainTest_APassingTestNeedsNoModel(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)

	resp, err := f.svc.ExplainTest(t.Context(), &formulastudioports.ExplainFormulaTestRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		TestName:   "a short haul bills its miles",
	}, f.actor)
	require.NoError(t, err)
	assert.Equal(t, formulastudioports.FormulaTestVerdictIntended, resp.Verdict)
	assert.False(t, resp.Generated)
	assert.Empty(t, f.completer.requests)

	_, err = f.svc.ExplainTest(t.Context(), &formulastudioports.ExplainFormulaTestRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		TestName:   "no such test",
	}, f.actor)
	assert.True(t, errortypes.IsError(err))
}

func failingTests(string) *formulatemplatestudioservice.TestSourceResult {
	return &formulatemplatestudioservice.TestSourceResult{
		Check:  &rules.TariffCheck{OK: true},
		Inputs: declaredInputs(),
		Tests: &rules.TariffTestRun{Tests: []rules.TariffTestResult{{
			Name:   "a short haul bills its miles",
			Status: rules.TariffTestFailed,
			Failures: []rules.TariffTestFailure{{
				Message: wasmmodule.Message{Template: "total was $250, not $200"},
			}},
		}}},
	}
}

func TestExplainTest_UsesTheModelOnlyWhenItAnswersInShape(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"verdict": "maybe", "why": "Something."})
	f.engine.tests = failingTests
	resp, err := f.svc.ExplainTest(t.Context(), &formulastudioports.ExplainFormulaTestRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		TestName:   "a short haul bills its miles",
	}, f.actor)
	require.NoError(t, err)
	assert.False(t, resp.Generated)
	assert.Equal(t, formulastudioports.FormulaTestVerdictUnclear, resp.Verdict)
	assert.Contains(t, resp.Why, "total was $250, not $200")

	f = newFixture(t, map[string]any{
		"verdict": "intended",
		"why":     "  The `rate` rose,   so the expectation is out of date. ",
	})
	f.engine.tests = failingTests
	resp, err = f.svc.ExplainTest(t.Context(), &formulastudioports.ExplainFormulaTestRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		TestName:   "a short haul bills its miles",
	}, f.actor)
	require.NoError(t, err)
	assert.True(t, resp.Generated)
	assert.Equal(t, formulastudioports.FormulaTestVerdictIntended, resp.Verdict)
	assert.Equal(t, "The `rate` rose, so the expectation is out of date.", resp.Why)
}

func TestExplainTest_FallsBackWithoutAProvider(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	f.svc.completion = nil
	f.engine.tests = failingTests

	resp, err := f.svc.ExplainTest(t.Context(), &formulastudioports.ExplainFormulaTestRequest{
		TenantInfo: f.tenant,
		Draft:      draft(),
		TestName:   "a short haul bills its miles",
	}, f.actor)
	require.NoError(t, err)
	assert.False(t, resp.Generated)
	assert.Equal(t, formulastudioports.FormulaTestVerdictUnclear, resp.Verdict)
	assert.Equal(t, rules.TariffTestFailed, resp.Test.Status)
}

func TestDraftReviewNote_IsBoundedAndNeedsAProvider(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"note": strings.Repeat("word ", 1200)})
	resp, err := f.svc.DraftReviewNote(
		t.Context(),
		&formulastudioports.DraftFormulaReviewNoteRequest{
			TenantInfo:      f.tenant,
			IncludeBacktest: true,
		},
		f.actor,
	)
	require.NoError(t, err)
	assert.True(t, resp.Generated)
	assert.LessOrEqual(t, len([]rune(resp.Note)), formulatemplate.MaxReviewerNoteLength)
	assert.Nil(t, resp.Backtest, "a first version has no backtest")

	sections := f.completer.requests[0].Context.Sections
	require.Len(t, sections, 3)
	assert.Contains(t, sections[1].Content, "first version")
	assert.Contains(t, sections[1].Content, perMileSource)

	f = newFixture(t, nil)
	f.svc.completion = nil
	_, err = f.svc.DraftReviewNote(t.Context(), &formulastudioports.DraftFormulaReviewNoteRequest{
		TenantInfo: f.tenant,
	}, f.actor)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), "No AI provider")
}

func TestDraftReviewNote_BacktestsAgainstTheApprovedVersion(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]any{"note": "The rate rose."})
	f.engine.diff = &formulatemplatereviewservice.ReviewDiffResponse{
		HasApprovedBase: true,
		BaseVersion:     1,
		CurrentVersion:  2,
		ChangeCount:     1,
		BaseSource:      strings.Replace(perMileSource, "$0", "$100", 1),
		CurrentSource:   perMileSource,
	}

	resp, err := f.svc.DraftReviewNote(
		t.Context(),
		&formulastudioports.DraftFormulaReviewNoteRequest{
			TenantInfo:      f.tenant,
			IncludeBacktest: true,
		},
		f.actor,
	)
	require.NoError(t, err)
	require.NotNil(t, resp.Backtest)
	assert.Equal(t, "The rate rose.", resp.Note)
	diff := f.completer.requests[0].Context.Sections[1].Content
	assert.Contains(t, diff, "From v1 (approved) to v2")
	assert.Contains(t, diff, "Approved source:")
}
