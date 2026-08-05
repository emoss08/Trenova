package mutate

import (
	"go/ast"
	"go/token"
	"go/types"
)

type Kind string

const (
	KindArithmetic Kind = "arithmetic"
	KindBoundary   Kind = "boundary"
	KindEquality   Kind = "equality"
	KindConnector  Kind = "connector"
	KindBoolean    Kind = "boolean"
	KindBranch     Kind = "branch"
	KindIncDec     Kind = "incdec"
)

func Kinds() []Kind {
	out := make([]Kind, 0, len(mutators))
	for _, m := range mutators {
		out = append(out, m.Kind())
	}

	return out
}

// edit is a byte-range replacement in the original source. Mutants are produced
// by splicing text rather than reprinting the AST, so comments, formatting and
// //go: directives survive exactly — losing a //go:embed line would not merely
// reformat the mutant, it would stop it compiling.
type edit struct {
	Offset int
	Length int
	Text   string
}

type candidate struct {
	Kind        Kind
	Pos         token.Pos
	Original    string
	Replacement string
	Edits       []edit
	// Schema is the mutant-schema rewrite for this candidate, when one can be
	// proven type-correct. Nil means the candidate is only reachable by compiling
	// its own binary.
	Schema *schemaForm
}

type mutator interface {
	Kind() Kind
	candidates(ctx *fileContext, node ast.Node) []candidate
}

type fileContext struct {
	fset   *token.FileSet
	info   *types.Info
	source []byte
	// stack is the chain of enclosing nodes, innermost last. It answers questions
	// the node alone cannot, such as whether the language demands a constant here.
	stack []ast.Node
}

func (c *fileContext) offset(pos token.Pos) int {
	return c.fset.Position(pos).Offset
}

// constantContext reports whether the enclosing syntax requires a constant, in
// which case no rewrite into a function call can compile. Reaching a function
// body first settles it: nothing inside one is constant-only.
func (c *fileContext) constantContext() bool {
	for i := len(c.stack) - 1; i >= 0; i-- {
		switch node := c.stack[i].(type) {
		case *ast.GenDecl:
			if node.Tok == token.CONST {
				return true
			}
		case *ast.ArrayType:
			return true
		case *ast.FuncDecl, *ast.FuncLit:
			return false
		}
	}

	return false
}

func (c *fileContext) text(from, to token.Pos) string {
	start, end := c.offset(from), c.offset(to)
	if start < 0 || end > len(c.source) || start >= end {
		return ""
	}

	return string(c.source[start:end])
}

var mutators = []mutator{
	arithmeticMutator{},
	boundaryMutator{},
	equalityMutator{},
	connectorMutator{},
	booleanMutator{},
	branchMutator{},
	incDecMutator{},
}

func replaceToken(kind Kind, ctx *fileContext, pos token.Pos, from, to token.Token) candidate {
	return candidate{
		Kind:        kind,
		Pos:         pos,
		Original:    from.String(),
		Replacement: to.String(),
		Edits: []edit{{
			Offset: ctx.offset(pos),
			Length: len(from.String()),
			Text:   to.String(),
		}},
	}
}

type arithmeticMutator struct{}

func (arithmeticMutator) Kind() Kind { return KindArithmetic }

var arithmeticSwaps = map[token.Token]token.Token{
	token.ADD: token.SUB,
	token.SUB: token.ADD,
	token.MUL: token.QUO,
	token.QUO: token.MUL,
	token.REM: token.MUL,
}

var arithmeticHelpers = map[token.Token]string{
	token.ADD: helperAddSub,
	token.SUB: helperSubAdd,
	token.MUL: helperMulQuo,
	token.QUO: helperQuoMul,
	token.REM: helperRemMul,
}

func (m arithmeticMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	to, ok := arithmeticSwaps[expr.Op]
	if !ok {
		return nil
	}

	// `+` is also string concatenation, so swapping it for `-` on strings would not
	// compile. Only provably numeric operands qualify.
	if !isNumeric(ctx.info, expr.X) || !isNumeric(ctx.info, expr.Y) {
		return nil
	}

	out := replaceToken(m.Kind(), ctx, expr.OpPos, expr.Op, to)
	out.Schema = arithmeticSchema(ctx, expr)

	return []candidate{out}
}

// arithmeticSchema declines whenever the helper's return type would not match
// what the expression's context expects. The helper returns its type parameter, so
// the operands' joint type has to be the expression's own type.
func arithmeticSchema(ctx *fileContext, expr *ast.BinaryExpr) *schemaForm {
	if ctx.constantContext() || constantExpr(ctx.info, expr) {
		return nil
	}

	joint, ok := schemaOperandType(ctx.info, expr.X, expr.Y)
	if !ok || !types.Identical(ctx.info.TypeOf(expr), joint) {
		return nil
	}

	if expr.Op == token.REM {
		if !integerType(joint) {
			return nil
		}
	} else if !numericType(joint) {
		return nil
	}

	return binarySchema(ctx, expr, arithmeticHelpers[expr.Op])
}

type boundaryMutator struct{}

func (boundaryMutator) Kind() Kind { return KindBoundary }

var boundarySwaps = map[token.Token]token.Token{
	token.LSS: token.LEQ,
	token.LEQ: token.LSS,
	token.GTR: token.GEQ,
	token.GEQ: token.GTR,
}

var boundaryHelpers = map[token.Token]string{
	token.LSS: helperLssLeq,
	token.LEQ: helperLeqLss,
	token.GTR: helperGtrGeq,
	token.GEQ: helperGeqGtr,
}

func (m boundaryMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	to, ok := boundarySwaps[expr.Op]
	if !ok {
		return nil
	}

	out := replaceToken(m.Kind(), ctx, expr.OpPos, expr.Op, to)
	out.Schema = boundarySchema(ctx, expr)

	return []candidate{out}
}

// boundarySchema needs a helper rather than the XOR trick, because `<` and `<=`
// differ only where the operands are equal and no boolean rewrite of the result
// can recover that. The helper returns `bool`, so a comparison whose context
// converts it to a named boolean type is declined.
func boundarySchema(ctx *fileContext, expr *ast.BinaryExpr) *schemaForm {
	if ctx.constantContext() || constantExpr(ctx.info, expr) {
		return nil
	}
	if !plainBool(ctx.info.TypeOf(expr)) {
		return nil
	}

	joint, ok := schemaOperandType(ctx.info, expr.X, expr.Y)
	if !ok || !orderedType(joint) {
		return nil
	}

	return binarySchema(ctx, expr, boundaryHelpers[expr.Op])
}

type equalityMutator struct{}

func (equalityMutator) Kind() Kind { return KindEquality }

func (m equalityMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	var out candidate
	switch expr.Op {
	case token.EQL:
		out = replaceToken(m.Kind(), ctx, expr.OpPos, token.EQL, token.NEQ)
	case token.NEQ:
		out = replaceToken(m.Kind(), ctx, expr.OpPos, token.NEQ, token.EQL)
	default:
		return nil
	}

	// Negating the result is the same mutation as swapping the operator, and needs
	// no type analysis at all: operands of any comparable type are untouched.
	if !ctx.constantContext() && !constantExpr(ctx.info, expr) {
		out.Schema = xorSchema(ctx, expr)
	}

	return []candidate{out}
}

type connectorMutator struct{}

func (connectorMutator) Kind() Kind { return KindConnector }

func (m connectorMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	var (
		out    candidate
		helper string
	)
	switch expr.Op {
	case token.LAND:
		out, helper = replaceToken(m.Kind(), ctx, expr.OpPos, token.LAND, token.LOR), helperLandLor
	case token.LOR:
		out, helper = replaceToken(m.Kind(), ctx, expr.OpPos, token.LOR, token.LAND), helperLorLand
	default:
		return nil
	}

	out.Schema = connectorSchemaFor(ctx, expr, helper)

	return []candidate{out}
}

// connectorSchemaFor requires plain `bool` throughout: the helper's parameters and
// its thunk are spelled `bool`, and a named boolean type would neither be
// accepted nor returned.
func connectorSchemaFor(ctx *fileContext, expr *ast.BinaryExpr, helper string) *schemaForm {
	if ctx.constantContext() || constantExpr(ctx.info, expr) {
		return nil
	}
	if !plainBool(ctx.info.TypeOf(expr)) ||
		!plainBool(ctx.info.TypeOf(expr.X)) ||
		!plainBool(ctx.info.TypeOf(expr.Y)) {
		return nil
	}

	return connectorSchema(ctx, expr, helper)
}

type booleanMutator struct{}

func (booleanMutator) Kind() Kind { return KindBoolean }

func (m booleanMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return nil
	}

	var to string
	switch ident.Name {
	case "true":
		to = "false"
	case "false":
		to = "true"
	default:
		return nil
	}

	// true and false are ordinary identifiers in Go and can be shadowed; only the
	// universe constants may be flipped.
	if !isUniverseBool(ctx.info, ident) {
		return nil
	}

	out := candidate{
		Kind:        m.Kind(),
		Pos:         ident.NamePos,
		Original:    ident.Name,
		Replacement: to,
		Edits: []edit{{
			Offset: ctx.offset(ident.NamePos),
			Length: len(ident.Name),
			Text:   to,
		}},
	}

	// A boolean literal is always a constant, so the value gate the other mutators
	// use would reject every one of them. Only the syntax that actually demands a
	// constant disqualifies the rewrite.
	if !ctx.constantContext() {
		out.Schema = xorSchema(ctx, ident)
	}

	return []candidate{out}
}

type branchMutator struct{}

func (branchMutator) Kind() Kind { return KindBranch }

// candidates forces a branch never or always taken. Deleting the branch would be
// the obvious mutation, but removing `if err != nil { return err }` routinely
// leaves `err` unused and the mutant fails to compile. Rewriting the condition
// keeps every identifier live and generalises to any branch.
func (m branchMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	stmt, ok := node.(*ast.IfStmt)
	if !ok || stmt.Cond == nil {
		return nil
	}
	if !isBoolean(ctx.info, stmt.Cond) {
		return nil
	}

	condText := ctx.text(stmt.Cond.Pos(), stmt.Cond.End())
	if condText == "" {
		return nil
	}

	start := ctx.offset(stmt.Cond.Pos())
	length := ctx.offset(stmt.Cond.End()) - start

	// The two directions rewrite the same byte range, which the renderer resolves by
	// nesting them: forcing one direction wraps the other, and each still answers to
	// its own id.
	schemable := !ctx.constantContext() && plainBool(ctx.info.TypeOf(stmt.Cond))

	force := func(suffix string, taken bool) candidate {
		out := candidate{
			Kind:        m.Kind(),
			Pos:         stmt.Cond.Pos(),
			Original:    condText,
			Replacement: "(" + condText + ")" + suffix,
			Edits: []edit{{
				Offset: start,
				Length: length,
				Text:   "(" + condText + ")" + suffix,
			}},
		}
		if schemable {
			out.Schema = wrapSchema(ctx, stmt.Cond, taken)
		}

		return out
	}

	return []candidate{force(" && false", false), force(" || true", true)}
}

type incDecMutator struct{}

func (incDecMutator) Kind() Kind { return KindIncDec }

func (m incDecMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	stmt, ok := node.(*ast.IncDecStmt)
	if !ok {
		return nil
	}
	if !isNumeric(ctx.info, stmt.X) {
		return nil
	}

	to, helper := token.INC, helperDecInc
	if stmt.Tok == token.INC {
		to, helper = token.DEC, helperIncDec
	}

	out := replaceToken(m.Kind(), ctx, stmt.TokPos, stmt.Tok, to)
	out.Schema = incDecSchema(ctx, stmt, helper)

	return []candidate{out}
}

// incDecSchema gates on addressability, because the helper steps the operand
// through a pointer and `m[k]++` — legal Go — has no address to take.
func incDecSchema(ctx *fileContext, stmt *ast.IncDecStmt, helper string) *schemaForm {
	tv, ok := ctx.info.Types[stmt.X]
	if !ok || !tv.Addressable() || !numericType(tv.Type) {
		return nil
	}
	if _, isParam := types.Unalias(tv.Type).(*types.TypeParam); isParam {
		return nil
	}

	return stepSchema(ctx, stmt, helper)
}

func isNumeric(info *types.Info, expr ast.Expr) bool {
	basic, ok := underlyingBasic(info, expr)
	if !ok {
		return false
	}

	return basic.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0
}

func isBoolean(info *types.Info, expr ast.Expr) bool {
	basic, ok := underlyingBasic(info, expr)
	if !ok {
		return false
	}

	return basic.Info()&types.IsBoolean != 0
}

func underlyingBasic(info *types.Info, expr ast.Expr) (*types.Basic, bool) {
	if info == nil {
		return nil, false
	}
	typ := info.TypeOf(expr)
	if typ == nil {
		return nil, false
	}
	basic, ok := typ.Underlying().(*types.Basic)

	return basic, ok
}

func isUniverseBool(info *types.Info, ident *ast.Ident) bool {
	if info == nil {
		return false
	}
	obj := info.ObjectOf(ident)
	if obj == nil {
		return false
	}

	return obj.Parent() == types.Universe
}
