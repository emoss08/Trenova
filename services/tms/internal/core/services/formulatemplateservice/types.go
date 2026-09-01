package formulatemplateservice

import (
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

type templateSnapshot struct {
	Description          string
	Type                 formulatemplate.TemplateType
	Expression           string
	SchemaID             string
	VariableDefinitions  []*formulatypes.VariableDefinition
	BreakdownDefinitions []*formulatypes.BreakdownDefinition
	MinCharge            decimal.NullDecimal
	MaxCharge            decimal.NullDecimal
	RoundingMode         ratetypes.RoundingMode
	RoundingPrecision    int32
	Metadata             map[string]any
}

func snapshotFromVersion(v *formulatemplate.FormulaTemplateVersion) templateSnapshot {
	return templateSnapshot{
		Description:          v.Description,
		Type:                 v.Type,
		Expression:           v.Expression,
		SchemaID:             v.SchemaID,
		VariableDefinitions:  v.VariableDefinitions,
		BreakdownDefinitions: v.BreakdownDefinitions,
		MinCharge:            v.MinCharge,
		MaxCharge:            v.MaxCharge,
		RoundingMode:         v.RoundingMode,
		RoundingPrecision:    v.RoundingPrecision,
		Metadata:             v.Metadata,
	}
}

func snapshotFromTemplate(t *formulatemplate.FormulaTemplate) templateSnapshot {
	return templateSnapshot{
		Description:          t.Description,
		Type:                 t.Type,
		Expression:           t.Expression,
		SchemaID:             t.SchemaID,
		VariableDefinitions:  t.VariableDefinitions,
		BreakdownDefinitions: t.BreakdownDefinitions,
		MinCharge:            t.MinCharge,
		MaxCharge:            t.MaxCharge,
		RoundingMode:         t.RoundingMode,
		RoundingPrecision:    t.RoundingPrecision,
		Metadata:             t.Metadata,
	}
}
