//go:build integration

package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmentlinkrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRejectTransferChange_CommentsOnBothOrganizationsShipmentsUnderRowLevelSecurity(
	t *testing.T,
) {
	tender := editest.SeedInternalTender(t)
	admin := tender.Admin
	seedtest.NewUser(tender.Source.Info.OrgID, tender.Source.Info.BuID).
		WithUsername(tenant.SystemUsername).
		WithEmail("system@example.com").
		Build(t, tender.Ctx, admin)

	transfer := tender.NewTransfer(t, edi.TransferStatusApproved)
	received := testutil.CreateShipmentGraph(
		t,
		tender.Ctx,
		admin,
		tender.TargetFixture,
		tender.Target.Info,
		testutil.ShipmentGraphParams{
			BOL:          "BOL-RECEIVED",
			ProNumber:    "PRO-RECEIVED",
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
		},
	).Shipment
	link := &edi.ShipmentLink{
		BusinessUnitID:       tender.Source.Info.BuID,
		SourceOrganizationID: tender.Source.Info.OrgID,
		TargetOrganizationID: tender.Target.Info.OrgID,
		SourceShipmentID:     tender.SourceShipment.ID,
		TargetShipmentID:     received.ID,
		TenderTransferID:     transfer.ID,
	}
	require.NoError(t, edidb.ShipmentLinks.Create(tender.Source.Ctx, admin, link))
	change := &edi.TransferChange{
		ShipmentLinkID: link.ID,
		Direction:      edi.TransferChangeDirectionSourceToTarget,
		ChangeType:     edi.TransferChangeTypeShipmentStatus214,
		IdempotencyKey: "status-" + link.ID.String(),
		Payload:        map[string]any{"newStatus": string(shipment.StatusInTransit)},
	}
	require.NoError(t, edidb.TransferChanges.Create(tender.Source.Ctx, admin, change))

	conn := dbtest.NewRLSConnection(t, admin, tender.DSN)
	logger := zap.NewNop()
	service := New(Params{
		Logger: logger,
		TransferChangeRepo: editransferchangerepository.New(
			editransferchangerepository.Params{DB: conn, Logger: logger},
		),
		ShipmentLinkRepo: edishipmentlinkrepository.New(
			edishipmentlinkrepository.Params{DB: conn, Logger: logger},
		),
		ShipmentCommentRepo: shipmentcommentrepository.New(
			shipmentcommentrepository.Params{DB: conn, Logger: logger},
		),
		UserRepo: userrepository.New(userrepository.Params{DB: conn, Logger: logger}),
		DB:       conn,
	})

	reviewed, err := service.RejectTransferChange(
		tender.Target.Ctx,
		&TransferChangeActionRequest{
			TenantInfo: tender.Target.Info,
			ChangeID:   change.ID,
			Reason:     "Driver has not departed",
		},
		testutil.NewSessionActor(
			tender.Target.Info.UserID,
			tender.Target.Info.OrgID,
			tender.Target.Info.BuID,
		),
	)

	require.NoError(t, err)
	assert.Equal(t, edi.TransferChangeStatusRejected, reviewed.Status)

	sc := shipmentdb.ShipmentComments
	for name, side := range map[string]struct {
		scope      editest.Tenant
		shipmentID pulid.ID
	}{
		"sender":   {scope: tender.Source, shipmentID: tender.SourceShipment.ID},
		"receiver": {scope: tender.Target, shipmentID: received.ID},
	} {
		comments, listErr := sc.Where(sc.ShipmentID.Eq(side.shipmentID)).
			All(side.scope.Ctx, admin)
		require.NoError(t, listErr, name)
		require.Len(t, comments, 1, name)
		assert.Equal(t, side.scope.Info.OrgID, comments[0].OrganizationID, name)
		assert.Equal(t, shipment.CommentSourceSystem, comments[0].Source, name)
	}
}
