//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEDISummaryCounts(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)

	scope := tenantWithoutPartners(t, ctx, db)
	ctx = scope.Ctx
	org := scope.Info
	documentType := findDocumentType(t, ctx, db, edi.TransactionSet204, edi.DocumentDirectionOutbound)
	tenantInfo := org

	partner := &edi.EDIPartner{
		BusinessUnitID: org.BuID,
		OrganizationID: org.OrgID,
		Kind:           edi.PartnerKindExternal,
		Code:           "SUMMARY-TEST",
		Name:           "Summary Test Partner",
	}
	require.NoError(t, create(ctx, db, partner))

	profile := &edi.EDICommunicationProfile{
		BusinessUnitID: org.BuID,
		OrganizationID: org.OrgID,
		EDIPartnerID:   partner.ID,
		Method:         edi.ConnectionMethodSFTP,
		Name:           "Summary Test Profile",
	}
	require.NoError(t, create(ctx, db, profile))

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
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet204,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
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
		require.NoError(t, create(ctx, db, message), "message %d", index)
	}

	messageRepo := edimessagerepository.New(edimessagerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	deliveryCounts, err := messageRepo.GetDeliveryStatusCounts(
		ctx,
		ediports.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, deliveryCounts[edi.MessageDeliveryStatusSent])
	require.Equal(t, 1, deliveryCounts[edi.MessageDeliveryStatusFailed])
	require.Equal(t, 2, deliveryCounts[edi.MessageDeliveryStatusDeadLettered])

	ackCounts, err := messageRepo.GetAckStatusCounts(
		ctx,
		ediports.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, ackCounts[edi.MessageAcknowledgmentStatusAccepted])
	require.Equal(t, 1, ackCounts[edi.MessageAcknowledgmentStatusPending])
	require.Equal(t, 3, ackCounts[edi.MessageAcknowledgmentStatusNotExpected])

	overdue, err := messageRepo.GetOverdueAckCount(
		ctx,
		ediports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenantInfo,
			PendingSince: now - 4*60*60,
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, overdue)

	deadLettered, err := messageRepo.ListRecentDeadLettered(
		ctx,
		&ediports.ListRecentEDIMessageFailuresRequest{TenantInfo: tenantInfo, Limit: 5},
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
			BusinessUnitID:         org.BuID,
			OrganizationID:         org.OrgID,
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
		require.NoError(t, create(ctx, db, file), "file %d", index)
	}

	fileRepo := ediinboundfilerepository.New(ediinboundfilerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	fileCounts, err := fileRepo.GetInboundFileStatusCounts(
		ctx,
		ediports.GetEDIInboundFileStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, fileCounts[edi.InboundFileStatusProcessed])
	require.Equal(t, 2, fileCounts[edi.InboundFileStatusQuarantined])
	require.Equal(t, 1, fileCounts[edi.InboundFileStatusPartiallyProcessed])

	quarantined, err := fileRepo.ListRecentQuarantined(
		ctx,
		ediports.ListRecentQuarantinedEDIInboundFilesRequest{TenantInfo: tenantInfo, Limit: 5},
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
			SourceOrganizationID: org.OrgID,
			SourceBusinessUnitID: org.BuID,
			TargetOrganizationID: org.OrgID,
			TargetBusinessUnitID: org.BuID,
			SourcePartnerID:      partner.ID,
			TargetPartnerID:      partner.ID,
			Status:               status,
			TenderPayload:        edi.LoadTenderPayload{},
			SubmittedAt:          now - int64(index),
		}
		require.NoError(t, create(ctx, db, transfer), "transfer %d", index)
	}

	transferRepo := editransferrepository.New(editransferrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	transferCounts, err := transferRepo.GetInboundStatusCounts(
		ctx,
		ediports.GetEDITransferStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, transferCounts[edi.TransferStatusMappingRequired])
	require.Equal(t, 1, transferCounts[edi.TransferStatusPendingApproval])
	require.Equal(t, 1, transferCounts[edi.TransferStatusFailed])
}
