package formula_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockFormulaTemplateRepo struct {
	getByIDFunc func(ctx context.Context, req formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error)
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
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, req)
	}
	return nil, errors.New("not found")
}

func (m *mockFormulaTemplateRepo) GetByIDs(
	_ context.Context,
	_ formulatemplateports.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) FindByNames(
	_ context.Context,
	_ formulatemplateports.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) List(
	_ context.Context,
	_ *formulatemplateports.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) ListConnection(
	_ context.Context,
	_ *formulatemplateports.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) BulkUpdateStatus(
	_ context.Context,
	_ *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) BulkDuplicate(
	_ context.Context,
	_ *formulatemplateports.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) CountUsages(
	_ context.Context,
	_ *formulatemplateports.GetTemplateUsageRequest,
) (*formulatemplateports.GetTemplateUsageResponse, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepo) SelectOptions(
	_ context.Context,
	_ *formulatemplateports.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return nil, nil
}

func newTestRegistry(t *testing.T) *schema.Registry {
	t.Helper()

	registry := schema.NewRegistry()

	shipmentSchema, err := os.ReadFile("schema/definitions/shipment.schema.json")
	require.NoError(t, err)
	require.NoError(t, registry.Register("shipment", shipmentSchema))

	return registry
}

func setupService(t *testing.T) *formula.Service {
	t.Helper()
	return setupServiceWithRepo(t, nil)
}

func setupServiceWithRepo(
	t *testing.T,
	repo formulatemplateports.FormulaTemplateRepository,
) *formula.Service {
	t.Helper()
	return setupServiceWithRepoAndMatrixData(t, repo, nil)
}

func setupServiceWithMatrixData(
	t *testing.T,
	data []*ratematrixports.RateMatrixLookupData,
) *formula.Service {
	t.Helper()
	return setupServiceWithRepoAndMatrixData(t, nil, data)
}

func setupServiceWithRepoAndMatrixData(
	t *testing.T,
	repo formulatemplateports.FormulaTemplateRepository,
	data []*ratematrixports.RateMatrixLookupData,
) *formula.Service {
	t.Helper()

	registry := newTestRegistry(t)
	res := resolver.NewResolver()

	return formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Environment: environment.NewBuilder(environment.Params{
			Registry: registry,
			Resolver: res,
		}),
		Resolver:       res,
		Repo:           repo,
		VersionRepo:    &stubVersionRepo{},
		RateMatrixRepo: &stubMatrixRepo{data: data},
	})
}

func checkOne(t *testing.T, svc *formula.Service, expression string) *rules.FormulaCheck {
	t.Helper()

	checked, err := svc.Check(t.Context(), &formula.CheckRequest{
		SchemaID: "shipment",
		Expect:   rules.ExpectAmount,
		Sources:  []formula.CheckSource{{Scope: formula.MainScope, Source: expression}},
	})
	require.NoError(t, err)
	require.Len(t, checked, 1)
	return checked[0].Check
}

type stubVersionRepo struct {
	formulatemplateports.FormulaTemplateVersionRepository
	effectiveVersion *formulatemplate.FormulaTemplateVersion
}

func (s *stubVersionRepo) GetEffectiveVersion(
	_ context.Context,
	_ *formulatemplateports.GetEffectiveVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return s.effectiveVersion, nil
}

func (s *stubVersionRepo) ListApproved(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	if s.effectiveVersion == nil {
		return nil, nil
	}
	return []*formulatemplate.FormulaTemplateVersion{s.effectiveVersion}, nil
}

func (s *stubVersionRepo) ListScheduled(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	if s.effectiveVersion == nil {
		return nil, nil
	}
	return []*formulatemplate.FormulaTemplateVersion{s.effectiveVersion}, nil
}

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

func TestService_RequiredPreloads(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	t.Run("unknown schema returns nil", func(t *testing.T) {
		t.Parallel()
		preloads := svc.RequiredPreloads("nonexistent")
		assert.Nil(t, preloads)
	})
}

func TestService_Calculate_Success(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, req formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			assert.Equal(t, templateID, req.TemplateID)
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Status:     formulatemplate.StatusActive,
				Name:       "Distance Rate",
				SchemaID:   "shipment",
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
		getByIDFunc: func(_ context.Context, _ formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
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
		getByIDFunc: func(_ context.Context, _ formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Status:     formulatemplate.StatusActive,
				Name:       "Bad Formula",
				SchemaID:   "shipment",
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

func TestService_Calculate_UnknownSchema(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Status:     formulatemplate.StatusActive,
				Name:       "Unknown Schema",
				SchemaID:   "nonexistent",
				Expression: "1 + 1",
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
	require.ErrorIs(t, err, environment.ErrSchemaNotFound)
	assert.Nil(t, resp)
}

func TestService_Calculate_UndeclaredVariableShadowsField(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	repo := &mockFormulaTemplateRepo{
		getByIDFunc: func(_ context.Context, _ formulatemplateports.GetFormulaTemplateByIDRequest) (*formulatemplate.FormulaTemplate, error) {
			return &formulatemplate.FormulaTemplate{
				ID:         templateID,
				Status:     formulatemplate.StatusActive,
				Name:       "Weight Rate",
				SchemaID:   "shipment",
				Expression: "weight * rate",
				VariableDefinitions: []*formulatypes.VariableDefinition{
					{Name: "rate", Type: "number", DefaultValue: 1.0},
				},
			}, nil
		},
	}

	svc := setupServiceWithRepo(t, repo)

	weight := int64(5000)
	entity := &struct{ Weight *int64 }{Weight: &weight}

	tests := []struct {
		name      string
		variables map[string]any
		want      decimal.Decimal
		wantErr   bool
	}{
		{
			name:      "undeclared variable shadowing schema field is rejected",
			variables: map[string]any{"weight": 1.0},
			wantErr:   true,
		},
		{
			name:      "undeclared nested variable shadowing schema field root is rejected",
			variables: map[string]any{"weight.override": 1.0},
			wantErr:   true,
		},
		{
			name:      "declared variable override is allowed",
			variables: map[string]any{"rate": 2.0},
			want:      decimal.NewFromFloat(10000.0),
		},
		{
			name:      "undeclared variable without field collision is allowed",
			variables: map[string]any{"customRate": 9.99},
			want:      decimal.NewFromFloat(5000.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
				TemplateID: templateID,
				Entity:     entity,
				Variables:  tt.variables,
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is already a shipment value")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, tt.want.Equal(resp.Amount), "expected %s, got %s", tt.want, resp.Amount)
		})
	}
}

func TestService_EvaluatePredicates_AnswersEveryConditionInOneBatch(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	entity := &shipment.Shipment{
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{Type: shipment.StopTypePickup},
					{Type: shipment.StopTypeDelivery},
					{Type: shipment.StopTypeDelivery},
				},
			},
		},
	}

	expressions := []string{
		"totalStops > 2",
		"totalStops > 5",
		"true",
		"0",
		"totalStops",
		"invalid +++",
		"weight > 0",
	}
	outcomes, err := svc.EvaluatePredicates(
		t.Context(),
		&formulatemplateports.EvaluatePredicatesRequest{
			Expressions: expressions,
			SchemaID:    "shipment",
			Entity:      entity,
		},
	)
	require.NoError(t, err)
	require.Len(t, outcomes, len(expressions))

	for index, want := range []bool{true, false, true, false, true} {
		require.NoError(t, outcomes[index].Err, expressions[index])
		assert.Equal(t, want, outcomes[index].Holds, expressions[index])
	}

	require.Error(t, outcomes[5].Err, "a condition that does not parse fails on its own")
	var evaluationErr *formulaerrors.EvaluationError
	require.ErrorAs(t, outcomes[5].Err, &evaluationErr)
	assert.Equal(t, "syntax", evaluationErr.Err.Code)

	require.Error(t, outcomes[6].Err, "an empty weight is missing, not zero")
	require.ErrorAs(t, outcomes[6].Err, &evaluationErr)
	assert.Equal(t, rules.ErrorMissingField, evaluationErr.Err.Code)
	assert.Equal(t, []string{"weight"}, evaluationErr.Err.Fields)
}

func TestApplyChargePolicy_GuardrailsThenRounding(t *testing.T) {
	t.Parallel()

	policy := formulatypes.ChargePolicy{
		MinCharge:         decimal.NewNullDecimal(decimal.NewFromInt(250)),
		RoundingMode:      ratetypes.RoundingModeUp,
		RoundingPrecision: 0,
	}

	amount, guardrail, rounding, err := formula.ApplyChargePolicy(
		policy,
		decimal.RequireFromString("249.996"),
	)
	require.NoError(t, err)
	require.NotNil(t, guardrail)
	assert.True(t, guardrail.Applied)
	assert.True(t, decimal.NewFromInt(250).Equal(amount), amount.String())
	require.NotNil(t, rounding)
	assert.False(t, rounding.Applied, "the floor is already whole")
	assert.Equal(t, "Up", rounding.Mode)

	amount, _, rounding, err = formula.ApplyChargePolicy(policy, decimal.RequireFromString("312.01"))
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(313).Equal(amount), amount.String())
	assert.True(t, rounding.Applied)
	assert.True(t, decimal.RequireFromString("312.01").Equal(rounding.UnroundedAmount))
}

func TestApplyChargePolicy_DefaultsAndNone(t *testing.T) {
	t.Parallel()

	amount, _, rounding, err := formula.ApplyChargePolicy(
		formulatypes.ChargePolicy{},
		decimal.RequireFromString("2.675"),
	)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2.68").Equal(amount), amount.String())
	assert.Equal(t, "HalfUp", rounding.Mode)
	assert.Equal(t, int32(2), rounding.Precision)

	amount, _, rounding, err = formula.ApplyChargePolicy(
		formulatypes.ChargePolicy{RoundingMode: ratetypes.RoundingModeNone},
		decimal.RequireFromString("2.675"),
	)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("2.675").Equal(amount))
	assert.False(t, rounding.Applied)
}

func TestService_Rate_AppliesTemplateRoundingPolicy(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	template := &formulatemplate.FormulaTemplate{
		ID:                pulid.MustNew("ft_"),
		Name:              "thirds",
		Expression:        "10 / 3",
		SchemaID:          "shipment",
		Status:            formulatemplate.StatusActive,
		RoundingMode:      ratetypes.RoundingModeHalfEven,
		RoundingPrecision: 3,
	}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   &shipment.Shipment{},
	})

	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("3.333").Equal(resp.Amount), resp.Amount.String())
	require.NotNil(t, resp.Rounding)
	assert.True(t, resp.Rounding.Applied)
	assert.Equal(t, "HalfEven", resp.Rounding.Mode)
}

func TestService_Rate_MissingNullableFieldIsAValidationError(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("ft_"),
		Name:       "per pound",
		Expression: "weight * 0.5",
		SchemaID:   "shipment",
		Status:     formulatemplate.StatusActive,
	}

	_, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   &shipment.Shipment{},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "must surface as a validation problem, not a 500")
	assert.Contains(t, err.Error(), "weight")
	assert.Contains(t, err.Error(), "coalesce(weight, 0)")

	var evaluationErr *formulaerrors.EvaluationError
	require.ErrorAs(t, err, &evaluationErr)
	assert.Equal(t, rules.ErrorMissingField, evaluationErr.Err.Code)
	assert.Equal(t, []string{"weight"}, evaluationErr.Err.Fields)
	assert.Equal(t, []string{"coalesce(weight, 0)"}, evaluationErr.Err.Guards)
}

func (m *mockFormulaTemplateRepo) CountStatsByIDs(
	context.Context,
	*formulatemplateports.GetFormulaTemplateStatsRequest,
) (map[pulid.ID]formulatemplateports.TemplateStats, error) {
	return map[pulid.ID]formulatemplateports.TemplateStats{}, nil
}

func (*stubMatrixRepo) GetLookupStamp(context.Context, pagination.TenantInfo) (string, error) {
	return "", nil
}
