package formulaassistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubMatrixRepo struct {
	repositories.RateMatrixRepository

	data []*repositories.RateMatrixLookupData
}

func (s *stubMatrixRepo) GetLookupData(
	_ context.Context,
	_ *repositories.GetRateMatrixLookupDataRequest,
) ([]*repositories.RateMatrixLookupData, error) {
	return s.data, nil
}

type stubVersionRepo struct {
	repositories.FormulaTemplateVersionRepository
}

func (s *stubVersionRepo) ListScheduled(
	_ context.Context,
	_ *repositories.ListScheduledVersionsRequest,
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
		"baseRate": {"description": "Rate", "type": "number"}
	}
}`

func newTestService(t *testing.T, matrices ...*repositories.RateMatrixLookupData) *Service {
	t.Helper()

	registry := schema.NewRegistry()
	require.NoError(t, registry.Register("shipment", []byte(testSchema)))
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	eng, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	matrixRepo := &stubMatrixRepo{data: matrices}
	formulaSvc := formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
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

func matrix(code string, axes int) *repositories.RateMatrixLookupData {
	dimensions := make([]*ratematrix.RateMatrixDimension, axes)
	for idx := range dimensions {
		dimensions[idx] = &ratematrix.RateMatrixDimension{}
	}

	return &repositories.RateMatrixLookupData{
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

func TestAmountOf(t *testing.T) {
	t.Parallel()

	for value, want := range map[any]string{
		int64(12):  "12",
		12.5:       "12.5",
		int(7):     "7",
		float32(2): "2",
	} {
		got, ok := AmountOf(value)
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
	_, ok := AmountOf("12")
	assert.False(t, ok, "text is not an amount")
}
