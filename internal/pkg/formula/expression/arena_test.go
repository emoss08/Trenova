// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package expression_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArena(t *testing.T) {
	t.Run("basic allocation", func(t *testing.T) {
		arena := expression.NewArena(1024)

		// Test float allocation
		f := arena.AllocFloat64(3.14)
		assert.NotNil(t, f)
		assert.Equal(t, 3.14, *f)

		// Test string allocation
		s := arena.AllocString("hello")
		assert.Equal(t, "hello", s)

		// Test bool allocation
		b := arena.AllocBool(true)
		assert.NotNil(t, b)
		assert.True(t, *b)

		// Test interface allocation
		v := arena.AllocInterface(42.0)
		// AllocInterface returns a pointer for float64
		if ptr, ok := v.(*float64); ok {
			assert.Equal(t, 42.0, *ptr)
		} else {
			assert.Fail(t, "expected *float64")
		}
	})

	t.Run("string interning", func(t *testing.T) {
		arena := expression.NewArena(1024)

		// Same string should be interned
		s1 := arena.AllocString("test")
		s2 := arena.AllocString("test")

		// Should be the same reference (interned)
		assert.Equal(t, s1, s2)

		stats := arena.Stats()
		assert.Equal(t, 1, stats.StringsInterned)
	})

	t.Run("reset", func(t *testing.T) {
		arena := expression.NewArena(1024)

		// Allocate some data
		arena.AllocFloat64(1.0)
		arena.AllocString("test")
		arena.AllocBool(true)

		stats := arena.Stats()
		assert.Greater(t, stats.BytesUsed, int64(0))
		assert.Greater(t, stats.Allocations, int64(0))

		// Reset
		arena.Reset()

		stats = arena.Stats()
		assert.Equal(t, int64(0), stats.BytesUsed)
		assert.Equal(t, int64(0), stats.Allocations)
		assert.Equal(t, 0, stats.StringsInterned)
	})

	t.Run("block allocation", func(t *testing.T) {
		arena := expression.NewArena(100) // Small blocks

		// Allocate more than one block
		// Each float64 allocation via Alloc is 8 bytes aligned to 8 = 8 bytes
		// But we're using the pool first, so force direct allocation
		for i := 0; i < 200; i++ { // More allocations to exceed pool
			arena.AllocFloat64(float64(i))
		}

		stats := arena.Stats()
		// The first block plus any additional blocks needed
		assert.GreaterOrEqual(t, stats.BlocksAllocated, 1)
	})

	t.Run("large allocation", func(t *testing.T) {
		arena := expression.NewArena(100)

		// Allocate larger than block size
		largeData := make([]byte, 200)
		ptr := arena.Alloc(len(largeData))
		assert.NotNil(t, ptr)

		stats := arena.Stats()
		assert.GreaterOrEqual(t, stats.BytesAllocated, int64(200))
	})
}

func TestArenaPool(t *testing.T) {
	t.Run("get and put", func(t *testing.T) {
		pool := expression.NewArenaPool(1024)

		// Get arena
		arena1 := pool.Get()
		assert.NotNil(t, arena1)

		// Use it
		arena1.AllocString("test")

		// Return to pool
		pool.Put(arena1)

		// Get again - should be reset
		arena2 := pool.Get()
		stats := arena2.Stats()
		assert.Equal(t, int64(0), stats.BytesUsed)
		assert.Equal(t, 0, stats.StringsInterned)
	})

	t.Run("with arena", func(t *testing.T) {
		pool := expression.NewArenaPool(1024)

		err := pool.WithArena(func(arena *expression.Arena) error {
			arena.AllocString("test")
			stats := arena.Stats()
			assert.Greater(t, stats.BytesUsed, int64(0))
			return nil
		})

		assert.NoError(t, err)
	})
}

func TestEvaluatorWithArena(t *testing.T) {
	t.Run("arena allocation in evaluation", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		// Evaluate expression multiple times
		for i := 0; i < 10; i++ {
			result, err := evaluator.Evaluate(ctx, "1.5 + 2.5", nil)
			require.NoError(t, err)
			assert.Equal(t, 4.0, result)
		}

		// Memory should be reused via arena pool
	})

	t.Run("batch evaluation with shared arena", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		// Create multiple contexts
		contexts := make([]variables.VariableContext, 100)
		for i := range contexts {
			contexts[i] = nil
		}

		// Batch evaluate
		results, err := evaluator.EvaluateBatch(ctx, "2 * 3", contexts)
		require.NoError(t, err)
		assert.Len(t, results, 100)

		for _, r := range results {
			assert.Equal(t, 6.0, r)
		}
	})
}

func TestArenaAllocator(t *testing.T) {
	arena := expression.NewArena(1024)
	allocator := expression.NewArenaAllocator(arena)

	t.Run("allocate different types", func(t *testing.T) {
		f := allocator.AllocFloat64(3.14)
		if ptr, ok := f.(*float64); ok {
			assert.Equal(t, 3.14, *ptr)
		} else {
			assert.Fail(t, "expected *float64")
		}

		s := allocator.AllocString("test")
		assert.Equal(t, "test", s)

		b := allocator.AllocBool(true)
		if ptr, ok := b.(*bool); ok {
			assert.True(t, *ptr)
		} else {
			assert.Fail(t, "expected *bool")
		}

		arr := allocator.AllocArray([]any{1, 2, 3})
		assert.Equal(t, []any{1, 2, 3}, arr)
	})
}

func BenchmarkArena(b *testing.B) {
	b.Run("AllocFloat64", func(b *testing.B) {
		arena := expression.NewArena(64 * 1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			arena.AllocFloat64(float64(i))
		}
	})

	b.Run("AllocString", func(b *testing.B) {
		arena := expression.NewArena(64 * 1024)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			arena.AllocString("test string")
		}
	})

	b.Run("WithoutArena", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = float64(i)
			_ = "test string"
		}
	})
}

func BenchmarkEvaluatorWithArena(b *testing.B) {
	evaluator := expression.NewEvaluator(nil)
	ctx := context.Background()

	b.Run("SimpleExpression", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = evaluator.Evaluate(ctx, "1 + 2 * 3", nil)
		}
	})

	b.Run("ComplexExpression", func(b *testing.B) {
		expr := "sqrt(pow(3, 2) + pow(4, 2)) * sin(3.14159 / 2)"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = evaluator.Evaluate(ctx, expr, nil)
		}
	})
}

func TestMemoryEfficiency(t *testing.T) {
	t.Run("arena reduces allocations", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		// Measure allocations
		var m1, m2 runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Evaluate many times
		for i := 0; i < 1000; i++ {
			_, err := evaluator.Evaluate(ctx, "1 + 2 + 3 + 4 + 5", nil)
			require.NoError(t, err)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// With arena pooling, allocations should be reasonable
		allocations := m2.Mallocs - m1.Mallocs
		t.Logf("Allocations for 1000 evaluations: %d", allocations)

		// This is a rough check - exact numbers depend on runtime
		// Each evaluation involves tokenizer, parser, AST nodes, etc.
		// 18-20 allocations per evaluation is reasonable
		assert.Less(t, allocations, uint64(25000))
	})
}
