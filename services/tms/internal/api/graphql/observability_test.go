package graphql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type stubTracer struct{ enabled bool }

func (stubTracer) StartSpan(
	ctx context.Context,
	_ string,
	_ ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

func (stubTracer) AddEvent(context.Context, string, ...attribute.KeyValue) {}

func (stubTracer) SetAttributes(context.Context, ...attribute.KeyValue) {}

func (stubTracer) RecordError(context.Context, error, ...trace.EventOption) {}

func (t stubTracer) IsEnabled() bool { return t.enabled }

func newTestExtension(
	t *testing.T,
	resolverMetrics bool,
) (*ObservabilityExtension, *prometheus.Registry) {
	t.Helper()

	registry := prometheus.NewRegistry()
	gql := metrics.NewGraphQL(registry, zap.NewNop(), true, metrics.GraphQLOptions{
		ResolverMetrics: resolverMetrics,
		MaxOperations:   100,
	})

	return &ObservabilityExtension{
		metrics:       gql,
		l:             zap.NewNop(),
		tracer:        stubTracer{},
		slowThreshold: time.Second,
	}, registry
}

func counterValue(t *testing.T, registry *prometheus.Registry, name string) float64 {
	t.Helper()

	gathered, err := registry.Gather()
	require.NoError(t, err)

	total := 0.0
	for _, mf := range gathered {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if c := m.GetCounter(); c != nil {
				total += c.GetValue()
			}
			if h := m.GetHistogram(); h != nil {
				total += float64(h.GetSampleCount())
			}
		}
	}

	return total
}

func operationCtx(name string, op ast.Operation) context.Context {
	now := time.Now()

	return graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
		OperationName: name,
		Operation:     &ast.OperationDefinition{Operation: op},
		Stats: graphql.Stats{
			Parsing:    graphql.TraceTiming{Start: now, End: now.Add(time.Millisecond)},
			Validation: graphql.TraceTiming{Start: now, End: now.Add(2 * time.Millisecond)},
		},
	})
}

func TestObservabilityExtension_Contract(t *testing.T) {
	t.Parallel()

	e, _ := newTestExtension(t, false)

	assert.Equal(t, observabilityExtensionName, e.ExtensionName())
	assert.NoError(t, e.Validate(nil))
}

func TestNewObservabilityExtension_UsesConfiguredThreshold(t *testing.T) {
	t.Parallel()

	registry := &metrics.Registry{
		GraphQL: metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
	}

	e := NewObservabilityExtension(ObservabilityParams{
		Config: &config.Config{
			Monitoring: config.MonitoringConfig{
				GraphQL: config.GraphQLObservConfig{SlowOperationAfter: 5 * time.Second},
			},
		},
		Logger:  zap.NewNop(),
		Metrics: registry,
		Tracer:  stubTracer{},
	})

	assert.Equal(t, 5*time.Second, e.slowThreshold)
	assert.NotNil(t, e.metrics)
}

func TestObservabilityExtension_InterceptResponse_RecordsSuccess(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, false)
	ctx := operationCtx("ListShipments", ast.Query)

	resp := e.InterceptResponse(ctx, func(context.Context) *graphql.Response {
		return &graphql.Response{}
	})

	require.NotNil(t, resp)
	assert.InDelta(t, 1.0, counterValue(t, registry, "trenova_graphql_operations_total"), 0.001)
	assert.InDelta(
		t,
		1.0,
		counterValue(t, registry, "trenova_graphql_operation_duration_seconds"),
		0.001,
	)
	assert.Zero(t, counterValue(t, registry, "trenova_graphql_operation_errors_total"))
	assert.InDelta(
		t,
		1.0,
		counterValue(t, registry, "trenova_graphql_parse_duration_seconds"),
		0.001,
	)
}

func TestObservabilityExtension_InterceptResponse_RecordsErrors(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, false)
	ctx := operationCtx("PostPayment", ast.Mutation)

	resp := e.InterceptResponse(ctx, func(context.Context) *graphql.Response {
		return &graphql.Response{
			Errors: gqlerror.List{
				{Message: "boom", Extensions: map[string]any{"code": "Invalid"}},
			},
		}
	})

	require.NotNil(t, resp)
	assert.InDelta(
		t,
		1.0,
		counterValue(t, registry, "trenova_graphql_operation_errors_total"),
		0.001,
	)
}

func TestObservabilityExtension_InterceptResponse_WithoutOperationContext(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, false)
	called := false

	resp := e.InterceptResponse(context.Background(), func(context.Context) *graphql.Response {
		called = true
		return &graphql.Response{}
	})

	assert.True(t, called)
	require.NotNil(t, resp)
	assert.Zero(t, counterValue(t, registry, "trenova_graphql_operations_total"))
}

func TestObservabilityExtension_InterceptResponse_NilResponse(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, false)
	ctx := operationCtx("Anything", ast.Query)

	assert.NotPanics(t, func() {
		e.InterceptResponse(ctx, func(context.Context) *graphql.Response { return nil })
	})
	assert.InDelta(t, 1.0, counterValue(t, registry, "trenova_graphql_operations_total"), 0.001)
}

func TestObservabilityExtension_InterceptField_OnlyResolverBackedFields(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, true)

	plain := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object:     "Shipment",
		Field:      graphql.CollectedField{Field: &ast.Field{Name: "id"}},
		IsResolver: false,
	})
	_, err := e.InterceptField(plain, func(context.Context) (any, error) { return "x", nil })
	require.NoError(t, err)
	assert.Zero(t, counterValue(t, registry, "trenova_graphql_resolver_calls_total"))

	resolved := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object:     "CustomerPayment",
		Field:      graphql.CollectedField{Field: &ast.Field{Name: "customer"}},
		IsResolver: true,
	})
	_, err = e.InterceptField(resolved, func(context.Context) (any, error) { return "y", nil })
	require.NoError(t, err)
	assert.InDelta(
		t,
		1.0,
		counterValue(t, registry, "trenova_graphql_resolver_calls_total"),
		0.001,
	)
}

func TestObservabilityExtension_InterceptField_RecordsErrors(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, true)
	ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object:     "Tender",
		Field:      graphql.CollectedField{Field: &ast.Field{Name: "routingGuide"}},
		IsResolver: true,
	})

	_, err := e.InterceptField(ctx, func(context.Context) (any, error) {
		return nil, errors.New("lookup failed")
	})

	require.Error(t, err)
	assert.InDelta(
		t,
		1.0,
		counterValue(t, registry, "trenova_graphql_resolver_errors_total"),
		0.001,
	)
}

func TestObservabilityExtension_InterceptField_DisabledIsPassthrough(t *testing.T) {
	t.Parallel()

	e, registry := newTestExtension(t, false)
	ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object:     "CustomerPayment",
		Field:      graphql.CollectedField{Field: &ast.Field{Name: "customer"}},
		IsResolver: true,
	})

	res, err := e.InterceptField(ctx, func(context.Context) (any, error) { return "v", nil })

	require.NoError(t, err)
	assert.Equal(t, "v", res)
	assert.Zero(t, counterValue(t, registry, "trenova_graphql_resolver_calls_total"))
}

func TestOperationType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, unknownOperationType, operationType(nil))
	assert.Equal(t, unknownOperationType, operationType(&graphql.OperationContext{}))
	assert.Equal(t, unknownOperationType, operationType(&graphql.OperationContext{
		Operation: &ast.OperationDefinition{},
	}))
	assert.Equal(t, "query", operationType(&graphql.OperationContext{
		Operation: &ast.OperationDefinition{Operation: ast.Query},
	}))
	assert.Equal(t, "mutation", operationType(&graphql.OperationContext{
		Operation: &ast.OperationDefinition{Operation: ast.Mutation},
	}))
}

func TestFirstErrorCode(t *testing.T) {
	t.Parallel()

	assert.Empty(t, firstErrorCode(nil))
	assert.Equal(t, unknownErrorCode, firstErrorCode(gqlerror.List{{Message: "no extensions"}}))
	assert.Equal(t, unknownErrorCode, firstErrorCode(gqlerror.List{
		{Message: "wrong type", Extensions: map[string]any{"code": 42}},
	}))
	assert.Equal(t, "Forbidden", firstErrorCode(gqlerror.List{
		{Message: "first has none"},
		{Message: "second has one", Extensions: map[string]any{"code": "Forbidden"}},
	}))
}

func TestPhaseSeconds(t *testing.T) {
	t.Parallel()

	assert.Zero(t, phaseSeconds(graphql.TraceTiming{}))
	assert.Zero(t, phaseSeconds(graphql.TraceTiming{Start: time.Now()}))

	start := time.Now()
	assert.InDelta(
		t,
		0.5,
		phaseSeconds(graphql.TraceTiming{Start: start, End: start.Add(500 * time.Millisecond)}),
		0.001,
	)
}

func TestResponseErrors(t *testing.T) {
	t.Parallel()

	assert.Nil(t, responseErrors(nil))
	assert.Len(t, responseErrors(&graphql.Response{
		Errors: gqlerror.List{{Message: "a"}},
	}), 1)
}
