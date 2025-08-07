/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package expression

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/formula/errors"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// EvaluationContext holds the state for expression evaluation
type EvaluationContext struct {
	// Context for cancellation
	ctx context.Context

	// Variable resolution
	variableContext  variables.VariableContext
	variableCache    map[string]any      // Cache resolved variables
	variableRegistry *variables.Registry // Registry to use for variable lookup

	// Function registry
	functions FunctionRegistry

	// Memory arena for allocations
	arena     *Arena
	ownsArena bool // whether we own the arena

	// Limits
	timeout     time.Duration
	memoryLimit int64
	maxDepth    int

	// Current state
	startTime   time.Time
	memoryUsed  int64
	depth       int
	evaluations int // Track number of evaluations for complexity limits
}

// NewEvaluationContext creates a new evaluation context
func NewEvaluationContext(
	ctx context.Context,
	varCtx variables.VariableContext,
) *EvaluationContext {
	return &EvaluationContext{
		ctx:             ctx,
		variableContext: varCtx,
		variableCache:   make(map[string]any),
		functions:       DefaultFunctionRegistry(),
		timeout:         100 * time.Millisecond,
		memoryLimit:     1 << 20, // 1MB
		maxDepth:        50,
		startTime:       time.Now(),
	}
}

// WithTimeout sets the evaluation timeout
func (ctx *EvaluationContext) WithTimeout(timeout time.Duration) *EvaluationContext {
	ctx.timeout = timeout
	return ctx
}

// WithMemoryLimit sets the memory limit
func (ctx *EvaluationContext) WithMemoryLimit(limit int64) *EvaluationContext {
	ctx.memoryLimit = limit
	return ctx
}

// WithFunctions sets a custom function registry
func (ctx *EvaluationContext) WithFunctions(registry FunctionRegistry) *EvaluationContext {
	ctx.functions = registry
	return ctx
}

// WithVariableRegistry sets a custom variable registry
func (ctx *EvaluationContext) WithVariableRegistry(
	registry *variables.Registry,
) *EvaluationContext {
	ctx.variableRegistry = registry
	return ctx
}

// WithArena sets a custom arena for memory allocation
func (ctx *EvaluationContext) WithArena(arena *Arena) *EvaluationContext {
	ctx.arena = arena
	ctx.ownsArena = false
	return ctx
}

// CheckLimits verifies execution limits haven't been exceeded
func (ctx *EvaluationContext) CheckLimits() error {
	// Check context cancellation
	if err := ctx.ctx.Err(); err != nil {
		return errors.NewComputeError("evaluation", "cancelled", err)
	}

	// Check timeout
	if time.Since(ctx.startTime) > ctx.timeout {
		return errors.NewComputeError("evaluation", "timeout",
			fmt.Errorf("evaluation exceeded timeout of %v", ctx.timeout))
	}

	// Check memory
	if ctx.memoryUsed > ctx.memoryLimit {
		return errors.NewComputeError("evaluation", "memory",
			fmt.Errorf("evaluation exceeded memory limit of %d bytes", ctx.memoryLimit))
	}

	// Check depth
	if ctx.depth > ctx.maxDepth {
		return errors.NewComputeError("evaluation", "depth",
			fmt.Errorf("evaluation exceeded maximum depth of %d", ctx.maxDepth))
	}

	// Increment evaluation counter
	ctx.evaluations++
	if ctx.evaluations > MaxEvaluations {
		return errors.NewComputeError("evaluation", "complexity",
			fmt.Errorf("expression too complex: exceeded %d evaluations", MaxEvaluations))
	}

	return nil
}

// ResolveVariable resolves a variable by name
func (ctx *EvaluationContext) ResolveVariable(name string) (any, error) {
	// Check cache first
	if val, ok := ctx.variableCache[name]; ok {
		return val, nil
	}

	// Get variable from registry
	var varDef variables.Variable
	var err error
	if ctx.variableRegistry != nil {
		varDef, err = ctx.variableRegistry.Get(name)
	} else {
		varDef, err = variables.Get(name)
	}
	if err != nil {
		return nil, errors.NewVariableError(name, "resolve", err)
	}

	// Resolve value
	val, err := varDef.Resolve(ctx.variableContext)
	if err != nil {
		return nil, errors.NewResolveError(name, "variable", err)
	}

	// Validate value
	if err = varDef.Validate(val); err != nil {
		return nil, errors.NewVariableError(name, "validate", err)
	}

	// Cache the result
	ctx.variableCache[name] = val

	// Track memory usage (approximate)
	ctx.memoryUsed += estimateMemoryUsage(val)

	return val, nil
}

// CallFunction calls a registered function
func (ctx *EvaluationContext) CallFunction(name string, args ...any) (any, error) {
	fn, ok := ctx.functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}

	// Track depth for recursive functions
	ctx.depth++
	defer func() { ctx.depth-- }()

	result, err := fn.Call(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("function %s: %w", name, err)
	}

	// Track memory usage
	ctx.memoryUsed += estimateMemoryUsage(result)

	return result, nil
}

// Clone creates a shallow copy of the context for parallel evaluation
func (ctx *EvaluationContext) Clone() *EvaluationContext {
	return &EvaluationContext{
		ctx:             ctx.ctx,
		variableContext: ctx.variableContext,
		variableCache:   make(map[string]any), // Don't share cache
		functions:       ctx.functions,
		arena:           ctx.arena, // Share arena for parallel evaluation
		ownsArena:       false,     // Clones don't own the arena
		timeout:         ctx.timeout,
		memoryLimit:     ctx.memoryLimit,
		maxDepth:        ctx.maxDepth,
		startTime:       ctx.startTime, // Share start time
		memoryUsed:      ctx.memoryUsed,
		depth:           ctx.depth,
		evaluations:     ctx.evaluations,
	}
}

// * estimateMemoryUsage provides a rough estimate of memory usage
func estimateMemoryUsage(v any) int64 {
	switch val := v.(type) {
	case bool:
		return 1
	case int, int32, int64, uint, uint32, uint64:
		return 8
	case float32:
		return 4
	case float64:
		return 8
	case string:
		return int64(len(val) + 16) // String header + content
	case []any:
		size := int64(24) // Slice header
		for _, elem := range val {
			size += estimateMemoryUsage(elem)
		}
		return size
	case map[string]any:
		size := int64(48) // Map header (approximate)
		for k, v := range val {
			size += int64(len(k)) + estimateMemoryUsage(v)
		}
		return size
	default:
		return 16 // Default estimate
	}
}

// GetArena returns the arena for this context, creating one if needed
func (ctx *EvaluationContext) GetArena() *Arena {
	if ctx.arena == nil {
		ctx.arena = GetArena()
		ctx.ownsArena = true
	}
	return ctx.arena
}

// ReleaseArena returns the arena to the pool if owned
func (ctx *EvaluationContext) ReleaseArena() {
	if ctx.arena != nil && ctx.ownsArena {
		PutArena(ctx.arena)
		ctx.arena = nil
		ctx.ownsArena = false
	}
}

// AllocValue allocates a value using the arena if available
func (ctx *EvaluationContext) AllocValue(v any) any {
	// For now, just return the value as-is
	// The arena is used internally for memory pooling but we return values
	return v
}

// * Limits
const (
	MaxExpressionLength = 10000
	MaxTokenCount       = 1000
	MaxNestingDepth     = 50
	MaxEvaluations      = 10000
)
