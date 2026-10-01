package gqlexec

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

type (
	TypedMarshal[T any]   func(ctx context.Context, ec *Exec, sel ast.SelectionSet, v T) graphql.Marshaler
	TypedUnmarshal[T any] func(ctx context.Context, ec *Exec, v any) (T, error)
)

func Marshal[T any](marshal TypedMarshal[T]) MarshalFunc {
	return func(ctx context.Context, ec *Exec, sel ast.SelectionSet, v any) graphql.Marshaler {
		typed, ok := v.(T)
		if !ok {
			var want T
			graphql.AddErrorf(
				ctx,
				`unexpected type %T from middleware/directive chain, should be %T`,
				v,
				want,
			)
			return graphql.Null
		}
		return marshal(ctx, ec, sel, typed)
	}
}

func Unmarshal[T any](unmarshal TypedUnmarshal[T]) UnmarshalFunc {
	return func(ctx context.Context, ec *Exec, v any) (any, error) {
		return unmarshal(ctx, ec, v)
	}
}

func UnmarshalInput[T any](ctx context.Context, ec *Exec, typeName string, v any) (T, error) {
	res, err := ec.UnmarshalInput(ctx, typeName, v)
	typed, ok := res.(T)
	if !ok {
		return typed, fmt.Errorf("gqlexec: input %s decoded to %T, want %T", typeName, res, typed)
	}
	return typed, err
}

type List[T any] struct {
	Elem        TypedMarshal[T]
	NonNull     bool
	NonNullElem bool
	Scalar      bool
}

func (l List[T]) Marshal(
	ctx context.Context,
	ec *Exec,
	sel ast.SelectionSet,
	v []T,
) graphql.Marshaler {
	if !l.NonNull && v == nil {
		return graphql.Null
	}

	var ret graphql.Array
	if l.Scalar {
		ret = make(graphql.Array, len(v))
		for i := range v {
			ret[i] = l.Elem(ctx, ec, sel, v[i])
		}
	} else {
		ret = graphql.MarshalSliceConcurrently(
			ctx,
			len(v),
			ec.WorkerLimit(),
			false,
			func(ctx context.Context, i int) graphql.Marshaler {
				fc := graphql.GetFieldContext(ctx)
				fc.Result = &v[i]
				return l.Elem(ctx, ec, sel, v[i])
			},
		)
	}

	if l.NonNullElem {
		for _, e := range ret {
			if e == graphql.Null {
				return graphql.Null
			}
		}
	}
	return ret
}

func UnmarshalList[T any](
	ctx context.Context,
	ec *Exec,
	v any,
	elem TypedUnmarshal[T],
) ([]T, error) {
	vSlice := graphql.CoerceList(v)
	res := make([]T, len(vSlice))
	for i := range vSlice {
		elemCtx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		var err error
		if res[i], err = elem(elemCtx, ec, vSlice[i]); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func NullViolation(ctx context.Context) {
	if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
		graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
	}
}
