//go:build integration

package editest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
)

type InternalTender struct {
	Ctx            context.Context
	Admin          *keel.DB
	DSN            string
	StateID        pulid.ID
	Source         Tenant
	Target         Tenant
	SourcePartner  *edi.EDIPartner
	TargetPartner  *edi.EDIPartner
	SourceShipment *shipment.Shipment
	TargetFixture  *testutil.ShipmentIntegrationFixture
}

func SeedInternalTender(t *testing.T) *InternalTender {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	sourceData := seedtest.SeedFullTestData(t, ctx, admin)
	targetOrg := seedtest.NewOrganization(sourceData.BusinessUnit.ID, sourceData.State.ID).
		WithName("Receiving Carrier").
		WithScacCode("RCVC").
		WithDOTNumber("9990042").
		WithBucketName("test-bucket-rcvc").
		Build(t, ctx, admin)
	targetUser := seedtest.NewUser(targetOrg.ID, sourceData.BusinessUnit.ID).
		WithUsername("receivingdispatcher").
		WithEmail("receiving-dispatcher@example.com").
		WithPassword("password123").
		Build(t, ctx, admin)
	targetData := &seedtest.TestData{
		BusinessUnit: sourceData.BusinessUnit,
		Organization: targetOrg,
		User:         targetUser,
		State:        sourceData.State,
	}

	tender := &InternalTender{
		Ctx:     ctx,
		Admin:   admin,
		DSN:     dsn,
		StateID: sourceData.State.ID,
		Source:  TenantOf(ctx, sourceData.Info()),
		Target:  TenantOf(ctx, targetData.Info()),
	}
	sourceFixture := testutil.SeedShipmentIntegrationFixture(
		t,
		ctx,
		admin,
		sourceData,
		tender.Source.Info,
	)
	tender.TargetFixture = testutil.SeedShipmentIntegrationFixture(
		t,
		ctx,
		admin,
		targetData,
		tender.Target.Info,
	)
	tender.SourceShipment = testutil.CreateShipmentGraph(
		t,
		ctx,
		admin,
		sourceFixture,
		tender.Source.Info,
		testutil.ShipmentGraphParams{
			BOL:          "BOL-TENDER",
			ProNumber:    "PRO-TENDER",
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
		},
	).Shipment
	tender.SetSourceTenderStatus(t, shipment.TenderStatusTendered)

	tender.SourcePartner = internalPartner(t, tender.Source, admin, "RCVC", targetOrg.ID)
	tender.TargetPartner = internalPartner(
		t,
		tender.Target,
		admin,
		"SHIPPER",
		sourceData.Organization.ID,
	)

	return tender
}

func (it *InternalTender) SetSourceTenderStatus(t *testing.T, status shipment.TenderStatus) {
	t.Helper()

	sp := shipmentdb.Shipments
	n, err := sp.Where(sp.ID.Eq(it.SourceShipment.ID)).
		Update(it.Source.Ctx, it.Admin, sp.TenderStatus.To(status))
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func (it *InternalTender) SourceTenderStatus(t *testing.T) *shipment.TenderStatus {
	t.Helper()

	stored, err := shipmentdb.Shipments.Get(it.Source.Ctx, it.Admin, it.SourceShipment.ID)
	require.NoError(t, err)

	return stored.TenderStatus
}

func (it *InternalTender) NewTransfer(t *testing.T, status edi.TransferStatus) *edi.EDITransfer {
	t.Helper()

	transfer := &edi.EDITransfer{
		SourceOrganizationID: it.Source.Info.OrgID,
		SourceBusinessUnitID: it.Source.Info.BuID,
		TargetOrganizationID: it.Target.Info.OrgID,
		TargetBusinessUnitID: it.Target.Info.BuID,
		SourcePartnerID:      it.SourcePartner.ID,
		TargetPartnerID:      it.TargetPartner.ID,
		SourceShipmentID:     it.SourceShipment.ID,
		Status:               status,
		TenderPayload: edi.LoadTenderPayload{
			ShipmentID:     it.SourceShipment.ID,
			BusinessUnitID: it.Source.Info.BuID,
			OrganizationID: it.Source.Info.OrgID,
			BOL:            it.SourceShipment.BOL,
		},
		SubmittedByID: it.Source.Info.UserID,
		SubmittedAt:   timeutils.NowUnix(),
	}
	require.NoError(t, edidb.EDITransfers.Create(it.Source.Ctx, it.Admin, transfer))

	return transfer
}

func (it *InternalTender) Recipient(transfer *edi.EDITransfer) *edi.TenderRecipient {
	return &edi.TenderRecipient{
		BusinessUnitID:          transfer.SourceBusinessUnitID,
		SourceOrganizationID:    transfer.SourceOrganizationID,
		SourceShipmentID:        transfer.SourceShipmentID,
		RecipientKind:           edi.TenderRecipientKindInternal,
		RecipientOrganizationID: transfer.TargetOrganizationID,
		EDIPartnerID:            transfer.TargetPartnerID,
		OriginalTransferID:      transfer.ID,
		LatestBaselinePayload:   transfer.TenderPayload,
		LatestBaselineHash:      transfer.ID.String(),
		BaselineRecordedAt:      timeutils.NowUnix(),
		BaselineStatus:          edi.TenderRecipientBaselineStatusSent,
		Status:                  edi.TenderRecipientStatusActive,
	}
}

func (it *InternalTender) NewRecipient(t *testing.T, transfer *edi.EDITransfer) *edi.TenderRecipient {
	t.Helper()

	recipient := it.Recipient(transfer)
	require.NoError(t, edidb.TenderRecipients.Create(it.Source.Ctx, it.Admin, recipient))

	return recipient
}

func internalPartner(
	t *testing.T,
	owner Tenant,
	db *keel.DB,
	code string,
	counterparty pulid.ID,
) *edi.EDIPartner {
	t.Helper()

	partner := &edi.EDIPartner{
		Kind:                   edi.PartnerKindInternal,
		Code:                   code,
		Name:                   code + " internal partner",
		InternalOrganizationID: counterparty,
		EnabledForInbound:      true,
		EnabledForOutbound:     true,
	}
	require.NoError(t, edidb.EDIPartners.Create(owner.Ctx, db, partner))

	return partner
}
