// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package expression_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// Mock variable context for benchmarks
type benchmarkVarContext struct {
	data map[string]any
}

func (c *benchmarkVarContext) ResolveVariable(name string) (any, error) {
	if val, ok := c.data[name]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("unknown variable: %s", name)
}

func (c *benchmarkVarContext) GetFieldSources() map[string]any {
	return c.data
}

func (c *benchmarkVarContext) GetComputed(name string) (any, error) {
	return nil, fmt.Errorf("no computed fields")
}

func (c *benchmarkVarContext) GetEntity() any {
	return c.data
}

func (c *benchmarkVarContext) GetField(path string) (any, error) {
	if val, ok := c.data[path]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("field not found: %s", path)
}

func (c *benchmarkVarContext) GetMetadata() map[string]any {
	return nil
}

func BenchmarkTokenizer(b *testing.B) {
	expressions := []struct {
		name string
		expr string
	}{
		{"simple", "1 + 2"},
		{"medium", "(price * quantity) - discount"},
		{"complex", "if(weight > 1000, base_rate * 1.5, base_rate) + fuel_surcharge"},
		{"very_complex", "sqrt(pow(x, 2) + pow(y, 2)) * sin(angle * 3.14159 / 180) + offset"},
	}

	for _, tc := range expressions {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tokenizer := expression.NewTokenizer(tc.expr)
				_, _ = tokenizer.Tokenize()
			}
		})
	}
}

func BenchmarkParser(b *testing.B) {
	expressions := []struct {
		name string
		expr string
	}{
		{"simple", "1 + 2"},
		{"nested", "((1 + 2) * 3) / 4"},
		{"function", "max(min(x, 100), 0)"},
		{"conditional", "if(x > 0, x * 2, -x)"},
		{"array", "[1, 2, 3, 4, 5]"},
	}

	for _, tc := range expressions {
		b.Run(tc.name, func(b *testing.B) {
			tokenizer := expression.NewTokenizer(tc.expr)
			tokens, _ := tokenizer.Tokenize()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				parser := expression.NewParser(tokens)
				_, _ = parser.Parse()
			}
		})
	}
}

func BenchmarkEvaluator(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	testCases := []struct {
		name string
		expr string
		vars map[string]any
	}{
		{
			name: "arithmetic",
			expr: "1 + 2 * 3 - 4 / 2",
			vars: nil,
		},
		{
			name: "variables",
			expr: "price * quantity * (1 - discount_rate)",
			vars: map[string]any{
				"price":         10.50,
				"quantity":      100.0,
				"discount_rate": 0.15,
			},
		},
		{
			name: "functions",
			expr: "round(sqrt(pow(x, 2) + pow(y, 2)), 2)",
			vars: map[string]any{
				"x": 3.0,
				"y": 4.0,
			},
		},
		{
			name: "conditional",
			expr: "if(weight > 1000, base_rate * 1.5, if(weight > 500, base_rate * 1.25, base_rate))",
			vars: map[string]any{
				"weight":    750.0,
				"base_rate": 2.50,
			},
		},
		{
			name: "array_operations",
			expr: "sum([1, 2, 3, 4, 5]) / len([1, 2, 3, 4, 5])",
			vars: nil,
		},
		{
			name: "string_operations",
			expr: `len("hello" + " " + "world")`,
			vars: nil,
		},
		{
			name: "complex_formula",
			expr: `base_rate * distance * if(has_hazmat, 1.25, 1) * (1 + fuel_surcharge/100) + accessorial_charges`,
			vars: map[string]any{
				"base_rate":           2.50,
				"distance":            500.0,
				"has_hazmat":          true,
				"fuel_surcharge":      15.0,
				"accessorial_charges": 125.0,
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			varCtx := &benchmarkVarContext{data: tc.vars}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = evaluator.Evaluate(ctx, tc.expr, varCtx)
			}
		})
	}
}

func BenchmarkEvaluatorCached(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	// Pre-compile expressions
	expressions := []string{
		"1 + 2 * 3",
		"price * quantity",
		"sqrt(x*x + y*y)",
	}

	for _, expr := range expressions {
		evaluator.Evaluate(ctx, expr, nil) // Warm up cache
	}

	b.Run("cache_hits", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expr := expressions[i%len(expressions)]
			_, _ = evaluator.Evaluate(ctx, expr, nil)
		}
	})
}

func BenchmarkBatchEvaluation(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	// Create variable contexts
	makeContexts := func(n int) []variables.VariableContext {
		contexts := make([]variables.VariableContext, n)
		for i := range contexts {
			contexts[i] = &benchmarkVarContext{
				data: map[string]any{
					"price":    float64(10 + i),
					"quantity": float64(100 - i),
				},
			}
		}
		return contexts
	}

	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			contexts := makeContexts(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = evaluator.EvaluateBatch(ctx, "price * quantity", contexts)
			}
		})
	}
}

func BenchmarkLRUCacheOperations(b *testing.B) {
	cache := expression.NewLRUCache(1000)

	// Pre-populate cache
	for i := 0; i < 500; i++ {
		expr := fmt.Sprintf("expr_%d", i)
		cache.Put(expr, &expression.CompiledExpression{})
	}

	b.Run("hits", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expr := fmt.Sprintf("expr_%d", i%500)
			cache.Get(expr)
		}
	})

	b.Run("misses", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expr := fmt.Sprintf("missing_%d", i)
			cache.Get(expr)
		}
	})

	b.Run("mixed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				expr := fmt.Sprintf("expr_%d", i%500)
				cache.Get(expr)
			} else {
				expr := fmt.Sprintf("new_%d", i)
				cache.Put(expr, &expression.CompiledExpression{})
			}
		}
	})
}

func BenchmarkComplexityScaling(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	depths := []int{1, 2, 3, 4, 5}

	// Generate expressions of increasing complexity
	var generateExpr func(int) string
	generateExpr = func(depth int) string {
		if depth <= 0 {
			return "1"
		}
		return fmt.Sprintf("(%s + %s)", generateExpr(depth-1), generateExpr(depth-1))
	}

	for _, depth := range depths {
		expr := generateExpr(depth)
		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = evaluator.Evaluate(ctx, expr, nil)
			}
		})
	}
}

func BenchmarkFunctions(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	functions := []struct {
		name string
		expr string
		vars map[string]any
	}{
		{"abs", "abs(-42)", nil},
		{"sqrt", "sqrt(16)", nil},
		{"pow", "pow(2, 10)", nil},
		{"sin", "sin(3.14159/2)", nil},
		{"round", "round(3.14159, 2)", nil},
		{"max", "max(1, 2, 3, 4, 5)", nil},
		{"sum", "sum([1, 2, 3, 4, 5])", nil},
		{"contains", `contains(["a", "b", "c"], "b")`, nil},
	}

	for _, fn := range functions {
		b.Run(fn.name, func(b *testing.B) {
			varCtx := &benchmarkVarContext{data: fn.vars}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = evaluator.Evaluate(ctx, fn.expr, varCtx)
			}
		})
	}
}

func BenchmarkMemoryAllocation(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	expressions := []struct {
		name string
		expr string
	}{
		{"numbers", "1 + 2 + 3 + 4 + 5"},
		{"strings", `"a" + "b" + "c" + "d" + "e"`},
		{"arrays", "[1, 2, 3] + [4, 5, 6]"},
		{"mixed", `if(true, "yes", "no") + " " + string(42)`},
	}

	for _, tc := range expressions {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = evaluator.Evaluate(ctx, tc.expr, nil)
			}
		})
	}
}

func BenchmarkParallelEvaluation(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	expr := "price * quantity * (1 - discount)"
	varCtx := &benchmarkVarContext{
		data: map[string]any{
			"price":    10.50,
			"quantity": 100.0,
			"discount": 0.15,
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = evaluator.Evaluate(ctx, expr, varCtx)
		}
	})
}

// Benchmark comparison: with vs without optimizations
func BenchmarkOptimizations(b *testing.B) {
	ctx := context.Background()
	expr := "sqrt(pow(x, 2) + pow(y, 2)) * factor"
	varCtx := &benchmarkVarContext{
		data: map[string]any{
			"x":      3.0,
			"y":      4.0,
			"factor": 2.5,
		},
	}

	b.Run("with_cache", func(b *testing.B) {
		evaluator := expression.NewEvaluator(nil)
		// Warm up cache
		evaluator.Evaluate(ctx, expr, varCtx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = evaluator.Evaluate(ctx, expr, varCtx)
		}
	})

	b.Run("without_cache", func(b *testing.B) {
		evaluator := expression.NewEvaluator(nil)
		evaluator.ResizeCache(0) // Disable cache

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = evaluator.Evaluate(ctx, expr, varCtx)
		}
	})
}
