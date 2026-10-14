//go:build integration

package editransferrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type transferFixture struct {
	tenant   pagination.TenantInfo
	sender   *edi.EDIPartner
	fileA    pulid.ID
	fromA    pulid.ID
	fromB    pulid.ID
	repo     ediports.EDILoadTenderTransferRepository
	receiver *edi.EDIPartner
}

func seedTransfers(t *testing.T) *transferFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))
	var documentType struct {
		ID pulid.ID `bun:"id"`
	}
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("id").
		Where("transaction_set = ?", edi.TransactionSet204).
		Limit(1).
		Scan(ctx, &documentType))

	fixture := &transferFixture{
		tenant: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		repo: editransferrepository.New(editransferrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
	}
	partner := func(code, name string) *edi.EDIPartner {
		row := &edi.EDIPartner{
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.ID,
			Kind:           edi.PartnerKindExternal,
			Code:           code,
			Name:           name,
		}
		_, insertErr := db.NewInsert().Model(row).Exec(ctx)
		require.NoError(t, insertErr)
		return row
	}
	fixture.sender = partner("ACME-TENDERS", "Acme Foods")
	fixture.receiver = partner("TRENOVA-SELF", "Trenova Freight")

	profile := &edi.EDICommunicationProfile{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		EDIPartnerID:   fixture.sender.ID,
		Method:         edi.ConnectionMethodSFTP,
		Name:           "Acme Tenders Mailbox",
	}
	_, err = db.NewInsert().Model(profile).Exec(ctx)
	require.NoError(t, err)

	now := timeutils.NowUnix()
	file := func(name string) pulid.ID {
		row := &edi.EDIInboundFile{
			BusinessUnitID:         org.BusinessUnitID,
			OrganizationID:         org.ID,
			CommunicationProfileID: profile.ID,
			EDIPartnerID:           fixture.sender.ID,
			Method:                 edi.ConnectionMethodSFTP,
			RemotePath:             "/inbound/" + name,
			FileName:               name,
			Checksum:               name,
			RawContent:             "ISA~",
			Status:                 edi.InboundFileStatusProcessed,
			ReceivedAt:             now,
		}
		_, insertErr := db.NewInsert().Model(row).Exec(ctx)
		require.NoError(t, insertErr)
		return row.ID
	}
	fixture.fileA = file("ACME_A.x12")
	fileB := file("ACME_B.x12")

	tender := func(fileID pulid.ID, bol string) pulid.ID {
		message := &edi.EDIMessage{
			BusinessUnitID:           org.BusinessUnitID,
			OrganizationID:           org.ID,
			EDIPartnerID:             fixture.sender.ID,
			DocumentTypeID:           documentType.ID,
			InboundFileID:            fileID,
			Direction:                edi.DocumentDirectionInbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			InterchangeControlNumber: "000000001",
			GroupControlNumber:       "1",
			TransactionControlNumber: bol,
			GeneratedAt:              now,
		}
		_, insertErr := db.NewInsert().Model(message).Exec(ctx)
		require.NoError(t, insertErr)

		transfer := &edi.EDITransfer{
			SourceOrganizationID: org.ID,
			SourceBusinessUnitID: org.BusinessUnitID,
			TargetOrganizationID: org.ID,
			TargetBusinessUnitID: org.BusinessUnitID,
			SourcePartnerID:      fixture.sender.ID,
			TargetPartnerID:      fixture.receiver.ID,
			InboundMessageID:     message.ID,
			Status:               edi.TransferStatusMappingRequired,
			TenderPayload:        edi.LoadTenderPayload{BOL: bol},
			SubmittedAt:          now,
		}
		_, insertErr = db.NewInsert().Model(transfer).Exec(ctx)
		require.NoError(t, insertErr)
		return transfer.ID
	}
	fixture.fromA = tender(fixture.fileA, "BOL-A")
	fixture.fromB = tender(fileB, "BOL-B")
	return fixture
}

func TestGetTransferByIDNamesBothPartners(t *testing.T) {
	fixture := seedTransfers(t)

	transfer, err := fixture.repo.GetTransferByID(t.Context(), ediports.GetEDITransferByIDRequest{
		ID:         fixture.fromA,
		TenantInfo: fixture.tenant,
		Direction:  "inbound",
	})

	require.NoError(t, err)
	require.NotNil(t, transfer.SourcePartner)
	require.Equal(t, "Acme Foods", transfer.SourcePartner.Name)
	require.NotNil(t, transfer.TargetPartner)
	require.Equal(t, "Trenova Freight", transfer.TargetPartner.Name)
}

func TestListInboundTransfersFromOneInboundFile(t *testing.T) {
	fixture := seedTransfers(t)

	result, err := fixture.repo.ListInbound(t.Context(), &ediports.ListEDITransfersRequest{
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
