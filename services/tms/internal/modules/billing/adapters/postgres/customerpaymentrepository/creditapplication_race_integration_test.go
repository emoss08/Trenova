//go:build integration

package customerpaymentrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUnapplyingAnApplicationTwiceConflicts_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	orgID, err := keel.QueryValue[pulid.ID](ctx, db, "SELECT id FROM organizations LIMIT 1")
	require.NoError(t, err)
	buID, err := keel.QueryValue[pulid.ID](ctx, db, "SELECT id FROM business_units LIMIT 1")
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	repo := customerpaymentrepository.New(customerpaymentrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})
	unappliedAt := int64(1_790_000_000)

	_, err = repo.UpdateCreditMemoApplication(ctx, &customerpayment.CreditMemoApplication{
		ID:              pulid.MustNew("cma_"),
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		Status:          customerpayment.CreditApplicationStatusUnapplied,
		UnappliedAt:     &unappliedAt,
		UnappliedByID:   pulid.MustNew("usr_"),
		UnappliedReason: "applied to the wrong invoice",
	})

	assert.True(t, errortypes.IsConflictError(err), "got %v", err)
}
