package dataretentionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) (*Service, *mocks.MockDataRetentionRepository) {
	t.Helper()

	repo := mocks.NewMockDataRetentionRepository(t)

	return New(Params{Logger: zap.NewNop(), Repo: repo}), repo
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func intPtr(value int) *int {
	return &value
}

func upsertReturnsEntity(repo *mocks.MockDataRetentionRepository) *tenant.DataRetention {
	saved := &tenant.DataRetention{}
	repo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*tenant.DataRetention")).
		RunAndReturn(func(_ context.Context, entity *tenant.DataRetention) (*tenant.DataRetention, error) {
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
		Get(mock.Anything, repositories.GetDataRetentionRequest{
			UserID: info.UserID,
			OrgID:  info.OrgID,
			BuID:   info.BuID,
		}).
		Return(nil, errortypes.NewNotFoundError("data retention not found")).
		Once()

	entity, err := svc.Get(t.Context(), info)

	require.NoError(t, err)
	assert.Equal(t, tenant.DefaultAIAuditRetentionDays, entity.AIAuditRetentionPeriod)
}

func TestUpdate_SavesTheAIAuditTrailRetentionItWasGiven(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	saved := upsertReturnsEntity(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                   testTenant(),
		AuditRetentionPeriod:         120,
		AgentEvalCaseRetentionPeriod: intPtr(365),
		AIAuditRetentionPeriod:       intPtr(3650),
	})

	require.NoError(t, err)
	assert.Equal(t, 3650, saved.AIAuditRetentionPeriod)
	assert.Equal(t, 365, saved.AgentEvalCaseRetentionPeriod)
}

func TestUpdate_KeepsTheCurrentAIAuditTrailRetentionWhenNotSent(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	info := testTenant()
	repo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("repositories.GetDataRetentionRequest")).
		Return(&tenant.DataRetention{
			AuditRetentionPeriod:         120,
			AgentEvalCaseRetentionPeriod: 90,
			AIAuditRetentionPeriod:       1825,
		}, nil).
		Once()
	saved := upsertReturnsEntity(repo)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:           info,
		AuditRetentionPeriod: 120,
	})

	require.NoError(t, err)
	assert.Equal(t, 1825, saved.AIAuditRetentionPeriod)
	assert.Equal(t, 90, saved.AgentEvalCaseRetentionPeriod)
}

func TestUpdate_RefusesAnAIAuditTrailKeptUnderAYear(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	_, err := svc.Update(t.Context(), &UpdateDataRetentionRequest{
		TenantInfo:                   testTenant(),
		AuditRetentionPeriod:         120,
		AgentEvalCaseRetentionPeriod: intPtr(365),
		AIAuditRetentionPeriod:       intPtr(30),
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "aiAuditRetentionPeriod", multiErr.Errors[0].Field)
}
