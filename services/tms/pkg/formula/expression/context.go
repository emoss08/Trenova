package expression

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/formula/errors"
	"github.com/emoss08/trenova/pkg/formula/variables"
)

type EvaluationContext struct {
	ctx              context.Context
	variableContext  variables.VariableContext
	variableCache    map[string]any
	variableRegistry *variables.Registry
	functions        FunctionRegistry
	arena            *Arena
	ownsArena        bool
	timeout          time.Duration
	memoryLimit      int64
	maxDepth         int
	startTime        time.Time
	memoryUsed       int64
	depth            int
	evaluations      int
}

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

func (ctx *EvaluationContext) WithTimeout(timeout time.Duration) *EvaluationContext {
	ctx.timeout = timeout
	return ctx
}

func (ctx *EvaluationContext) WithMemoryLimit(limit int64) *EvaluationContext {
	ctx.memoryLimit = limit
	return ctx
}

func (ctx *EvaluationContext) WithFunctions(registry FunctionRegistry) *EvaluationContext {
	ctx.functions = registry
	return ctx
}

func (ctx *EvaluationContext) WithVariableRegistry(
	registry *variables.Registry,
) *EvaluationContext {
	ctx.variableRegistry = registry
	return ctx
}

func (ctx *EvaluationContext) WithArena(arena *Arena) *EvaluationContext {
	ctx.arena = arena
	ctx.ownsArena = false
	return ctx
}

func (ctx *EvaluationContext) CheckLimits() error {
	if err := ctx.ctx.Err(); err != nil {
		return errors.NewComputeError("evaluation", "cancelled", err)
	}

	if time.Since(ctx.startTime) > ctx.timeout {
		return errors.NewComputeError("evaluation", "timeout",
			fmt.Errorf("evaluation exceeded timeout of %v", ctx.timeout))
	}

	if ctx.memoryUsed > ctx.memoryLimit {
		return errors.NewComputeError("evaluation", "memory",
			fmt.Errorf("evaluation exceeded memory limit of %d bytes", ctx.memoryLimit))
	}

	if ctx.depth > ctx.maxDepth {
		return errors.NewComputeError("evaluation", "depth",
			fmt.Errorf("evaluation exceeded maximum depth of %d", ctx.maxDepth))
	}

	ctx.evaluations++
	if ctx.evaluations > MaxEvaluations {
		return errors.NewComputeError("evaluation", "complexity",
			fmt.Errorf("expression too complex: exceeded %d evaluations", MaxEvaluations))
	}

	return nil
}

func (ctx *EvaluationContext) ResolveVariable(name string) (any, error) {
	if val, ok := ctx.variableCache[name]; ok {
		return val, nil
	}

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

	val, err := varDef.Resolve(ctx.variableContext)
	if err != nil {
		return nil, errors.NewResolveError(name, "variable", err)
	}

	if err = varDef.Validate(val); err != nil {
		return nil, errors.NewVariableError(name, "validate", err)
	}

	ctx.variableCache[name] = val

	ctx.memoryUsed += estimateMemoryUsage(val)

	return val, nil
}

func (ctx *EvaluationContext) CallFunction(name string, args ...any) (any, error) {
	fn, ok := ctx.functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}

	ctx.depth++
	defer func() { ctx.depth-- }()

	result, err := fn.Call(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("function %s: %w", name, err)
	}

	ctx.memoryUsed += estimateMemoryUsage(result)

	return result, nil
}

func (ctx *EvaluationContext) Clone() *EvaluationContext {
	return &EvaluationContext{
		ctx:             ctx.ctx,
		variableContext: ctx.variableContext,
		variableCache:   make(map[string]any),
		functions:       ctx.functions,
		arena:           ctx.arena,
		ownsArena:       false,
		timeout:         ctx.timeout,
		memoryLimit:     ctx.memoryLimit,
		maxDepth:        ctx.maxDepth,
		startTime:       ctx.startTime,
		memoryUsed:      ctx.memoryUsed,
		depth:           ctx.depth,
		evaluations:     ctx.evaluations,
	}
}

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
		return int64(len(val) + 16)
	case []any:
		size := int64(24)
		for _, elem := range val {
			size += estimateMemoryUsage(elem)
		}
		return size
	case map[string]any:
		size := int64(48)
		for k, v := range val {
			size += int64(len(k)) + estimateMemoryUsage(v)
		}
		return size
	default:
		return 16
	}
}

func (ctx *EvaluationContext) GetArena() *Arena {
	if ctx.arena == nil {
		ctx.arena = GetArena()
		ctx.ownsArena = true
	}
	return ctx.arena
}

func (ctx *EvaluationContext) ReleaseArena() {
	if ctx.arena != nil && ctx.ownsArena {
		PutArena(ctx.arena)
		ctx.arena = nil
		ctx.ownsArena = false
	}
}

func (ctx *EvaluationContext) AllocValue(v any) any {
	// ! For now, just return the value as-is
	// ! The arena is used internally for memory pooling but we return values
	return v
}

const (
	MaxExpressionLength = 10000
	MaxTokenCount       = 1000
	MaxNestingDepth     = 50
	MaxEvaluations      = 10000
)
