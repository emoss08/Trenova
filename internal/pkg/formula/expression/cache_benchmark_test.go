package expression_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/expression"
)

func BenchmarkCachePerformance(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()
	expr := "sqrt(pow(x, 2) + pow(y, 2)) * factor"

	b.Run("compile_only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This will compile every time since we're using different expressions
			evaluator.Evaluate(ctx, expr+"_"+string(rune(i%100)), nil)
		}
	})

	b.Run("compile_and_cache_hit", func(b *testing.B) {
		// Warm up cache with 10 expressions
		for i := 0; i < 10; i++ {
			evaluator.Evaluate(ctx, expr+"_"+string(rune(i)), nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This will hit cache most of the time
			evaluator.Evaluate(ctx, expr+"_"+string(rune(i%10)), nil)
		}
	})

	b.Run("single_expression_cached", func(b *testing.B) {
		// Compile once
		evaluator.Evaluate(ctx, expr, nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Always hits cache
			evaluator.Evaluate(ctx, expr, nil)
		}
	})
}
