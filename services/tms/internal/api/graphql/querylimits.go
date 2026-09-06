package graphql

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const depthLimitExtensionName = "OperationDepthLimit"

func init() {
	errcode.RegisterErrorType(querycost.DepthLimitErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(querycost.ComplexityLimitErrorCode, errcode.KindProtocol)
}

type costLimitedSchema struct {
	graphql.ExecutableSchema
	index *querycost.Index
}

func newCostLimitedSchema(es graphql.ExecutableSchema) costLimitedSchema {
	return costLimitedSchema{
		ExecutableSchema: es,
		index:            querycost.NewIndex(es.Schema()),
	}
}

func (s costLimitedSchema) Complexity(
	ctx context.Context,
	typeName, fieldName string,
	childComplexity int,
	args map[string]any,
) (int, bool) {
	if cost, ok := s.index.Complexity(typeName, fieldName, childComplexity, args); ok {
		return cost, true
	}

	return s.ExecutableSchema.Complexity(ctx, typeName, fieldName, childComplexity, args)
}

type operationDepthLimit struct {
	max int
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = operationDepthLimit{}

func (operationDepthLimit) ExtensionName() string {
	return depthLimitExtensionName
}

func (operationDepthLimit) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e operationDepthLimit) MutateOperationContext(
	_ context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	depth := querycost.Depth(operationDefinition(opCtx))
	if depth <= e.max {
		return nil
	}

	err := gqlerror.Errorf(
		"operation has depth %d, which exceeds the limit of %d",
		depth,
		e.max,
	)
	errcode.Set(err, querycost.DepthLimitErrorCode)

	return err
}

func operationDefinition(opCtx *graphql.OperationContext) *ast.OperationDefinition {
	if opCtx == nil {
		return nil
	}
	if opCtx.Operation != nil {
		return opCtx.Operation
	}
	if opCtx.Doc == nil {
		return nil
	}

	return opCtx.Doc.Operations.ForName(opCtx.OperationName)
}
