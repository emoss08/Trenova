package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardCatalog_EveryTemplatePassesTheModulesCheckAndPrices(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	require.NotEmpty(t, catalog)

	svc := newFormulaService(t)
	for _, entry := range catalog {
		t.Run(entry.Name, func(t *testing.T) {
			t.Parallel()

			template := formulatemplate.Seed{
				Name:                entry.Name,
				Type:                entry.Type,
				Expression:          entry.Expression,
				SchemaID:            entry.SchemaID,
				VariableDefinitions: entry.VariableDefinitions,
			}.Build()

			checked, checkErr := svc.CheckTemplate(t.Context(), template)
			require.NoError(t, checkErr)
			for _, item := range checked {
				refusal := item.Check.FirstError()
				assert.Nil(t, refusal, "%s %s: %+v", entry.Name, item.Scope, refusal)
			}

			preview, previewErr := svc.Preview(t.Context(), &formula.PreviewRequest{
				SchemaID:   entry.SchemaID,
				Expression: entry.Expression,
				Variables:  template.ScenarioContent().VariableDefaults(),
			})
			require.NoError(t, previewErr)
			require.NotNil(t, preview.Evaluated, "%s does not run on the sample load", entry.Name)
			if preview.Error != nil {
				assert.NotEqual(t, rules.ErrorMissingField, preview.Error.Err.Code,
					"%s needs a value the sample load lacks: %v", entry.Name, preview.Error)
			}
		})
	}
}
