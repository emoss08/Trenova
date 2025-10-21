package expression

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/formula/variables"
)

type Evaluator struct {
	variables *variables.Registry
	functions FunctionRegistry
	cache     *LRUCache
	interner  *StringInterner
	metrics   *EvaluatorMetrics
}

type CompiledExpression struct {
	ast         Node
	expression  string
	variables   []string
	complexity  int
	fingerprint string
}

func NewEvaluator(vars *variables.Registry) *Evaluator {
	return &Evaluator{
		variables: vars,
		functions: DefaultFunctionRegistry(),
		cache:     NewLRUCache(1000),
		interner:  NewStringInterner(),
		metrics:   NewEvaluatorMetrics(),
	}
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	expr string,
	varCtx variables.VariableContext,
) (float64, error) {
	compiled, err := e.compile(expr)
	if err != nil {
		return 0, fmt.Errorf("compilation error: %w", err)
	}

	arena := GetArena()
	defer PutArena(arena)

	evalCtx := NewEvaluationContext(ctx, varCtx).
		WithFunctions(e.functions).
		WithVariableRegistry(e.variables).
		WithArena(arena)

	result, err := compiled.ast.Evaluate(evalCtx)
	if err != nil {
		return 0, fmt.Errorf("evaluation error: %w", err)
	}

	switch v := result.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("expression must return a numeric value, got %T", result)
	}
}

func (e *Evaluator) EvaluateBatch(
	ctx context.Context,
	expr string,
	contexts []variables.VariableContext,
) ([]float64, error) {
	compiled, err := e.compile(expr)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	arena := GetArena()
	defer PutArena(arena)

	results := make([]float64, len(contexts))

	numWorkers := min(
		// ! Could be configurable
		len(contexts), 4)

	type job struct {
		index int
		ctx   variables.VariableContext
	}

	jobs := make(chan job, len(contexts))
	errors := make(chan error, len(contexts))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := range jobs {
				evalCtx := NewEvaluationContext(ctx, j.ctx).
					WithFunctions(e.functions).
					WithVariableRegistry(e.variables).
					WithArena(arena) // Share arena across workers

				result, resultErr := compiled.ast.Evaluate(evalCtx)
				if resultErr != nil {
					errors <- fmt.Errorf("context %d: %w", j.index, resultErr)
					continue
				}

				switch v := result.(type) {
				case float64:
					results[j.index] = v
				case int:
					results[j.index] = float64(v)
				case bool:
					if v {
						results[j.index] = 1
					}
				default:
					errors <- fmt.Errorf("context %d: non-numeric result", j.index)
				}
			}
		}()
	}

	for i, ctx := range contexts {
		jobs <- job{index: i, ctx: ctx}
	}
	close(jobs)

	wg.Wait()
	close(errors)

	var firstErr error
	for err := range errors {
		if firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

func (e *Evaluator) compile(expr string) (*CompiledExpression, error) {
	if cached, found := e.cache.Get(expr); found {
		return cached, nil
	}

	tokenizer := NewTokenizer(expr)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("tokenization error: %w", err)
	}

	parser := NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	vars := e.extractVariables(ast)
	complexity := ast.Complexity()

	if complexity > MaxExpressionComplexity {
		return nil, fmt.Errorf("expression too complex: complexity %d exceeds limit %d",
			complexity, MaxExpressionComplexity)
	}

	compiled := &CompiledExpression{
		ast:         ast,
		expression:  expr,
		variables:   vars,
		complexity:  complexity,
		fingerprint: expr,
	}

	e.cache.Put(expr, compiled)

	return compiled, nil
}

func (e *Evaluator) extractVariables(node Node) []string {
	vars := make(map[string]bool)
	e.extractVariablesRecursive(node, vars)

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}

	return result
}

func (e *Evaluator) extractVariablesRecursive(node Node, vars map[string]bool) {
	switch n := node.(type) {
	case *IdentifierNode:
		vars[n.Name] = true

	case *BinaryOpNode:
		e.extractVariablesRecursive(n.Left, vars)
		e.extractVariablesRecursive(n.Right, vars)

	case *UnaryOpNode:
		e.extractVariablesRecursive(n.Operand, vars)

	case *ConditionalNode:
		e.extractVariablesRecursive(n.Condition, vars)
		e.extractVariablesRecursive(n.TrueExpr, vars)
		e.extractVariablesRecursive(n.FalseExpr, vars)

	case *FunctionCallNode:
		for _, arg := range n.Arguments {
			e.extractVariablesRecursive(arg, vars)
		}

	case *ArrayNode:
		for _, elem := range n.Elements {
			e.extractVariablesRecursive(elem, vars)
		}
	}
}

type EvaluatorMetrics struct{}

func NewEvaluatorMetrics() *EvaluatorMetrics {
	return &EvaluatorMetrics{}
}

const MaxExpressionComplexity = 1000

func (e *Evaluator) GetCacheStats() CacheStats {
	return e.cache.Stats()
}

func (e *Evaluator) ClearCache() {
	e.cache.Clear()
}

func (e *Evaluator) ResizeCache(newCapacity int) {
	e.cache.Resize(newCapacity)
}

func (e *Evaluator) PreloadExpressions(expressions []string) error {
	precompiled := make(map[string]*CompiledExpression)

	for _, expr := range expressions {
		compiled, err := e.compile(expr)
		if err != nil {
			return fmt.Errorf("failed to precompile %q: %w", expr, err)
		}
		precompiled[expr] = compiled
	}

	e.cache.Preload(precompiled)
	return nil
}

func (e *Evaluator) GetCachedExpressions() []string {
	// This would require adding a method to LRUCache to list keys
	// For now, return empty
	return []string{}
}
