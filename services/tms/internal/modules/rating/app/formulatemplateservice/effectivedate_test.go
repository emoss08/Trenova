package formulatemplateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func scheduleRequest(
	template *formulatemplate.FormulaTemplate,
	version int64,
	effectiveFrom *int64,
) *formulatemplateports.UpdateEffectiveDateRequest {
	return &formulatemplateports.UpdateEffectiveDateRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    template.ID,
		VersionNumber: version,
		EffectiveFrom: effectiveFrom,
	}
}

func approvedSnapshot(
	template *formulatemplate.FormulaTemplate,
	number int64,
	effectiveFrom int64,
) *formulatemplate.FormulaTemplateVersion {
	return &formulatemplate.FormulaTemplateVersion{
		TemplateID:    template.ID,
		VersionNumber: number,
		Name:          template.Name,
		Description:   template.Description,
		Status:        formulatemplate.StatusActive,
		Expression:    "totalDistance * 2",
		SchemaID:      template.SchemaID,
		Type:          template.Type,
		CreatedAt:     effectiveFrom,
		EffectiveFrom: &effectiveFrom,
	}
}

func TestUpdateVersionEffectiveDate_AWindowThatBilledCannotBeUnscheduled(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	started := timeutils.NowUnix() - 30*86_400
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(approvedSnapshot(template, 2, started), nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.UpdateVersionEffectiveDate(t.Context(), scheduleRequest(template, 2, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already billed")
	deps.versionRepo.AssertNotCalled(t, "UpdateEffectiveDate")
}

func TestUpdateVersionEffectiveDate_ClearingAPendingWindowStartsItNow(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	pending := timeutils.NowUnix() + 30*86_400
	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(approvedSnapshot(template, 2, pending), nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	before := timeutils.NowUnix()
	var sent *formulatemplateports.UpdateEffectiveDateRequest
	deps.versionRepo.On("UpdateEffectiveDate", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			sent = args.Get(1).(*formulatemplateports.UpdateEffectiveDateRequest)
		}).
		Return(approvedSnapshot(template, 2, before), nil)

	_, err := deps.svc.UpdateVersionEffectiveDate(t.Context(), scheduleRequest(template, 2, nil))
	require.NoError(t, err)
	require.NotNil(t, sent)
	require.NotNil(t, sent.EffectiveFrom, "a cleared schedule must not reach back to the approval")
	assert.GreaterOrEqual(t, *sent.EffectiveFrom, before)
}

func TestUpdateVersionEffectiveDate_SchedulingBilledContentStartsANewWindow(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.CurrentVersionNumber = 4
	deps.testCaseRepo.cases = append(deps.testCaseRepo.cases, newTestCase("exact match", 250, 0.01))
	started := timeutils.NowUnix() - 30*86_400
	old := approvedSnapshot(template, 2, started)
	old.Expression = "totalDistance * 2.5"

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).Return(old, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(func(_ context.Context, entity *formulatemplate.FormulaTemplate) *formulatemplate.FormulaTemplate {
			return entity
		}, nil)
	var created *formulatemplate.FormulaTemplateVersion
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(func(_ context.Context, v *formulatemplate.FormulaTemplateVersion) *formulatemplate.FormulaTemplateVersion {
			return v
		}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	next := timeutils.NowUnix() + 14*86_400
	result, err := deps.svc.UpdateVersionEffectiveDate(
		t.Context(),
		scheduleRequest(template, 2, &next),
	)
	require.NoError(t, err)

	require.NotNil(t, created)
	assert.EqualValues(t, 5, created.VersionNumber, "a new window, not a moved one")
	assert.Equal(t, "totalDistance * 2.5", created.Expression)
	assert.Equal(t, formulatemplate.StatusActive, created.Status)
	require.NotNil(t, created.EffectiveFrom)
	assert.Equal(t, next, *created.EffectiveFrom)
	assert.EqualValues(t, 5, result.VersionNumber)
	assert.Equal(t, started, *old.EffectiveFrom, "the billed window keeps its start")
	deps.versionRepo.AssertNotCalled(t, "UpdateEffectiveDate")
}

func TestUpdateVersionEffectiveDate_SchedulingBilledContentRunsItsScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.CurrentVersionNumber = 4
	deps.testCaseRepo.cases = append(deps.testCaseRepo.cases, newTestCase("exact match", 250, 0.01))
	old := approvedSnapshot(template, 2, timeutils.NowUnix()-30*86_400)
	old.Expression = "totalDistance * 9"

	deps.versionRepo.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).Return(old, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	next := timeutils.NowUnix() + 14*86_400
	_, err := deps.svc.UpdateVersionEffectiveDate(
		t.Context(),
		scheduleRequest(template, 2, &next),
	)
	require.Error(t, err, "the old content no longer passes the template's scenarios")
	deps.versionRepo.AssertNotCalled(t, "Create")
	deps.repo.AssertNotCalled(t, "Update")
}
