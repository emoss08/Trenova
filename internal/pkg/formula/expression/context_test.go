package expression

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockVariableContext is a test implementation of variables.VariableContext
type mockVariableContext struct {
	entity   any
	fields   map[string]any
	computed map[string]any
	metadata map[string]any
}

func (m *mockVariableContext) GetEntity() any {
	return m.entity
}

func (m *mockVariableContext) GetField(path string) (any, error) {
	if val, ok := m.fields[path]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("field not found: %s", path)
}

func (m *mockVariableContext) GetComputed(function string) (any, error) {
	if val, ok := m.computed[function]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("computed function not found: %s", function)
}

func (m *mockVariableContext) GetMetadata() map[string]any {
	if m.metadata == nil {
		return make(map[string]any)
	}
	return m.metadata
}

func TestEvaluationContext(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{
				"test": 42.0,
			},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		
		// Test function registry
		if evalCtx.functions == nil {
			t.Error("functions registry is nil")
		}
		
		// Test that default functions are loaded
		if _, exists := evalCtx.functions["abs"]; !exists {
			t.Error("abs function not found in registry")
		}
		
		// Test timeout setting
		evalCtx = evalCtx.WithTimeout(200 * time.Millisecond)
		
		// Test memory limit setting
		evalCtx = evalCtx.WithMemoryLimit(2 << 20)
	})

	t.Run("function calls", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		
		// Test calling a function
		result, err := evalCtx.CallFunction("abs", -5.0)
		if err != nil {
			t.Errorf("CallFunction() error = %v", err)
		}
		if result != 5.0 {
			t.Errorf("CallFunction() = %v, want 5.0", result)
		}
		
		// Test calling non-existent function
		_, err = evalCtx.CallFunction("nonexistent", 42)
		if err == nil {
			t.Error("CallFunction() expected error for non-existent function")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(ctx, mockVarCtx)
		
		// Context should work normally
		err := evalCtx.CheckLimits()
		if err != nil {
			t.Errorf("CheckLimits() error before cancellation: %v", err)
		}
		
		// Cancel the context
		cancel()
		
		// Context should return error
		err = evalCtx.CheckLimits()
		if err == nil {
			t.Error("CheckLimits() expected error after cancellation")
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx).
			WithTimeout(50 * time.Millisecond)
		
		// Context should work initially
		err := evalCtx.CheckLimits()
		if err != nil {
			t.Errorf("CheckLimits() error before timeout: %v", err)
		}
		
		// Wait for timeout
		time.Sleep(100 * time.Millisecond)
		
		// Context should return timeout error
		err = evalCtx.CheckLimits()
		if err == nil {
			t.Error("CheckLimits() expected timeout error")
		}
	})

	t.Run("memory limits", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx).
			WithMemoryLimit(100) // Very small limit
		
		// Manually set memory used to exceed limit
		evalCtx.memoryUsed = 200
		
		err := evalCtx.CheckLimits()
		if err == nil {
			t.Error("CheckLimits() expected memory limit error")
		}
	})

	t.Run("depth limits", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		
		// Manually set depth to exceed limit
		evalCtx.depth = 100
		
		err := evalCtx.CheckLimits()
		if err == nil {
			t.Error("CheckLimits() expected depth limit error")
		}
	})

	t.Run("custom function registry", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		
		// Create custom registry
		customRegistry := make(FunctionRegistry)
		customRegistry["test"] = &testFunction{}
		
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx).
			WithFunctions(customRegistry)
		
		// Should have custom function
		if _, exists := evalCtx.functions["test"]; !exists {
			t.Error("custom function not found")
		}
		
		// Should not have default functions
		if _, exists := evalCtx.functions["abs"]; exists {
			t.Error("default function found in custom registry")
		}
	})

	t.Run("clone context", func(t *testing.T) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{
				"original": true,
			},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		
		// Clone the context
		cloned := evalCtx.Clone()
		
		// Should share configuration
		if cloned.timeout != evalCtx.timeout {
			t.Error("Clone() did not copy timeout")
		}
		if cloned.memoryLimit != evalCtx.memoryLimit {
			t.Error("Clone() did not copy memoryLimit")
		}
		
		// Should have separate variable cache
		if &cloned.variableCache == &evalCtx.variableCache {
			t.Error("Clone() shared variable cache")
		}
	})
}

// testFunction is a simple test function implementation
type testFunction struct{}

func (f *testFunction) Name() string { return "test" }
func (f *testFunction) MinArgs() int { return 0 }
func (f *testFunction) MaxArgs() int { return -1 }
func (f *testFunction) Call(_ *EvaluationContext, args ...any) (any, error) {
	return "test result", nil
}

func BenchmarkEvaluationContext(b *testing.B) {
	b.Run("CallFunction", func(b *testing.B) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			evalCtx.CallFunction("abs", -42.0)
		}
	})
	
	b.Run("CheckLimits", func(b *testing.B) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			evalCtx.CheckLimits()
		}
	})
	
	b.Run("Clone", func(b *testing.B) {
		mockVarCtx := &mockVariableContext{
			fields: map[string]any{},
		}
		evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			evalCtx.Clone()
		}
	})
}