//go:build integration

package editransferservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmentlinkrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmenttenderrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrecipientrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync/edisyncdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange/editenderchangedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type receivingShipments struct {
	shipmentserviceports.ShipmentReader
	shipmentserviceports.ShipmentWriter
	conn    *postgres.Connection
	tender  *editest.InternalTender
	created []pulid.ID
}

func (r *receivingShipments) Create(
	ctx context.Context,
	_ *shipment.Shipment,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	fixture := r.tender.TargetFixture
	entity := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    r.tender.Target.Info.BuID,
		OrganizationID:    r.tender.Target.Info.OrgID,
		ServiceTypeID:     fixture.ServiceType.ID,
		ShipmentTypeID:    fixture.ShipmentType.ID,
		CustomerID:        fixture.Customer.ID,
		FormulaTemplateID: fixture.FormulaTemplate.ID,
		Status:            shipmentvocab.StatusNew,
		TenderStatus:      new(shipmentvocab.TenderStatusAccepted),
		EntryMethod:       shipmentvocab.EntryMethodEDI,
		ProNumber:         "PRO-RECEIVED",
		BOL:               r.tender.SourceShipment.BOL,
	}
	if err := shipmentdb.Shipments.Create(ctx, r.conn.Keel(ctx), entity); err != nil {
		return nil, err
	}
	r.created = append(r.created, entity.ID)

	return entity, nil
}

type approvalHarness struct {
	tender    *editest.InternalTender
	service   *Service
	shipments *receivingShipments
	tenders   editenderports.EDIShipmentTenderRepository
	transfer  *editransfer.EDITransfer
	payload   *ApproveLoadTenderTransferWorkflowPayload
}

func newApprovalHarness(t *testing.T) *approvalHarness {
	t.Helper()

	tender := editest.SeedInternalTender(t)
	transfer := tender.NewTransfer(t, editransfer.TransferStatusProcessing)
	tender.NewRecipient(t, transfer)

	conn := dbtest.NewRLSConnection(t, tender.Admin, tender.DSN)
	logger := zap.NewNop()
	shipments := &receivingShipments{conn: conn, tender: tender}
	tenders := edishipmenttenderrepository.New(
		edishipmenttenderrepository.Params{DB: conn, Logger: logger},
	)
	partners := edipartnerrepository.New(
		edipartnerrepository.Params{DB: conn, Logger: logger},
	)
	mappings := edimappingprofilerepository.New(
		edimappingprofilerepository.Params{DB: conn, Logger: logger},
	)
	transfers := editransferrepository.New(
		editransferrepository.Params{DB: conn, Logger: logger},
	)
	recipients := editenderrecipient.New(editenderrecipient.Params{
		TenderRecipientRepo: editenderrecipientrepository.New(
			editenderrecipientrepository.Params{DB: conn, Logger: logger},
		),
	})
	alerts := edialert.New(edialert.Params{Logger: logger})
	service := New(Params{
		Logger:       logger,
		PartnerRepo:  partners,
		TransferRepo: transfers,
		ShipmentLinkRepo: edishipmentlinkrepository.New(
			edishipmentlinkrepository.Params{DB: conn, Logger: logger},
		),
		ShipmentSvc:   shipments,
		DB:            conn,
		Savepoints:    conn,
		Recipient:     recipients,
		TenderMapping: editendermapping.New(editendermapping.Params{MappingProfileRepo: mappings}),
		Shipments:     editendershipment.New(editendershipment.Params{ShipmentTenderRepo: tenders}),
		Setup: edisetup.New(edisetup.Params{
			Logger:             logger,
			PartnerRepo:        partners,
			MappingProfileRepo: mappings,
		}),
		Outbound: edioutbound.New(edioutbound.Params{
			Logger:       logger,
			PartnerRepo:  partners,
			TransferRepo: transfers,
			ShipmentSvc:  shipments,
			Alert:        alerts,
			Recipient:    recipients,
			Payload:      edipayloadbuild.New(edipayloadbuild.Params{ShipmentSvc: shipments}),
			Delivery: edidelivery.New(edidelivery.Params{
				Logger:      logger,
				PartnerRepo: partners,
				Alert:       alerts,
				Recipient:   recipients,
				Secret:      ediprofilesecret.New(ediprofilesecret.Params{}),
			}),
		}),
	})

	return &approvalHarness{
		tender:    tender,
		service:   service,
		shipments: shipments,
		tenders:   tenders,
		transfer:  transfer,
		payload: &ApproveLoadTenderTransferWorkflowPayload{
			TransferID: transfer.ID,
			TenantInfo: tender.Target.Info,
			Actor: testutil.NewSessionActor(
				tender.Target.Info.UserID,
				tender.Target.Info.OrgID,
				tender.Target.Info.BuID,
			),
		},
	}
}

func (h *approvalHarness) approve(t *testing.T) *ApproveLoadTenderTransferWorkflowResult {
	t.Helper()

	result, err := h.service.ProcessLoadTenderApproval(h.tender.Target.Ctx, h.payload)
	require.NoError(t, err)

	return result
}

func TestProcessLoadTenderApproval_AcceptsTheSourceTenderUnderRowLevelSecurity(t *testing.T) {
	h := newApprovalHarness(t)

	result := h.approve(t)

	require.Len(t, h.shipments.created, 1)
	assert.Equal(t, h.transfer.ID, result.TransferID)
	assert.Equal(t, h.shipments.created[0], result.TargetShipmentID)

	status := h.tender.SourceTenderStatus(t)
	require.NotNil(t, status)
	assert.Equal(
		t,
		shipmentvocab.TenderStatusAccepted,
		*status,
		"the receiving organization's approval accepts the sending organization's tender",
	)

	admin := h.tender.Admin
	stored, err := editransferdb.EDITransfers.Get(h.tender.Source.Ctx, admin, h.transfer.ID)
	require.NoError(t, err)
	assert.Equal(t, editransfer.TransferStatusApproved, stored.Status)
	assert.Equal(t, result.TargetShipmentID, stored.TargetShipmentID)

	sl := edisyncdb.ShipmentLinks
	links, err := sl.Where(sl.TenderTransferID.Eq(h.transfer.ID)).All(h.tender.Source.Ctx, admin)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, h.tender.SourceShipment.ID, links[0].SourceShipmentID)
	assert.Equal(t, result.TargetShipmentID, links[0].TargetShipmentID)

	tr := editenderchangedb.TenderRecipients
	recipients, err := tr.Where(tr.OriginalTransferID.Eq(h.transfer.ID)).
		All(h.tender.Source.Ctx, admin)
	require.NoError(t, err)
	require.Len(t, recipients, 1)
	assert.Equal(
		t,
		editenderchange.TenderRecipientBaselineStatusAccepted,
		recipients[0].BaselineStatus,
	)
	assert.Equal(t, links[0].ID, recipients[0].ShipmentLinkID)
}

func TestProcessLoadTenderApproval_ARetryAcceptsASourceTenderTheFirstRunLeftTendered(
	t *testing.T,
) {
	h := newApprovalHarness(t)
	first := h.approve(t)
	h.tender.SetSourceTenderStatus(t, shipmentvocab.TenderStatusTendered)

	retried := h.approve(t)

	assert.Equal(t, first.TargetShipmentID, retried.TargetShipmentID)
	assert.Len(t, h.shipments.created, 1, "a retry never creates a second receiving shipment")
	status := h.tender.SourceTenderStatus(t)
	require.NotNil(t, status)
	assert.Equal(t, shipmentvocab.TenderStatusAccepted, *status)

	again := h.approve(t)
	assert.Equal(t, first.TargetShipmentID, again.TargetShipmentID)
	assert.Len(t, h.shipments.created, 1)
}

func TestSetShipmentTenderStatus_FailsWhenTheScopeCannotSeeTheShipment(t *testing.T) {
	h := newApprovalHarness(t)

	err := h.tenders.SetShipmentTenderStatus(
		h.tender.Target.Ctx,
		editenderports.SetEDIShipmentTenderStatusRequest{
			ShipmentID: h.tender.SourceShipment.ID,
			TenantInfo: h.tender.Source.Info,
			Status:     shipmentvocab.TenderStatusAccepted,
		},
	)

	require.Error(t, err)
	assert.True(
		t,
		errortypes.IsNotFoundError(err),
		"an update that matches no row is not found: %v",
		err,
	)
	status := h.tender.SourceTenderStatus(t)
	require.NotNil(t, status)
	assert.Equal(t, shipmentvocab.TenderStatusTendered, *status)
}
