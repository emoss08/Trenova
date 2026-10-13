package formulatemplate

import (
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

type ScenarioContent struct {
	Expression           string                              `json:"expression"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions"`
	MinCharge            decimal.NullDecimal                 `json:"minCharge"`
	MaxCharge            decimal.NullDecimal                 `json:"maxCharge"`
	RoundingMode         ratetypes.RoundingMode              `json:"roundingMode"`
	RoundingPrecision    int32                               `json:"roundingPrecision"`
}

func (ft *FormulaTemplate) ScenarioContent() *ScenarioContent {
	return &ScenarioContent{
		Expression:           ft.Expression,
		VariableDefinitions:  ft.VariableDefinitions,
		BreakdownDefinitions: ft.BreakdownDefinitions,
		MinCharge:            ft.MinCharge,
		MaxCharge:            ft.MaxCharge,
		RoundingMode:         ft.RoundingMode,
		RoundingPrecision:    ft.RoundingPrecision,
	}
}

func (v *FormulaTemplateVersion) ScenarioContent() *ScenarioContent {
	return &ScenarioContent{
		Expression:           v.Expression,
		VariableDefinitions:  v.VariableDefinitions,
		BreakdownDefinitions: v.BreakdownDefinitions,
		MinCharge:            v.MinCharge,
		MaxCharge:            v.MaxCharge,
		RoundingMode:         v.RoundingMode,
		RoundingPrecision:    v.RoundingPrecision,
	}
}

func (c *ScenarioContent) ChargePolicy() formulatypes.ChargePolicy {
	return formulatypes.ChargePolicy{
		MinCharge:         c.MinCharge,
		MaxCharge:         c.MaxCharge,
		RoundingMode:      c.RoundingMode,
		RoundingPrecision: c.RoundingPrecision,
	}
}

func (c *ScenarioContent) VariableDefaults() map[string]any {
	defaults := make(map[string]any, len(c.VariableDefinitions))
	for _, definition := range c.VariableDefinitions {
		if definition != nil && definition.Name != "" && definition.DefaultValue != nil {
			defaults[definition.Name] = definition.DefaultValue
		}
	}

	return defaults
}
