package formulaassistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubMatrixRepo struct {
	ratematrixports.RateMatrixRepository

	data []*ratematrixports.RateMatrixLookupData
}

func (s *stubMatrixRepo) GetLookupData(
	_ context.Context,
	_ *ratematrixports.GetRateMatrixLookupDataRequest,
) ([]*ratematrixports.RateMatrixLookupData, error) {
	return s.data, nil
}

type stubVersionRepo struct {
	formulatemplateports.FormulaTemplateVersionRepository
}

func (s *stubVersionRepo) ListScheduled(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return nil, nil
}

const testSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"$id": "assistant-test-schema",
	"type": "object",
	"x-formula-context": {"category": "shipment"},
	"x-data-source": {"table": "shipments", "preloads": []},
	"properties": {
		"totalDistance": {"description": "Distance", "type": "number"},
		"baseRate": {"description": "Rate", "type": "number"},
		"serviceType": {
			"description": "Service type",
			"type": "object",
			"properties": {"code": {"description": "Service type code", "type": "string"}}
		}
	}
}`

func newTestService(t *testing.T, matrices ...*ratematrixports.RateMatrixLookupData) *Service {
	t.Helper()

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register("shipment", []byte(testSchema)))
	res := resolver.NewResolver()

	matrixRepo := &stubMatrixRepo{data: matrices}
	formulaSvc := formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Environment: environment.NewBuilder(environment.Params{Registry: registry, Resolver: res}),
		Resolver:       res,
		VersionRepo:    &stubVersionRepo{},
		RateMatrixRepo: matrixRepo,
	})

	return &Service{
		formulaService: formulaSvc,
		templateService: formulatemplateservice.New(formulatemplateservice.Params{
			Logger:         zap.NewNop(),
			FormulaService: formulaSvc,
		}),
		rateMatrixRepo: matrixRepo,
	}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func matrix(code string, axes int) *ratematrixports.RateMatrixLookupData {
	dimensions := make([]*ratematrix.RateMatrixDimension, axes)
	for idx := range dimensions {
		dimensions[idx] = &ratematrix.RateMatrixDimension{}
	}

	return &ratematrixports.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{Code: code, Name: code + " table", Dimensions: dimensions},
	}
}

func TestReference_NamesTheVariablesFunctionsAndTablesAFormulaMayUse(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, matrix("ZONES", 1), matrix("LANES", 2), matrix("CUBE", 3), nil)

	reference, err := svc.Reference(t.Context(), tenant(), "")
	require.NoError(t, err)

	assert.Equal(t, DefaultSchemaID, reference.SchemaID)
	names := make([]string, 0, len(reference.Variables))
	for _, variable := range reference.Variables {
		names = append(names, variable.Name)
	}
	assert.Contains(t, names, "totalDistance")
	assert.Contains(t, names, "baseRate")
	assert.NotEmpty(t, reference.Functions)
	assert.Equal(t, []RateTable{
		{Code: "ZONES", Name: "ZONES table", Axes: 1, Functions: "lookup, lookupOr"},
		{Code: "LANES", Name: "LANES table", Axes: 2, Functions: "lookup2, lookup2Or"},
	}, reference.RateTables, "a table a formula cannot look up is not offered")

	_, err = svc.Reference(t.Context(), tenant(), "nothing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no formula schema named nothing")
}

func TestTest_TheEnginePricesEveryScenario(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	outcome, err := svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * baseRate",
		Variables:  map[string]any{"totalDistance": 100, "baseRate": 2.5},
		Scenarios: []Scenario{
			{Name: "Long haul", Variables: map[string]any{"totalDistance": 900}},
			{Name: "Own rate", Description: "A cheaper lane",
				Variables: map[string]any{"baseRate": 1.25}},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, pagedraft.FormulaCheck{Valid: true, Result: "250"}, outcome.Check)
	require.Len(t, outcome.Scenarios, 2)
	assert.Equal(t, "Long haul", outcome.Scenarios[0].Name)
	assert.Equal(t, "2250", outcome.Scenarios[0].Amount)
	assert.True(t, outcome.Scenarios[0].Valid)
	assert.Equal(t, 2.5, outcome.Scenarios[0].Variables["baseRate"],
		"a scenario starts from the defaults and overrides what it names")
	assert.Equal(t, "125", outcome.Scenarios[1].Amount)
	assert.Equal(t, "A cheaper lane", outcome.Scenarios[1].Description)
}

/*
gpt-6-luna gave a template variable a value in each scenario but not in
variables, and the check refused the expression as "unknown name teamService".
It then set serviceType.code per scenario, a schema field the reference had
listed, and was told the variable name was invalid. It asked the person which
service type meant team service instead of proposing the formula.
*/
func TestTest_ScenarioValuesCountAsDeclaredAndMayNameASchemaField(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	outcome, err := svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: `totalDistance * baseRate + (teamService ? teamRate * totalDistance : 0) + ` +
			`(serviceType.code == "TEAM" ? 10 : 0)`,
		Variables: map[string]any{"baseRate": 2, "teamRate": 0.25},
		Scenarios: []Scenario{
			{Name: "Solo", Variables: map[string]any{
				"totalDistance": 100, "teamService": false, "serviceType.code": "STD",
			}},
			{Name: "Team", Variables: map[string]any{
				"totalDistance": 100, "teamService": true, "serviceType.code": "TEAM",
			}},
		},
	})
	require.NoError(t, err)

	assert.True(t, outcome.Check.Valid, outcome.Check.Error)
	require.Len(t, outcome.Scenarios, 2)
	assert.Equal(t, "200", outcome.Scenarios[0].Amount)
	assert.Equal(t, "235", outcome.Scenarios[1].Amount)
}

// Asked what the studio's per-mile-with-minimum template comes to on a 90-mile
// run at 3 a mile, gpt-6-luna set every value on the one load it described.
// The check ran without them, max met an empty minimum, and the model said the
// engine could not price it.
func TestTest_PricesALoadWhoseValuesAreAllOnTheLoad(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	outcome, err := svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "max(minimumCharge, baseRate * totalDistance)",
		Scenarios: []Scenario{{Name: "90-mile run", Variables: map[string]any{
			"baseRate": 3, "minimumCharge": 250, "totalDistance": 90,
		}}},
	})
	require.NoError(t, err)

	assert.True(t, outcome.Check.Valid, outcome.Check.Error)
	require.Len(t, outcome.Scenarios, 1)
	assert.Equal(t, "270", outcome.Scenarios[0].Amount)
}

func TestTest_PricesUnderThePageDraftsChargePolicy(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	minCharge, maxCharge, precision := "300", "2000.00", 0

	outcome, err := svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * baseRate",
		Variables:  map[string]any{"totalDistance": 100, "baseRate": 2.555},
		Scenarios: []Scenario{
			{Name: "Short", Variables: map[string]any{"totalDistance": 10}},
			{Name: "Long", Variables: map[string]any{"totalDistance": 1000}},
		},
		Policy: &pagedraft.FormulaPolicy{
			MinCharge:         &minCharge,
			MaxCharge:         &maxCharge,
			RoundingMode:      "Up",
			RoundingPrecision: &precision,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, pagedraft.FormulaCheck{Valid: true, Result: "300"}, outcome.Check,
		"255.5 is lifted to the minimum")
	require.Len(t, outcome.Scenarios, 2)
	assert.Equal(t, "300", outcome.Scenarios[0].Amount)
	assert.Equal(t, "2000", outcome.Scenarios[1].Amount, "2555 is held to the maximum")
	require.NotNil(t, outcome.Policy)
	assert.Equal(t, "300", *outcome.Policy.MinCharge)
	assert.Equal(t, "2000", *outcome.Policy.MaxCharge)
	assert.Equal(t, "Up", outcome.Policy.RoundingMode)
	assert.Equal(t, 0, *outcome.Policy.RoundingPrecision)

	outcome, err = svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * baseRate",
		Variables:  map[string]any{"totalDistance": 100, "baseRate": 2.555},
		Policy:     &pagedraft.FormulaPolicy{RoundingMode: "Up", RoundingPrecision: &precision},
	})
	require.NoError(t, err)
	assert.Equal(t, "256", outcome.Check.Result, "rounded up to whole dollars")

	outcome, err = svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * baseRate",
		Variables:  map[string]any{"totalDistance": 100, "baseRate": 2.555},
	})
	require.NoError(t, err)
	assert.Equal(t, "255.5", outcome.Check.Result, "no policy is the default rounding")
	assert.Equal(t, "HalfUp", outcome.Policy.RoundingMode)
	assert.Equal(t, 2, *outcome.Policy.RoundingPrecision)

	bad := "lots"
	_, err = svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance",
		Policy:     &pagedraft.FormulaPolicy{MinCharge: &bad},
	})
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "policy.minCharge", multiErr.Errors[0].Field)
}

func TestTest_ReportsAnExpressionThatWillNotEvaluate(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	outcome, err := svc.Test(t.Context(), &TestRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance *",
		Scenarios:  []Scenario{{Name: "Any"}},
	})
	require.NoError(t, err)

	assert.False(t, outcome.Check.Valid)
	assert.NotEmpty(t, outcome.Check.Error)
	assert.Empty(t, outcome.Check.Result)
	assert.Empty(t, outcome.Scenarios, "nothing is priced with an expression that does not run")
}

func TestTest_RefusesWhatCannotBeAFormulaValue(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	scenarios := make([]Scenario, MaxScenarios+1)
	for idx := range scenarios {
		scenarios[idx] = Scenario{Name: "Scenario"}
	}

	cases := map[string]*TestRequest{
		"no expression":  {Expression: "  "},
		"a bad name":     {Expression: "1", Variables: map[string]any{"9lives": 1}},
		"a bad path":     {Expression: "1", Variables: map[string]any{"serviceType..code": 1}},
		"a list value":   {Expression: "1", Variables: map[string]any{"rate": []any{1, 2}}},
		"a nameless one": {Expression: "1", Scenarios: []Scenario{{Name: " "}}},
		"too many":       {Expression: "1", Scenarios: scenarios},
	}
	for name, req := range cases {
		req.TenantInfo = tenant()
		_, err := svc.Test(t.Context(), req)
		require.Error(t, err, name)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr, name)
	}
}

func TestPropose_PricesTheDraftWithItsOwnDefaults(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	proposal, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "max(totalDistance * perMile, minimum)",
		Variables: []pagedraft.FormulaVariable{
			{Name: "perMile", Type: pagedraft.VariableNumber, DefaultValue: 2.85},
			{Name: "minimum", Type: pagedraft.VariableNumber, DefaultValue: 350},
		},
		Explanation: "Charges per mile, never less than the minimum.",
		Scenarios: []Scenario{
			{Name: "Short run", Variables: map[string]any{"totalDistance": 10}},
			{Name: "Long run", Variables: map[string]any{"totalDistance": 1000}},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, DefaultSchemaID, proposal.SchemaID)
	assert.Equal(t, "Charges per mile, never less than the minimum.", proposal.Explanation)
	require.Len(t, proposal.Scenarios, 2)
	assert.Equal(t, "350", proposal.Scenarios[0].Amount, "the minimum holds on a short run")
	assert.Equal(t, "2850", proposal.Scenarios[1].Amount)
	assert.Len(t, proposal.Variables, 2)
}

// A proposal goes into the studio, where only its declared variables exist,
// so a name only its sample loads give a value is still unknown there.
func TestPropose_ChecksWithTheDeclaredVariablesOnly(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	proposal, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  tenant(),
		Expression:  "totalDistance * perMile",
		Explanation: "Charges per mile.",
		Scenarios: []Scenario{
			{Name: "Run", Variables: map[string]any{"totalDistance": 10, "perMile": 2}},
		},
	})
	require.NoError(t, err)

	assert.False(t, proposal.Check.Valid, "perMile is not declared, so the studio could not run it")
}

func TestPropose_RefusesADraftTheStudioCouldNotHold(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	_, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance",
		Variables: []pagedraft.FormulaVariable{
			{Name: "rate", Type: "Money"},
			{Name: "rate", Type: pagedraft.VariableNumber},
		},
	})
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.Contains(t, fields, "explanation")
	assert.Contains(t, fields, "formula.variables[0].type")
	assert.Contains(t, fields, "formula.variables[1].name")
}

func TestPropose_CarriesTheTemplateDetailsAndPricesTheBreakdown(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	proposal, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * perMile + handling",
		Variables: []pagedraft.FormulaVariable{
			{Name: "perMile", Type: pagedraft.VariableNumber, DefaultValue: 2},
			{Name: "handling", Type: pagedraft.VariableNumber, DefaultValue: 50},
		},
		Details: pagedraft.FormulaDetails{
			Name:         "  Per mile with handling ",
			Description:  "Linehaul by the mile plus a handling fee.",
			TemplateType: pagedraft.TemplateTypeFreightCharge,
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "linehaul", Label: "Linehaul", Expression: "totalDistance * perMile"},
				{Name: "handlingFee", Label: "Handling", Expression: "handling"},
			},
		},
		Explanation: "Per mile plus handling.",
	})
	require.NoError(t, err)

	assert.True(t, proposal.Check.Valid, proposal.Check.Error)
	assert.Equal(t, "Per mile with handling", proposal.Name)
	assert.Equal(t, pagedraft.TemplateTypeFreightCharge, proposal.TemplateType)
	require.Len(t, proposal.Breakdown, 2)
	assert.Equal(t, "handlingFee", proposal.Breakdown[1].Name)
	assert.Nil(t, proposal.Policy, "no policy proposed leaves the editor's alone")
}

func TestPropose_PricesTheSampleLoadsUnderTheProposedPolicy(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	minCharge, precision := " 350 ", 1

	proposal, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * perMile",
		Variables: []pagedraft.FormulaVariable{
			{Name: "perMile", Type: pagedraft.VariableNumber, DefaultValue: 2.85},
		},
		Details: pagedraft.FormulaDetails{
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "linehaul", Label: "Linehaul", Expression: "totalDistance * perMile"},
			},
			Policy: &pagedraft.FormulaPolicy{
				MinCharge:         &minCharge,
				RoundingMode:      "HalfEven",
				RoundingPrecision: &precision,
			},
		},
		Explanation: "Per mile, never less than the minimum.",
		Scenarios: []Scenario{
			{Name: "Short run", Variables: map[string]any{"totalDistance": 10}},
			{Name: "Long run", Variables: map[string]any{"totalDistance": 1001}},
		},
	})
	require.NoError(t, err)

	assert.True(t, proposal.Check.Valid, proposal.Check.Error)
	require.Len(t, proposal.Scenarios, 2)
	assert.Equal(t, "350", proposal.Scenarios[0].Amount, "the policy's minimum holds")
	assert.Equal(t, "2852.8", proposal.Scenarios[1].Amount, "2852.85 rounded half even to 1 place")
	require.NotNil(t, proposal.Policy)
	assert.Equal(t, "350", *proposal.Policy.MinCharge, "the proposal carries the policy, trimmed")
	assert.Equal(t, "HalfEven", proposal.Policy.RoundingMode)

	bad := "-1"
	_, err = svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo:  tenant(),
		Expression:  "totalDistance",
		Details:     pagedraft.FormulaDetails{Policy: &pagedraft.FormulaPolicy{MaxCharge: &bad}},
		Explanation: "Distance.",
	})
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "policy.maxCharge", multiErr.Errors[0].Field)
}

func TestPropose_ABreakdownLineThatFailsFailsTheCheck(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	proposal, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance * 2",
		Details: pagedraft.FormulaDetails{
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "fuel", Label: "Fuel surcharge", Expression: "unknownThing * 2"},
			},
		},
		Explanation: "Two per mile.",
	})
	require.NoError(t, err)

	assert.False(t, proposal.Check.Valid)
	assert.Contains(t, proposal.Check.Error, `"Fuel surcharge" (fuel)`)
}

func TestPropose_RefusesDetailsTheStudioCouldNotHold(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	_, err := svc.Propose(t.Context(), &ProposeRequest{
		TenantInfo: tenant(),
		Expression: "totalDistance",
		Details: pagedraft.FormulaDetails{
			Name:         strings.Repeat("n", pagedraft.MaxTemplateNameLength+1),
			TemplateType: "Discount",
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "_hidden", Label: "", Expression: "1"},
				{Name: "line", Label: "Line", Expression: ""},
				{Name: "line", Label: "Again", Expression: "2"},
			},
		},
		Explanation: "Distance.",
	})
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make(map[string]bool, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["name"])
	assert.True(t, fields["templateType"])
	assert.True(t, fields["breakdown[0].name"])
	assert.True(t, fields["breakdown[0].label"])
	assert.True(t, fields["breakdown[1].expression"])
	assert.True(t, fields["breakdown[2].name"])
}

