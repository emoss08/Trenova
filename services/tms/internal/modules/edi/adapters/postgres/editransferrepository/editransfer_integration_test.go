//go:build integration

package editransferrepository_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type transferFixture struct {
	ctx      context.Context
	tenant   pagination.TenantInfo
	sender   *edi.EDIPartner
	fileA    pulid.ID
	fromA    pulid.ID
	fromB    pulid.ID
	repo     editenderports.EDILoadTenderTransferRepository
	receiver *edi.EDIPartner
	tender   func(fileID pulid.ID, bol string) pulid.ID
	stampAll func(createdAt int64)
}

func seedTransfers(t *testing.T) *transferFixture {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	scope := editest.FirstTenant(t, ctx, db)
	ctx = scope.Ctx
	org := scope.Info
	dt := edidocumentdb.EDIDocumentTypes
	documentType, err := dt.Where(dt.TransactionSet.Eq(edidocument.TransactionSet204)).
		First(ctx, db)
	require.NoError(t, err)

	fixture := &transferFixture{
		ctx:    ctx,
		tenant: org,
		repo: editransferrepository.New(editransferrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
	}
	partner := func(code, name string) *edi.EDIPartner {
		row := &edi.EDIPartner{
			Kind: edi.PartnerKindExternal,
			Code: code,
			Name: name,
		}
		require.NoError(t, edidb.EDIPartners.Create(ctx, db, row))
		return row
	}
	fixture.sender = partner("ACME-TENDERS", "Acme Foods")
	fixture.receiver = partner("TRENOVA-SELF", "Trenova Freight")

	profile := &edi.EDICommunicationProfile{
		EDIPartnerID: fixture.sender.ID,
		Method:       edi.ConnectionMethodSFTP,
		Name:         "Acme Tenders Mailbox",
	}
	require.NoError(t, edidb.EDICommunicationProfiles.Create(ctx, db, profile))

	now := timeutils.NowUnix()
	file := func(name string) pulid.ID {
		row := &ediinboundfile.EDIInboundFile{
			CommunicationProfileID: profile.ID,
			EDIPartnerID:           fixture.sender.ID,
			Method:                 edi.ConnectionMethodSFTP,
			RemotePath:             "/inbound/" + name,
			FileName:               name,
			Checksum:               name,
			RawContent:             "ISA~",
			Status:                 ediinboundfile.InboundFileStatusProcessed,
			ReceivedAt:             now,
		}
		require.NoError(t, ediinboundfiledb.EDIInboundFiles.Create(ctx, db, row))
		return row.ID
	}
	fixture.fileA = file("ACME_A.x12")
	fileB := file("ACME_B.x12")

	tender := func(fileID pulid.ID, bol string) pulid.ID {
		message := &edimessage.EDIMessage{
			EDIPartnerID:             fixture.sender.ID,
			DocumentTypeID:           documentType.ID,
			InboundFileID:            fileID,
			Direction:                edidocument.DocumentDirectionInbound,
			Standard:                 edidocument.EDIStandardX12,
			TransactionSet:           edidocument.TransactionSet204,
			X12Version:               edidocument.DefaultX12204Version,
			Status:                   edimessage.MessageStatusGenerated,
			InterchangeControlNumber: "000000001",
			GroupControlNumber:       "1",
			TransactionControlNumber: bol,
			GeneratedAt:              now,
		}
		require.NoError(t, edimessagedb.EDIMessages.Create(ctx, db, message))

		transfer := &editransfer.EDITransfer{
			SourceOrganizationID: org.OrgID,
			SourceBusinessUnitID: org.BuID,
			TargetOrganizationID: org.OrgID,
			TargetBusinessUnitID: org.BuID,
			SourcePartnerID:      fixture.sender.ID,
			TargetPartnerID:      fixture.receiver.ID,
			InboundMessageID:     message.ID,
			Status:               editransfer.TransferStatusMappingRequired,
			TenderPayload:        editransfer.LoadTenderPayload{BOL: bol},
			SubmittedAt:          now,
		}
		require.NoError(t, editransferdb.EDITransfers.Create(ctx, db, transfer))
		return transfer.ID
	}
	fixture.fromA = tender(fixture.fileA, "BOL-A")
	fixture.fromB = tender(fileB, "BOL-B")
	fixture.tender = tender
	fixture.stampAll = func(createdAt int64) {
		editest.Exec(
			t,
			ctx,
			db,
			"UPDATE edi_load_tender_transfers SET created_at = $1 WHERE target_organization_id = $2",
			createdAt,
			org.OrgID.String(),
		)
	}
	return fixture
}

func TestGetTransferByIDNamesBothPartners(t *testing.T) {
	fixture := seedTransfers(t)

	transfer, err := fixture.repo.GetTransferByID(
		fixture.ctx,
		editenderports.GetEDITransferByIDRequest{
			ID:         fixture.fromA,
			TenantInfo: fixture.tenant,
			Direction:  "inbound",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, transfer.SourcePartner)
	require.Equal(t, "Acme Foods", transfer.SourcePartner.Name)
	require.NotNil(t, transfer.TargetPartner)
	require.Equal(t, "Trenova Freight", transfer.TargetPartner.Name)
}

func TestListInboundTransfersFromOneInboundFile(t *testing.T) {
	fixture := seedTransfers(t)

	result, err := fixture.repo.ListInbound(fixture.ctx, &editenderports.ListEDITransfersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: fixture.tenant,
			Pagination: pagination.Info{Limit: 20},
		},
		InboundFileID: fixture.fileA,
	})

	require.NoError(t, err)
	ids := make([]pulid.ID, 0, len(result.Items))
	for _, transfer := range result.Items {
		ids = append(ids, transfer.ID)
	}
	require.Equal(t, []pulid.ID{fixture.fromA}, ids)
	require.Equal(t, 1, result.Total)
}

func TestListInboundTransfersPagesAFileWhoseTendersArrivedTogether(t *testing.T) {
	fixture := seedTransfers(t)
	want := []pulid.ID{fixture.fromA}
	for _, bol := range []string{"BOL-A2", "BOL-A3", "BOL-A4", "BOL-A5"} {
		want = append(want, fixture.tender(fixture.fileA, bol))
	}
	fixture.stampAll(timeutils.NowUnix())
	slices.SortFunc(
		want,
		func(a, b pulid.ID) int { return strings.Compare(b.String(), a.String()) },
	)

	read := make([]pulid.ID, 0, len(want))
	for offset := 0; offset < len(want); offset += 2 {
		page, err := fixture.repo.ListInbound(fixture.ctx, &editenderports.ListEDITransfersRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: fixture.tenant,
				Pagination: pagination.Info{Limit: 2, Offset: offset},
			},
			InboundFileID: fixture.fileA,
		})
		require.NoError(t, err)
		require.Equal(t, len(want), page.Total)
		for _, transfer := range page.Items {
			read = append(read, transfer.ID)
		}
	}

	require.Equal(t, want, read, "every tender once, newest first and by id within a second")
}
