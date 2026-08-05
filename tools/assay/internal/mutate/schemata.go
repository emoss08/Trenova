package mutate

import (
	"go/ast"
	"go/types"
	"strconv"
)

// schemaSpan is a byte range of the original file that must be re-rendered when a
// site is expanded, so that mutation sites nested inside it survive.
type schemaSpan struct {
	from int
	to   int
}

// schemaForm is a candidate's mutant-schema rewrite: one byte range replaced by a
// template that selects between the original and the mutated behaviour at run
// time. Every site in a package compiles into the same binary, so a whole package
// costs one build instead of one build per mutant.
//
// A form is only produced when the rewrite is provably type-correct. A candidate
// without one is still a valid mutant; it just falls back to compiling its own
// binary through -overlay, which is slow but has no gates.
type schemaForm struct {
	from    int
	to      int
	spans   []schemaSpan
	helpers []string
	render  func(id int, parts []string) string
}

// wholeRange reports whether the form re-renders everything it replaces, and so
// can host any site that shares its range.
func (f *schemaForm) wholeRange() bool {
	return len(f.spans) == 1 && f.spans[0].from == f.from && f.spans[0].to == f.to
}

// xorSchema flips a boolean-valued expression by comparing it against the switch.
//
// `(x) != on(id)` is the original value when the mutant is inactive and its
// negation when active, and because a comparison yields an *untyped* boolean the
// result stays assignable everywhere the original was — including to named
// boolean types, which a helper returning `bool` could not manage.
func xorSchema(ctx *fileContext, node ast.Node) *schemaForm {
	from, to := ctx.offset(node.Pos()), ctx.offset(node.End())
	form := &schemaForm{
		from:    from,
		to:      to,
		spans:   []schemaSpan{{from: from, to: to}},
		helpers: []string{helperOn},
	}
	form.render = func(id int, parts []string) string {
		return "((" + parts[0] + ") != " + helperOn + "(" + strconv.Itoa(id) + "))"
	}

	return form
}

// wrapSchema forces a condition in one direction while still evaluating it, which
// is what makes a branch mutant compile: every identifier the condition mentions
// stays live.
func wrapSchema(ctx *fileContext, cond ast.Expr, force bool) *schemaForm {
	from, to := ctx.offset(cond.Pos()), ctx.offset(cond.End())
	form := &schemaForm{
		from:    from,
		to:      to,
		spans:   []schemaSpan{{from: from, to: to}},
		helpers: []string{helperOn},
	}
	form.render = func(id int, parts []string) string {
		if force {
			return "((" + parts[0] + ") || " + helperOn + "(" + strconv.Itoa(id) + "))"
		}

		return "((" + parts[0] + ") && !" + helperOn + "(" + strconv.Itoa(id) + "))"
	}

	return form
}

// binarySchema routes both operands through a generic helper that picks the
// operator at run time. The operands are passed as values, so each is evaluated
// exactly once and the type parameter is inferred from the arguments — no type
// name is ever rendered, which is what keeps this safe across named types and
// dot-imports.
func binarySchema(ctx *fileContext, expr *ast.BinaryExpr, helper string) *schemaForm {
	form := &schemaForm{
		from: ctx.offset(expr.Pos()),
		to:   ctx.offset(expr.End()),
		spans: []schemaSpan{
			{from: ctx.offset(expr.X.Pos()), to: ctx.offset(expr.X.End())},
			{from: ctx.offset(expr.Y.Pos()), to: ctx.offset(expr.Y.End())},
		},
		helpers: []string{helper},
	}
	form.render = func(id int, parts []string) string {
		return helper + "(" + strconv.Itoa(id) + ", " + parts[0] + ", " + parts[1] + ")"
	}

	return form
}

// connectorSchema passes the right operand as a thunk so `&&` and `||` keep their
// short-circuit semantics: a helper taking two values would evaluate both, which
// is a behaviour change of its own and would manufacture kills.
func connectorSchema(ctx *fileContext, expr *ast.BinaryExpr, helper string) *schemaForm {
	form := &schemaForm{
		from: ctx.offset(expr.Pos()),
		to:   ctx.offset(expr.End()),
		spans: []schemaSpan{
			{from: ctx.offset(expr.X.Pos()), to: ctx.offset(expr.X.End())},
			{from: ctx.offset(expr.Y.Pos()), to: ctx.offset(expr.Y.End())},
		},
		helpers: []string{helper},
	}
	form.render = func(id int, parts []string) string {
		return helper + "(" + strconv.Itoa(id) + ", " + parts[0] +
			", func() bool { return " + parts[1] + " })"
	}

	return form
}

// stepSchema hands the operand over by address so the helper can step it either
// way. That is why addressability is checked: `m[k]++` is legal Go but `&m[k]` is
// not.
func stepSchema(ctx *fileContext, stmt *ast.IncDecStmt, helper string) *schemaForm {
	form := &schemaForm{
		from:    ctx.offset(stmt.Pos()),
		to:      ctx.offset(stmt.End()),
		spans:   []schemaSpan{{from: ctx.offset(stmt.X.Pos()), to: ctx.offset(stmt.X.End())}},
		helpers: []string{helper},
	}
	form.render = func(id int, parts []string) string {
		return helper + "(" + strconv.Itoa(id) + ", &(" + parts[0] + "))"
	}

	return form
}

// schemaOperandType reduces a binary expression's two operands to the single type
// its helper will be instantiated with, or reports that no such type exists.
//
// Untyped operands take the other side's type, matching what the compiler does.
// Two untyped operands mean the expression is constant and is rejected elsewhere.
// Genuinely mixed types are refused rather than guessed at: inference would fail
// and the whole schemata binary would stop compiling.
func schemaOperandType(info *types.Info, x, y ast.Expr) (types.Type, bool) {
	tx, ty := info.TypeOf(x), info.TypeOf(y)
	if tx == nil || ty == nil {
		return nil, false
	}

	var joint types.Type
	switch {
	case isUntypedType(tx) && isUntypedType(ty):
		return nil, false
	case isUntypedType(tx):
		joint = ty
	case isUntypedType(ty):
		joint = tx
	case types.Identical(tx, ty):
		joint = tx
	default:
		return nil, false
	}

	// A type parameter's own constraint would have to imply the helper's, and
	// `constraints.Ordered` admits strings while the arithmetic helpers do not.
	if _, ok := types.Unalias(joint).(*types.TypeParam); ok {
		return nil, false
	}

	return joint, true
}

func isUntypedType(t types.Type) bool {
	basic, ok := types.Unalias(t).(*types.Basic)

	return ok && basic.Info()&types.IsUntyped != 0
}

// plainBool reports whether a type is exactly `bool`, as opposed to a named type
// whose underlying type is bool. Helpers return `bool`, so a named boolean result
// would not be assignable and the mutant would not compile.
func plainBool(t types.Type) bool {
	basic, ok := types.Unalias(t).(*types.Basic)
	if !ok {
		return false
	}

	return basic.Kind() == types.Bool || basic.Kind() == types.UntypedBool
}

func basicInfo(t types.Type) (types.BasicInfo, bool) {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return 0, false
	}

	return basic.Info(), true
}

func numericType(t types.Type) bool {
	info, ok := basicInfo(t)

	return ok && info&(types.IsInteger|types.IsFloat|types.IsComplex) != 0
}

func integerType(t types.Type) bool {
	info, ok := basicInfo(t)

	return ok && info&types.IsInteger != 0
}

// orderedType matches the helper constraint for `<` and friends: integers, floats
// and strings. Complex numbers are numeric but unordered.
func orderedType(t types.Type) bool {
	info, ok := basicInfo(t)

	return ok && info&(types.IsInteger|types.IsFloat|types.IsString) != 0
}

// constantExpr reports whether the expression is a constant. Replacing one with a
// function call is fine in a statement but not where the language demands a
// constant, and the value being recorded is exactly the signal for that.
func constantExpr(info *types.Info, expr ast.Expr) bool {
	tv, ok := info.Types[expr]

	return ok && tv.Value != nil
}
