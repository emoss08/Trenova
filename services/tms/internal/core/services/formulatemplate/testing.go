package formulatemplate

import "github.com/emoss08/trenova/pkg/pulid"

type TestFormulaRequest struct {
	OrgID      pulid.ID       `json:"orgId"`
	BuID       pulid.ID       `json:"buId"`
	UserID     pulid.ID       `json:"userId"`
	Expression string         `json:"expression"`
	TestData   map[string]any `json:"testData"`
	Parameters map[string]any `json:"parameters,omitempty"`
	MinRate    *float64       `json:"minRate,omitempty"`
	MaxRate    *float64       `json:"maxRate,omitempty"`
}

type TestFormulaResponse struct {
	Success          bool             `json:"success"`
	Result           float64          `json:"result,omitempty"`
	RawResult        any              `json:"rawResult,omitempty"`
	Error            string           `json:"error,omitempty"`
	UsedVariables    []string         `json:"usedVariables,omitempty"`
	EvaluationSteps  []EvaluationStep `json:"evaluationSteps,omitempty"`
	AvailableContext map[string]any   `json:"availableContext,omitempty"`
}

type EvaluationStep struct {
	Step        string `json:"step"`
	Description string `json:"description"`
	Result      string `json:"result"`
}

type VariableInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Example     any    `json:"example,omitempty"`
}

type GetAvailableVariablesResponse struct {
	Variables []VariableInfo `json:"variables"`
	Functions []FunctionInfo `json:"functions"`
}

type FunctionInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	Example     string   `json:"example"`
}
