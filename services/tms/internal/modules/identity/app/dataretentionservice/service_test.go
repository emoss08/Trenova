package dataretentionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) (*Service, *tenantportsmock.MockDataRetentionRepository) {
	t.Helper()

	repo := tenantportsmock.NewMockDataRetentionRepository(t)

	return New(Params{Logger: zap.NewNop(), Repo: repo}), repo
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

type recordingAuditor struct {
	changes []*permissionports.SecurityChange
}

func (a *recordingAuditor) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	a.changes = append(a.changes, change)
}

func getFindsNothing(repo *tenantportsmock.MockDataRetentionRepository) {
	repo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("tenantports.GetDataRetentionRequest")).
		Return(nil, errortypes.NewNotFoundError("data retention not found")).
		Once()
}

func intPtr(value int) *int {
	return &value
}

func upsertReturnsEntity(
	repo *tenantportsmock.MockDataRetentionRepository,
) *dataretention.DataRetention {
	saved := &dataretention.DataRetention{}
	repo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*dataretention.DataRetention")).
		RunAndReturn(func(_ context.Context, entity *dataretention.DataRetention) (*dataretention.DataRetention, error) {
			*saved = *entity
			return entity, nil
		}).
		Once()

	return saved
}

func TestGet_DefaultsTheAIAuditTrailToSevenYears(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	info := testTenant()
	repo.EXPECT().
		Get(mock.Anything, tenantports.GetDataRetentionRequest{
			UserID: info.UserID,
			OrgID:  info.OrgID,
			BuID:   info.BuID,
		}).
		Return(nil, errortypes.NewNotFoundError("data retention not found")).
		Once()

	entity, err := svc.Get(t.Context(), info)

	require.NoError(t, err)
	assert.Equal(t, dataretention.DefaultAIAuditRetentionDays, entity.AIAuditRetentionPeriod)
}

func TestUpdate_SavesTheAIAuditTrailRetentionItWasGiven(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	saved := upsertReturnsEntity(repo)
	getFindsNothing(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                    testTenant(),
		AuditOperationalRetentionDays: 730,
		AgentEvalCaseRetentionPeriod:  intPtr(365),
		AIAuditRetentionPeriod:        intPtr(3650),
		AICorrectionRetentionPeriod:   intPtr(365),
	})

	require.NoError(t, err)
	assert.Equal(t, 3650, saved.AIAuditRetentionPeriod)
	assert.Equal(t, 365, saved.AgentEvalCaseRetentionPeriod)
	assert.Equal(t, 365, saved.AICorrectionRetentionPeriod)
}

func TestUpdate_KeepsTheCurrentAIAuditTrailRetentionWhenNotSent(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	info := testTenant()
	repo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("tenantports.GetDataRetentionRequest")).
		Return(&dataretention.DataRetention{
			AuditOperationalRetentionDays: 730,
			AgentEvalCaseRetentionPeriod:  90,
			AIAuditRetentionPeriod:        1825,
			AICorrectionRetentionPeriod:   400,
		}, nil).
		Once()
	saved := upsertReturnsEntity(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                    info,
		AuditOperationalRetentionDays: 730,
	})

	require.NoError(t, err)
	assert.Equal(t, 1825, saved.AIAuditRetentionPeriod)
	assert.Equal(t, 90, saved.AgentEvalCaseRetentionPeriod)
	assert.Equal(t, 400, saved.AICorrectionRetentionPeriod)
}

func TestUpdate_RefusesAnAIAuditTrailKeptUnderAYear(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	getFindsNothing(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                    testTenant(),
		AuditOperationalRetentionDays: 730,
		AgentEvalCaseRetentionPeriod:  intPtr(365),
		AIAuditRetentionPeriod:        intPtr(30),
		AICorrectionRetentionPeriod:   intPtr(730),
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiAuditRetentionPeriod", multiErr.Errors[0].Field)
}

func TestUpdate_RefusesAICorrectionsKeptUnderAMonth(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	getFindsNothing(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                    testTenant(),
		AuditOperationalRetentionDays: 730,
		AgentEvalCaseRetentionPeriod:  intPtr(365),
		AIAuditRetentionPeriod:        intPtr(2555),
		AICorrectionRetentionPeriod:   intPtr(7),
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiCorrectionRetentionPeriod", multiErr.Errors[0].Field)
}

func TestGet_DefaultsAICorrectionRetentionWhenNothingIsSaved(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	repo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("tenantports.GetDataRetentionRequest")).
		Return(nil, errortypes.NewNotFoundError("data retention not found")).
		Once()

	entity, err := svc.Get(t.Context(), testTenant())

	require.NoError(t, err)
	assert.Equal(
		t,
		dataretention.DefaultAICorrectionRetentionDays,
		entity.AICorrectionRetentionPeriod,
	)
}

func TestUpdate_RefusesFinancialAuditEntriesKeptUnderSevenYears(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	getFindsNothing(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                  testTenant(),
		AuditFinancialRetentionDays: 365,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "auditFinancialRetentionDays", multiErr.Errors[0].Field)
}

func TestUpdate_KeepsAnAuditClassLeftUnsetAtItsFloor(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	getFindsNothing(repo)
	saved := upsertReturnsEntity(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                 testTenant(),
		AuditSecurityRetentionDays: 3650,
	})

	require.NoError(t, err)
	assert.Equal(t, 3650, saved.AuditSecurityRetentionDays)
	assert.Equal(t, dataretention.MinAuditAccessRetentionDays, saved.AuditAccessRetentionDays)
	assert.Equal(t, dataretention.MinAuditFinancialRetentionDays, saved.AuditFinancialRetentionDays)
}

func TestUpdate_RecordsTheChangeAsASecurityEntry(t *testing.T) {
	t.Parallel()

	repo := tenantportsmock.NewMockDataRetentionRepository(t)
	auditor := &recordingAuditor{}
	svc := New(Params{Logger: zap.NewNop(), Repo: repo, Auditor: auditor})
	info := testTenant()
	getFindsNothing(repo)
	upsertReturnsEntity(repo)
	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{TenantInfo: info})

	require.NoError(t, err)
	require.Len(t, auditor.changes, 1)
	change := auditor.changes[0]
	assert.Equal(t, "data_retention.updated", change.Action)
	assert.Equal(t, info.OrgID.String(), change.ResourceID)
	assert.NotNil(t, change.Before)
	assert.NotNil(t, change.After)
}
