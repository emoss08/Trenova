package validationtest

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

type contextConfig struct {
	mode           validationframework.Mode
	original       any
	organizationID pulid.ID
	businessUnitID pulid.ID
}

type ContextOption func(*contextConfig)

func Update() ContextOption {
	return func(cfg *contextConfig) {
		cfg.mode = validationframework.ModeUpdate
	}
}

func UpdateFrom(original validationframework.Entity) ContextOption {
	return func(cfg *contextConfig) {
		cfg.mode = validationframework.ModeUpdate
		cfg.original = original
	}
}

func InTenant(organizationID, businessUnitID pulid.ID) ContextOption {
	return func(cfg *contextConfig) {
		cfg.organizationID = organizationID
		cfg.businessUnitID = businessUnitID
	}
}

func NewContext(
	t testing.TB,
	entity validationframework.Entity,
	opts ...ContextOption,
) *validationframework.Context {
	t.Helper()

	cfg := contextConfig{mode: validationframework.ModeCreate}
	for _, opt := range opts {
		opt(&cfg)
	}

	valCtx := validationframework.NewContext(t.Context(), entity, cfg.mode)
	valCtx.Original = cfg.original
	if cfg.organizationID.IsNotNil() || cfg.businessUnitID.IsNotNil() {
		valCtx.OrganizationID = cfg.organizationID
		valCtx.BusinessUnitID = cfg.businessUnitID
	}

	return valCtx
}

func RunRule[T validationframework.Entity](
	t testing.TB,
	rule validationframework.Rule[T],
	entity T,
	opts ...ContextOption,
) *Result {
	t.Helper()

	multiErr, err := runRule(t, rule, entity, opts)
	if err != nil {
		t.Fatalf("rule %s could not complete: %v", rule.Name(), err)
		return nil
	}

	return Report(t, multiErr)
}

func RunRuleFailure[T validationframework.Entity](
	t testing.TB,
	rule validationframework.Rule[T],
	entity T,
	opts ...ContextOption,
) error {
	t.Helper()

	multiErr, err := runRule(t, rule, entity, opts)
	if err == nil {
		err = multiErr.Err()
	}

	return Failed(t, err)
}

func runRule[T validationframework.Entity](
	t testing.TB,
	rule validationframework.Rule[T],
	entity T,
	opts []ContextOption,
) (*errortypes.MultiError, error) {
	t.Helper()

	valCtx := NewContext(t, entity, opts...)
	multiErr := errortypes.NewMultiError()
	if !rule.ShouldRun(valCtx.Mode) {
		return multiErr, nil
	}

	return multiErr, rule.Validate(t.Context(), entity, valCtx, multiErr)
}
