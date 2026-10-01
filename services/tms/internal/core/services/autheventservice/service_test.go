package autheventservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepository struct {
	events []*iam.AuthEvent
	scopes []dbscope.Scope
	ctxErr []error
	err    error
}

func (f *fakeRepository) Create(ctx context.Context, event *iam.AuthEvent) error {
	f.events = append(f.events, event)
	f.scopes = append(f.scopes, dbscope.From(ctx))
	f.ctxErr = append(f.ctxErr, ctx.Err())
	return f.err
}

func (f *fakeRepository) DeleteBefore(context.Context, int64) (int64, error) {
	return 0, nil
}

func newTestService(repo *fakeRepository) services.AuthEventRecorder {
	return New(Params{
		Repository: repo,
		Metrics:    &metrics.Registry{Audit: metrics.NewAudit(nil, zap.NewNop(), false)},
		Logger:     zap.NewNop(),
	})
}

func TestRecordBindsTheEventTenantAndRequest(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	srv := newTestService(repo)

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	ctx := requestmeta.With(t.Context(), requestmeta.New("req-1", "203.0.113.5", "browser"))
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	srv.Record(ctx, &services.AuthEventRecord{
		Provider:       services.AuthEventProviderPassword,
		Outcome:        iam.AuthEventOutcomeSuccess,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.Len(t, repo.events, 1)
	event := repo.events[0]
	assert.Equal(t, orgID, event.OrganizationID)
	assert.Equal(t, buID, event.BusinessUnitID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "203.0.113.5", event.IPAddress)
	assert.Equal(t, "browser", event.UserAgent)
	assert.Equal(t, 1, event.AuthenticatorAAL)
	assert.Equal(t, 1, event.FederationFAL)
	assert.Equal(t, iam.RiskOutcomeAllow, event.RiskOutcome)
	assert.NotZero(t, event.OccurredAt)
	assert.NoError(t, repo.ctxErr[0], "a cancelled request must not drop the event")

	tenant, ok := repo.scopes[0].Tenant()
	require.True(t, ok)
	assert.Equal(t, orgID, tenant.OrganizationID)
	assert.Equal(t, buID, tenant.BusinessUnitID)
}

func TestRecordWithoutATenantLeavesTheScopeToTheRepository(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	srv := newTestService(repo)

	srv.Record(t.Context(), &services.AuthEventRecord{
		Provider:       services.AuthEventProviderPassword,
		Outcome:        "bogus",
		OrganizationID: pulid.MustNew("org_"),
		ErrorCode:      strings.Repeat("x", 200),
	})

	require.Len(t, repo.events, 1)
	event := repo.events[0]
	assert.True(t, event.OrganizationID.IsNil(), "an organization without a business unit is not a tenant")
	assert.Equal(t, iam.AuthEventOutcomeFailed, event.Outcome)
	assert.Len(t, event.ErrorCode, maxErrorCodeLen)
	assert.Empty(t, event.IPAddress)
	_, isTenant := repo.scopes[0].Tenant()
	assert.False(t, isTenant)
}

func TestRecordDropsAnUnparseableClientIP(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	srv := newTestService(repo)

	ctx := requestmeta.With(t.Context(), requestmeta.New("req", "not-an-ip", "agent"))
	srv.Record(ctx, &services.AuthEventRecord{Outcome: iam.AuthEventOutcomeDenied})

	require.Len(t, repo.events, 1)
	assert.Empty(t, repo.events[0].IPAddress)
	assert.Equal(t, "agent", repo.events[0].UserAgent)
}

func TestRecordSwallowsRepositoryFailures(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{err: errors.New("database unavailable")}
	srv := newTestService(repo)

	assert.NotPanics(t, func() {
		srv.Record(t.Context(), &services.AuthEventRecord{Outcome: iam.AuthEventOutcomeFailed})
	})
	assert.Len(t, repo.events, 1)
}
