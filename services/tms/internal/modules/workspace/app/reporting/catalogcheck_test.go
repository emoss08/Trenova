package reporting

import (
	"errors"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/stretchr/testify/assert"
)

func TestCatalogDiagnostics_NameThePartAtFault(t *testing.T) {
	t.Parallel()

	assert.Nil(t, CatalogDiagnostics(nil))
	unknown := `unknown field "statsu" on shipment; did you mean "status"?`
	assert.Equal(t,
		[]string{
			"definition.columns[2].dimension.field: " + unknown,
			"definition: the report has no columns",
		},
		CatalogDiagnostics(ir.Errors{
			{Message: unknown, Pointer: "/columns/2/dimension/field"},
			{Message: "the report has no columns"},
		}),
	)
	assert.Equal(t, []string{"boom"}, CatalogDiagnostics(errors.New("boom")))
}

func TestSavingADefinitionClearsNeedsAttention(t *testing.T) {
	t.Parallel()

	existing := &report.ReportDefinition{
		Status:         report.DefinitionStatusNeedsAttention,
		Diagnostics:    []string{`definition.columns[0].dimension.field: unknown field "x"`},
		CatalogVersion: "v1",
	}
	ApplyDefinitionSave(existing, &SaveDefinitionRequest{
		Status: report.DefinitionStatusNeedsAttention,
	}, "v2")

	assert.Equal(t, report.DefinitionStatusActive, existing.Status,
		"a saved definition has just been validated, so it cannot still need attention")
	assert.Empty(t, existing.Diagnostics)
	assert.Equal(t, "v2", existing.CatalogVersion)

	created := NewDefinition(&SaveDefinitionRequest{
		Status: report.DefinitionStatusNeedsAttention,
	}, "v2")
	assert.Equal(t, report.DefinitionStatusActive, created.Status)

	draft := NewDefinition(&SaveDefinitionRequest{Status: report.DefinitionStatusDraft}, "v2")
	assert.Equal(t, report.DefinitionStatusDraft, draft.Status)
}
