package gqlexec

import (
	"bytes"
	"context"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

type SchemaConfig struct {
	Registry    *Registry
	Parsed      *ast.Schema
	Override    *ast.Schema
	Resolvers   map[string]func() any
	WorkerLimit int64
}

type Schema struct {
	state       *graphql.ExecutableSchemaState[struct{}, struct{}, struct{}]
	parsed      *ast.Schema
	reg         *Registry
	resolvers   map[string]func() any
	workerLimit int64
	query       *Object
	mutation    *Object
}

func NewSchema(cfg SchemaConfig) *Schema {
	s := &Schema{
		state:       &graphql.ExecutableSchemaState[struct{}, struct{}, struct{}]{SchemaData: cfg.Override},
		parsed:      cfg.Parsed,
		reg:         cfg.Registry,
		resolvers:   cfg.Resolvers,
		workerLimit: cfg.WorkerLimit,
	}

	schema := s.Schema()
	if err := s.reg.validate(schema, s.resolvers); err != nil {
		panic(err)
	}
	if schema.Query != nil {
		s.query = s.reg.objects[schema.Query.Name]
	}
	if schema.Mutation != nil {
		s.mutation = s.reg.objects[schema.Mutation.Name]
	}

	return s
}

func (s *Schema) Schema() *ast.Schema {
	if s.state.SchemaData != nil {
		return s.state.SchemaData
	}
	return s.parsed
}

func (s *Schema) Complexity(
	_ context.Context,
	_, _ string,
	_ int,
	_ map[string]any,
) (int, bool) {
	return 0, false
}

func (s *Schema) Exec(ctx context.Context) graphql.ResponseHandler {
	opCtx := graphql.GetOperationContext(ctx)
	ec := s.newExec(opCtx)
	first := true

	switch opCtx.Operation.Operation {
	case ast.Query:
		if s.query == nil {
			break
		}
		return func(ctx context.Context) *graphql.Response {
			var response graphql.Response
			var data graphql.Marshaler
			if first {
				first = false
				data = ec.execObject(ctx, opCtx.Operation.SelectionSet, s.query, nil)
			} else {
				if atomic.LoadInt32(&ec.PendingDeferred) <= 0 {
					return nil
				}
				result := <-ec.DeferredResults
				atomic.AddInt32(&ec.PendingDeferred, -1)
				data = result.Result
				response.Path = result.Path
				response.Label = result.Label
				response.Errors = result.Errors
			}
			var buf bytes.Buffer
			data.MarshalGQL(&buf)
			response.Data = buf.Bytes()
			if atomic.LoadInt32(&ec.Deferred) > 0 {
				hasNext := atomic.LoadInt32(&ec.PendingDeferred) > 0
				response.HasNext = &hasNext
			}
			return &response
		}
	case ast.Mutation:
		if s.mutation == nil {
			break
		}
		return func(ctx context.Context) *graphql.Response {
			if !first {
				return nil
			}
			first = false
			data := ec.execObject(ctx, opCtx.Operation.SelectionSet, s.mutation, nil)
			var buf bytes.Buffer
			data.MarshalGQL(&buf)
			return &graphql.Response{Data: buf.Bytes()}
		}
	}

	return graphql.OneShot(graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
}

func (s *Schema) newExec(opCtx *graphql.OperationContext) *Exec {
	return &Exec{
		ExecutionContextState: graphql.NewExecutionContextState(
			opCtx,
			s.state,
			s.parsed,
			make(chan graphql.DeferredResult),
		),
		schema: s,
	}
}
