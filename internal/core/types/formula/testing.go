package formula

import (
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// * TestFormulaRequest represents a request to test a formula expression
type TestFormulaRequest struct {
	// Organization and business unit context
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
	UserID pulid.ID `json:"userId"`

	// Formula expression to test
	Expression string `json:"expression"`

	// Test data to use for evaluation
	TestData map[string]any `json:"testData"`

	// Optional parameters (like template parameters)
	Parameters map[string]any `json:"parameters,omitempty"`

	// Optional constraints to test
	MinRate *float64 `json:"minRate,omitempty"`
	MaxRate *float64 `json:"maxRate,omitempty"`
}

// * TestFormulaResponse represents the result of testing a formula
type TestFormulaResponse struct {
	// Whether the test was successful
	Success bool `json:"success"`

	// The calculated result (if successful)
	Result float64 `json:"result,omitempty"`

	// Raw result before conversion
	RawResult any `json:"rawResult,omitempty"`

	// Error message (if failed)
	Error string `json:"error,omitempty"`

	// Variables that were used in the expression
	UsedVariables []string `json:"usedVariables,omitempty"`

	// Step-by-step evaluation trace
	EvaluationSteps []EvaluationStep `json:"evaluationSteps,omitempty"`

	// Available context (variables and their values)
	AvailableContext map[string]any `json:"availableContext,omitempty"`
}

// * EvaluationStep represents a step in the evaluation process
type EvaluationStep struct {
	Step        string `json:"step"`
	Description string `json:"description"`
	Result      string `json:"result"`
}

// * VariableInfo provides information about an available variable
type VariableInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Example     any    `json:"example,omitempty"`
}

// * GetAvailableVariablesResponse returns all available variables
type GetAvailableVariablesResponse struct {
	Variables []VariableInfo `json:"variables"`
	Functions []FunctionInfo `json:"functions"`
}

// * FunctionInfo provides information about an available function
type FunctionInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	Example     string   `json:"example"`
}
