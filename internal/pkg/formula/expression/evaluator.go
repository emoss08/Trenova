package expression

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// * Evaluator evaluates formula expressions
type Evaluator struct {
	// Variable registry for resolving variables
	variables *variables.Registry
	
	// Function registry
	functions FunctionRegistry
	
	// Expression cache
	cache *ExpressionCache
	
	// String interner for memory efficiency
	interner *StringInterner
	
	// Metrics (optional)
	metrics *EvaluatorMetrics
}

// * CompiledExpression represents a parsed and validated expression
type CompiledExpression struct {
	ast         Node
	expression  string
	variables   []string // Pre-extracted variable names
	complexity  int      // Pre-calculated complexity
	fingerprint string   // For cache key
}

// * ExpressionCache caches compiled expressions
type ExpressionCache struct {
	cache map[string]*CompiledExpression
	mu    sync.RWMutex
	
	maxSize int
	hits    int64
	misses  int64
}

// * NewEvaluator creates a new expression evaluator
func NewEvaluator(vars *variables.Registry) *Evaluator {
	return &Evaluator{
		variables: vars,
		functions: DefaultFunctionRegistry(),
		cache:     NewExpressionCache(1000), // Cache up to 1000 expressions
		interner:  NewStringInterner(),
		metrics:   NewEvaluatorMetrics(),
	}
}

// * Evaluate parses and evaluates an expression
func (e *Evaluator) Evaluate(ctx context.Context, expr string, varCtx variables.VariableContext) (float64, error) {
	// Get or compile the expression
	compiled, err := e.compile(expr)
	if err != nil {
		return 0, fmt.Errorf("compilation error: %w", err)
	}
	
	// Create evaluation context
	evalCtx := NewEvaluationContext(ctx, varCtx).
		WithFunctions(e.functions)
	
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

// * EvaluateBatch evaluates the same expression for multiple contexts
func (e *Evaluator) EvaluateBatch(ctx context.Context, expr string, contexts []variables.VariableContext) ([]float64, error) {
	// Compile once
	compiled, err := e.compile(expr)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}
	
	// Pre-allocate results
	results := make([]float64, len(contexts))
	
	// Use worker pool for parallel evaluation
	numWorkers := 4 // Could be configurable
	if len(contexts) < numWorkers {
		numWorkers = len(contexts)
	}
	
	type job struct {
		index int
		ctx   variables.VariableContext
	}
	
	jobs := make(chan job, len(contexts))
	errors := make(chan error, len(contexts))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := range jobs {
				evalCtx := NewEvaluationContext(ctx, j.ctx).
					WithFunctions(e.functions)
				
				result, err := compiled.ast.Evaluate(evalCtx)
				if err != nil {
					errors <- fmt.Errorf("context %d: %w", j.index, err)
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
	if cached := e.cache.Get(expr); cached != nil {
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
	variables := e.extractVariables(ast)
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
		variables:   variables,
		complexity:  complexity,
		fingerprint: expr, // Could use hash for longer expressions
	}
	
	// Cache it
	e.cache.Put(expr, compiled)
	
	return compiled, nil
}

// * extractVariables extracts all variable names from an AST
func (e *Evaluator) extractVariables(node Node) []string {
	variables := make(map[string]bool)
	e.extractVariablesRecursive(node, variables)
	
	// Convert to slice
	result := make([]string, 0, len(variables))
	for v := range variables {
		result = append(result, v)
	}
	
	return result
}

func (e *Evaluator) extractVariablesRecursive(node Node, variables map[string]bool) {
	switch n := node.(type) {
	case *IdentifierNode:
		variables[n.Name] = true
		
	case *BinaryOpNode:
		e.extractVariablesRecursive(n.Left, variables)
		e.extractVariablesRecursive(n.Right, variables)
		
	case *UnaryOpNode:
		e.extractVariablesRecursive(n.Operand, variables)
		
	case *ConditionalNode:
		e.extractVariablesRecursive(n.Condition, variables)
		e.extractVariablesRecursive(n.TrueExpr, variables)
		e.extractVariablesRecursive(n.FalseExpr, variables)
		
	case *FunctionCallNode:
		for _, arg := range n.Arguments {
			e.extractVariablesRecursive(arg, variables)
		}
		
	case *ArrayNode:
		for _, elem := range n.Elements {
			e.extractVariablesRecursive(elem, variables)
		}
	}
}

// * ExpressionCache implementation

func NewExpressionCache(maxSize int) *ExpressionCache {
	return &ExpressionCache{
		cache:   make(map[string]*CompiledExpression),
		maxSize: maxSize,
	}
}

func (c *ExpressionCache) Get(expr string) *CompiledExpression {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if compiled, ok := c.cache[expr]; ok {
		c.hits++
		return compiled
	}
	
	c.misses++
	return nil
}

func (c *ExpressionCache) Put(expr string, compiled *CompiledExpression) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple eviction: remove all if at capacity
	if len(c.cache) >= c.maxSize {
		// In production, use LRU eviction
		c.cache = make(map[string]*CompiledExpression)
	}
	
	c.cache[expr] = compiled
}

func (c *ExpressionCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	
	return float64(c.hits) / float64(total)
}

// * EvaluatorMetrics tracks performance metrics
type EvaluatorMetrics struct {
	compilations   int64
	evaluations    int64
	cacheHits      int64
	cacheMisses    int64
	totalDuration  int64 // nanoseconds
	mu             sync.Mutex
}

func NewEvaluatorMetrics() *EvaluatorMetrics {
	return &EvaluatorMetrics{}
}

// * Constants
const MaxExpressionComplexity = 1000