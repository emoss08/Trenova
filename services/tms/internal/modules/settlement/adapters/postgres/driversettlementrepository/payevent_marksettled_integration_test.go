//go:build integration

package driversettlementrepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driversettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type shipmentRow struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

// Two settlements built at once for one driver read the same accrued pay
// events. The second to mark them settled must fail rather than pay the
// driver for the same loads twice.
func TestMarkSettled_RefusesEventsAnotherSettlementTook_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	shipments, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowToStructByPos[shipmentRow],
		"SELECT id, organization_id, business_unit_id FROM shipments LIMIT 1",
	)
	require.NoError(t, err)
	require.Len(t, shipments, 1)
	tenant := pagination.TenantInfo{OrgID: shipments[0].OrganizationID, BuID: shipments[0].BusinessUnitID}
	workerID, err := keel.QueryValue[pulid.ID](
		ctx,
		db,
		"SELECT id FROM workers WHERE organization_id = $1 AND business_unit_id = $2 LIMIT 1",
		tenant.OrgID.String(),
		tenant.BuID.String(),
	)
	require.NoError(t, err)
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	newSettlement := func(number string) *driversettlement.Settlement {
		settlement := &driversettlement.Settlement{
			ID:               pulid.MustNew("dstl_"),
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			WorkerID:         workerID,
			SettlementNumber: number,
			Status:           driversettlement.StatusDraft,
			Classification:   driverpay.PayeeClassificationCompanyDriver,
			PeriodStart:      1_790_000_000,
			PeriodEnd:        1_790_600_000,
			PayDate:          1_790_900_000,
			CurrencyCode:     "USD",
		}
		require.NoError(t, driversettlementdb.Settlements.Create(ctx, db, settlement))
		return settlement
	}
	newEvent := func() *driversettlement.PayEvent {
		event := &driversettlement.PayEvent{
			ID:               pulid.MustNew("dpe_"),
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			WorkerID:         workerID,
			ShipmentID:       shipments[0].ID,
			IdempotencyKey:   pulid.MustNew("idem_").String(),
			Status:           driversettlement.PayEventStatusAccrued,
			EventDate:        1_790_100_000,
			GrossAmountMinor: 42500,
			CurrencyCode:     "USD",
		}
		require.NoError(t, driversettlementdb.PayEvents.Create(ctx, db, event))
		return event
	}

	first, second := newSettlement("DS-RACE-1"), newSettlement("DS-RACE-2")
	taken, also, untouched := newEvent(), newEvent(), newEvent()

	repo := driversettlementrepository.NewPayEvent(driversettlementrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	require.NoError(t, repo.MarkSettled(ctx, tenant, []pulid.ID{taken.ID, also.ID, taken.ID}, first.ID))

	err = repo.MarkSettled(ctx, tenant, []pulid.ID{taken.ID, untouched.ID}, second.ID)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "a lost race is the caller's to retry: %v", err)

	status, err := keel.QueryValue[string](
		ctx,
		db,
		"SELECT status FROM driver_pay_events WHERE id = $1",
		untouched.ID.String(),
	)
	require.NoError(t, err)
	assert.Equal(t, string(driversettlement.PayEventStatusAccrued), status,
		"a refused mark settles none of the events it named")
	owner, err := keel.QueryValue[string](
		ctx,
		db,
		"SELECT settlement_id FROM driver_pay_events WHERE id = $1",
		taken.ID.String(),
	)
	require.NoError(t, err)
	assert.Equal(t, first.ID.String(), owner)
}
