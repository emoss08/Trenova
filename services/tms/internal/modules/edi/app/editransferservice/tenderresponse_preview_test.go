package editransferservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func externalTender(status editransfer.TransferStatus) *editransfer.EDITransfer {
	return &editransfer.EDITransfer{
		ID:               pulid.MustNew("edilt_"),
		TargetPartnerID:  pulid.MustNew("edip_"),
		InboundMessageID: pulid.MustNew("edimsg_"),
		SourceShipmentID: pulid.MustNew("shp_"),
		Status:           status,
		TenderPayload:    editransfer.LoadTenderPayload{BOL: "BOL-990"},
	}
}

func TestTenderResponseRequest_IsTheRequestTheSendMakes(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	transfer := externalTender(editransfer.TransferStatusRejected)
	transfer.RejectionReason = "No capacity Friday"
	actor := pulid.MustNew("usr_")

	req := tenderResponseRequest(tenant, transfer, actor)

	assert.Equal(t, transfer.TargetPartnerID, req.EDIPartnerID)
	assert.Equal(t, transfer.ID, req.TransferID)
	assert.Equal(t, edidocument.TransactionSet990, req.TransactionSet)
	assert.Equal(t, edidocument.DocumentDirectionOutbound, req.Direction)
	assert.Equal(t, actor, req.GeneratedByID)
	require.NotNil(t, req.Payload)
	require.NotNil(t, req.Payload.TenderResponse)
	assert.Equal(t, "D", req.Payload.TenderResponse.ResponseCode)
	assert.Equal(t, "No capacity Friday", req.Payload.TenderResponse.RejectionReason)
}

func TestTenderResponseToPreview_SettlesWhichAnswerTheDocumentCarries(t *testing.T) {
	t.Parallel()

	t.Run("an undecided tender needs the answer to preview", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusPendingApproval)
		_, err := tenderResponseToPreview(transfer, transfer.TargetPartnerID, "", "")
		require.Error(t, err)
		assert.True(t, errortypes.IsError(err))
	})

	t.Run("accepting one", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusPendingApproval)
		previewed, err := tenderResponseToPreview(
			transfer, transfer.TargetPartnerID, TenderResponseAccept, "",
		)
		require.NoError(t, err)
		assert.Equal(t, editransfer.TransferStatusApproved, previewed.Status)
		assert.Equal(t, editransfer.TransferStatusPendingApproval, transfer.Status,
			"the stored transfer is never changed")
	})

	t.Run("declining one with a reason", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusMappingRequired)
		previewed, err := tenderResponseToPreview(
			transfer, transfer.TargetPartnerID, TenderResponseDecline, "Lane not served",
		)
		require.NoError(t, err)
		assert.Equal(t, editransfer.TransferStatusRejected, previewed.Status)
		assert.Equal(t, "Lane not served", previewed.RejectionReason)
		assert.Equal(
			t,
			"D",
			edipayloadbuild.BuildTenderResponsePayload(previewed).TenderResponse.ResponseCode,
		)
	})

	t.Run("a decided tender answers as it was decided", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusApproved)
		previewed, err := tenderResponseToPreview(transfer, transfer.TargetPartnerID, "", "")
		require.NoError(t, err)
		assert.Equal(t, editransfer.TransferStatusApproved, previewed.Status)

		_, err = tenderResponseToPreview(
			transfer, transfer.TargetPartnerID, TenderResponseDecline, "",
		)
		require.Error(t, err, "an accepted tender cannot be previewed as declined")
	})

	t.Run("a tender from another Trenova organization has no 990", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusPendingApproval)
		transfer.InboundMessageID = pulid.Nil
		_, err := tenderResponseToPreview(
			transfer, transfer.TargetPartnerID, TenderResponseAccept, "",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inside Trenova")
	})

	t.Run("another partner's tender", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusPendingApproval)
		_, err := tenderResponseToPreview(
			transfer, pulid.MustNew("edip_"), TenderResponseAccept, "",
		)
		require.Error(t, err)
		assert.True(t, errortypes.IsError(err))
	})

	t.Run("a tender in flight", func(t *testing.T) {
		t.Parallel()

		transfer := externalTender(editransfer.TransferStatusProcessing)
		_, err := tenderResponseToPreview(
			transfer, transfer.TargetPartnerID, TenderResponseAccept, "",
		)
		require.Error(t, err)
	})
}
