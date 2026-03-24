package shipmentholdrepository

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
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

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestListByShipmentID_ReturnsHolds(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	holdID := pulid.MustNew("shh_")
	reasonID := pulid.MustNew("hr_")
	userID := pulid.MustNew("usr_")

	mock.ExpectQuery(`SELECT count\(\*\) FROM "shipment_holds" AS "shh".*shipment_id = .*organization_id = .*business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .* FROM "shipment_holds" AS "shh".*released_at IS NULL DESC.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "shipment_id", "business_unit_id", "organization_id", "hold_reason_id", "type", "severity", "reason_code", "notes", "source", "blocks_dispatch", "blocks_delivery", "blocks_billing", "visible_to_customer", "started_at", "created_at", "updated_at", "version", "released_at", "created_by_id", "released_by_id",
			"hold_reason__id", "hold_reason__business_unit_id", "hold_reason__organization_id", "hold_reason__type", "hold_reason__code", "hold_reason__label", "hold_reason__active", "hold_reason__default_severity", "hold_reason__default_blocks_dispatch", "hold_reason__default_blocks_delivery", "hold_reason__default_blocks_billing", "hold_reason__default_visible_to_customer", "hold_reason__sort_order", "hold_reason__version", "hold_reason__created_at", "hold_reason__updated_at",
			"created_by__id", "created_by__business_unit_id", "created_by__current_organization_id", "created_by__status", "created_by__name", "created_by__username", "created_by__time_format", "created_by__password", "created_by__email_address", "created_by__profile_pic_url", "created_by__thumbnail_url", "created_by__timezone", "created_by__is_locked", "created_by__must_change_password", "created_by__is_platform_admin", "created_by__version", "created_by__created_at", "created_by__updated_at", "created_by__last_login_at",
		}).AddRow(holdID, shipmentID, buID, orgID, reasonID, "OperationalHold", "Blocking", "APPT", "dock issue", "User", true, false, false, false, 10, 10, 10, 0, nil, userID, nil, reasonID, buID, orgID, "OperationalHold", "APPT", "Appointment", true, "Blocking", true, false, false, false, 100, 0, 10, 10, userID, buID, orgID, "Active", "Alice", "alice", "12-hour", "secret", "a@example.com", "", "", "UTC", false, false, false, 0, 1, 1, nil))

	result, err := repo.ListByShipmentID(t.Context(), &repositories.ListShipmentHoldsRequest{
		ShipmentID: shipmentID,
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			Pagination: pagination.Info{Limit: 20},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, holdID, result.Items[0].ID)
	assert.Equal(t, 1, result.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DuplicateActiveTypeMapsBusinessError(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &shipment.ShipmentHold{
		ID:             pulid.MustNew("shh_"),
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Type:           holdreason.HoldTypeOperational,
		Severity:       holdreason.HoldSeverityBlocking,
		ReasonCode:     "APPT",
		Source:         shipment.HoldSourceUser,
		BlocksDispatch: true,
		StartedAt:      1,
	}

	mock.ExpectQuery(`INSERT INTO "shipment_holds".*`).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "ux_shipment_holds_active_by_type"})

	created, err := repo.Create(t.Context(), entity)

	require.Error(t, err)
	assert.Nil(t, created)
	assert.True(t, errortypes.IsBusinessError(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRelease_UpdatesHold(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &shipment.ShipmentHold{
		ID:             pulid.MustNew("shh_"),
		ShipmentID:     pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	releasedAt := int64(99)
	releasedBy := pulid.MustNew("usr_")
	entity.ReleasedAt = &releasedAt
	entity.ReleasedByID = &releasedBy

	mock.ExpectExec(`UPDATE "shipment_holds" AS "shh".*released_at = .*released_by_id = .*released_at IS NULL`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM "shipment_holds" AS "shh".*shh\.id = .*shh\.shipment_id = .*shh\.organization_id = .*shh\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "shipment_id", "business_unit_id", "organization_id", "hold_reason_id", "type", "severity", "reason_code", "notes", "source", "blocks_dispatch", "blocks_delivery", "blocks_billing", "visible_to_customer", "started_at", "created_at", "updated_at", "version", "released_at", "created_by_id", "released_by_id",
		}).AddRow(entity.ID, entity.ShipmentID, entity.BusinessUnitID, entity.OrganizationID, nil, "OperationalHold", "Blocking", "APPT", "", "User", true, false, false, false, 1, 1, 1, 1, releasedAt, releasedBy, releasedBy))

	released, err := repo.Release(t.Context(), entity)

	require.NoError(t, err)
	require.NotNil(t, released)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHasActiveDispatchHold_ReturnsTrue(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	mock.ExpectQuery(fmt.Sprintf(`SELECT count\(\*\) FROM "shipment_holds" AS "shh".*shipment_id = .*organization_id = .*business_unit_id = .*%s.*`, "blocks_dispatch")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	hasHold, err := repo.HasActiveDispatchHold(t.Context(), &repositories.ActiveShipmentHoldRequest{
		ShipmentID: shipmentID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})

	require.NoError(t, err)
	assert.True(t, hasHold)
	require.NoError(t, mock.ExpectationsWereMet())
}
