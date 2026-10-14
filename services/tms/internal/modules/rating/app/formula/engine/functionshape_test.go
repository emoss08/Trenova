package engine_test

import (
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/expr-lang/expr/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func specsByName() map[string]engine.FunctionSpec {
	specs := engine.FunctionSpecs()
	byName := make(map[string]engine.FunctionSpec, len(specs))
	for _, spec := range specs {
		byName[spec.Name] = spec
	}
	return byName
}

func TestBuiltinSpecsDescribeEveryExprBuiltinTheEngineKeeps(t *testing.T) {
	t.Parallel()

	documented := specsByName()
	disabled := make(map[string]struct{})
	for _, fn := range engine.DisabledFunctions() {
		disabled[fn.Name] = struct{}{}
	}

	want := make([]string, 0, len(builtin.Builtins))
	for _, fn := range builtin.Builtins {
		if _, off := disabled[fn.Name]; off {
			continue
		}
		if _, ours := documented[fn.Name]; ours {
			continue
		}
		want = append(want, fn.Name)
	}

	got := make([]string, 0, len(engine.BuiltinSpecs()))
	for _, spec := range engine.BuiltinSpecs() {
		got = append(got, spec.Name)
	}

	sort.Strings(want)
	sort.Strings(got)
	assert.Equal(t, want, got)
}

func TestBuiltinSpecsMarkTheBuiltinsThatTakeAPredicate(t *testing.T) {
	t.Parallel()

	predicates := make(map[string]bool, len(builtin.Builtins))
	for _, fn := range builtin.Builtins {
		predicates[fn.Name] = fn.Predicate
	}

	for _, spec := range engine.BuiltinSpecs() {
		assert.Equal(t, predicates[spec.Name], spec.Predicate, spec.Name)
		assert.NotEmpty(t, spec.Signature, spec.Name)
		assert.NotEmpty(t, spec.Returns, spec.Name)
		assert.True(t, spec.MaxArgs == engine.VariadicArgs || spec.MaxArgs >= spec.MinArgs, spec.Name)
	}
}

func TestFunctionSpecsDeriveTheirShapeFromTheirSignatures(t *testing.T) {
	t.Parallel()

	specs := specsByName()

	cases := map[string]engine.FunctionShape{
		"round":    {MinArgs: 1, MaxArgs: 2, Returns: engine.ValueNumber, NumericArgs: true},
		"clamp":    {MinArgs: 3, MaxArgs: 3, Returns: engine.ValueNumber, NumericArgs: true},
		"min":      {MinArgs: 1, MaxArgs: engine.VariadicArgs, Returns: engine.ValueNumber, NumericArgs: true},
		"sum":      {MinArgs: 1, MaxArgs: engine.VariadicArgs, Returns: engine.ValueNumber},
		"avg":      {MinArgs: 1, MaxArgs: engine.VariadicArgs, Returns: engine.ValueNumber},
		"coalesce": {MinArgs: 1, MaxArgs: engine.VariadicArgs, Returns: engine.ValueAny},
	}
	for name, want := range cases {
		spec, ok := specs[name]
		require.True(t, ok, name)
		assert.Equal(t, want, spec.FunctionShape, name)
	}

	for _, spec := range specs {
		assert.NotEmpty(t, spec.Returns, "%s publishes what it returns", spec.Name)
		assert.True(
			t,
			spec.MaxArgs == engine.VariadicArgs || spec.MaxArgs >= spec.MinArgs,
			"%s has a coherent arity",
			spec.Name,
		)
	}
}

func TestDisabledFunctionsSayWhyAndWhatToUseInstead(t *testing.T) {
	t.Parallel()

	disabled := engine.DisabledFunctions()
	names := make([]string, 0, len(disabled))
	for _, fn := range disabled {
		names = append(names, fn.Name)
		assert.NotEmpty(t, fn.Reason, fn.Name)
		if fn.Name == "now" {
			assert.Equal(t, engine.RatingTimeVariable, fn.Replacement)
		}
	}
	assert.Equal(t, []string{"fromJSON", "now", "repeat", "toJSON"}, names)
}
