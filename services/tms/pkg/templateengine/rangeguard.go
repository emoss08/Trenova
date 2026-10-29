package templateengine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"text/template/parse"

	"github.com/emoss08/trenova/pkg/templatediag"
	"github.com/emoss08/trenova/pkg/templatefuncs"
)

// rangeGuardName is the function every {{range}} pipeline is routed through.
// It is bound per render, never in the author's FuncMap, so a template that
// names it fails to parse.
const rangeGuardName = "__trenovaRangeGuard"

const (
	defaultMaxRangeIterations = 1_000_000
	maxRangeDepth             = 4
)

// ErrRenderBudget means a render looped more than its iteration budget allows.
var ErrRenderBudget = errors.New("template loops exceed the iteration limit")

// rangeGuard bounds a render's loops. A loop body that writes nothing never
// reaches the limited writer, so without this a range over a large count, or
// ranges nested over the same collection, run on after the deadline. Each
// range pipeline is evaluated once per pass of the loop around it, so charging
// the collection's length every time it is evaluated counts every iteration
// of every nested loop.
type rangeGuard struct {
	ctx       context.Context
	remaining int64
}

func newRangeGuard(ctx context.Context, budget int64) *rangeGuard {
	return &rangeGuard{ctx: ctx, remaining: budget}
}

func (g *rangeGuard) check(value any) (any, error) {
	if err := g.ctx.Err(); err != nil {
		return nil, fmt.Errorf("template render canceled: %w", err)
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return value, nil
		}
		v = v.Elem()
	}

	var iterations int64
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		iterations = int64(v.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iterations = v.Int()
		if iterations > templatefuncs.MaxSeqLength {
			return nil, fmt.Errorf("%w: range over %d is more than %d", ErrRenderBudget,
				iterations, templatefuncs.MaxSeqLength)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		if v.Uint() > templatefuncs.MaxSeqLength {
			return nil, fmt.Errorf("%w: range over %d is more than %d", ErrRenderBudget,
				v.Uint(), templatefuncs.MaxSeqLength)
		}
		iterations = int64(v.Uint()) //nolint:gosec // bounded by MaxSeqLength just above
	case reflect.Invalid, reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64,
		reflect.Complex128, reflect.String, reflect.Struct, reflect.Interface, reflect.Pointer,
		reflect.UnsafePointer:
		return value, nil
	case reflect.Chan, reflect.Func:
		return nil, fmt.Errorf("%w: a template cannot range over a %s", ErrRenderBudget, v.Kind())
	}

	g.remaining -= max(iterations, 0)
	if g.remaining < 0 {
		return nil, ErrRenderBudget
	}
	return value, nil
}

// guardRanges routes the pipeline of every {{range}} in tree through
// rangeGuardName, keeping the loop's own variable declarations. It runs once,
// on the compiled template, before anything executes it.
func guardRanges(tree *parse.Tree) {
	if tree == nil || tree.Root == nil {
		return
	}
	guardList(tree, tree.Root)
}

func guardList(tree *parse.Tree, list *parse.ListNode) {
	if list == nil {
		return
	}
	for _, node := range list.Nodes {
		switch n := node.(type) {
		case *parse.RangeNode:
			n.Pipe = guardedPipe(tree, n.Pipe)
			guardList(tree, n.List)
			guardList(tree, n.ElseList)
		case *parse.IfNode:
			guardList(tree, n.List)
			guardList(tree, n.ElseList)
		case *parse.WithNode:
			guardList(tree, n.List)
			guardList(tree, n.ElseList)
		case *parse.ListNode:
			guardList(tree, n)
		}
	}
}

func guardedPipe(tree *parse.Tree, pipe *parse.PipeNode) *parse.PipeNode {
	if pipe == nil || len(pipe.Cmds) == 0 {
		return pipe
	}

	inner := *pipe
	inner.IsAssign = false
	inner.Decl = nil

	command := *pipe.Cmds[0]
	command.Args = []parse.Node{
		parse.NewIdentifier(rangeGuardName).SetTree(tree).SetPos(pipe.Pos),
		&inner,
	}

	outer := *pipe
	outer.Cmds = []*parse.CommandNode{&command}
	return &outer
}

// rangeChecks reports the loops a template may not have: a range over a
// number written into the template, which seq exists for, and ranges nested
// deeper than maxRangeDepth.
func rangeChecks(field string, tree *parse.Tree) templatediag.Diagnostics {
	if tree == nil || tree.Root == nil {
		return nil
	}
	var diags templatediag.Diagnostics
	rangeCheckList(field, tree.Root, 0, &diags)
	return diags
}

func rangeCheckList(
	field string,
	list *parse.ListNode,
	depth int,
	diags *templatediag.Diagnostics,
) {
	if list == nil {
		return
	}
	for _, node := range list.Nodes {
		switch n := node.(type) {
		case *parse.RangeNode:
			if depth+1 > maxRangeDepth {
				*diags = append(*diags, templatediag.Error(
					templatediag.CodeRangeTooDeep,
					field,
					fmt.Sprintf("ranges are nested more than %d deep", maxRangeDepth),
				))
				return
			}
			if rangesOverNumber(n.Pipe) {
				*diags = append(*diags, templatediag.Error(
					templatediag.CodeRangeOverNumber,
					field,
					fmt.Sprintf(
						"{{range}} over a number is not allowed; use seq, which stops at %d",
						templatefuncs.MaxSeqLength,
					),
				))
			}
			rangeCheckList(field, n.List, depth+1, diags)
			rangeCheckList(field, n.ElseList, depth, diags)
		case *parse.IfNode:
			rangeCheckList(field, n.List, depth, diags)
			rangeCheckList(field, n.ElseList, depth, diags)
		case *parse.WithNode:
			rangeCheckList(field, n.List, depth, diags)
			rangeCheckList(field, n.ElseList, depth, diags)
		case *parse.ListNode:
			rangeCheckList(field, n, depth, diags)
		}
	}
}

// rangesOverNumber reports a pipeline whose value is a number written into the
// template: its last command is a bare number, or a parenthesized pipeline
// that is. A call such as seq 10 is not, because seq caps what it returns.
func rangesOverNumber(pipe *parse.PipeNode) bool {
	if pipe == nil || len(pipe.Cmds) == 0 {
		return false
	}
	last := pipe.Cmds[len(pipe.Cmds)-1]
	if len(last.Args) != 1 {
		return false
	}
	switch node := last.Args[0].(type) {
	case *parse.NumberNode:
		return true
	case *parse.PipeNode:
		return rangesOverNumber(node)
	default:
		return false
	}
}
