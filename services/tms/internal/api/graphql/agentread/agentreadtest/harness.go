// Package agentreadtest builds the agent read path for a test: the generated
// read executor over the services a test supplies, behind the same
// extensions a request runs through, so a converted tool's tests read
// exactly as the tool does.
package agentreadtest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type Options struct {
	// Resolvers are the services the bound operations' resolvers use; a test
	// sets the ones its operation reaches. Logger and PermissionEngine are
	// filled in.
	Resolvers agentread.ResolverParams
	Engine    permissionports.PermissionEngine
}

// NewReader is the reader a converted tool reads through, built as the
// application builds it. It attaches no loaders of its own: a test whose
// operation resolves a field through one attaches that loader's factory to
// the context it reads with.
func NewReader(t testing.TB, opts Options) *agentread.Reader {
	t.Helper()

	monitoring := &monitoringconfig.Config{}
	logger := zap.NewNop()
	registry := &metrics.Registry{
		GraphQL: httpmetrics.NewGraphQL(nil, logger, false, httpmetrics.GraphQLOptions{}),
	}

	resolvers := opts.Resolvers
	resolvers.Logger = logger
	resolvers.PermissionEngine = opts.Engine
	resolvers.MonitoringConfig = monitoring

	return agentread.New(agentread.Params{
		Params: gqlext.Params{
			Observability: gqlext.NewObservabilityExtension(gqlext.ObservabilityParams{
				MonitoringConfig: monitoring, Logger: logger, Metrics: registry, Tracer: noopTracer{},
			}),
			CostBudget: gqlext.NewCostBudgetExtension(gqlext.CostBudgetParams{
				SecurityConfig: &securityconfig.Config{}, Logger: logger, Metrics: registry,
			}),
			ReadOnly: gqlext.NewReadOnlyExtension(gqlext.ReadOnlyParams{Logger: logger}),
			FieldGuard: gqlext.NewFieldGuardExtension(gqlext.FieldGuardParams{
				PermissionEngine: opts.Engine,
			}),
		},
		AppConfig:   &appconfig.Config{},
		Schema:      agentread.NewSchema(resolvers),
		Permissions: opts.Engine,
	})
}

// Read runs an operation as the actor, with the read scope a tool call
// carries, at the given data access ceiling.
func Read(
	ctx context.Context,
	reader graphqlreadports.GraphQLReader,
	actor *permissionports.RequestActor,
	operation string,
	variables map[string]any,
	ceiling permission.FieldSensitivity,
) (*graphqlreadports.GraphQLReadResult, error) {
	return reader.Read(readscope.With(ctx, nil), &graphqlreadports.GraphQLReadRequest{
		Actor:             actor,
		Operation:         operation,
		Variables:         variables,
		DataAccessCeiling: ceiling,
		RequestID:         "agentreadtest",
	})
}

// Person is a person in one tenant.
func Person() *permissionports.RequestActor {
	user := pulid.MustNew("usr_")

	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    user,
		UserID:         user,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

// Agent is an unattended agent in the person's tenant.
func Agent(person *permissionports.RequestActor) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agd_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: person.OrganizationID,
		BusinessUnitID: person.BusinessUnitID,
	}
}

type noopTracer struct{}

func (noopTracer) StartSpan(
	ctx context.Context,
	_ string,
	_ ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return ctx, noop.Span{}
}

func (noopTracer) AddEvent(context.Context, string, ...attribute.KeyValue) {}

func (noopTracer) SetAttributes(context.Context, ...attribute.KeyValue) {}

func (noopTracer) RecordError(context.Context, error, ...trace.EventOption) {}

func (noopTracer) IsEnabled() bool { return false }
