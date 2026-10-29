package editenderrecipient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) AdvanceBaseline(
	ctx context.Context,
	recipient *editenderchange.TenderRecipient,
	payload *editransfer.LoadTenderPayload,
	status editenderchange.TenderRecipientBaselineStatus,
) error {
	baseline := *payload
	baseline.PurposeCode = editransfer.LoadTenderPurposeOriginal
	recipient.LatestBaselinePayload = baseline
	recipient.LatestBaselineHash = PayloadHash(&baseline)
	recipient.BaselineRecordedAt = timeutils.NowUnix()
	recipient.BaselineStatus = status
	_, err := s.tenderRecipientRepo.UpdateTenderRecipient(ctx, recipient)
	return err
}

func (s *Service) UpsertInternal(
	ctx context.Context,
	transfer *editransfer.EDITransfer,
	link *edisync.ShipmentLink,
	status editenderchange.TenderRecipientBaselineStatus,
) error {
	if s.tenderRecipientRepo == nil || transfer == nil {
		return nil
	}
	payload := transfer.TenderPayload
	payload.PurposeCode = editransfer.LoadTenderPurposeOriginal
	recipient := &editenderchange.TenderRecipient{
		BusinessUnitID:          transfer.SourceBusinessUnitID,
		SourceOrganizationID:    transfer.SourceOrganizationID,
		SourceShipmentID:        transfer.SourceShipmentID,
		RecipientKind:           editenderchange.TenderRecipientKindInternal,
		RecipientOrganizationID: transfer.TargetOrganizationID,
		EDIPartnerID:            transfer.TargetPartnerID,
		OriginalTransferID:      transfer.ID,
		LatestBaselinePayload:   payload,
		LatestBaselineHash:      PayloadHash(&payload),
		BaselineRecordedAt:      timeutils.NowUnix(),
		BaselineStatus:          status,
		Status:                  editenderchange.TenderRecipientStatusActive,
	}
	if link != nil {
		recipient.ShipmentLinkID = link.ID
	}
	_, err := s.tenderRecipientRepo.UpsertTenderRecipient(
		ctx,
		editenderports.UpsertEDITenderRecipientRequest{Recipient: recipient},
	)
	return err
}

func (s *Service) UpsertExternal(
	ctx context.Context,
	message *edimessage.EDIMessage,
	partner *edi.EDIPartner,
) error {
	if s.tenderRecipientRepo == nil || message == nil || partner == nil ||
		message.TransactionSet != edidocument.TransactionSet204 ||
		message.Direction != edidocument.DocumentDirectionOutbound ||
		message.PayloadSnapshot.LoadTender == nil ||
		message.PayloadSnapshot.LoadTender.PurposeCode == editransfer.LoadTenderPurposeChange {
		return nil
	}
	if partner.Kind != edi.PartnerKindExternal {
		return nil
	}
	commProfile, err := s.profileRepo.GetActiveProfileByPartner(
		ctx,
		edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID: message.EDIPartnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: message.OrganizationID,
				BuID:  message.BusinessUnitID,
			},
			Method: edi.ConnectionMethodSFTP,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return err
		}
		commProfile = nil
	}
	payload := *message.PayloadSnapshot.LoadTender
	payload.PurposeCode = editransfer.LoadTenderPurposeOriginal
	recipient := &editenderchange.TenderRecipient{
		BusinessUnitID:        message.BusinessUnitID,
		SourceOrganizationID:  message.OrganizationID,
		SourceShipmentID:      payload.ShipmentID,
		RecipientKind:         editenderchange.TenderRecipientKindExternal,
		EDIPartnerID:          message.EDIPartnerID,
		OriginalMessageID:     message.ID,
		LatestBaselinePayload: payload,
		LatestBaselineHash:    PayloadHash(&payload),
		BaselineRecordedAt:    timeutils.NowUnix(),
		BaselineStatus:        editenderchange.TenderRecipientBaselineStatusSent,
		Status:                editenderchange.TenderRecipientStatusActive,
	}
	if commProfile != nil {
		recipient.CommunicationProfileID = commProfile.ID
	}
	_, err = s.tenderRecipientRepo.UpsertTenderRecipient(
		ctx,
		editenderports.UpsertEDITenderRecipientRequest{Recipient: recipient},
	)
	return err
}

func PayloadHash(payload *editransfer.LoadTenderPayload) string {
	if payload == nil {
		return ""
	}
	hashPayload := *payload
	hashPayload.PurposeCode = ""
	data, err := sonic.ConfigStd.Marshal(hashPayload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
