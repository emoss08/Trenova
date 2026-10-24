//go:build integration

package editenderrepairrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrepairrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrepairservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	misreadTransaction = "ST*204*0001~B2**ACME**LT-88240**PP~B2A*00~SE*4*0001~"
	legacyTransaction  = "ST*204*0002~B2*ABCD*shp_01JLEGACY**PP~B2A*00~SE*4*0002~"
	interchange        = "ISA*00*          *00*          *ZZ*ACMEFOODS      *ZZ*TRENOVA        " +
		"*261013*0912*^*00401*000000317*0*P*>~" +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~" +
		misreadTransaction + legacyTransaction +
		"GE*2*317~IEA*1*000000317~"
)

type tenderFixture struct {
	orgID      pulid.ID
	buID       pulid.ID
	misread    pulid.ID
	legacy     pulid.ID
	purged     pulid.ID
	transfer   pulid.ID
	edited     pulid.ID
	shipment   pulid.ID
	untouched  pulid.ID
	documentID pulid.ID
	partnerID  pulid.ID
}

func seedTenders(t *testing.T) (*tenderFixture, *bun.DB, *postgres.Connection) {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var shipments []struct {
		ID             pulid.ID `bun:"id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("shipments").
		Column("id", "organization_id", "business_unit_id").
		Order("id").
		Limit(2).
		Scan(ctx, &shipments))
	require.Len(t, shipments, 2)
	fixture := &tenderFixture{
		orgID:     shipments[0].OrganizationID,
		buID:      shipments[0].BusinessUnitID,
		shipment:  shipments[0].ID,
		untouched: shipments[1].ID,
	}

	var documentType struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("id").
		Where("transaction_set = ?", edi.TransactionSet204).
		Limit(1).
		Scan(ctx, &documentType))
	fixture.documentID = documentType.ID

	partner := &edi.EDIPartner{
		BusinessUnitID: fixture.buID,
		OrganizationID: fixture.orgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "ACME-REPAIR",
		Name:           "Acme Repair Partner",
	}
	_, err := db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)
	fixture.partnerID = partner.ID

	profile := &edi.EDICommunicationProfile{
		BusinessUnitID: fixture.buID,
		OrganizationID: fixture.orgID,
		EDIPartnerID:   partner.ID,
		Method:         edi.ConnectionMethodSFTP,
		Name:           "Acme Repair Mailbox",
	}
	_, err = db.NewInsert().Model(profile).Exec(ctx)
	require.NoError(t, err)

	now := timeutils.NowUnix()
	file := &edi.EDIInboundFile{
		BusinessUnitID:         fixture.buID,
		OrganizationID:         fixture.orgID,
		CommunicationProfileID: profile.ID,
		EDIPartnerID:           partner.ID,
		Method:                 edi.ConnectionMethodSFTP,
		RemotePath:             "/inbound/acme/ACME_REPAIR.x12",
		FileName:               "ACME_REPAIR.x12",
		Checksum:               "repair-checksum",
		SizeBytes:              int64(len(interchange)),
		RawContent:             interchange,
		Status:                 edi.InboundFileStatusProcessed,
		ReceivedAt:             now,
	}
	_, err = db.NewInsert().Model(file).Exec(ctx)
	require.NoError(t, err)

	message := func(control, raw string, ratingDetail map[string]any) pulid.ID {
		row := &edi.EDIMessage{
			BusinessUnitID:           fixture.buID,
			OrganizationID:           fixture.orgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			InboundFileID:            file.ID,
			Direction:                edi.DocumentDirectionInbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			InterchangeControlNumber: "000000317",
			GroupControlNumber:       "317",
			TransactionControlNumber: control,
			RawX12:                   raw,
			PayloadSnapshot: edi.NewLoadTenderDocumentPayload(edi.LoadTenderPayload{
				BOL:          "BOL-" + control,
				RatingDetail: ratingDetail,
			}),
			GeneratedAt: now,
		}
		_, insertErr := db.NewInsert().Model(row).Exec(ctx)
		require.NoError(t, insertErr)
		return row.ID
	}
	fixture.misread = message("0001", misreadTransaction, map[string]any{
		"externalShipmentId": "ACME",
		"paymentMethod":      "LT-88240",
		"rateBasis":          "flat",
	})
	fixture.legacy = message("0002", legacyTransaction, map[string]any{
		"externalShipmentId": "shp_01JLEGACY",
		"paymentMethod":      "PP",
	})
	fixture.purged = message("0003", "", map[string]any{
		"externalShipmentId": "ACME",
		"paymentMethod":      "LT-88242",
	})

	transfer := func(payment string, shipmentID pulid.ID, ref string) pulid.ID {
		row := &edi.EDITransfer{
			SourceOrganizationID: fixture.orgID,
			SourceBusinessUnitID: fixture.buID,
			TargetOrganizationID: fixture.orgID,
			TargetBusinessUnitID: fixture.buID,
			SourcePartnerID:      partner.ID,
			TargetPartnerID:      partner.ID,
			TargetShipmentID:     shipmentID,
			InboundMessageID:     fixture.misread,
			Status:               edi.TransferStatusApproved,
			TenderPayload: edi.LoadTenderPayload{
				BOL: "BOL-0001",
				RatingDetail: map[string]any{
					"externalShipmentId": ref,
					"paymentMethod":      payment,
				},
			},
			SubmittedAt: now,
		}
		_, insertErr := db.NewInsert().Model(row).Exec(ctx)
		require.NoError(t, insertErr)
		return row.ID
	}
	fixture.transfer = transfer("LT-88240", fixture.shipment, "ACME")
	fixture.edited = transfer("PP", fixture.untouched, "ACME-FIXED-BY-HAND")

	for _, id := range []pulid.ID{fixture.shipment, fixture.untouched} {
		_, err = db.NewUpdate().
			Table("shipments").
			Set("external_reference = ?", "ACME").
			Where("id = ?", id).
			Exec(ctx)
		require.NoError(t, err)
	}

	return fixture, db, postgres.NewTestConnection(db)
}

type repairRun struct {
	*editenderrepairservice.Report

	repairs []editenderrepairservice.Repair
}

func runRepair(
	t *testing.T,
	conn *postgres.Connection,
	orgID pulid.ID,
	dryRun bool,
) (*repairRun, error) {
	t.Helper()

	run := &repairRun{}
	report, err := editenderrepairservice.New(editenderrepairservice.Deps{
		Repo: editenderrepairrepository.New(editenderrepairrepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
	}).Repair(t.Context(), &editenderrepairservice.Request{
		OrganizationID: orgID,
		DryRun:         dryRun,
		OnRepair: func(repair editenderrepairservice.Repair) {
			run.repairs = append(run.repairs, repair)
		},
	})
	run.Report = report
	return run, err
}

func repair(t *testing.T, conn *postgres.Connection, orgID pulid.ID, dryRun bool) *repairRun {
	t.Helper()

	run, err := runRepair(t, conn, orgID, dryRun)
	require.NoError(t, err)
	return run
}

func ratingDetail(t *testing.T, db *bun.DB, table, column, path string, id pulid.ID) string {
	t.Helper()

	var value string
	require.NoError(t, db.NewSelect().
		Table(table).
		ColumnExpr("("+column+" #> ?)::text", bun.Safe(path)).
		Where("id = ?", id).
		Scan(t.Context(), &value))
	return value
}

func externalReference(t *testing.T, db *bun.DB, id pulid.ID) string {
	t.Helper()

	var value string
	require.NoError(t, db.NewSelect().
		Table("shipments").
		ColumnExpr("COALESCE(external_reference, '')").
		Where("id = ?", id).
		Scan(t.Context(), &value))
	return value
}

func TestRepairEDITenderReferencesCorrectsATenderReadOneElementEarly(t *testing.T) {
	fixture, db, conn := seedTenders(t)

	preview := repair(t, conn, fixture.orgID, true)
	require.Equal(t, 1, preview.Repaired)
	require.Len(t, preview.repairs, 1)
	require.Equal(t, fixture.misread, preview.repairs[0].MessageID)
	require.Equal(
		t,
		editenderrepairservice.Reference{ShipmentID: "LT-88240", PaymentMethod: "PP"},
		preview.repairs[0].To,
	)
	require.Equal(t, "ACME", externalReference(t, db, fixture.shipment))

	report := repair(t, conn, fixture.orgID, false)

	require.Equal(t, 1, report.Repaired)
	require.Equal(t, 1, report.Unchanged)
	require.Equal(t, 1, report.Unreadable)
	require.Equal(t, 1, report.TransfersRepaired)
	require.Equal(t, 1, report.ShipmentsRepaired)

	require.JSONEq(
		t,
		`{"externalShipmentId":"LT-88240","paymentMethod":"PP","rateBasis":"flat"}`,
		ratingDetail(t, db, "edi_messages", "payload_snapshot",
			"'{loadTender,ratingDetail}'", fixture.misread),
	)
	require.JSONEq(
		t,
		`{"externalShipmentId":"LT-88240","paymentMethod":"PP"}`,
		ratingDetail(t, db, "edi_load_tender_transfers", "tender_payload",
			"'{ratingDetail}'", fixture.transfer),
	)
	require.Equal(t, "LT-88240", externalReference(t, db, fixture.shipment))

	require.JSONEq(
		t,
		`{"externalShipmentId":"ACME-FIXED-BY-HAND","paymentMethod":"PP"}`,
		ratingDetail(t, db, "edi_load_tender_transfers", "tender_payload",
			"'{ratingDetail}'", fixture.edited),
	)
	require.Equal(t, "ACME", externalReference(t, db, fixture.untouched))
	require.JSONEq(
		t,
		`{"externalShipmentId":"shp_01JLEGACY","paymentMethod":"PP"}`,
		ratingDetail(t, db, "edi_messages", "payload_snapshot",
			"'{loadTender,ratingDetail}'", fixture.legacy),
	)
	require.JSONEq(
		t,
		`{"externalShipmentId":"ACME","paymentMethod":"LT-88242"}`,
		ratingDetail(t, db, "edi_messages", "payload_snapshot",
			"'{loadTender,ratingDetail}'", fixture.purged),
	)

	again := repair(t, conn, fixture.orgID, false)
	require.Zero(t, again.Repaired)
	require.Equal(t, 2, again.Unchanged)
}

func TestRepairEDITenderReferencesReadsAPurgedFileFromItsTransaction(t *testing.T) {
	fixture, db, conn := seedTenders(t)
	_, err := db.NewUpdate().
		Table("edi_inbound_files").
		Set("raw_content = ''").
		Where("organization_id = ?", fixture.orgID).
		Where("file_name = ?", "ACME_REPAIR.x12").
		Exec(t.Context())
	require.NoError(t, err)

	report := repair(t, conn, fixture.orgID, false)

	require.Equal(t, 1, report.Repaired)
	require.Equal(t, "LT-88240", externalReference(t, db, fixture.shipment))
}

func TestRepairEDITenderReferencesLeavesATenderWholeWhenOneWriteFails(t *testing.T) {
	fixture, db, conn := seedTenders(t)
	for _, statement := range []string{
		`CREATE FUNCTION refuse_message_update() RETURNS trigger LANGUAGE plpgsql AS
			$$ BEGIN RAISE EXCEPTION 'refused for the test'; END $$`,
		`CREATE TRIGGER refuse_message_update BEFORE UPDATE ON edi_messages
			FOR EACH ROW EXECUTE FUNCTION refuse_message_update()`,
	} {
		_, err := db.ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}

	_, err := runRepair(t, conn, fixture.orgID, false)

	require.ErrorContains(t, err, "refused for the test")
	require.Equal(t, "ACME", externalReference(t, db, fixture.shipment),
		"the shipment keeps its reference when the message could not be corrected")
	require.JSONEq(
		t,
		`{"externalShipmentId":"ACME","paymentMethod":"LT-88240"}`,
		ratingDetail(t, db, "edi_load_tender_transfers", "tender_payload",
			"'{ratingDetail}'", fixture.transfer),
	)
}
