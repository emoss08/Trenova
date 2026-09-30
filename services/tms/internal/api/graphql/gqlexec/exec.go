package gqlexec

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

type Exec struct {
	*graphql.ExecutionContextState[struct{}, struct{}, struct{}]
	schema *Schema
}

func (ec *Exec) WorkerLimit() int64 {
	return ec.schema.workerLimit
}

func Resolver[T any](ec *Exec, root string) T {
	return ec.schema.resolvers[root]().(T)
}

func (ec *Exec) MarshalType(
	ctx context.Context,
	sel ast.SelectionSet,
	typeName string,
	v any,
) graphql.Marshaler {
	if obj, ok := ec.schema.reg.objects[typeName]; ok {
		return ec.execObject(ctx, sel, obj, v)
	}
	if marshal, ok := ec.schema.reg.abstracts[typeName]; ok {
		return marshal(ctx, ec, sel, v)
	}
	panic(fmt.Sprintf("gqlexec: no executor for type %q", typeName))
}

func (ec *Exec) MarshalRoot(ctx context.Context, sel ast.SelectionSet, typeName string) graphql.Marshaler {
	return ec.execObject(ctx, sel, ec.schema.reg.objects[typeName], nil)
}

func (ec *Exec) UnmarshalInput(ctx context.Context, typeName string, v any) (any, error) {
	unmarshal, ok := ec.schema.reg.inputs[typeName]
	if !ok {
		panic(fmt.Sprintf("gqlexec: no unmarshaler for input %q", typeName))
	}
	return unmarshal(ctx, ec, v)
}

func (ec *Exec) execObject(
	ctx context.Context,
	sel ast.SelectionSet,
	obj *Object,
	v any,
) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, obj.Implementors)
	if obj.Root {
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Object: obj.Name})
	}

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)

	for i, field := range fields {
		if field.Name == "__typename" {
			out.Values[i] = graphql.MarshalString(obj.Name)
			continue
		}

		f := obj.field(field.Name)
		if f == nil {
			panic("unknown field " + strconv.Quote(field.Name))
		}

		if !f.Concurrent {
			if obj.Root {
				out.Values[i] = ec.RootResolverMiddleware(
					rootFieldContext(ctx, field),
					func(ctx context.Context) graphql.Marshaler {
						return ec.resolveField(ctx, obj, f, field, nil)
					},
				)
			} else {
				out.Values[i] = ec.resolveField(ctx, obj, f, field, v)
			}
			if invalid(f, out.Values[i]) {
				atomic.AddUint32(&out.Invalids, 1)
			}
			continue
		}

		innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
			defer func() {
				if r := recover(); r != nil {
					ec.Error(ctx, ec.Recover(ctx, r))
				}
			}()
			res = ec.resolveField(ctx, obj, f, field, v)
			if invalid(f, res) {
				atomic.AddUint32(&fs.Invalids, 1)
			}
			return res
		}

		if obj.Root {
			innerCtx := rootFieldContext(ctx, field)
			out.Concurrently(i, func(context.Context) graphql.Marshaler {
				return ec.RootResolverMiddleware(innerCtx, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, out)
				})
			})
			continue
		}

		if field.IsDeferred() {
			deferredFieldSet.AddField(field)
			fieldIndex := len(deferredFieldSet.Values) - 1
			deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
				return innerFunc(ctx, deferredFieldSet)
			})
			for _, deferrable := range field.Deferrables {
				view, ok := deferLabelToView[deferrable.Label]
				if !ok {
					view = deferredFieldSet.NewView()
					deferLabelToView[deferrable.Label] = view
				}
				view.AddIndices(fieldIndex)
			}
			out.Values[i] = graphql.Null
			continue
		}

		out.Concurrently(i, func(ctx context.Context) graphql.Marshaler {
			return innerFunc(ctx, out)
		})
	}

	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

func rootFieldContext(ctx context.Context, field graphql.CollectedField) context.Context {
	return graphql.WithRootFieldContext(ctx, &graphql.RootFieldContext{
		Object: field.Name,
		Field:  field,
	})
}

func invalid(f *Field, res graphql.Marshaler) bool {
	if f.NonNull {
		return res == graphql.Null
	}
	return res == graphql.RequiredNull
}

func (ec *Exec) resolveField(
	ctx context.Context,
	obj *Object,
	f *Field,
	field graphql.CollectedField,
	v any,
) graphql.Marshaler {
	if f.RootValue != nil {
		return ec.resolveRootTypedField(ctx, obj, f, field)
	}

	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext(ctx, obj.Name, f, field)
		},
		func(ctx context.Context) (any, error) {
			return f.Resolve(ctx, ec, v)
		},
		nil,
		func(ctx context.Context, sel ast.SelectionSet, res any) graphql.Marshaler {
			return f.Marshal(ctx, ec, sel, res)
		},
		true,
		f.NonNull,
	)
}

func (ec *Exec) resolveRootTypedField(
	ctx context.Context,
	obj *Object,
	f *Field,
	field graphql.CollectedField,
) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext(ctx, obj.Name, f, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	res := f.RootValue()
	fc.Result = res
	return f.Marshal(ctx, ec, field.Selections, res)
}

func (ec *Exec) fieldContext(
	ctx context.Context,
	objectName string,
	f *Field,
	field graphql.CollectedField,
) (fc *graphql.FieldContext, err error) {
	isMethod := f.IsMethod || f.IsResolver
	if f.Args == nil && !f.HasChild {
		return graphql.NewScalarFieldContext(objectName, field, isMethod, f.IsResolver, f.ChildErr)
	}

	fc = &graphql.FieldContext{
		Object:     objectName,
		Field:      field,
		IsMethod:   isMethod,
		IsResolver: f.IsResolver,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			if f.ChildErr != nil {
				return nil, f.ChildErr
			}
			return ec.childFieldContext(ctx, f.ChildType, field)
		},
	}

	if f.Args == nil {
		return fc, nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = f.Args(ctx, ec, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *Exec) childFieldContext(
	ctx context.Context,
	typeName string,
	field graphql.CollectedField,
) (*graphql.FieldContext, error) {
	obj := ec.schema.reg.objects[typeName]
	if f := obj.field(field.Name); f != nil {
		return ec.fieldContext(ctx, typeName, f, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type %s", field.Name, typeName)
}
