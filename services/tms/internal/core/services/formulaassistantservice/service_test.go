package formulaassistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCompletion struct {
	text string
	err  error

	lastRequest *serviceports.StructuredCompletionRequest
}

func (s *stubCompletion) Diagnose(
	_ context.Context,
	_ *serviceports.DiagnoseRequest,
) (*serviceports.DiagnoseResult, error) {
	return nil, nil
}

func (s *stubCompletion) CompleteStructured(
	_ context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	s.lastRequest = req
	if s.err != nil {
		return nil, s.err
	}
	return &serviceports.StructuredCompletionResult{
		Text:            s.text,
		ModelIdentifier: "claude-opus-5",
		InputTokens:     100,
		OutputTokens:    50,
	}, nil
}

type stubMatrixRepo struct {
	repositories.RateMatrixRepository
}

func (s *stubMatrixRepo) GetLookupData(
	_ context.Context,
	_ *repositories.GetRateMatrixLookupDataRequest,
) ([]*repositories.RateMatrixLookupData, error) {
	return nil, nil
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

func newTestService(t *testing.T, completion serviceports.CompletionService) *Service {
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

	formulaSvc := formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		VersionRepo:    &stubVersionRepo{},
		RateMatrixRepo: &stubMatrixRepo{},
	})

	aiLogRepo := &noopAILogRepo{}

	return &Service{
		l:               zap.NewNop(),
		completion:      completion,
		formulaService:  formulaSvc,
		templateService: newTemplateService(formulaSvc),
		rateMatrixRepo:  &stubMatrixRepo{},
		aiLogRepo:       aiLogRepo,
	}
}

func newTemplateService(formulaSvc *formula.Service) *formulatemplateservice.Service {
	return formulatemplateservice.New(formulatemplateservice.Params{
		Logger:         zap.NewNop(),
		FormulaService: formulaSvc,
	})
}

type noopAILogRepo struct {
	repositories.AILogRepository
	created []*ailog.Log
}

func (n *noopAILogRepo) Create(_ context.Context, entry *ailog.Log) (*ailog.Log, error) {
	n.created = append(n.created, entry)
	return entry, nil
}

func newTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestGenerateFormula_RequiresInstruction(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubCompletion{})

	_, err := svc.GenerateFormula(t.Context(), &GenerateFormulaRequest{
		TenantInfo: newTenant(),
	})

	require.Error(t, err)
}

func TestGenerateFormula_ParsesAndValidatesModelOutput(t *testing.T) {
	t.Parallel()

	completion := &stubCompletion{
		text: `{
			"expression": "round(baseRate * totalDistance * (1 + fuelPct / 100), 2)",
			"variables": [
				{"name": "fuelPct", "type": "Number", "description": "Fuel percent", "defaultValue": 18}
			],
			"explanation": "Multiplies the per-mile rate by distance and adds a fuel surcharge."
		}`,
	}
	svc := newTestService(t, completion)

	result, err := svc.GenerateFormula(t.Context(), &GenerateFormulaRequest{
		TenantInfo:   newTenant(),
		Instruction:  "per mile with fuel surcharge",
		TemplateType: formulatemplate.TemplateTypeFreightCharge,
	})

	require.NoError(t, err)
	assert.Contains(t, result.Expression, "fuelPct")
	require.Len(t, result.VariableDefinitions, 1)
	assert.Equal(t, "fuelPct", result.VariableDefinitions[0].Name)
	assert.Equal(t, formulatypes.VariableValueTypeNumber, result.VariableDefinitions[0].Type)
	assert.NotEmpty(t, result.Explanation)
	require.NotNil(t, result.Validation)
	assert.True(t, result.Validation.Valid)

	require.NotNil(t, completion.lastRequest)
	var untrusted bool
	for _, section := range completion.lastRequest.Context.Sections {
		if !section.Trusted && section.Content == "per mile with fuel surcharge" {
			untrusted = true
		}
	}
	assert.True(t, untrusted, "the user instruction must land in an untrusted section")
}

func TestGenerateFormula_ReportsInvalidModelJSON(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubCompletion{text: "not json"})

	_, err := svc.GenerateFormula(t.Context(), &GenerateFormulaRequest{
		TenantInfo:  newTenant(),
		Instruction: "anything",
	})

	require.Error(t, err)
	require.ErrorIs(t, err, serviceports.ErrModelSchemaValidation)
}

func TestExplainFormula_ParsesExplanation(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubCompletion{
		text: `{"explanation": "Charges by distance."}`,
	})

	result, err := svc.ExplainFormula(t.Context(), &ExplainFormulaRequest{
		TenantInfo: newTenant(),
		Expression: "baseRate * totalDistance",
	})

	require.NoError(t, err)
	assert.Equal(t, "Charges by distance.", result.Explanation)
}

func TestExplainFormula_RequiresExpression(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubCompletion{})

	_, err := svc.ExplainFormula(t.Context(), &ExplainFormulaRequest{TenantInfo: newTenant()})

	require.Error(t, err)
}

func TestMapGeneratedVariables(t *testing.T) {
	t.Parallel()

	variables, testValues := mapGeneratedVariables([]GeneratedVariable{
		{Name: "fuelPct", Type: "Number", DefaultValue: float64(18)},
		{Name: "flag", Type: "Boolean", DefaultValue: true},
		{Name: "weird", Type: "Blob", DefaultValue: nil},
		{Name: "", Type: "Number"},
	})

	require.Len(t, variables, 3)
	assert.Equal(t, formulatypes.VariableValueTypeNumber, variables[0].Type)
	assert.Equal(t, formulatypes.VariableValueTypeBoolean, variables[1].Type)
	assert.Equal(t, formulatypes.VariableValueTypeNumber, variables[2].Type)

	assert.Equal(t, float64(18), testValues["fuelPct"])
	assert.Equal(t, true, testValues["flag"])
	_, hasWeird := testValues["weird"]
	assert.False(t, hasWeird)
}

func TestLogCallRecordsUsage(t *testing.T) {
	t.Parallel()

	aiLogRepo := &noopAILogRepo{}
	svc := &Service{l: zap.NewNop(), aiLogRepo: aiLogRepo}

	svc.logCall(
		t.Context(),
		newTenant(),
		ailog.OperationFormulaGenerate,
		"shipment",
		"per mile",
		&serviceports.StructuredCompletionResult{
			Text:         "{}",
			InputTokens:  120,
			OutputTokens: 30,
		},
	)

	require.Len(t, aiLogRepo.created, 1)
	entry := aiLogRepo.created[0]
	assert.Equal(t, ailog.ModelClaudeOpus5, entry.Model)
	assert.Equal(t, ailog.OperationFormulaGenerate, entry.Operation)
	assert.Equal(t, 120, entry.PromptTokens)
	assert.Equal(t, 30, entry.CompletionTokens)
	assert.Equal(t, 150, entry.TotalTokens)
}
