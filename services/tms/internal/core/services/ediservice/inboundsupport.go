package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) BuildInboundMappingPreview(
	ctx context.Context,
	partner *edi.EDIPartner,
	payload edi.LoadTenderPayload,
) (*MappingPreview, error) {
	return s.buildMappingPreview(ctx, partner, payload)
}

type ApplyExternalTenderResponseRequest struct {
	Recipient       *edi.TenderRecipient
	Accepted        bool
	ReasonCode      string
	RejectionReason string
	ActorID         pulid.ID
}

func (s *Service) ApplyExternalTenderResponse(
	ctx context.Context,
	req *ApplyExternalTenderResponseRequest,
) error {
	if req == nil || req.Recipient == nil {
		return errortypes.NewValidationError(
			"recipient",
			errortypes.ErrRequired,
			"EDI tender recipient is required for tender response processing",
		)
	}
	recipient := req.Recipient
	tenantInfo := pagination.TenantInfo{
		OrgID:  recipient.SourceOrganizationID,
		BuID:   recipient.SourceBusinessUnitID,
		UserID: req.ActorID,
	}
	tenderStatus := shipment.TenderStatusAccepted
	comment := "EDI load tender accepted by trading partner."
	if !req.Accepted {
		tenderStatus = shipment.TenderStatusRejected
		reason := stringutils.FirstNonEmpty(req.RejectionReason, req.ReasonCode)
		comment = "EDI load tender rejected by trading partner."
		if reason != "" {
			comment = "EDI load tender rejected by trading partner: " + reason
		}
	}
	if recipient.SourceShipmentID.IsNotNil() {
		if err := s.setShipmentTenderStatus(
			ctx,
			recipient.SourceShipmentID,
			tenantInfo,
			tenderStatus,
		); err != nil {
			return err
		}
		if err := s.createSystemShipmentComment(
			ctx,
			recipient.SourceShipmentID,
			tenantInfo,
			comment,
			map[string]any{"recipientId": recipient.ID},
		); err != nil {
			s.l.Warn(
				"failed to record EDI tender response comment",
				zap.String("recipientId", recipient.ID.String()),
				zap.Error(err),
			)
		}
	}
	if req.Accepted {
		recipient.BaselineStatus = edi.TenderRecipientBaselineStatusAccepted
	} else {
		recipient.Status = edi.TenderRecipientStatusClosed
	}
	recipient.BaselineRecordedAt = timeutils.NowUnix()
	_, err := s.tenderRecipientRepo.UpdateTenderRecipient(ctx, recipient)
	return err
}

type RecordExternalShipmentStatusRequest struct {
	Recipient   *edi.TenderRecipient
	StatusCode  string
	ReasonCode  string
	ReferenceID string
	EventAt     int64
}

func (s *Service) RecordExternalShipmentStatus(
	ctx context.Context,
	req *RecordExternalShipmentStatusRequest,
) error {
	if req == nil || req.Recipient == nil || req.Recipient.SourceShipmentID.IsNil() {
		return errortypes.NewValidationError(
			"recipient",
			errortypes.ErrRequired,
			"EDI tender recipient with a source shipment is required for status processing",
		)
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: req.Recipient.SourceOrganizationID,
		BuID:  req.Recipient.SourceBusinessUnitID,
	}
	comment := "EDI 214 shipment status received from trading partner: " + req.StatusCode
	if description := externalShipmentStatusDescription(req.StatusCode); description != "" {
		comment += " (" + description + ")"
	}
	if req.ReasonCode != "" {
		comment += ", reason " + req.ReasonCode
	}
	metadata := map[string]any{
		"recipientId": req.Recipient.ID,
		"statusCode":  req.StatusCode,
	}
	if req.ReasonCode != "" {
		metadata["reasonCode"] = req.ReasonCode
	}
	if req.ReferenceID != "" {
		metadata["referenceId"] = req.ReferenceID
	}
	if req.EventAt > 0 {
		metadata["eventAt"] = req.EventAt
	}
	return s.createSystemShipmentComment(
		ctx,
		req.Recipient.SourceShipmentID,
		tenantInfo,
		comment,
		metadata,
	)
}

func externalShipmentStatusDescription(statusCode string) string {
	switch statusCode {
	case "AF":
		return "departed pickup location"
	case "X1":
		return "arrived at delivery location"
	case "X3":
		return "arrived at pickup location"
	case "D1":
		return "completed delivery"
	case "A3", "SD":
		return "shipment delayed"
	case "A7":
		return "shipment canceled"
	case "X6":
		return "en route to delivery location"
	default:
		return ""
	}
}

func (s *Service) EnsureOutboundDocumentProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	transactionSet edi.TransactionSet,
) (*edi.EDIPartnerDocumentProfile, error) {
	profile, err := s.documentProfileRepo.GetActivePartnerDocumentProfile(
		ctx,
		repositories.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      partnerID,
			TenantInfo:     tenantInfo,
			TransactionSet: transactionSet,
			Direction:      edi.DocumentDirectionOutbound,
		},
	)
	if err == nil {
		return profile, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	envelope, err := s.partnerEnvelopeSettings(ctx, tenantInfo, partnerID)
	if err != nil {
		return nil, err
	}
	template, _, err := s.templateRepo.EnsureBaseTemplate(ctx, tenantInfo, transactionSet)
	if err != nil {
		return nil, err
	}
	return s.UpsertPartnerDocumentProfile(ctx, &UpsertEDIPartnerDocumentProfileRequest{
		TenantInfo:     tenantInfo,
		EDIPartnerID:   partnerID,
		TemplateID:     template.ID,
		Status:         edi.DocumentStatusActive,
		Envelope:       envelope,
		Acknowledgment: edi.AcknowledgmentConfig{Type: edi.AcknowledgmentTypeNone},
		ValidationMode: edi.ValidationModeWarnOnly,
	}, nil)
}

func (s *Service) partnerEnvelopeSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
) (edi.X12EnvelopeSettings, error) {
	profiles, err := s.documentProfileRepo.ListPartnerDocumentProfiles(
		ctx,
		&repositories.ListEDIPartnerDocumentProfilesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 50},
			},
			PartnerID: partnerID,
		},
	)
	if err != nil {
		return edi.X12EnvelopeSettings{}, err
	}
	var inbound *edi.EDIPartnerDocumentProfile
	for _, candidate := range profiles.Items {
		if candidate == nil || candidate.Status != edi.DocumentStatusActive {
			continue
		}
		if candidate.Direction == edi.DocumentDirectionOutbound {
			return candidate.Envelope, nil
		}
		if inbound == nil {
			inbound = candidate
		}
	}
	if inbound != nil {
		envelope := inbound.Envelope
		envelope.InterchangeSenderID, envelope.InterchangeReceiverID =
			envelope.InterchangeReceiverID, envelope.InterchangeSenderID
		envelope.ApplicationSenderCode, envelope.ApplicationReceiverCode =
			envelope.ApplicationReceiverCode, envelope.ApplicationSenderCode
		return envelope, nil
	}
	return edi.X12EnvelopeSettings{}, errortypes.NewValidationError(
		"partnerDocumentProfileId",
		errortypes.ErrInvalidOperation,
		"Partner requires at least one document profile with envelope identifiers before acknowledgments or responses can be generated",
	)
}

func (s *Service) generateExternalTenderResponse(
	ctx context.Context,
	transfer *edi.EDITransfer,
	actorID pulid.ID,
) {
	if transfer == nil || transfer.InboundMessageID.IsNil() {
		return
	}
	tenantInfo := pagination.TenantInfo{
		OrgID:  transfer.TargetOrganizationID,
		BuID:   transfer.TargetBusinessUnitID,
		UserID: actorID,
	}
	profile, err := s.EnsureOutboundDocumentProfile(
		ctx,
		tenantInfo,
		transfer.TargetPartnerID,
		edi.TransactionSet990,
	)
	if err != nil {
		s.l.Warn(
			"failed to resolve outbound 990 document profile for EDI tender response",
			zap.String("transferId", transfer.ID.String()),
			zap.Error(err),
		)
		return
	}
	payload := buildTenderResponsePayload(transfer)
	if _, err = s.GenerateDocument(ctx, &GenerateEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: profile.ID,
		EDIPartnerID:             transfer.TargetPartnerID,
		TransferID:               transfer.ID,
		TransactionSet:           edi.TransactionSet990,
		Direction:                edi.DocumentDirectionOutbound,
		Payload:                  &payload,
		GeneratedByID:            actorID,
	}); err != nil {
		s.l.Warn(
			"failed to generate outbound 990 EDI tender response",
			zap.String("transferId", transfer.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) ListPartnersCursor(
	ctx context.Context,
	req *repositories.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.ListCursor(ctx, req)
}

func (s *Service) ListCommunicationProfilesCursor(
	ctx context.Context,
	req *repositories.ListEDICommunicationProfilesRequest,
) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
	return s.profileRepo.ListProfilesCursor(ctx, req)
}

func (s *Service) ListInboundTransfersCursor(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListInboundCursor(ctx, req)
}

func (s *Service) ListOutboundTransfersCursor(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListOutboundCursor(ctx, req)
}

func (s *Service) ListMessagesCursor(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	return s.messageRepo.ListMessagesCursor(ctx, req)
}

func (s *Service) ListTemplatesCursor(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.CursorListResult[*edi.EDITemplate], error) {
	return s.templateRepo.ListTemplatesCursor(ctx, req)
}

func (s *Service) ListMappingProfilesCursor(
	ctx context.Context,
	req *repositories.ListEDIMappingProfilesRequest,
) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfilesCursor(ctx, req)
}
