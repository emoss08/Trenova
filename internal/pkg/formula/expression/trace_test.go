package expression

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/emoss08/trenova/internal/pkg/formula/variables/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// * testVariable is a simple test variable for testing
type testVariable struct {
	name        string
	description string
	valueType   formula.ValueType
	resolver    func(variables.VariableContext) (any, error)
}

func (v *testVariable) Name() string                { return v.name }
func (v *testVariable) Description() string         { return v.description }
func (v *testVariable) Type() formula.ValueType     { return v.valueType }
func (v *testVariable) Category() string            { return "test" }
func (v *testVariable) Resolve(ctx variables.VariableContext) (any, error) {
	return v.resolver(ctx)
}
func (v *testVariable) Validate(value any) error { return nil }

func setupTracingEvaluator(t *testing.T) *TracingEvaluator {
	// Create variable registry
	registry := variables.NewRegistry()
	
	// Register builtin variables
	builtin.RegisterAll(registry)
	
	// Register some test variables that use metadata
	registry.MustRegister(&testVariable{
		name:        "weight",
		description: "Shipment weight",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["weight"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	registry.MustRegister(&testVariable{
		name:        "distance",
		description: "Shipment distance",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["distance"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	registry.MustRegister(&testVariable{
		name:        "rate",
		description: "Base rate",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["rate"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	return NewTracingEvaluator(registry)
}

func TestTracingEvaluator_BasicExpression(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Create a simple variable context
	varCtx := createTestContext(map[string]any{
		"distance": 100.0,
		"rate":     2.5,
	})
	
	result, trace, err := evaluator.EvaluateWithTrace(ctx, "distance * rate", varCtx)
	require.NoError(t, err)
	assert.Equal(t, 250.0, result)
	
	// Verify trace steps
	assert.Len(t, trace, 6) // Tokenization, Parsing, Variable Extraction, Resolution, Evaluation, Type Conversion
	
	// Check tokenization
	assert.Equal(t, "Tokenization", trace[0].Step)
	assert.Contains(t, trace[0].Result, "Success")
	
	// Check parsing
	assert.Equal(t, "Parsing", trace[1].Step)
	assert.Equal(t, "Success", trace[1].Result)
	
	// Check variable extraction
	assert.Equal(t, "Variable Extraction", trace[2].Step)
	assert.Contains(t, trace[2].Result, "2 variables")
	
	// Check variable resolution
	assert.Equal(t, "Variable Resolution", trace[3].Step)
	assert.Len(t, trace[3].Children, 2)
	
	// Check evaluation
	assert.Equal(t, "Expression Evaluation", trace[4].Step)
	assert.Equal(t, "Success", trace[4].Result)
	
	// Check the evaluation tree
	evalTree := trace[4].Children
	assert.Greater(t, len(evalTree), 0)
}

func TestTracingEvaluator_ComplexExpression(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Register additional test variables
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:        "base_rate",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["base_rate"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:        "fuel_charge",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["fuel_charge"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	varCtx := createTestContext(map[string]any{
		"weight":      1000.0,
		"distance":    50.0,
		"base_rate":   1.5,
		"fuel_charge": 0.5,
	})
	
	expr := "(weight / 100) * distance * (base_rate + fuel_charge)"
	result, trace, err := evaluator.EvaluateWithTrace(ctx, expr, varCtx)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, result) // (1000/100) * 50 * (1.5 + 0.5) = 10 * 50 * 2 = 1000
	
	// Verify we have all main steps
	assert.Greater(t, len(trace), 4)
	
	// Check that evaluation step has children
	evalStep := trace[4]
	assert.Equal(t, "Expression Evaluation", evalStep.Step)
	assert.Greater(t, len(evalStep.Children), 0)
}

func TestTracingEvaluator_ConditionalExpression(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Register rate variables
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:        "short_rate",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["short_rate"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:        "long_rate",
		valueType:   formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["long_rate"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	varCtx := createTestContext(map[string]any{
		"distance": 150.0,
		"short_rate": 2.0,
		"long_rate":  1.5,
	})
	
	expr := "distance > 100 ? long_rate : short_rate"
	result, trace, err := evaluator.EvaluateWithTrace(ctx, expr, varCtx)
	require.NoError(t, err)
	assert.Equal(t, 1.5, result)
	
	// Find the evaluation step
	var evalStep *TraceStep
	for i := range trace {
		if trace[i].Step == "Expression Evaluation" {
			evalStep = &trace[i]
			break
		}
	}
	require.NotNil(t, evalStep)
	
	// Check that conditional node was evaluated
	assert.Greater(t, len(evalStep.Children), 0)
	
	// Find conditional node in children
	var condNode *TraceStep
	for i := range evalStep.Children {
		if evalStep.Children[i].Step == "Conditional Expression" {
			condNode = &evalStep.Children[i]
			break
		}
	}
	
	if condNode != nil {
		// Should have condition and true branch children
		assert.GreaterOrEqual(t, len(condNode.Children), 2)
	}
}

func TestTracingEvaluator_FunctionCall(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Register value variables
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("value%d", i)
		evaluator.evaluator.variables.MustRegister(&testVariable{
			name:        name,
			valueType:   formula.ValueTypeNumber,
			resolver: func(varName string) func(variables.VariableContext) (any, error) {
				return func(ctx variables.VariableContext) (any, error) {
					if meta := ctx.GetMetadata(); meta != nil {
						if val, ok := meta[varName]; ok {
							return val, nil
						}
					}
					return 0.0, nil
				}
			}(name),
		})
	}
	
	varCtx := createTestContext(map[string]any{
		"value1": 10.0,
		"value2": 20.0,
		"value3": 15.0,
	})
	
	expr := "max(value1, value2, value3)"
	result, trace, err := evaluator.EvaluateWithTrace(ctx, expr, varCtx)
	require.NoError(t, err)
	assert.Equal(t, 20.0, result)
	
	// Find evaluation step
	var evalStep *TraceStep
	for i := range trace {
		if trace[i].Step == "Expression Evaluation" {
			evalStep = &trace[i]
			break
		}
	}
	require.NotNil(t, evalStep)
	
	// Find function call node
	var funcNode *TraceStep
	for i := range evalStep.Children {
		if evalStep.Children[i].Step == "Function Call: max" {
			funcNode = &evalStep.Children[i]
			break
		}
	}
	
	if funcNode != nil {
		// Should have 3 argument children
		assert.Equal(t, 3, len(funcNode.Children))
	}
}

func TestTracingEvaluator_ErrorHandling(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Register zero variable for division by zero test
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:      "zero",
		valueType: formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["zero"]; ok {
					return val, nil
				}
			}
			return 0.0, nil
		},
	})
	
	tests := []struct {
		name    string
		expr    string
		varCtx  variables.VariableContext
		errStep string
	}{
		{
			name: "Invalid syntax",
			expr: "2 +* 3",
			varCtx: createTestContext(nil),
			errStep: "Parsing",
		},
		{
			name: "Unknown variable", 
			expr: "unknown_var * 2",
			varCtx: createTestContext(nil),
			errStep: "Expression Evaluation",
		},
		{
			name: "Division by zero",
			expr: "10 / zero",
			varCtx: createTestContext(map[string]any{"zero": 0.0}),
			errStep: "Expression Evaluation",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, trace, err := evaluator.EvaluateWithTrace(ctx, tt.expr, tt.varCtx)
			assert.Error(t, err)
			
			// Find the failed step
			var failedStep *TraceStep
			for i := range trace {
				if trace[i].Step == tt.errStep {
					failedStep = &trace[i]
					break
				}
			}
			
			if failedStep != nil {
				assert.Contains(t, failedStep.Result, "Failed")
			}
		})
	}
}

// * testVariableContext implements VariableContext for testing tracing
type testVariableContext struct {
	entity   any
	fields   map[string]any
	computed map[string]any
	metadata map[string]any
}

func (c *testVariableContext) GetEntity() any {
	return c.entity
}

func (c *testVariableContext) GetField(path string) (any, error) {
	if c.fields != nil {
		if val, ok := c.fields[path]; ok {
			return val, nil
		}
	}
	return nil, nil
}

func (c *testVariableContext) GetComputed(function string) (any, error) {
	if c.computed != nil {
		if val, ok := c.computed[function]; ok {
			return val, nil
		}
	}
	return nil, nil
}

func (c *testVariableContext) GetMetadata() map[string]any {
	if c.metadata == nil {
		return make(map[string]any)
	}
	return c.metadata
}

func createTestContext(metadata map[string]any) *testVariableContext {
	return &testVariableContext{
		metadata: metadata,
	}
}

func TestTracingEvaluator_AllNodeTypes(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	// Register test variables
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:      "x",
		valueType: formula.ValueTypeNumber,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["x"]; ok {
					return val, nil
				}
			}
			return 10.0, nil
		},
	})
	
	evaluator.evaluator.variables.MustRegister(&testVariable{
		name:      "flag",
		valueType: formula.ValueTypeBoolean,
		resolver: func(ctx variables.VariableContext) (any, error) {
			if meta := ctx.GetMetadata(); meta != nil {
				if val, ok := meta["flag"]; ok {
					return val, nil
				}
			}
			return true, nil
		},
	})
	
	tests := []struct {
		name        string
		expr        string
		metadata    map[string]any
		expected    float64
		checkNodes  []string
	}{
		{
			name:       "Number literal",
			expr:       "42.5",
			expected:   42.5,
			checkNodes: []string{"Number"},
		},
		{
			name:       "Boolean literal",
			expr:       "true",
			expected:   1.0,
			checkNodes: []string{"Boolean"},
		},
		{
			name:       "String comparison",
			expr:       `"hello" == "hello"`,
			expected:   1.0,
			checkNodes: []string{"String", "Binary Operation: =="},
		},
		{
			name:       "Unary negation",
			expr:       "-x",
			metadata:   map[string]any{"x": 5.0},
			expected:   -5.0,
			checkNodes: []string{"Unary Operation: -"},
		},
		{
			name:       "Unary not",
			expr:       "!flag",
			metadata:   map[string]any{"flag": false},
			expected:   1.0,
			checkNodes: []string{"Unary Operation: !"},
		},
		{
			name:       "Array min",
			expr:       "min(5, 3, 8, 1)",
			expected:   1.0,
			checkNodes: []string{"Function Call: min"},
		},
		{
			name:       "Nested conditional",
			expr:       "x > 20 ? (x > 30 ? 3 : 2) : 1",
			metadata:   map[string]any{"x": 25.0},
			expected:   2.0,
			checkNodes: []string{"Conditional Expression"},
		},
		{
			name:       "Complex expression with all operators",
			expr:       "(x + 5) * 2 - 10 / 2",
			metadata:   map[string]any{"x": 10.0},
			expected:   25.0, // (10 + 5) * 2 - 10 / 2 = 15 * 2 - 5 = 30 - 5 = 25
			checkNodes: []string{"Binary Operation: +", "Binary Operation: *", "Binary Operation: /", "Binary Operation: -"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varCtx := createTestContext(tt.metadata)
			result, trace, err := evaluator.EvaluateWithTrace(ctx, tt.expr, varCtx)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result, 0.001)
			
			// Verify trace contains expected nodes
			var evalStep *TraceStep
			for i := range trace {
				if trace[i].Step == "Expression Evaluation" {
					evalStep = &trace[i]
					break
				}
			}
			require.NotNil(t, evalStep)
			
			// Check for expected node types in the trace
			foundNodes := collectNodeSteps(evalStep)
			for _, expectedNode := range tt.checkNodes {
				found := false
				for _, node := range foundNodes {
					if node == expectedNode {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find node '%s' in trace", expectedNode)
			}
		})
	}
}

func TestTracingEvaluator_TraceStructure(t *testing.T) {
	evaluator := setupTracingEvaluator(t)
	ctx := context.Background()
	
	varCtx := createTestContext(map[string]any{
		"weight": 100.0,
		"rate":   2.0,
	})
	
	_, trace, err := evaluator.EvaluateWithTrace(ctx, "weight * rate + 10", varCtx)
	require.NoError(t, err)
	
	// Verify trace structure
	assert.GreaterOrEqual(t, len(trace), 6, "Should have at least 6 main steps")
	
	// Check main steps in order
	expectedSteps := []string{
		"Tokenization",
		"Parsing", 
		"Variable Extraction",
		"Variable Resolution",
		"Expression Evaluation",
		"Type Conversion",
	}
	
	for i, expected := range expectedSteps {
		if i < len(trace) {
			assert.Equal(t, expected, trace[i].Step)
		}
	}
	
	// Check tokenization details
	assert.Contains(t, trace[0].Result, "Success")
	assert.NotNil(t, trace[0].Value)
	
	// Check variable resolution has children
	varResStep := trace[3]
	assert.Equal(t, "Variable Resolution", varResStep.Step)
	assert.Len(t, varResStep.Children, 2) // weight and rate
	
	// Check evaluation step has tree structure
	evalStep := trace[4]
	assert.Greater(t, len(evalStep.Children), 0)
}

// Helper function to collect all node steps from a trace tree
func collectNodeSteps(step *TraceStep) []string {
	nodes := []string{}
	if step.Step != "" {
		nodes = append(nodes, step.Step)
	}
	for i := range step.Children {
		nodes = append(nodes, collectNodeSteps(&step.Children[i])...)
	}
	return nodes
}