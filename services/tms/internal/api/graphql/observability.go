package graphql

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var complexityStatsKey = extension.ComplexityLimit{}.ExtensionName()

const (
	observabilityExtensionName = "TrenovaObservability"
	unknownOperationType       = "unknown"
	unknownErrorCode           = "unknown"
)

type ObservabilityParams struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Metrics *metrics.Registry
	Tracer  observability.Tracer
}

type ObservabilityExtension struct {
	metrics       *metrics.GraphQL
	l             *zap.Logger
	tracer        observability.Tracer
	slowThreshold time.Duration
}

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
	graphql.FieldInterceptor
} = (*ObservabilityExtension)(nil)

func NewObservabilityExtension(p ObservabilityParams) *ObservabilityExtension {
	return &ObservabilityExtension{
		metrics:       p.Metrics.GraphQL,
		l:             p.Logger.Named("api.graphql.observability"),
		tracer:        p.Tracer,
		slowThreshold: p.Config.GetGraphQLObservConfig().GetSlowOperationAfter(),
	}
}

func (*ObservabilityExtension) ExtensionName() string {
	return observabilityExtensionName
}

func (*ObservabilityExtension) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e *ObservabilityExtension) InterceptResponse(
	ctx context.Context,
	next graphql.ResponseHandler,
) *graphql.Response {
	if !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}

	opCtx := graphql.GetOperationContext(ctx)
	name := e.metrics.BoundOperationName(opCtx.OperationName)
	opType := operationType(opCtx)

	e.metrics.IncrementActiveOperations()
	defer e.metrics.DecrementActiveOperations()

	execCtx, span := e.startOperationSpan(ctx, name, opType)
	if span != nil {
		defer span.End()
	}

	start := time.Now()
	resp := next(execCtx)
	duration := time.Since(start)

	errs := responseErrors(resp)
	e.metrics.RecordOperation(metrics.RecordOperationParams{
		Operation: name,
		Type:      opType,
		ErrorCode: firstErrorCode(errs),
		Duration:  duration.Seconds(),
		Errors:    len(errs),
	})
	e.metrics.RecordParsePhases(
		phaseSeconds(opCtx.Stats.Parsing),
		phaseSeconds(opCtx.Stats.Validation),
	)
	if cost, ok := operationCost(opCtx); ok {
		e.metrics.RecordOperationCost(name, opType, float64(cost))
	}

	if span != nil {
		span.SetAttributes(attribute.Int("graphql.errors.count", len(errs)))
		if len(errs) > 0 {
			span.SetStatus(codes.Error, errs[0].Message)
		}
	}

	e.logOperation(ctx, name, opType, duration, errs)

	return resp
}

func (e *ObservabilityExtension) InterceptField(
	ctx context.Context,
	next graphql.Resolver,
) (any, error) {
	if !e.metrics.ResolverMetricsEnabled() {
		return next(ctx)
	}

	fc := graphql.GetFieldContext(ctx)
	if fc == nil || !fc.IsResolver {
		return next(ctx)
	}

	resolverCtx, span := e.startResolverSpan(ctx, fc.Object, fc.Field.Name)
	if span != nil {
		defer span.End()
	}

	start := time.Now()
	res, err := next(resolverCtx)

	e.metrics.RecordResolver(metrics.RecordResolverParams{
		Object:   fc.Object,
		Field:    fc.Field.Name,
		Duration: time.Since(start).Seconds(),
		Failed:   err != nil,
	})

	if err != nil && span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return res, err
}

func (e *ObservabilityExtension) startOperationSpan(
	ctx context.Context,
	name, opType string,
) (context.Context, trace.Span) {
	if !e.tracer.IsEnabled() {
		return ctx, nil
	}

	return e.tracer.StartSpan(
		ctx,
		"graphql."+opType+" "+name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("graphql.operation.name", name),
			attribute.String("graphql.operation.type", opType),
		),
	)
}

func (e *ObservabilityExtension) startResolverSpan(
	ctx context.Context,
	object, field string,
) (context.Context, trace.Span) {
	if !e.tracer.IsEnabled() {
		return ctx, nil
	}

	return e.tracer.StartSpan(
		ctx,
		"graphql.resolver "+object+"."+field,
		trace.WithAttributes(
			attribute.String("graphql.field.object", object),
			attribute.String("graphql.field.name", field),
		),
	)
}

func (e *ObservabilityExtension) logOperation(
	ctx context.Context,
	name, opType string,
	duration time.Duration,
	errs gqlerror.List,
) {
	slow := duration >= e.slowThreshold
	if len(errs) == 0 && !slow {
		return
	}

	fields := []zap.Field{
		zap.String("operation", name),
		zap.String("type", opType),
		zap.Duration("duration", duration),
		zap.String("request_id", gqlctx.RequestID(ctx)),
	}

	if len(errs) > 0 {
		fields = append(fields,
			zap.Int("error_count", len(errs)),
			zap.String("error_code", firstErrorCode(errs)),
			zap.String("error", errs[0].Message),
		)
		e.l.Warn("GraphQL operation returned errors", fields...)
		return
	}

	e.l.Warn("slow GraphQL operation", append(fields,
		zap.Duration("threshold", e.slowThreshold))...)
}

func operationCost(opCtx *graphql.OperationContext) (int, bool) {
	stats, ok := opCtx.Stats.GetExtension(complexityStatsKey).(*extension.ComplexityStats)
	if !ok || stats == nil {
		return 0, false
	}

	return stats.Complexity, true
}

func operationType(opCtx *graphql.OperationContext) string {
	if opCtx == nil || opCtx.Operation == nil {
		return unknownOperationType
	}
	if opCtx.Operation.Operation == "" {
		return unknownOperationType
	}

	return string(opCtx.Operation.Operation)
}

func responseErrors(resp *graphql.Response) gqlerror.List {
	if resp == nil {
		return nil
	}

	return resp.Errors
}

func firstErrorCode(errs gqlerror.List) string {
	if len(errs) == 0 {
		return ""
	}

	for _, err := range errs {
		if err == nil || err.Extensions == nil {
			continue
		}
		if code, ok := err.Extensions["code"].(string); ok && code != "" {
			return code
		}
	}

	return unknownErrorCode
}

func phaseSeconds(timing graphql.TraceTiming) float64 {
	if timing.Start.IsZero() || timing.End.IsZero() {
		return 0
	}

	return timing.End.Sub(timing.Start).Seconds()
}
