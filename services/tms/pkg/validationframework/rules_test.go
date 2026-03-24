package validationframework

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     ValidationMode
		expected int
	}{
		{"ModeCreate is 0", ModeCreate, 0},
		{"ModeUpdate is 1", ModeUpdate, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.mode))
		})
	}
}

func TestTenantedValidationContext(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	entityID := pulid.MustNew("ent_")

	t.Run("IsCreate returns true for ModeCreate", func(t *testing.T) {
		ctx := &TenantedValidationContext{
			Mode:           ModeCreate,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		}

		assert.True(t, ctx.IsCreate())
		assert.False(t, ctx.IsUpdate())
	})

	t.Run("IsUpdate returns true for ModeUpdate", func(t *testing.T) {
		ctx := &TenantedValidationContext{
			Mode:           ModeUpdate,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EntityID:       entityID,
		}

		assert.False(t, ctx.IsCreate())
		assert.True(t, ctx.IsUpdate())
	})
}

func TestNewTenantedRule(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule")

	require.NotNil(t, rule)
	assert.Equal(t, "test_rule", rule.Name())
	assert.Equal(t, ValidationStageBusinessRules, rule.Stage())
	assert.Equal(t, ValidationPriorityMedium, rule.Priority())
}

func TestBaseTenantedRule_OnCreate(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").OnCreate()

	createCtx := &TenantedValidationContext{Mode: ModeCreate}
	updateCtx := &TenantedValidationContext{Mode: ModeUpdate}

	assert.True(t, rule.ShouldRun(createCtx))
	assert.False(t, rule.ShouldRun(updateCtx))
}

func TestBaseTenantedRule_OnUpdate(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").OnUpdate()

	createCtx := &TenantedValidationContext{Mode: ModeCreate}
	updateCtx := &TenantedValidationContext{Mode: ModeUpdate}

	assert.False(t, rule.ShouldRun(createCtx))
	assert.True(t, rule.ShouldRun(updateCtx))
}

func TestBaseTenantedRule_OnBoth(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").
		OnCreate().
		OnBoth()

	createCtx := &TenantedValidationContext{Mode: ModeCreate}
	updateCtx := &TenantedValidationContext{Mode: ModeUpdate}

	assert.True(t, rule.ShouldRun(createCtx))
	assert.True(t, rule.ShouldRun(updateCtx))
}

func TestBaseTenantedRule_WithStage(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").
		WithStage(ValidationStageCompliance)

	assert.Equal(t, ValidationStageCompliance, rule.Stage())
}

func TestBaseTenantedRule_WithPriority(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").
		WithPriority(ValidationPriorityHigh)

	assert.Equal(t, ValidationPriorityHigh, rule.Priority())
}

func TestBaseTenantedRule_WithValidation(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").
		WithValidation(func(
			_ context.Context,
			entity *mockTenantedEntity,
			_ *TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			executed = true
			if entity.Name == "" {
				multiErr.Add("name", errortypes.ErrRequired, "Name is required")
			}
			return nil
		})

	entity := &mockTenantedEntity{Name: ""}
	valCtx := &TenantedValidationContext{Mode: ModeCreate}
	multiErr := errortypes.NewMultiError()

	err := rule.Validate(t.Context(), entity, valCtx, multiErr)

	require.NoError(t, err)
	assert.True(t, executed)
	assert.True(t, multiErr.HasErrors())
}

func TestBaseTenantedRule_ValidateWithNilFunction(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule")

	entity := &mockTenantedEntity{}
	valCtx := &TenantedValidationContext{Mode: ModeCreate}
	multiErr := errortypes.NewMultiError()

	err := rule.Validate(t.Context(), entity, valCtx, multiErr)

	require.NoError(t, err)
	assert.False(t, multiErr.HasErrors())
}

func TestBaseTenantedRule_FluentChaining(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("test_rule").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		OnCreate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			return nil
		})

	assert.Equal(t, "test_rule", rule.Name())
	assert.Equal(t, ValidationStageDataIntegrity, rule.Stage())
	assert.Equal(t, ValidationPriorityHigh, rule.Priority())
	assert.True(t, rule.ShouldRun(&TenantedValidationContext{Mode: ModeCreate}))
	assert.False(t, rule.ShouldRun(&TenantedValidationContext{Mode: ModeUpdate}))
}
