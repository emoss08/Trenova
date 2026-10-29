package formulatemplatereviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAudit_ApprovalIsCriticalAndCarriesTheRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	logged := formulatemplatetest.CaptureAudit(deps.auditSvc)
	template := approvableTemplate(deps)

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)

	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Approve(formulatemplatetest.RequestContext(t), &ApprovalActionRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)

	entry := logged.Last(t)
	assert.True(t, entry.Critical)
	assert.Equal(t, "req-42", entry.Metadata["requestId"])
	assert.Equal(t, "10.0.0.7", entry.IPAddress)
}
