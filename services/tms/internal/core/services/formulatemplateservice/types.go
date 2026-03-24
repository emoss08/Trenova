package formulatemplateservice

import (
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

type templateSnapshot struct {
	Description         string
	Type                formulatemplate.TemplateType
	Expression          string
	SchemaID            string
	VariableDefinitions []*formulatypes.VariableDefinition
	Metadata            map[string]any
}

func snapshotFromVersion(v *formulatemplate.FormulaTemplateVersion) templateSnapshot {
	return templateSnapshot{
		Description:         v.Description,
		Type:                v.Type,
		Expression:          v.Expression,
		SchemaID:            v.SchemaID,
		VariableDefinitions: v.VariableDefinitions,
		Metadata:            v.Metadata,
	}
}

func snapshotFromTemplate(t *formulatemplate.FormulaTemplate) templateSnapshot {
	return templateSnapshot{
		Description:         t.Description,
		Type:                t.Type,
		Expression:          t.Expression,
		SchemaID:            t.SchemaID,
		VariableDefinitions: t.VariableDefinitions,
		Metadata:            t.Metadata,
	}
}

func applyVersionToTemplate(
	t *formulatemplate.FormulaTemplate,
	v *formulatemplate.FormulaTemplateVersion,
) {
	t.Name = v.Name
	t.Description = v.Description
	t.Type = v.Type
	t.Expression = v.Expression
	t.Status = v.Status
	t.SchemaID = v.SchemaID
	t.VariableDefinitions = v.VariableDefinitions
	t.Metadata = v.Metadata
}
