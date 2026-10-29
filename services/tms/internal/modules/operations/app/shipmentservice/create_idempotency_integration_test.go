//go:build integration

package shipmentservice

import (
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentwrite"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIntegration_IdempotencyKeyBooksOneShipment(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = false
			sc.CheckForDuplicateBOLs = false
		},
	)
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)
	key := idempotency.NewHasher("test-secret").ScopedKey(idempotency.Scope{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PrincipalType:  "user",
		PrincipalID:    data.User.ID,
	}, "create-shipment-1")

	first := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	first.BOL = "BOL-IDEMPOTENT-1"
	first.IdempotencyKey = key
	created, err := svc.Create(ctx, first, actor)
	require.NoError(t, err)

	retry := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	retry.BOL = "BOL-IDEMPOTENT-1"
	retry.IdempotencyKey = key
	replayed, err := svc.Create(ctx, retry, actor)
	require.NoError(t, err)
	assert.Equal(t, created.ID, replayed.ID)
	assert.Equal(t, created.ProNumber, replayed.ProNumber)
	assert.NotEmpty(t, replayed.Moves, "the replay returns the shipment with its details")

	count, err := keel.QueryValue[int64](ctx, db,
		"SELECT count(*) FROM shipments WHERE organization_id = $1 AND bol = $2",
		tenantInfo.OrgID.String(), "BOL-IDEMPOTENT-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	created.BOL = "BOL-IDEMPOTENT-EDITED"
	created.IdempotencyKey = ""
	_, err = svc.Update(ctx, created, actor)
	require.NoError(t, err)
	stored, err := keel.QueryValue[string](ctx, db,
		"SELECT idempotency_key FROM shipments WHERE id = $1", created.ID.String())
	require.NoError(t, err)
	assert.Equal(t, key, stored, "an edit keeps the key the shipment was booked under")

	err = shipmentdb.Shipments.Create(ctx, db, &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     created.ServiceTypeID,
		ShipmentTypeID:    created.ShipmentTypeID,
		CustomerID:        created.CustomerID,
		FormulaTemplateID: created.FormulaTemplateID,
		ProNumber:         "PRO-RACE-IDEMPOTENT",
		IdempotencyKey:    key,
		Status:            shipment.StatusNew,
	})
	require.Error(t, err, "the unique index holds when the lookup is raced")
	assert.True(t, shipmentwrite.IsIdempotencyKeyConflict(err))
	assert.False(t, shipmentwrite.IsIdempotencyKeyConflict(errors.New("unrelated")))

	err = shipmentdb.Shipments.Create(ctx, db, &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     created.ServiceTypeID,
		ShipmentTypeID:    created.ShipmentTypeID,
		CustomerID:        created.CustomerID,
		FormulaTemplateID: created.FormulaTemplateID,
		ProNumber:         "PRO-OTHER-IDEMPOTENT",
		IdempotencyKey: idempotency.NewHasher("test-secret").ScopedKey(idempotency.Scope{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			PrincipalType:  "user",
			PrincipalID:    data.User.ID,
		}, "create-shipment-2"),
		Status: shipment.StatusNew,
	})
	require.NoError(t, err, "another key books another shipment")
}
