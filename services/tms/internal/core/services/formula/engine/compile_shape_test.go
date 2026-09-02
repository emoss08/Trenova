package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// weighed lacks its weight; weighedAt has one. Both resolve into the same
// schema shape, which is the point: one program must serve either record.
type weighed struct {
	Weight *float64
}

type weighedAt struct {
	Weight float64
}

func compileShapeEngine(t *testing.T, cacheSize int) *engine.Engine {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	require.NoError(t, registry.Register("test", []byte(testSchemaJSON)))
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	e, err := engine.NewEngine(engine.Params{
		Registry:         registry,
		Resolver:         res,
		EnvBuilder:       envBuilder,
		CompileCacheSize: cacheSize,
	})
	require.NoError(t, err)
	return e
}

func evaluateWeight(t *testing.T, e *engine.Engine, expression string, entity any) float64 {
	t.Helper()

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template: &formulatemplate.FormulaTemplate{
			ID:         pulid.MustNew("ft_"),
			Expression: expression,
			SchemaID:   "test",
		},
		Entity: entity,
		Lookup: engine.StubLookup{},
	})
	require.NoError(t, err, expression)
	return result.Value.InexactFloat64()
}

func TestCompile_NullableFieldShapesShareOneProgram(t *testing.T) {
	t.Parallel()

	e := compileShapeEngine(t, 0)

	assert.InDelta(t, 0, evaluateWeight(t, e, "coalesce(weight, 0) * 2", &weighed{}), 0.0001)
	assert.InDelta(t, 20,
		evaluateWeight(t, e, "coalesce(weight, 0) * 2", &weighedAt{Weight: 10}), 0.0001)

	assert.Equal(t, 1, e.CacheLen(),
		"a record with and a record without a weight compile to the same program")
}

func TestCompile_NilComparisonsWorkWhateverRecordCompiledFirst(t *testing.T) {
	t.Parallel()

	e := compileShapeEngine(t, 0)
	expression := "weight == nil ? 5 : weight * 2"

	assert.InDelta(t, 20, evaluateWeight(t, e, expression, &weighedAt{Weight: 10}), 0.0001)
	assert.InDelta(t, 5, evaluateWeight(t, e, expression, &weighed{}), 0.0001)
	assert.Equal(t, 1, e.CacheLen())
}

func TestNewEngine_CompileCacheSizeIsConfigurable(t *testing.T) {
	t.Parallel()

	e := compileShapeEngine(t, 1)

	evaluateWeight(t, e, "coalesce(weight, 0) + 1", &weighed{})
	evaluateWeight(t, e, "coalesce(weight, 0) + 2", &weighed{})

	assert.Equal(t, 1, e.CacheLen(), "the cache holds what it was sized for")
}
