//go:build integration

package shipmentservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
		sc.CheckForDuplicateBOLs = false
	})
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)
	key := idempotency.ScopedKey(idempotency.Scope{
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

	count, err := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("bol = ?", "BOL-IDEMPOTENT-1").
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	created.BOL = "BOL-IDEMPOTENT-EDITED"
	created.IdempotencyKey = ""
	_, err = svc.Update(ctx, created, actor)
	require.NoError(t, err)
	var stored string
	err = db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Column("idempotency_key").
		Where("id = ?", created.ID).
		Scan(ctx, &stored)
	require.NoError(t, err)
	assert.Equal(t, key, stored, "an edit keeps the key the shipment was booked under")

	_, err = db.NewInsert().Model(&shipment.Shipment{
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
	}).Exec(ctx)
	require.Error(t, err, "the unique index holds when the lookup is raced")
	assert.True(t, isIdempotencyKeyConflict(err))
	assert.False(t, isIdempotencyKeyConflict(errors.New("unrelated")))

	_, err = db.NewInsert().Model(&shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     created.ServiceTypeID,
		ShipmentTypeID:    created.ShipmentTypeID,
		CustomerID:        created.CustomerID,
		FormulaTemplateID: created.FormulaTemplateID,
		ProNumber:         "PRO-OTHER-IDEMPOTENT",
		IdempotencyKey: idempotency.ScopedKey(idempotency.Scope{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			PrincipalType:  "user",
			PrincipalID:    data.User.ID,
		}, "create-shipment-2"),
		Status: shipment.StatusNew,
	}).Exec(ctx)
	require.NoError(t, err, "another key books another shipment")
}
