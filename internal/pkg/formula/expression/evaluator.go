package expression

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// Evaluator evaluates formula expressions
type Evaluator struct {
	// Variable registry for resolving variables
	variables *variables.Registry

	// Function registry
	functions FunctionRegistry

	// Expression cache
	cache *LRUCache

	// String interner for memory efficiency
	interner *StringInterner

	// Metrics (optional)
	metrics *EvaluatorMetrics
}

// CompiledExpression represents a parsed and validated expression
type CompiledExpression struct {
	ast         Node
	expression  string
	variables   []string // Pre-extracted variable names
	complexity  int      // Pre-calculated complexity
	fingerprint string   // For cache key
}

// NewEvaluator creates a new expression evaluator
func NewEvaluator(vars *variables.Registry) *Evaluator {
	return &Evaluator{
		variables: vars,
		functions: DefaultFunctionRegistry(),
		cache:     NewLRUCache(1000), // Cache up to 1000 expressions
		interner:  NewStringInterner(),
		metrics:   NewEvaluatorMetrics(),
	}
}

// Evaluate parses and evaluates an expression
func (e *Evaluator) Evaluate(
	ctx context.Context,
	expr string,
	varCtx variables.VariableContext,
) (float64, error) {
	// Get or compile the expression
	compiled, err := e.compile(expr)
	if err != nil {
		return 0, fmt.Errorf("compilation error: %w", err)
	}

	// Get arena from pool
	arena := GetArena()
	defer PutArena(arena)

	// Create evaluation context
	evalCtx := NewEvaluationContext(ctx, varCtx).
		WithFunctions(e.functions).
		WithVariableRegistry(e.variables).
		WithArena(arena)

	// Evaluate the AST
	result, err := compiled.ast.Evaluate(evalCtx)
	if err != nil {
		return 0, fmt.Errorf("evaluation error: %w", err)
	}

	// Convert result to float64
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

// EvaluateBatch evaluates the same expression for multiple contexts
func (e *Evaluator) EvaluateBatch( //nolint:gocognit // this is fine
	ctx context.Context,
	expr string,
	contexts []variables.VariableContext,
) ([]float64, error) {
	// Compile once
	compiled, err := e.compile(expr)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	// Get arena from pool for batch operation
	arena := GetArena()
	defer PutArena(arena)

	// Pre-allocate results
	results := make([]float64, len(contexts))

	// Use worker pool for parallel evaluation
	numWorkers := min(
		// Could be configurable
		len(contexts), 4)

	type job struct {
		index int
		ctx   variables.VariableContext
	}

	jobs := make(chan job, len(contexts))
	errors := make(chan error, len(contexts))

	// Start workers
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

				// Convert to float64
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

	// Send jobs
	for i, ctx := range contexts {
		jobs <- job{index: i, ctx: ctx}
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(errors)

	// Check for errors
	var firstErr error
	for err := range errors {
		if firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// * compile parses and caches an expression
func (e *Evaluator) compile(expr string) (*CompiledExpression, error) {
	// Check cache first
	if cached, found := e.cache.Get(expr); found {
		return cached, nil
	}

	// Tokenize
	tokenizer := NewTokenizer(expr)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("tokenization error: %w", err)
	}

	// Parse
	parser := NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Extract metadata
	vars := e.extractVariables(ast)
	complexity := ast.Complexity()

	// Validate complexity
	if complexity > MaxExpressionComplexity {
		return nil, fmt.Errorf("expression too complex: complexity %d exceeds limit %d",
			complexity, MaxExpressionComplexity)
	}

	// Create compiled expression
	compiled := &CompiledExpression{
		ast:         ast,
		expression:  expr,
		variables:   vars,
		complexity:  complexity,
		fingerprint: expr, // Could use hash for longer expressions
	}

	// Cache it
	e.cache.Put(expr, compiled)

	return compiled, nil
}

// * extractVariables extracts all variable names from an AST
func (e *Evaluator) extractVariables(node Node) []string {
	vars := make(map[string]bool)
	e.extractVariablesRecursive(node, vars)

	// Convert to slice
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

// EvaluatorMetrics tracks performance metrics
type EvaluatorMetrics struct{}

func NewEvaluatorMetrics() *EvaluatorMetrics {
	return &EvaluatorMetrics{}
}

const MaxExpressionComplexity = 1000

// GetCacheStats returns cache performance statistics
func (e *Evaluator) GetCacheStats() CacheStats {
	return e.cache.Stats()
}

// ClearCache removes all entries from the expression cache
func (e *Evaluator) ClearCache() {
	e.cache.Clear()
}

// ResizeCache changes the maximum capacity of the expression cache
func (e *Evaluator) ResizeCache(newCapacity int) {
	e.cache.Resize(newCapacity)
}

// PreloadExpressions adds multiple expressions to the cache
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

// GetCachedExpressions returns a list of all cached expression keys
func (e *Evaluator) GetCachedExpressions() []string {
	// This would require adding a method to LRUCache to list keys
	// For now, return empty
	return []string{}
}
