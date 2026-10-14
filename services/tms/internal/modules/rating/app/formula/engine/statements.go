package engine

import (
	"context"
	goErrors "errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	"github.com/shopspring/decimal"
)

const (
	traceMaxItems     = 50
	traceMaxDepth     = 3
	traceMaxTextRunes = 500
)

var (
	ErrNoStatements       = goErrors.New("expression has no statements")
	ErrStatementsMismatch = goErrors.New("expression statements could not be located")
)

type TraceRequest struct {
	Expression      string
	Env             map[string]any
	SchemaID        string
	Lookup          formulatemplatetypes.RateTableLookup
	ResolveFailures map[string]error
}

type statementSpan struct {
	statement formulatemplatetypes.Statement
	probeEnd  int
}

type sourceOffsets struct {
	bytes []int
	utf16 []int
}

func newSourceOffsets(source string) sourceOffsets {
	count := utf8.RuneCountInString(source)
	offsets := sourceOffsets{
		bytes: make([]int, 0, count+1),
		utf16: make([]int, 0, count+1),
	}

	units := 0
	for index, r := range source {
		offsets.bytes = append(offsets.bytes, index)
		offsets.utf16 = append(offsets.utf16, units)
		units += utf16.RuneLen(r)
	}
	offsets.bytes = append(offsets.bytes, len(source))
	offsets.utf16 = append(offsets.utf16, units)

	return offsets
}

func (o sourceOffsets) byteAt(runeIndex int) int {
	return o.bytes[clampIndex(runeIndex, len(o.bytes)-1)]
}

func (o sourceOffsets) utf16At(runeIndex int) int {
	return o.utf16[clampIndex(runeIndex, len(o.utf16)-1)]
}

func clampIndex(index, upper int) int {
	if index < 0 {
		return 0
	}
	if index > upper {
		return upper
	}
	return index
}

func SplitStatements(expression string) ([]formulatemplatetypes.Statement, error) {
	spans, err := splitStatements(expression)
	if err != nil {
		return nil, err
	}

	statements := make([]formulatemplatetypes.Statement, 0, len(spans))
	for _, span := range spans {
		statements = append(statements, span.statement)
	}

	return statements, nil
}

func splitStatements(expression string) ([]statementSpan, error) {
	tree, err := parseExpression(expression)
	if err != nil {
		return nil, err
	}

	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return nil, err
	}

	end := len(tokens)
	for end > 0 && tokens[end-1].Kind == lexer.EOF {
		end--
	}
	tokens = tokens[:end]
	if len(tokens) == 0 {
		return nil, ErrNoStatements
	}

	names := letChain(tree.Node)
	offsets := newSourceOffsets(expression)
	spans := make([]statementSpan, 0, len(names)+1)

	pos := 0
	for index, name := range names {
		if pos+1 >= len(tokens) ||
			!tokens[pos].Is(lexer.Operator, "let") ||
			!tokens[pos+1].Is(lexer.Identifier, name) {
			return nil, ErrStatementsMismatch
		}

		semicolon := topLevelSemicolon(tokens, pos)
		if semicolon <= pos {
			return nil, ErrStatementsMismatch
		}

		span := newSpan(expression, offsets, tokens[pos], tokens[semicolon-1])
		span.statement.Index = index
		span.statement.Kind = formulatemplatetypes.StatementLet
		span.statement.Name = name
		span.probeEnd = offsets.byteAt(tokens[semicolon].To)
		spans = append(spans, span)

		pos = semicolon + 1
	}

	last := len(tokens) - 1
	if tokens[last].Is(lexer.Operator, ";") {
		last--
	}
	if pos > last {
		return nil, ErrStatementsMismatch
	}

	result := newSpan(expression, offsets, tokens[pos], tokens[last])
	result.statement.Index = len(names)
	result.statement.Kind = formulatemplatetypes.StatementResult
	result.probeEnd = len(expression)
	spans = append(spans, result)

	return spans, nil
}

func letChain(node ast.Node) []string {
	names := make([]string, 0, 4)
	for {
		declarator, ok := node.(*ast.VariableDeclaratorNode)
		if !ok {
			return names
		}
		names = append(names, declarator.Name)
		node = declarator.Expr
	}
}

func topLevelSemicolon(tokens []lexer.Token, from int) int {
	depth := 0
	for index := from; index < len(tokens); index++ {
		token := tokens[index]
		switch {
		case token.Is(lexer.Bracket, "(", "[", "{"):
			depth++
		case token.Is(lexer.Bracket, ")", "]", "}"):
			depth--
		case depth == 0 && token.Is(lexer.Operator, ";"):
			return index
		}
	}

	return -1
}

func newSpan(
	expression string,
	offsets sourceOffsets,
	first, last lexer.Token,
) statementSpan {
	startRune := first.From
	endRune := last.To

	return statementSpan{
		statement: formulatemplatetypes.Statement{
			Code:  strings.TrimSpace(expression[offsets.byteAt(startRune):offsets.byteAt(endRune)]),
			Start: offsets.utf16At(startRune),
			End:   offsets.utf16At(endRune),
		},
	}
}

func (e *Engine) TraceStatements(
	ctx context.Context,
	req *TraceRequest,
) ([]formulatemplatetypes.Statement, error) {
	spans, err := splitStatements(req.Expression)
	if err != nil {
		return nil, err
	}

	env := req.Env
	if env == nil {
		env = make(map[string]any)
	}

	var definition *formulatypes.Definition
	if req.SchemaID != "" {
		if known, ok := e.registry.Get(req.SchemaID); ok {
			definition = known
		}
	}
	shape := e.shapeFor(ctx, definition, env, req.Lookup)

	statements := make([]formulatemplatetypes.Statement, 0, len(spans))
	for _, span := range spans {
		statement := span.statement
		if statement.Kind == formulatemplatetypes.StatementLet {
			e.traceLet(ctx, &statement, req.Expression[:span.probeEnd], env, shape, req)
		} else {
			e.traceResult(ctx, &statement, req.Expression, env, shape, req)
		}
		statements = append(statements, statement)
	}

	return statements, nil
}

func (e *Engine) traceLet(
	ctx context.Context,
	statement *formulatemplatetypes.Statement,
	prefix string,
	env map[string]any,
	shape *compileShape,
	req *TraceRequest,
) {
	probe := prefix + "\n" + statement.Name
	output, err := e.runShaped(ctx, probe, env, shape, req.ResolveFailures)
	if err != nil {
		statement.Error = err.Error()
		return
	}

	value, err := traceValue(output, 0)
	if err != nil {
		statement.Error = err.Error()
		return
	}
	statement.Value = value
}

func (e *Engine) traceResult(
	ctx context.Context,
	statement *formulatemplatetypes.Statement,
	expression string,
	env map[string]any,
	shape *compileShape,
	req *TraceRequest,
) {
	result, err := e.evaluateShaped(ctx, expression, env, shape, req.ResolveFailures)
	if err == nil {
		err = rejectBooleanAmount(expression, result)
	}
	if err != nil {
		statement.Error = err.Error()
		return
	}

	statement.Value = result.Value.String()
}

func traceValue(value any, depth int) (any, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case bool:
		return typed, nil
	case string:
		if utf8.RuneCountInString(typed) <= traceMaxTextRunes {
			return typed, nil
		}
		return stringutils.Ellipsize(typed, traceMaxTextRunes), nil
	case decimal.Decimal:
		return typed.String(), nil
	case *decimal.Decimal:
		if typed == nil {
			return nil, nil
		}
		return typed.String(), nil
	case decimal.NullDecimal:
		if !typed.Valid {
			return nil, nil
		}
		return typed.Decimal.String(), nil
	case float64:
		return traceFloat(typed)
	case float32:
		return traceFloat(float64(typed))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(typed), nil
	case time.Time:
		return typed.UTC().Format(time.RFC3339), nil
	case map[string]any:
		return traceMap(typed, depth)
	case []any:
		return traceSlice(typed, depth)
	}

	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Func || kind == reflect.Chan {
		return nil, nil
	}

	return stringutils.Ellipsize(fmt.Sprintf("%v", value), traceMaxTextRunes), nil
}

func traceFloat(value float64) (any, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("%w: %v", ErrNonFiniteResult, value)
	}

	return decimal.NewFromFloat(value).String(), nil
}

func traceMap(values map[string]any, depth int) (any, error) {
	if depth >= traceMaxDepth {
		return fmt.Sprintf("{%d fields}", len(values)), nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > traceMaxItems {
		keys = keys[:traceMaxItems]
	}

	out := make(map[string]any, len(keys))
	for _, key := range keys {
		item, err := traceValue(values[key], depth+1)
		if err != nil {
			return nil, err
		}
		if item != nil {
			out[key] = item
		}
	}

	return out, nil
}

func traceSlice(values []any, depth int) (any, error) {
	if depth >= traceMaxDepth {
		return fmt.Sprintf("[%d items]", len(values)), nil
	}

	limit := min(len(values), traceMaxItems)
	out := make([]any, 0, limit)
	for _, value := range values[:limit] {
		item, err := traceValue(value, depth+1)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, nil
}
