//go:build integration

package driversettlementrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driversettlementrepository"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMarkSettledRefusesAnEventItDidNotMove_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	orgID, err := keel.QueryValue[pulid.ID](ctx, db, "SELECT id FROM organizations LIMIT 1")
	require.NoError(t, err)
	buID, err := keel.QueryValue[pulid.ID](ctx, db, "SELECT id FROM business_units LIMIT 1")
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	repo := driversettlementrepository.NewPayEvent(driversettlementrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})

	err = repo.MarkSettled(
		ctx,
		tenant,
		[]pulid.ID{pulid.MustNew("dpe_")},
		pulid.MustNew("dset_"),
	)

	assert.True(t, errortypes.IsConflictError(err), "got %v", err)
}
