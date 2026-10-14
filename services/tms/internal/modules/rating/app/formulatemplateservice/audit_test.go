package formulatemplateservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type auditLog struct {
	mu      sync.Mutex
	entries []*audit.Entry
}

func (l *auditLog) last(t *testing.T) *audit.Entry {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	require.NotEmpty(t, l.entries)
	return l.entries[len(l.entries)-1]
}

func captureAudit(deps *testDeps) *auditLog {
	log := &auditLog{}
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			params := args.Get(0).(*auditports.LogActionParams)
			entry := &audit.Entry{
				ResourceID: params.ResourceID,
				Operation:  params.Operation,
			}
			if opts, ok := args.Get(1).([]auditports.LogOption); ok {
				for _, opt := range opts {
					_ = opt(entry)
				}
			}
			log.mu.Lock()
			log.entries = append(log.entries, entry)
			log.mu.Unlock()
		}).
		Return(nil)
	return log
}

func requestContext(t *testing.T) context.Context {
	t.Helper()
	return requestmeta.With(t.Context(), requestmeta.New("req-42", "10.0.0.7", "studio"))
}

func TestAudit_ApprovalIsCriticalAndCarriesTheRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	logged := captureAudit(deps)
	template := approvableTemplate(deps)

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)

	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Approve(requestContext(t), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)

	entry := logged.last(t)
	assert.True(t, entry.Critical)
	assert.Equal(t, "req-42", entry.Metadata["requestId"])
	assert.Equal(t, "10.0.0.7", entry.IPAddress)
}

func TestAudit_ANonMaterialEditIsNotCriticalButCarriesTheRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	logged := captureAudit(deps)

	original := newTestTemplate()
	edited := *original
	edited.Description = "clearer words"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(&edited, nil)

	_, err := deps.svc.Update(requestContext(t), &edited, newTenantInfo().UserID)
	require.NoError(t, err)

	entry := logged.last(t)
	assert.False(t, entry.Critical)
	assert.Equal(t, "req-42", entry.Metadata["requestId"])
}

func TestAudit_AMaterialEditIsCritical(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	logged := captureAudit(deps)

	original := newTestTemplate()
	edited := *original
	edited.Expression = "totalDistance * 3"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(&edited, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)

	_, err := deps.svc.Update(requestContext(t), &edited, newTenantInfo().UserID)
	require.NoError(t, err)
	assert.True(t, logged.last(t).Critical)
}

func TestAudit_ScenarioChangesAreRecordedWithBeforeAndAfter(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	logged := captureAudit(deps)

	template := newTestTemplate()
	scenario := newTestCase("exact match", 250, 0.01)
	scenario.TemplateID = template.ID
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{scenario}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	tenant := newTenantInfo()
	_, err := deps.svc.UpdateTestCase(requestContext(t), &UpdateTestCaseRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
		TestCaseID: scenario.ID,
		TestCaseInput: TestCaseInput{
			Name:           "exact match",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(300),
			Tolerance:      decimal.RequireFromString("0.01"),
		},
	})
	require.NoError(t, err)

	updated := logged.last(t)
	assert.Equal(t, template.ID.String(), updated.ResourceID)
	before, ok := updated.Metadata["scenarioBefore"].(map[string]any)
	require.True(t, ok, "the scenario as it was")
	assert.Equal(t, "250", before["expectedAmount"])
	after, ok := updated.Metadata["scenarioAfter"].(map[string]any)
	require.True(t, ok, "the scenario as it is")
	assert.Equal(t, "300", after["expectedAmount"])

	err = deps.svc.DeleteTestCase(requestContext(t), formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
		TestCaseID: scenario.ID,
	})
	require.NoError(t, err)

	deleted := logged.last(t)
	assert.Equal(t, template.ID.String(), deleted.ResourceID)
	removed, ok := deleted.Metadata["scenarioBefore"].(map[string]any)
	require.True(t, ok, "a deleted scenario is recorded as it was")
	assert.Equal(t, "exact match", removed["name"])
	assert.Equal(t, "req-42", deleted.Metadata["requestId"])
}
