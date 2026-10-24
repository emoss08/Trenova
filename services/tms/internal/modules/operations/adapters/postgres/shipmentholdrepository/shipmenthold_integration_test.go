//go:build integration

package shipmentholdrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentholdrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx      context.Context
	repo     shipmentports.ShipmentHoldRepository
	tenant   pagination.TenantInfo
	entity   *shipment.Shipment
	otherCtx context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-HOLD",
		ProNumber:    "PRO-HOLD",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	return &harness{
		ctx: data.Ctx(ctx),
		repo: shipmentholdrepository.New(shipmentholdrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant:   tenant,
		entity:   graph.Shipment,
		otherCtx: seedtest.SeedAdditionalTenant(t, ctx, db, "OT").Ctx(ctx),
	}
}

func (h *harness) hold(severity holdreason.HoldSeverity) *shipment.ShipmentHold {
	return &shipment.ShipmentHold{
		ShipmentID:     h.entity.ID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Type:           holdreason.HoldTypeOperational,
		Severity:       severity,
		BlocksDispatch: true,
		StartedAt:      timeutils.NowUnix(),
	}
}

func TestCreate_StoresTheHoldWithItsDefaults(t *testing.T) {
	h := newHarness(t)
	hold := h.hold("")

	created, err := h.repo.Create(h.ctx, hold)
	require.NoError(t, err)
	assert.False(t, created.ID.IsNil())
	assert.Equal(t, holdreason.HoldSeverityAdvisory, created.Severity)
	assert.Equal(t, shipment.HoldSourceUser, created.Source)

	_, err = h.repo.Create(h.ctx, h.hold(holdreason.HoldSeverityBlocking))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err), "one active hold of a type per shipment")
}

func TestListAndBlockingHolds(t *testing.T) {
	h := newHarness(t)
	created, err := h.repo.Create(h.ctx, h.hold(holdreason.HoldSeverityBlocking))
	require.NoError(t, err)

	list, err := h.repo.ListByShipmentID(h.ctx, &shipmentports.ListShipmentHoldsRequest{
		Filter:     &pagination.QueryOptions{TenantInfo: h.tenant},
		ShipmentID: h.entity.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	assert.Equal(t, created.ID, list.Items[0].ID)

	active := &shipmentports.ActiveShipmentHoldRequest{TenantInfo: h.tenant, ShipmentID: h.entity.ID}
	blocked, err := h.repo.HasActiveDispatchHold(h.ctx, active)
	require.NoError(t, err)
	assert.True(t, blocked)
	delivery, err := h.repo.HasActiveDeliveryHold(h.ctx, active)
	require.NoError(t, err)
	assert.False(t, delivery)

	otherBlocked, err := h.repo.HasActiveDispatchHold(h.otherCtx, active)
	require.NoError(t, err)
	assert.False(t, otherBlocked, "another tenant never sees the hold")
}

func TestUpdateAndRelease(t *testing.T) {
	h := newHarness(t)
	created, err := h.repo.Create(h.ctx, h.hold(holdreason.HoldSeverityBlocking))
	require.NoError(t, err)

	created.Notes = "Waiting on paperwork"
	updated, err := h.repo.Update(h.ctx, created)
	require.NoError(t, err)
	assert.Equal(t, "Waiting on paperwork", updated.Notes)
	assert.Equal(t, created.Version+1, updated.Version)

	_, err = h.repo.Update(h.ctx, created)
	require.Error(t, err, "a stale version is a conflict")

	releasedAt := timeutils.NowUnix()
	updated.ReleasedAt = &releasedAt
	released, err := h.repo.Release(h.ctx, updated)
	require.NoError(t, err)
	require.NotNil(t, released.ReleasedAt)

	_, err = h.repo.Release(h.ctx, updated)
	require.Error(t, err, "a released hold cannot be released again")
}
