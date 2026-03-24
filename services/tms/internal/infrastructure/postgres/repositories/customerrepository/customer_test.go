package customerrepository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestBulkUpdateStatusReturnsVersionMismatchWhenRowsAffectedZero(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	customerIDs := []pulid.ID{
		pulid.MustNew("cus_"),
		pulid.MustNew("cus_"),
	}

	mock.ExpectQuery(`UPDATE .*customers.*`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	updated, err := repo.BulkUpdateStatus(
		t.Context(),
		&repositories.BulkUpdateCustomerStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			CustomerIDs: customerIDs,
			Status:      domaintypes.StatusInactive,
		},
	)

	require.Nil(t, updated)
	require.Error(t, err)

	var valErr *errortypes.Error
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, "version", valErr.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, valErr.Code)
	assert.Contains(t, valErr.Message, customerIDs[0].String())
	assert.Contains(t, valErr.Message, customerIDs[1].String())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCustomerBillingProfileModelDoesNotPanicWhenInitializingM2MRelation(t *testing.T) {
	t.Parallel()

	repo, _ := newTestRepository(t)

	assert.NotPanics(t, func() {
		repo.db.DB().NewInsert().Model(&customer.CustomerBillingProfile{})
	})
}

func TestSaveBillingProfileSkipsWhenProfileMissing(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	repo.m2mSync = m2msync.NewSyncer(m2msync.SyncerParams{Logger: zap.NewNop()})

	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	err := repo.saveBillingProfile(t.Context(), repo.db.DB(), entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveEmailProfileSkipsWhenProfileMissing(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	err := repo.saveEmailProfile(t.Context(), repo.db.DB(), entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAddsDefaultBillingProfileWhenMissing(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	repo.m2mSync = m2msync.NewSyncer(m2msync.SyncerParams{Logger: zap.NewNop()})

	entity := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "CUST1",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "customers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(entity.ID.String()))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "customer_billing_profiles"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(pulid.MustNew("cbp_").String()))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM customer_billing_profile_document_types`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	created, err := repo.Create(t.Context(), entity)

	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, created.BillingProfile)
	assert.Equal(t, entity.ID, created.BillingProfile.CustomerID)
	assert.Equal(t, entity.OrganizationID, created.BillingProfile.OrganizationID)
	assert.Equal(t, entity.BusinessUnitID, created.BillingProfile.BusinessUnitID)
	require.NoError(t, mock.ExpectationsWereMet())
}
