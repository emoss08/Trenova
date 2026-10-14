package engine

import (
	"context"
	goErrors "errors"
	"fmt"
	"reflect"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/vm/runtime"
)

const (
	vmMemoryBudget        uint = 100_000
	maxEvaluationStrBytes      = 1 << 20
	concatFuncName             = "__concat"
)

var ErrStringBudgetExceeded = goErrors.New(
	"expression built more text than a formula may (1 MiB); formulas compare codes, they do not assemble documents",
)

var meteredStringBuiltins = []string{
	"replace",
	"join",
	"string", //nolint:goconst // expr's builtin is named string; this is not the category
	"toBase64",
	"upper",
	"lower",
}

type stringMeterKey struct{}

type stringMeter struct {
	used int
}

func withStringMeter(ctx context.Context) context.Context {
	return context.WithValue(ctx, stringMeterKey{}, &stringMeter{})
}

func chargeString(ctx context.Context, size int) error {
	if size > maxEvaluationStrBytes {
		return ErrStringBudgetExceeded
	}
	meter, ok := ctx.Value(stringMeterKey{}).(*stringMeter)
	if !ok {
		return nil
	}
	meter.used += size
	if meter.used > maxEvaluationStrBytes {
		return ErrStringBudgetExceeded
	}
	return nil
}

func limitOptions() []expr.Option {
	options := make([]expr.Option, 0, len(meteredStringBuiltins)+2)
	options = append(options,
		expr.Function(concatFuncName, concatFn, new(func(context.Context, any, any) any)),
		expr.Patch(concatPatcher{}),
	)
	for _, name := range meteredStringBuiltins {
		options = append(options, meteredBuiltin(name))
	}
	return options
}

func concatFn(args ...any) (any, error) {
	ctx, _ := args[0].(context.Context)
	left, right := args[1], args[2]
	if l, ok := left.(string); ok {
		if r, isString := right.(string); isString {
			if err := chargeString(ctx, len(l)+len(r)); err != nil {
				return nil, err
			}
			return l + r, nil
		}
	}
	return runtime.Add(left, right), nil
}

type concatPatcher struct{}

//nolint:gocritic // expr ast.Visitor requires the pointer signature
func (concatPatcher) Visit(node *ast.Node) {
	binary, ok := (*node).(*ast.BinaryNode)
	if !ok || binary.Operator != "+" {
		return
	}
	if !mayBeString(binary.Left) || !mayBeString(binary.Right) {
		return
	}
	ast.Patch(node, &ast.CallNode{
		Callee:    &ast.IdentifierNode{Value: concatFuncName},
		Arguments: []ast.Node{binary.Left, binary.Right},
	})
}

func mayBeString(node ast.Node) bool {
	nodeType := node.Type()
	if nodeType == nil {
		return true
	}
	kind := nodeType.Kind()
	return kind == reflect.String || kind == reflect.Interface
}

func meteredBuiltin(name string) expr.Option {
	original := builtinNamed(name)

	types := make([]any, 0, len(original.Types))
	for _, signature := range original.Types {
		in := make([]reflect.Type, 0, signature.NumIn()+1)
		in = append(in, contextType)
		for i := range signature.NumIn() {
			in = append(in, signature.In(i))
		}
		out := make([]reflect.Type, 0, signature.NumOut())
		for i := range signature.NumOut() {
			out = append(out, signature.Out(i))
		}
		types = append(types,
			reflect.New(reflect.FuncOf(in, out, signature.IsVariadic())).Interface())
	}

	call := builtinCall(original)

	return expr.Function(name, func(args ...any) (any, error) {
		ctx, _ := args[0].(context.Context)
		value, err := call(args[1:]...)
		if err != nil {
			return nil, err
		}
		if text, ok := value.(string); ok {
			if err = chargeString(ctx, len(text)); err != nil {
				return nil, err
			}
		}
		return value, nil
	}, types...)
}

func builtinNamed(name string) *builtin.Function {
	for _, fn := range builtin.Builtins {
		if fn.Name == name {
			return fn
		}
	}
	panic(fmt.Sprintf("expr has no builtin %q", name))
}

func builtinCall(fn *builtin.Function) func(args ...any) (any, error) {
	switch {
	case fn.Func != nil:
		return fn.Func
	case fn.Fast != nil:
		return func(args ...any) (any, error) { return fn.Fast(args[0]), nil }
	case fn.Safe != nil:
		return func(args ...any) (any, error) {
			value, _, err := fn.Safe(args...)
			return value, err
		}
	default:
		panic(fmt.Sprintf("expr builtin %q has no implementation", fn.Name))
	}
}
