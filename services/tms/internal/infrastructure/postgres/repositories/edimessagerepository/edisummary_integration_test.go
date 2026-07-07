//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edimessagerepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/editransferrepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type seededSummaryOrg struct {
	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type seededSummaryDocumentType struct {
	ID pulid.ID `bun:"id"`
}

func TestEDISummaryCounts(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()
	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(
		db,
		seedRegistry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)

	var org seededSummaryOrg
	require.NoError(
		t,
		db.NewSelect().
			Table("organizations").
			Column("id", "business_unit_id").
			Limit(1).
			Scan(ctx, &org),
	)
	var documentType seededSummaryDocumentType
	require.NoError(
		t,
		db.NewSelect().
			Table("edi_document_types").
			Column("id").
			Where("transaction_set = ?", edi.TransactionSet204).
			Where("direction = ?", edi.DocumentDirectionOutbound).
			Limit(1).
			Scan(ctx, &documentType),
	)
	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Kind:           edi.PartnerKindExternal,
		Code:           "SUMMARY-TEST",
		Name:           "Summary Test Partner",
	}
	_, err = db.NewInsert().Model(partner).Exec(ctx)
	require.NoError(t, err)

	profile := &edi.EDICommunicationProfile{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		EDIPartnerID:   partner.ID,
		Method:         edi.ConnectionMethodSFTP,
		Name:           "Summary Test Profile",
	}
	_, err = db.NewInsert().Model(profile).Exec(ctx)
	require.NoError(t, err)

	now := timeutils.NowUnix()
	messageStatuses := []struct {
		delivery edi.MessageDeliveryStatus
		ack      edi.MessageAcknowledgmentStatus
		age      int64
	}{
		{edi.MessageDeliveryStatusSent, edi.MessageAcknowledgmentStatusAccepted, 60},
		{edi.MessageDeliveryStatusSent, edi.MessageAcknowledgmentStatusPending, 10 * 60 * 60},
		{edi.MessageDeliveryStatusFailed, edi.MessageAcknowledgmentStatusNotExpected, 60},
		{edi.MessageDeliveryStatusDeadLettered, edi.MessageAcknowledgmentStatusNotExpected, 60},
		{edi.MessageDeliveryStatusDeadLettered, edi.MessageAcknowledgmentStatusNotExpected, 120},
	}
	for index, entry := range messageStatuses {
		message := &edi.EDIMessage{
			BusinessUnitID:           org.BusinessUnitID,
			OrganizationID:           org.ID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           entry.delivery,
			AckStatus:                entry.ack,
			GeneratedAt:              now - entry.age,
		}
		if entry.delivery == edi.MessageDeliveryStatusDeadLettered {
			message.DeliveryLastError = "delivery failed"
		}
		_, insertErr := db.NewInsert().Model(message).Exec(ctx)
		require.NoError(t, insertErr, "message %d", index)
	}

	messageRepo := edimessagerepository.New(edimessagerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	deliveryCounts, err := messageRepo.GetDeliveryStatusCounts(
		ctx,
		repositories.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, deliveryCounts[edi.MessageDeliveryStatusSent])
	require.Equal(t, 1, deliveryCounts[edi.MessageDeliveryStatusFailed])
	require.Equal(t, 2, deliveryCounts[edi.MessageDeliveryStatusDeadLettered])

	ackCounts, err := messageRepo.GetAckStatusCounts(
		ctx,
		repositories.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, ackCounts[edi.MessageAcknowledgmentStatusAccepted])
	require.Equal(t, 1, ackCounts[edi.MessageAcknowledgmentStatusPending])
	require.Equal(t, 3, ackCounts[edi.MessageAcknowledgmentStatusNotExpected])

	overdue, err := messageRepo.GetOverdueAckCount(
		ctx,
		repositories.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenantInfo,
			PendingSince: now - 4*60*60,
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, overdue)

	deadLettered, err := messageRepo.ListRecentDeadLettered(
		ctx,
		&repositories.ListRecentEDIMessageFailuresRequest{TenantInfo: tenantInfo, Limit: 5},
	)
	require.NoError(t, err)
	require.Len(t, deadLettered, 2)
	require.NotNil(t, deadLettered[0].Partner)
	require.Equal(t, "delivery failed", deadLettered[0].DeliveryLastError)

	fileStatuses := []edi.InboundFileStatus{
		edi.InboundFileStatusProcessed,
		edi.InboundFileStatusQuarantined,
		edi.InboundFileStatusQuarantined,
		edi.InboundFileStatusPartiallyProcessed,
	}
	for index, status := range fileStatuses {
		file := &edi.EDIInboundFile{
			BusinessUnitID:         org.BusinessUnitID,
			OrganizationID:         org.ID,
			CommunicationProfileID: profile.ID,
			EDIPartnerID:           partner.ID,
			Method:                 edi.ConnectionMethodSFTP,
			RemotePath:             "/inbound/test.edi",
			FileName:               "test.edi",
			Checksum:               pulid.MustNew("chk_").String(),
			RawContent:             "ISA*TEST~",
			Status:                 status,
			FailureReason:          "processing failed",
			ReceivedAt:             now - int64(index),
		}
		_, insertErr := db.NewInsert().Model(file).Exec(ctx)
		require.NoError(t, insertErr, "file %d", index)
	}

	fileRepo := ediinboundfilerepository.New(ediinboundfilerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	fileCounts, err := fileRepo.GetInboundFileStatusCounts(
		ctx,
		repositories.GetEDIInboundFileStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, fileCounts[edi.InboundFileStatusProcessed])
	require.Equal(t, 2, fileCounts[edi.InboundFileStatusQuarantined])
	require.Equal(t, 1, fileCounts[edi.InboundFileStatusPartiallyProcessed])

	quarantined, err := fileRepo.ListRecentQuarantined(
		ctx,
		repositories.ListRecentQuarantinedEDIInboundFilesRequest{TenantInfo: tenantInfo, Limit: 5},
	)
	require.NoError(t, err)
	require.Len(t, quarantined, 2)

	transferStatuses := []edi.TransferStatus{
		edi.TransferStatusMappingRequired,
		edi.TransferStatusMappingRequired,
		edi.TransferStatusPendingApproval,
		edi.TransferStatusFailed,
	}
	for index, status := range transferStatuses {
		transfer := &edi.EDITransfer{
			SourceOrganizationID: org.ID,
			SourceBusinessUnitID: org.BusinessUnitID,
			TargetOrganizationID: org.ID,
			TargetBusinessUnitID: org.BusinessUnitID,
			SourcePartnerID:      partner.ID,
			TargetPartnerID:      partner.ID,
			Status:               status,
			TenderPayload:        edi.LoadTenderPayload{},
			SubmittedAt:          now - int64(index),
		}
		_, insertErr := db.NewInsert().Model(transfer).Exec(ctx)
		require.NoError(t, insertErr, "transfer %d", index)
	}

	transferRepo := editransferrepository.New(editransferrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	transferCounts, err := transferRepo.GetInboundStatusCounts(
		ctx,
		repositories.GetEDITransferStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, transferCounts[edi.TransferStatusMappingRequired])
	require.Equal(t, 1, transferCounts[edi.TransferStatusPendingApproval])
	require.Equal(t, 1, transferCounts[edi.TransferStatusFailed])
}
