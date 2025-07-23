// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package expression_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		// Test initial state
		assert.Equal(t, 0, cache.Size())
		assert.Equal(t, 3, cache.Capacity())

		// Test Put and Get
		expr1 := &expression.CompiledExpression{}
		cache.Put("expr1", expr1)

		retrieved, found := cache.Get("expr1")
		assert.True(t, found)
		assert.Equal(t, expr1, retrieved)

		// Test miss
		_, found = cache.Get("notfound")
		assert.False(t, found)
	})

	t.Run("LRU eviction", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		// Fill cache to capacity
		expr1 := &expression.CompiledExpression{}
		expr2 := &expression.CompiledExpression{}
		expr3 := &expression.CompiledExpression{}

		cache.Put("expr1", expr1)
		cache.Put("expr2", expr2)
		cache.Put("expr3", expr3)

		assert.Equal(t, 3, cache.Size())

		// Access expr1 to make it recently used
		cache.Get("expr1")

		// Add new item, should evict expr2 (least recently used)
		expr4 := &expression.CompiledExpression{}
		cache.Put("expr4", expr4)

		// Verify eviction
		assert.Equal(t, 3, cache.Size())
		_, found := cache.Get("expr2")
		assert.False(t, found)

		// Verify others still exist
		_, found = cache.Get("expr1")
		assert.True(t, found)
		_, found = cache.Get("expr3")
		assert.True(t, found)
		_, found = cache.Get("expr4")
		assert.True(t, found)
	})

	t.Run("update existing entry", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		expr1 := &expression.CompiledExpression{}
		expr1Updated := &expression.CompiledExpression{}

		cache.Put("expr1", expr1)
		cache.Put("expr1", expr1Updated)

		assert.Equal(t, 1, cache.Size())

		retrieved, found := cache.Get("expr1")
		assert.True(t, found)
		assert.Equal(t, expr1Updated, retrieved)
	})

	t.Run("clear", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		cache.Put("expr1", &expression.CompiledExpression{})
		cache.Put("expr2", &expression.CompiledExpression{})

		assert.Equal(t, 2, cache.Size())

		cache.Clear()
		assert.Equal(t, 0, cache.Size())

		_, found := cache.Get("expr1")
		assert.False(t, found)
	})

	t.Run("resize", func(t *testing.T) {
		cache := expression.NewLRUCache(5)

		// Fill cache
		for i := 0; i < 5; i++ {
			cache.Put(string(rune('a'+i)), &expression.CompiledExpression{})
		}

		assert.Equal(t, 5, cache.Size())

		// Resize smaller
		cache.Resize(3)
		assert.Equal(t, 3, cache.Capacity())
		assert.Equal(t, 3, cache.Size())

		// Resize larger
		cache.Resize(10)
		assert.Equal(t, 10, cache.Capacity())
		assert.Equal(t, 3, cache.Size())
	})

	t.Run("stats", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		// Generate some activity
		cache.Put("expr1", &expression.CompiledExpression{})
		cache.Get("expr1") // hit
		cache.Get("expr2") // miss
		cache.Get("expr1") // hit

		stats := cache.Stats()
		assert.Equal(t, int64(2), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 1, stats.Size)
		assert.Equal(t, 3, stats.Capacity)
		assert.InDelta(t, 0.666, stats.HitRate, 0.01)
	})

	t.Run("contains", func(t *testing.T) {
		cache := expression.NewLRUCache(3)

		cache.Put("expr1", &expression.CompiledExpression{})

		assert.True(t, cache.Contains("expr1"))
		assert.False(t, cache.Contains("expr2"))
	})

	t.Run("preload", func(t *testing.T) {
		cache := expression.NewLRUCache(10)

		expressions := map[string]*expression.CompiledExpression{
			"expr1": {},
			"expr2": {},
			"expr3": {},
		}

		cache.Preload(expressions)
		assert.Equal(t, 3, cache.Size())

		for key := range expressions {
			assert.True(t, cache.Contains(key))
		}
	})

	t.Run("get multiple", func(t *testing.T) {
		cache := expression.NewLRUCache(10)

		cache.Put("expr1", &expression.CompiledExpression{})
		cache.Put("expr2", &expression.CompiledExpression{})
		cache.Put("expr3", &expression.CompiledExpression{})

		results := cache.GetMultiple([]string{"expr1", "expr2", "notfound"})
		assert.Len(t, results, 2)
		assert.Contains(t, results, "expr1")
		assert.Contains(t, results, "expr2")
		assert.NotContains(t, results, "notfound")
	})

	t.Run("zero capacity", func(t *testing.T) {
		cache := expression.NewLRUCache(0)
		assert.Equal(t, 100, cache.Capacity()) // Should use default
	})
}

func TestLRUCacheWithCallback(t *testing.T) {
	t.Run("eviction callback", func(t *testing.T) {
		evictedKeys := []string{}

		callback := func(key string, expr *expression.CompiledExpression) {
			evictedKeys = append(evictedKeys, key)
		}

		cache := expression.NewLRUCacheWithCallback(2, callback)

		cache.Put("expr1", &expression.CompiledExpression{})
		cache.Put("expr2", &expression.CompiledExpression{})
		cache.Put("expr3", &expression.CompiledExpression{}) // Should evict expr1

		assert.Equal(t, []string{"expr1"}, evictedKeys)

		cache.Put("expr4", &expression.CompiledExpression{}) // Should evict expr2
		assert.Equal(t, []string{"expr1", "expr2"}, evictedKeys)
	})
}

func TestEvaluatorWithLRUCache(t *testing.T) {
	// Integration test with evaluator
	t.Run("evaluator cache integration", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		// Compile same expression multiple times
		expr := "2 + 2"

		// First evaluation should miss cache
		stats1 := evaluator.GetCacheStats()
		initialMisses := stats1.Misses

		result, err := evaluator.Evaluate(ctx, expr, nil)
		require.NoError(t, err)
		assert.Equal(t, 4.0, result)

		// Second evaluation should hit cache
		result, err = evaluator.Evaluate(ctx, expr, nil)
		require.NoError(t, err)
		assert.Equal(t, 4.0, result)

		stats2 := evaluator.GetCacheStats()
		assert.Equal(t, initialMisses+1, stats2.Misses)
		assert.Greater(t, stats2.Hits, stats1.Hits)
		assert.Equal(t, 1, stats2.Size)
	})

	t.Run("cache clear", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		// Add some expressions to cache
		evaluator.Evaluate(ctx, "1 + 1", nil)
		evaluator.Evaluate(ctx, "2 + 2", nil)

		stats := evaluator.GetCacheStats()
		assert.Greater(t, stats.Size, 0)

		// Clear cache
		evaluator.ClearCache()

		stats = evaluator.GetCacheStats()
		assert.Equal(t, 0, stats.Size)
	})

	t.Run("cache resize", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)

		// Resize cache
		evaluator.ResizeCache(500)

		stats := evaluator.GetCacheStats()
		assert.Equal(t, 500, stats.Capacity)
	})

	t.Run("preload expressions", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)
		ctx := context.Background()

		expressions := []string{
			"1 + 1",
			"2 * 3",
			"sqrt(16)",
		}

		err := evaluator.PreloadExpressions(expressions)
		require.NoError(t, err)

		stats := evaluator.GetCacheStats()
		assert.Equal(t, 3, stats.Size)

		// All should hit cache now
		for _, expr := range expressions {
			_, err := evaluator.Evaluate(ctx, expr, nil)
			require.NoError(t, err)
		}

		stats = evaluator.GetCacheStats()
		assert.Equal(t, int64(3), stats.Hits)
	})

	t.Run("preload with error", func(t *testing.T) {
		evaluator := expression.NewEvaluator(nil)

		expressions := []string{
			"1 + 1",
			"invalid expression ++",
		}

		err := evaluator.PreloadExpressions(expressions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to precompile")
	})
}

func BenchmarkLRUCache(b *testing.B) {
	cache := expression.NewLRUCache(1000)

	// Preload some data
	for i := 0; i < 500; i++ {
		cache.Put(string(rune(i)), &expression.CompiledExpression{})
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Get(string(rune(i % 500)))
		}
	})

	b.Run("Put", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Put(string(rune(i)), &expression.CompiledExpression{})
		}
	})

	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				cache.Get(string(rune(i % 500)))
			} else {
				cache.Put(string(rune(i)), &expression.CompiledExpression{})
			}
		}
	})
}
