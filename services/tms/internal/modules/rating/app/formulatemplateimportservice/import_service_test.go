package formulatemplateimportservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestImport_ChecksEachSourceAndCreatesDrafts(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: ExportVersion,
		TenantInfo:    formulatemplatetest.NewTenantInfo(),
		Template: &ImportTemplatePayload{
			Name:        "Per Mile (imported)",
			Description: "From another organisation",
			Source:      formulatemplatetest.PerMileSource,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	assert.Equal(t, formulatemplate.StatusDraft, result.Created[0].Status)
	assert.Equal(t, "Per Mile (imported)", result.Created[0].Name)
	assert.Equal(
		t,
		strings.Replace(
			formulatemplatetest.PerMileSource,
			`tariff "Per Mile"`,
			`tariff "Per Mile (imported)"`,
			1,
		),
		result.Created[0].Source,
		"the file's name is written into the header",
	)
	assert.Len(t, result.Created[0].Inputs, 2)

	_, err = deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: ExportVersion,
		TenantInfo:    formulatemplatetest.NewTenantInfo(),
		Template: &ImportTemplatePayload{
			Name:   "Broken",
			Source: formulatemplatetest.BrokenSource,
		},
	})
	require.Error(t, err, "an import that does not check is refused whole")

	_, err = deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.0",
		TenantInfo:    formulatemplatetest.NewTenantInfo(),
		Template: &ImportTemplatePayload{
			Name:   "Old",
			Source: formulatemplatetest.PerMileSource,
		},
	})
	assert.Contains(t, formulatemplatetest.ValidationFields(t, err), "exportVersion")
}
