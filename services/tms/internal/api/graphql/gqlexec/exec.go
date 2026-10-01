package gqlexec

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/emoss08/trenova/shared/intutils"

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
	resolver, ok := ec.schema.resolvers[root]().(T)
	if !ok {
		var want T
		panic(fmt.Sprintf("gqlexec: resolver %s does not implement %T", root, want))
	}
	return resolver
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

func (ec *Exec) MarshalRoot(
	ctx context.Context,
	sel ast.SelectionSet,
	typeName string,
) graphql.Marshaler {
	return ec.execObject(ctx, sel, ec.schema.reg.objects[typeName], nil)
}

func (ec *Exec) UnmarshalInput(ctx context.Context, typeName string, v any) (any, error) {
	unmarshal, ok := ec.schema.reg.inputs[typeName]
	if !ok {
		panic(fmt.Sprintf("gqlexec: no unmarshaler for input %q", typeName))
	}
	return unmarshal(ctx, ec, v)
}

type objectRun struct {
	ec       *Exec
	ctx      context.Context
	obj      *Object
	v        any
	out      *graphql.FieldSet
	deferred *graphql.FieldSet
	views    map[string]*graphql.FieldSetView
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

	run := &objectRun{
		ec:       ec,
		ctx:      ctx,
		obj:      obj,
		v:        v,
		out:      graphql.NewFieldSet(fields),
		deferred: graphql.NewFieldSet(nil),
		views:    make(map[string]*graphql.FieldSetView),
	}

	for i, field := range fields {
		if field.Name == "__typename" {
			run.out.Values[i] = graphql.MarshalString(obj.Name)
			continue
		}

		f := obj.field(field.Name)
		if f == nil {
			panic("unknown field " + strconv.Quote(field.Name))
		}

		switch {
		case !f.Concurrent:
			run.serial(i, f, field)
		case obj.Root:
			run.concurrentRoot(i, f, field)
		case field.IsDeferred():
			run.deferField(i, f, field)
		default:
			run.out.Concurrently(i, func(ctx context.Context) graphql.Marshaler {
				return run.guarded(ctx, run.out, f, field)
			})
		}
	}

	run.out.Dispatch(ctx)
	if run.out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, intutils.SafeToInt32(len(run.views)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   run.views,
		Path:     graphql.GetPath(ctx),
		FieldSet: run.deferred,
		Context:  ctx,
	})

	return run.out
}

func (r *objectRun) serial(i int, f *Field, field graphql.CollectedField) {
	if r.obj.Root {
		r.out.Values[i] = r.ec.RootResolverMiddleware(
			rootFieldContext(r.ctx, field),
			func(ctx context.Context) graphql.Marshaler {
				return r.ec.resolveField(ctx, r.obj, f, field, nil)
			},
		)
	} else {
		r.out.Values[i] = r.ec.resolveField(r.ctx, r.obj, f, field, r.v)
	}
	if invalid(f, r.out.Values[i]) {
		atomic.AddUint32(&r.out.Invalids, 1)
	}
}

func (r *objectRun) concurrentRoot(i int, f *Field, field graphql.CollectedField) {
	innerCtx := rootFieldContext(r.ctx, field)
	r.out.Concurrently(i, func(context.Context) graphql.Marshaler {
		return r.ec.RootResolverMiddleware(
			innerCtx,
			func(ctx context.Context) graphql.Marshaler {
				return r.guarded(ctx, r.out, f, field)
			},
		)
	})
}

func (r *objectRun) deferField(i int, f *Field, field graphql.CollectedField) {
	r.deferred.AddField(field)
	fieldIndex := len(r.deferred.Values) - 1
	r.deferred.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
		return r.guarded(ctx, r.deferred, f, field)
	})
	for _, deferrable := range field.Deferrables {
		view, ok := r.views[deferrable.Label]
		if !ok {
			view = r.deferred.NewView()
			r.views[deferrable.Label] = view
		}
		view.AddIndices(fieldIndex)
	}
	r.out.Values[i] = graphql.Null
}

func (r *objectRun) guarded(
	ctx context.Context,
	fs *graphql.FieldSet,
	f *Field,
	field graphql.CollectedField,
) (res graphql.Marshaler) {
	defer func() {
		if rec := recover(); rec != nil {
			r.ec.Error(ctx, r.ec.Recover(ctx, rec))
		}
	}()
	res = r.ec.resolveField(ctx, r.obj, f, field, r.v)
	if invalid(f, res) {
		atomic.AddUint32(&fs.Invalids, 1)
	}
	return res
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
