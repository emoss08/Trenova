package formulatemplateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice/standardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestInstallStandards_CreatesMissingTemplates also proves every catalog
// template compiles against the shipment schema: InstallStandards validates
// each expression before inserting, so a broken catalog entry fails this test.
func TestInstallStandards_CreatesMissingTemplates(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.MatchedBy(
		func(entity *formulatemplate.FormulaTemplate) bool {
			return entity.Status == formulatemplate.StatusActive &&
				entity.CurrentVersionNumber == 1
		},
	)).Return(func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
		return entity, nil
	}).Times(len(catalog))
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil).
		Times(len(catalog))
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.InstallStandards(t.Context(), newTenantInfo())

	require.NoError(t, err)
	assert.Len(t, result.Installed, len(catalog))
	assert.Empty(t, result.Skipped)
	deps.repo.AssertExpectations(t)
	deps.versionRepo.AssertExpectations(t)
}

func TestInstallStandards_SkipsExistingTemplates(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	existing := make([]*formulatemplate.FormulaTemplate, 0, len(catalog))
	for _, entry := range catalog {
		template := newTestTemplate()
		template.Name = entry.Name
		existing = append(existing, template)
	}

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).Return(existing, nil)

	result, err := deps.svc.InstallStandards(t.Context(), newTenantInfo())

	require.NoError(t, err)
	assert.Empty(t, result.Installed)
	assert.Len(t, result.Skipped, len(catalog))
	deps.repo.AssertNotCalled(t, "Create")
}

func TestInstallStandards_FillsOnlyGaps(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	require.Greater(t, len(catalog), 1)

	alreadyPresent := newTestTemplate()
	alreadyPresent.Name = catalog[0].Name

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{alreadyPresent}, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			return entity, nil
		}).Times(len(catalog) - 1)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil).
		Times(len(catalog) - 1)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.InstallStandards(t.Context(), newTenantInfo())

	require.NoError(t, err)
	assert.Len(t, result.Installed, len(catalog)-1)
	assert.Equal(t, []string{catalog[0].Name}, result.Skipped)
}
