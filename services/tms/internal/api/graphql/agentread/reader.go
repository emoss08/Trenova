package agentread

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

const (
	queryCacheSize   = 256
	parserTokenLimit = 15000

	spanName   = "graphql.read"
	tracerName = "trenova.agentread"
)

var (
	ErrNoReadScope      = errors.New("an agent read needs the tool call's read scope")
	ErrUnsupportedActor = errors.New("an agent read runs as a person or an agent")
)

// Operation is one persisted operation an agent tool reads through: its name,
// the hash the client sends it under, and its text, which is the persisted
// document byte for byte.
type Operation struct {
	Name     string
	Hash     string
	Document string
}

// Schema is the executable schema agent reads run on: the whole parsed
// schema over only the executor shards the bound operations reach.
type Schema struct {
	graphql.ExecutableSchema
}

type Params struct {
	fx.In
	gqlext.Params

	Config      *config.Config
	Schema      Schema
	Loaders     *loaders.Factory
	Permissions permissionports.PermissionEngine
}

type Reader struct {
	exec        *executor.Executor
	operations  map[string]Operation
	loaders     *loaders.Factory
	permissions permissionports.PermissionEngine
	tracer      trace.Tracer
}

var _ graphqlreadports.GraphQLReader = (*Reader)(nil)

func New(p Params) *Reader { //nolint:gocritic // fx params are passed by value
	exec := executor.New(gqlext.NewCostLimitedSchema(p.Schema.ExecutableSchema))
	exec.SetQueryCache(lru.New[*ast.QueryDocument](queryCacheSize))
	exec.SetParserTokenLimit(parserTokenLimit)
	for _, ext := range gqlext.Extensions(p.Params) {
		exec.Use(ext)
	}
	exec.SetErrorPresenter(gqlext.ErrorPresenter(p.Config))
	exec.SetRecoverFunc(gqlext.Recover)

	operations := make(map[string]Operation, len(Operations()))
	for _, op := range Operations() {
		operations[op.Name] = op
	}

	return &Reader{
		exec:        exec,
		operations:  operations,
		loaders:     p.Loaders,
		permissions: p.Permissions,
		tracer:      otel.Tracer(tracerName),
	}
}

func (r *Reader) Read(
	ctx context.Context,
	req *graphqlreadports.GraphQLReadRequest,
) (*graphqlreadports.GraphQLReadResult, error) {
	op, ok := r.operations[req.Operation]
	if !ok {
		return nil, fmt.Errorf("%w: %s", graphqlreadports.ErrUnknownReadOperation, req.Operation)
	}
	if readscope.Current(ctx) == nil {
		return nil, ErrNoReadScope
	}
	auth, err := authContextFor(req.Actor)
	if err != nil {
		return nil, err
	}

	ctx, span := r.tracer.Start(ctx, spanName+" "+op.Name, trace.WithAttributes(
		attribute.String("graphql.operation.name", op.Name),
		attribute.String("trenova.graphql.persisted_hash", op.Hash),
	))
	defer span.End()

	ceiling := gqlctx.NewFieldCeiling(
		fieldsensitivity.NewAgentCeilings(r.permissions, req.Actor, req.DataAccessCeiling),
	)
	status := gqlctx.NewResponseStatus()
	ctx = gqlctx.WithAuthContext(ctx, auth)
	ctx = gqlctx.WithRequestID(ctx, req.RequestID)
	ctx = gqlctx.WithPermissionMemo(ctx, gqlctx.NewPermissionMemo())
	ctx = gqlctx.WithResponseStatus(ctx, status)
	ctx = gqlctx.WithFieldCeiling(ctx, ceiling)
	if r.loaders != nil {
		ctx = r.loaders.Attach(ctx, pagination.TenantInfo{
			OrgID:  auth.OrganizationID,
			BuID:   auth.BusinessUnitID,
			UserID: auth.UserID,
		})
	}

	response, cost := r.dispatch(ctx, op, req.Variables)
	result, err := decode(response, ceiling)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "result could not be decoded")
		return nil, err
	}
	result.Cost = cost
	result.RetryAfter = status.RetryAfter()

	span.SetAttributes(
		attribute.Int("trenova.graphql.cost", cost),
		attribute.Int("trenova.graphql.withheld", len(result.Withheld)),
	)
	if len(result.Errors) > 0 {
		span.SetAttributes(attribute.StringSlice("trenova.graphql.error_codes", errorCodes(result)))
		span.SetStatus(codes.Error, "the operation returned errors")
	}

	return result, nil
}

func (r *Reader) dispatch(
	ctx context.Context,
	op Operation,
	variables map[string]any,
) (response *graphql.Response, cost int) {
	ctx = graphql.StartOperationTrace(ctx)
	params := &graphql.RawParams{
		Query:         op.Document,
		OperationName: op.Name,
		Variables:     wireNumbers(variables),
		ReadTime: graphql.TraceTiming{
			Start: graphql.Now(),
			End:   graphql.Now(),
		},
	}

	rc, errs := r.exec.CreateOperationContext(ctx, params)
	if errs != nil {
		return r.exec.DispatchError(graphql.WithOperationContext(ctx, rc), errs), 0
	}

	if stats := extension.GetComplexityStats(graphql.WithOperationContext(ctx, rc)); stats != nil {
		cost = stats.Complexity
	}

	responses, ctx := r.exec.DispatchOperation(ctx, rc)

	return responses(ctx), cost
}

// maxExactInteger is the largest whole number a float64 holds exactly.
const maxExactInteger = 1 << 53

// wireNumbers sends a call's whole numbers as integers: a model's arguments
// arrive as float64, which gqlgen's Int refuses where the HTTP transport
// hands it a JSON number, and its Float takes an integer as well.
func wireNumbers(variables map[string]any) map[string]any {
	out := make(map[string]any, len(variables))
	for name, value := range variables {
		out[name] = wireNumber(value)
	}

	return out
}

func wireNumber(value any) any {
	switch v := value.(type) {
	case float64:
		if v == math.Trunc(v) && math.Abs(v) <= maxExactInteger {
			return int64(v)
		}
		return v
	case map[string]any:
		return wireNumbers(v)
	case []any:
		out := make([]any, len(v))
		for idx, element := range v {
			out[idx] = wireNumber(element)
		}
		return out
	}

	return value
}

// authContextFor is the GraphQL principal an actor reads as. A person reads
// as themselves. An unattended agent reads as itself and names no person, so
// a resolver that narrows to "my" records, or a team scope, finds none
// rather than the system user's.
func authContextFor(actor *permissionports.RequestActor) (*authctx.AuthContext, error) {
	if actor == nil {
		return nil, ErrUnsupportedActor
	}
	auth := &authctx.AuthContext{
		PrincipalID:    actor.PrincipalID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
	}
	switch {
	case actor.IsUser():
		auth.PrincipalType = authctx.PrincipalTypeUser
		auth.UserID = actor.UserID
		if auth.PrincipalID.IsNil() {
			auth.PrincipalID = actor.UserID
		}
	case actor.IsAgent():
		auth.PrincipalType = authctx.PrincipalTypeAgent
		auth.UserID = pulid.Nil
	default:
		return nil, ErrUnsupportedActor
	}
	if auth.PrincipalID.IsNil() || auth.OrganizationID.IsNil() || auth.BusinessUnitID.IsNil() {
		return nil, ErrUnsupportedActor
	}

	return auth, nil
}

func decode(
	response *graphql.Response,
	ceiling *gqlctx.FieldCeiling,
) (*graphqlreadports.GraphQLReadResult, error) {
	result := &graphqlreadports.GraphQLReadResult{Withheld: ceiling.Withheld()}
	if response == nil {
		return result, nil
	}
	if len(response.Data) > 0 && string(response.Data) != "null" {
		if err := sonic.Unmarshal(response.Data, &result.Data); err != nil {
			return nil, fmt.Errorf("decode agent read: %w", err)
		}
		for _, path := range ceiling.Stripped() {
			strip(result.Data, path)
		}
	}
	result.Errors = readErrors(response.Errors)

	return result, nil
}

func readErrors(list gqlerror.List) []graphqlreadports.GraphQLReadError {
	if len(list) == 0 {
		return nil
	}
	out := make([]graphqlreadports.GraphQLReadError, 0, len(list))
	for _, gqlErr := range list {
		code, _ := gqlErr.Extensions["code"].(string)
		out = append(out, graphqlreadports.GraphQLReadError{
			Message: gqlErr.Message,
			Path:    gqlErr.Path.String(),
			Code:    code,
		})
	}

	return out
}

func errorCodes(result *graphqlreadports.GraphQLReadResult) []string {
	out := make([]string, 0, len(result.Errors))
	for _, readErr := range result.Errors {
		out = append(out, readErr.Code)
	}

	return out
}

// strip removes a withheld non-null value from the decoded result. The
// executor had to resolve it, because nulling it would have nulled its
// parent; the reader makes sure it never leaves the adapter.
func strip(node any, path ast.Path) {
	for i, element := range path {
		last := i == len(path)-1
		switch e := element.(type) {
		case ast.PathName:
			object, ok := node.(map[string]any)
			if !ok {
				return
			}
			if last {
				delete(object, string(e))
				return
			}
			node = object[string(e)]
		case ast.PathIndex:
			list, ok := node.([]any)
			if !ok || int(e) < 0 || int(e) >= len(list) {
				return
			}
			if last {
				list[e] = nil
				return
			}
			node = list[e]
		}
	}
}
