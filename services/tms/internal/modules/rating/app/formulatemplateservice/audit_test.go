package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAudit_ANonMaterialEditIsNotCriticalButCarriesTheRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	logged := formulatemplatetest.CaptureAudit(deps.auditSvc)

	original := formulatemplatetest.NewTemplate()
	edited := *original
	edited.Description = "clearer words"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(&edited, nil)

	_, err := deps.svc.Update(
		formulatemplatetest.RequestContext(t),
		&edited,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)

	entry := logged.Last(t)
	assert.False(t, entry.Critical)
	assert.Equal(t, "req-42", entry.Metadata["requestId"])
}

func TestAudit_AMaterialEditIsCritical(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	logged := formulatemplatetest.CaptureAudit(deps.auditSvc)

	original := formulatemplatetest.NewTemplate()
	edited := *original
	edited.Source = formulatemplatetest.PerMileRetypedSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(&edited, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)

	_, err := deps.svc.Update(
		formulatemplatetest.RequestContext(t),
		&edited,
		formulatemplatetest.NewTenantInfo().UserID,
	)
	require.NoError(t, err)
	assert.True(t, logged.Last(t).Critical)
}
