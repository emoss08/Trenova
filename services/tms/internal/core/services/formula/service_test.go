package formula_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockFormulaTemplateRepo struct {
	getByIDFunc func(ctx context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error)
}

func (m *mockFormulaTemplateRepo) Create(
	_ context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return entity, nil
}

func (m *mockFormulaTemplateRepo) Update(
	_ context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return entity, nil
}

func (m *mockFormulaTemplateRepo) GetByID(
	ctx context.Context,
	req repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, req)
	}
	return nil, errors.New("not found")
}

func (m *mockFormulaTemplateRepo) GetByIDs(
	_ context.Context,
	_ repositories.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) List(
	_ context.Context,
	_ *repositories.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) BulkUpdateStatus(
	_ context.Context,
	_ *repositories.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) BulkDuplicate(
	_ context.Context,
	_ *repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) CountUsages(
	_ context.Context,
	_ *repositories.GetTemplateUsageRequest,
) (*repositories.GetTemplateUsageResponse, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) SelectOptions(
	_ context.Context,
	_ *repositories.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, nil
}

func setupService(t *testing.T) *formula.Service {
	t.Helper()
	return setupServiceWithRepo(t, nil)
}

func setupServiceWithRepo(
	t *testing.T,
	repo repositories.FormulaTemplateRepository,
) *formula.Service {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	eng := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})

	logger := zap.NewNop()

	return formula.NewService(formula.ServiceParams{
		Logger:   logger,
		Registry: registry,
		Engine:   eng,
		Resolver: res,
		Repo:     repo,
	})
}

func TestService_ValidateExpression(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	t.Run("invalid schema returns error", func(t *testing.T) {
		t.Parallel()
		err := svc.ValidateExpression("x + y", "nonexistent")
		require.Error(t, err)
	})
}

func TestService_ValidateExpressionWithEnv(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		wantErr    bool
	}{
		{
			name:       "valid expression",
			expression: "x + y",
			env:        map[string]any{"x": 0.0, "y": 0.0},
			wantErr:    false,
		},
		{
			name:       "undefined variable",
			expression: "x + z",
			env:        map[string]any{"x": 0.0},
			wantErr:    true,
		},
		{
			name:       "invalid syntax",
			expression: "x +",
			env:        map[string]any{"x": 0.0},
			wantErr:    true,
		},
		{
			name:       "complex valid expression",
			expression: "max(a, b) + min(c, d)",
			env:        map[string]any{"a": 0.0, "b": 0.0, "c": 0.0, "d": 0.0},
			wantErr:    false,
		},
		{
			name:       "ternary expression",
			expression: "flag ? a : b",
			env:        map[string]any{"flag": false, "a": 0.0, "b": 0.0},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := svc.ValidateExpressionWithEnv(tt.expression, tt.env)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_EvaluateWithEnv(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       decimal.Decimal
		wantErr    bool
	}{
		{
			name:       "simple addition",
			expression: "x + y",
			env:        map[string]any{"x": 10.0, "y": 20.0},
			want:       decimal.NewFromFloat(30.0),
			wantErr:    false,
		},
		{
			name:       "multiplication",
			expression: "rate * quantity",
			env:        map[string]any{"rate": 2.5, "quantity": 100.0},
			want:       decimal.NewFromFloat(250.0),
			wantErr:    false,
		},
		{
			name:       "with builtin functions",
			expression: "round(a * b, 2)",
			env:        map[string]any{"a": 3.14159, "b": 2.0},
			want:       decimal.NewFromFloat(6.28),
			wantErr:    false,
		},
		{
			name:       "ternary true",
			expression: "flag ? a : b",
			env:        map[string]any{"flag": true, "a": 100.0, "b": 0.0},
			want:       decimal.NewFromFloat(100.0),
			wantErr:    false,
		},
		{
			name:       "ternary false",
			expression: "flag ? a : b",
			env:        map[string]any{"flag": false, "a": 100.0, "b": 50.0},
			want:       decimal.NewFromFloat(50.0),
			wantErr:    false,
		},
		{
			name:       "complex freight formula",
			expression: "max(minCharge, baseRate + distance * ratePerMile)",
			env: map[string]any{
				"minCharge":   100.0,
				"baseRate":    50.0,
				"distance":    200.0,
				"ratePerMile": 1.5,
			},
			want:    decimal.NewFromFloat(350.0),
			wantErr: false,
		},
		{
			name:       "undefined variable error",
			expression: "x + undefined",
			env:        map[string]any{"x": 1.0},
			want:       decimal.Zero,
			wantErr:    true,
		},
		{
			name:       "invalid syntax error",
			expression: "x +",
			env:        map[string]any{"x": 1.0},
			want:       decimal.Zero,
			wantErr:    true,
		},
		{
			name:       "integer result",
			expression: "a + b",
			env:        map[string]any{"a": 5, "b": 3},
			want:       decimal.NewFromInt(8),
			wantErr:    false,
		},
		{
			name:       "boolean true result",
			expression: "a > b",
			env:        map[string]any{"a": 10.0, "b": 5.0},
			want:       decimal.NewFromInt(1),
			wantErr:    false,
		},
		{
			name:       "boolean false result",
			expression: "a > b",
			env:        map[string]any{"a": 3.0, "b": 5.0},
			want:       decimal.NewFromInt(0),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := svc.EvaluateWithEnv(tt.expression, tt.env)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, tt.want.Equal(resp.Amount), "expected %s, got %s", tt.want, resp.Amount)
		})
	}
}

func TestService_EvaluateExpression(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	tests := []struct {
		name    string
		req     *formula.EvaluateExpressionRequest
		want    decimal.Decimal
		wantErr bool
	}{
		{
			name: "simple with variables",
			req: &formula.EvaluateExpressionRequest{
				Expression: "rate * weight",
				Entity:     struct{}{},
				SchemaID:   "unknown",
				Variables:  map[string]any{"rate": 1.5, "weight": 1000.0},
			},
			want:    decimal.NewFromFloat(1500.0),
			wantErr: false,
		},
		{
			name: "with clamp",
			req: &formula.EvaluateExpressionRequest{
				Expression: "clamp(x, lo, hi)",
				Entity:     struct{}{},
				SchemaID:   "unknown",
				Variables:  map[string]any{"x": 500.0, "lo": 100.0, "hi": 300.0},
			},
			want:    decimal.NewFromFloat(300.0),
			wantErr: false,
		},
		{
			name: "invalid expression",
			req: &formula.EvaluateExpressionRequest{
				Expression: "invalid +++",
				Entity:     struct{}{},
				SchemaID:   "unknown",
				Variables:  map[string]any{},
			},
			want:    decimal.Zero,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := svc.EvaluateExpression(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, tt.want.Equal(resp.Amount), "expected %s, got %s", tt.want, resp.Amount)
		})
	}
}

func TestService_GetAvailableVariables(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	t.Run("unknown schema returns nil", func(t *testing.T) {
		t.Parallel()
		vars := svc.GetAvailableVariables("nonexistent")
		assert.Nil(t, vars)
	})
}

func TestService_GetRequiredPreloads(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	t.Run("unknown schema returns nil", func(t *testing.T) {
		t.Parallel()
		preloads := svc.GetRequiredPreloads("nonexistent")
		assert.Nil(t, preloads)
	})
}

func TestService_GetEngine(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	eng := svc.GetEngine()
	require.NotNil(t, eng)
}

func TestService_GetResolver(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	res := svc.GetResolver()
	require.NotNil(t, res)
}

func TestService_EvaluateWithEnvClampFormulas(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       decimal.Decimal
	}{
		{
			name:       "rate per mile clamped",
			expression: "clamp(distance * ratePerMile, minCharge, maxCharge)",
			env: map[string]any{
				"distance":    500.0,
				"ratePerMile": 1.5,
				"minCharge":   200.0,
				"maxCharge":   600.0,
			},
			want: decimal.NewFromFloat(600.0),
		},
		{
			name:       "below minimum charge",
			expression: "clamp(distance * ratePerMile, minCharge, maxCharge)",
			env: map[string]any{
				"distance":    10.0,
				"ratePerMile": 1.5,
				"minCharge":   200.0,
				"maxCharge":   600.0,
			},
			want: decimal.NewFromFloat(200.0),
		},
		{
			name:       "within range",
			expression: "clamp(distance * ratePerMile, minCharge, maxCharge)",
			env: map[string]any{
				"distance":    200.0,
				"ratePerMile": 1.5,
				"minCharge":   200.0,
				"maxCharge":   600.0,
			},
			want: decimal.NewFromFloat(300.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := svc.EvaluateWithEnv(tt.expression, tt.env)
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(resp.Amount), "expected %s, got %s", tt.want, resp.Amount)
		})
	}
}

func TestService_EvaluateWithEnvWeightBased(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       decimal.Decimal
	}{
		{
			name:       "weight based CWT rating",
			expression: "ceil(weight / 100.0) * ratePerCWT",
			env: map[string]any{
				"weight":     4550.0,
				"ratePerCWT": 15.0,
			},
			want: decimal.NewFromFloat(690.0),
		},
		{
			name:       "per piece pricing",
			expression: "pieces * pricePerPiece",
			env: map[string]any{
				"pieces":        50.0,
				"pricePerPiece": 2.5,
			},
			want: decimal.NewFromFloat(125.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := svc.EvaluateWithEnv(tt.expression, tt.env)
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(resp.Amount), "expected %s, got %s", tt.want, resp.Amount)
		})
	}
}

func TestService_Calculate_Success(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, req repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			assert.Equal(t, templateID, req.TemplateID)
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Name:       "Distance Rate",
				Expression: "distance * rate",
				VariableDefinitions: []*formulatypes.VariableDefinition{
					{Name: "distance", Type: "number", DefaultValue: 0},
					{Name: "rate", Type: "number", DefaultValue: 0},
				},
			}, nil
		},
	}

	svc := setupServiceWithRepo(t, repo)

	resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: templateID,
		Entity:     struct{}{},
		Variables:  map[string]any{"distance": 100.0, "rate": 2.5},
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(
		t,
		decimal.NewFromFloat(250.0).Equal(resp.Amount),
		"expected 250, got %s",
		resp.Amount,
	)
}

func TestService_Calculate_RepoError(t *testing.T) {
	t.Parallel()

	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return nil, errors.New("template not found")
		},
	}

	svc := setupServiceWithRepo(t, repo)

	resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: pulid.MustNew("ft_"),
		Entity:     struct{}{},
		Variables:  map[string]any{},
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "template not found")
}

func TestService_Calculate_EvaluateError(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ repositories.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Name:       "Bad Formula",
				Expression: "invalid +++",
			}, nil
		},
	}

	svc := setupServiceWithRepo(t, repo)

	resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: templateID,
		Entity:     struct{}{},
		Variables:  map[string]any{},
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	})

	require.Error(t, err)
	assert.Nil(t, resp)
}
