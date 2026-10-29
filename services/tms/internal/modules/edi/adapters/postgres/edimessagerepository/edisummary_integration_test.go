//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
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
	documentType := findDocumentType(
		t,
		ctx,
		db,
		edidocument.TransactionSet204,
		edidocument.DocumentDirectionOutbound,
	)
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
		delivery edimessage.MessageDeliveryStatus
		ack      edimessage.MessageAcknowledgmentStatus
		age      int64
	}{
		{edimessage.MessageDeliveryStatusSent, edimessage.MessageAcknowledgmentStatusAccepted, 60},
		{
			edimessage.MessageDeliveryStatusSent,
			edimessage.MessageAcknowledgmentStatusPending,
			10 * 60 * 60,
		},
		{
			edimessage.MessageDeliveryStatusFailed,
			edimessage.MessageAcknowledgmentStatusNotExpected,
			60,
		},
		{
			edimessage.MessageDeliveryStatusDeadLettered,
			edimessage.MessageAcknowledgmentStatusNotExpected,
			60,
		},
		{
			edimessage.MessageDeliveryStatusDeadLettered,
			edimessage.MessageAcknowledgmentStatusNotExpected,
			120,
		},
	}
	for index, entry := range messageStatuses {
		message := &edimessage.EDIMessage{
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             partner.ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edidocument.DocumentDirectionOutbound,
			Standard:                 edidocument.EDIStandardX12,
			TransactionSet:           edidocument.TransactionSet204,
			X12Version:               edidocument.DefaultX12204Version,
			Status:                   edimessage.MessageStatusGenerated,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           entry.delivery,
			AckStatus:                entry.ack,
			GeneratedAt:              now - entry.age,
		}
		if entry.delivery == edimessage.MessageDeliveryStatusDeadLettered {
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
		&edimessageports.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, deliveryCounts[edimessage.MessageDeliveryStatusSent])
	require.Equal(t, 1, deliveryCounts[edimessage.MessageDeliveryStatusFailed])
	require.Equal(t, 2, deliveryCounts[edimessage.MessageDeliveryStatusDeadLettered])

	ackCounts, err := messageRepo.GetAckStatusCounts(
		ctx,
		&edimessageports.GetEDIMessageStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, ackCounts[edimessage.MessageAcknowledgmentStatusAccepted])
	require.Equal(t, 1, ackCounts[edimessage.MessageAcknowledgmentStatusPending])
	require.Equal(t, 3, ackCounts[edimessage.MessageAcknowledgmentStatusNotExpected])

	overdue, err := messageRepo.GetOverdueAckCount(
		ctx,
		&edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenantInfo,
			PendingSince: now - 4*60*60,
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, overdue)

	deadLettered, err := messageRepo.ListRecentDeadLettered(
		ctx,
		&edimessageports.ListRecentEDIMessageFailuresRequest{TenantInfo: tenantInfo, Limit: 5},
	)
	require.NoError(t, err)
	require.Len(t, deadLettered, 2)
	require.NotNil(t, deadLettered[0].Partner)
	require.Equal(t, "delivery failed", deadLettered[0].DeliveryLastError)

	fileStatuses := []ediinboundfile.InboundFileStatus{
		ediinboundfile.InboundFileStatusProcessed,
		ediinboundfile.InboundFileStatusQuarantined,
		ediinboundfile.InboundFileStatusQuarantined,
		ediinboundfile.InboundFileStatusPartiallyProcessed,
	}
	for index, status := range fileStatuses {
		file := &ediinboundfile.EDIInboundFile{
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
		edimessageports.GetEDIInboundFileStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 1, fileCounts[ediinboundfile.InboundFileStatusProcessed])
	require.Equal(t, 2, fileCounts[ediinboundfile.InboundFileStatusQuarantined])
	require.Equal(t, 1, fileCounts[ediinboundfile.InboundFileStatusPartiallyProcessed])

	quarantined, err := fileRepo.ListRecentQuarantined(
		ctx,
		edimessageports.ListRecentQuarantinedEDIInboundFilesRequest{
			TenantInfo: tenantInfo,
			Limit:      5,
		},
	)
	require.NoError(t, err)
	require.Len(t, quarantined, 2)

	transferStatuses := []editransfer.TransferStatus{
		editransfer.TransferStatusMappingRequired,
		editransfer.TransferStatusMappingRequired,
		editransfer.TransferStatusPendingApproval,
		editransfer.TransferStatusFailed,
	}
	for index, status := range transferStatuses {
		transfer := &editransfer.EDITransfer{
			SourceOrganizationID: org.OrgID,
			SourceBusinessUnitID: org.BuID,
			TargetOrganizationID: org.OrgID,
			TargetBusinessUnitID: org.BuID,
			SourcePartnerID:      partner.ID,
			TargetPartnerID:      partner.ID,
			Status:               status,
			TenderPayload:        editransfer.LoadTenderPayload{},
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
		editenderports.GetEDITransferStatusCountsRequest{TenantInfo: tenantInfo},
	)
	require.NoError(t, err)
	require.Equal(t, 2, transferCounts[editransfer.TransferStatusMappingRequired])
	require.Equal(t, 1, transferCounts[editransfer.TransferStatusPendingApproval])
	require.Equal(t, 1, transferCounts[editransfer.TransferStatusFailed])
}
