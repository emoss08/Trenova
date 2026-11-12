//go:build integration

package workqueuerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testKind = "test-kind"

// A subject requested again while its work runs stays queued after the
// settle, even within the same second; settling what was claimed clears it.
func TestQueueKeepsASubjectRequestedWhileItsWorkRan(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	queue := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	subject := pulid.MustNew("shp_")
	const now = int64(1_000)

	enqueue := func() {
		t.Helper()
		require.NoError(t, queue.Enqueue(ctx, &workqueueports.EnqueueRequest{
			Kind: testKind, TenantInfo: tenant, SubjectIDs: []pulid.ID{subject, subject}, At: now,
		}))
	}
	due := func() []workqueueports.Due {
		t.Helper()
		items, err := queue.ListDue(ctx, &workqueueports.ListDueRequest{
			Kind: testKind, TenantInfo: tenant, Now: now + 10, Limit: 10,
		})
		require.NoError(t, err)

		return items
	}

	enqueue()
	claimed := due()
	require.Len(t, claimed, 1, "a subject queued twice is one row")

	enqueue()
	require.NoError(t, queue.SettleMany(ctx, &workqueueports.SettleManyRequest{
		Kind: testKind, TenantInfo: tenant, Done: claimed,
	}))
	left := due()
	require.Len(t, left, 1, "the request made meanwhile keeps the subject queued")
	assert.Greater(t, left[0].DueAt, claimed[0].DueAt)

	tenants, err := queue.ListDueTenants(ctx, &workqueueports.ListDueTenantsRequest{
		Kind: testKind, Now: now + 10, Limit: 10,
	})
	require.NoError(t, err)
	assert.Contains(t, tenants, tenant)

	require.NoError(t, queue.SettleMany(ctx, &workqueueports.SettleManyRequest{
		Kind: testKind, TenantInfo: tenant, Done: left,
	}))
	assert.Empty(t, due(), "settling what was claimed clears it")
}

// A failed subject waits out its retry.
func TestQueuePushesAFailedSubjectBack(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	queue := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	subject := pulid.MustNew("shp_")

	require.NoError(t, queue.Enqueue(ctx, &workqueueports.EnqueueRequest{
		Kind: testKind, TenantInfo: tenant, SubjectIDs: []pulid.ID{subject}, At: 1_000,
	}))
	require.NoError(t, queue.SettleMany(ctx, &workqueueports.SettleManyRequest{
		Kind:       testKind,
		TenantInfo: tenant,
		Retry:      []workqueueports.Due{{SubjectID: subject, DueAt: 1_000}},
		RetryAt:    2_000,
	}))

	early, err := queue.ListDue(ctx, &workqueueports.ListDueRequest{
		Kind: testKind, TenantInfo: tenant, Now: 1_500, Limit: 10,
	})
	require.NoError(t, err)
	assert.Empty(t, early)

	later, err := queue.ListDue(ctx, &workqueueports.ListDueRequest{
		Kind: testKind, TenantInfo: tenant, Now: 2_000, Limit: 10,
	})
	require.NoError(t, err)
	assert.Len(t, later, 1)
}

// A subject's detail is the latest request's, and the backstop lists the
// tenant and kind pairs under a prefix that have due work.
func TestQueueKeepsTheLatestDetailAndListsDueKinds(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	queue := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	subject := pulid.MustNew("shp_")
	kind := "agent-event:" + pulid.MustNew("agd_").String()

	for _, detail := range []string{"shipment.updated", "shipment.pod_missing"} {
		require.NoError(t, queue.Enqueue(ctx, &workqueueports.EnqueueRequest{
			Kind: kind, TenantInfo: tenant, SubjectIDs: []pulid.ID{subject}, At: 1_000, Detail: detail,
		}))
	}

	due, err := queue.ListDue(ctx, &workqueueports.ListDueRequest{
		Kind: kind, TenantInfo: tenant, Now: 5_000, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, "shipment.pod_missing", due[0].Detail, "the latest request's detail wins")

	kinds, err := queue.ListDueKinds(ctx, &workqueueports.ListDueKindsRequest{
		Prefix: "agent-event:", Now: 5_000, Limit: 100,
	})
	require.NoError(t, err)
	assert.Contains(t, kinds, workqueueports.TenantKind{TenantInfo: tenant, Kind: kind})
}
