package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/dbscope"
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
		BuID:   recipient.BusinessUnitID,
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
		BuID:  req.Recipient.BusinessUnitID,
	}
	metadata := map[string]any{"recipientId": req.Recipient.ID}
	comment := externalShipmentStatusComment(externalShipmentStatusCommentParams{
		StatusCode:  req.StatusCode,
		ReasonCode:  req.ReasonCode,
		ReferenceID: req.ReferenceID,
		EventAt:     req.EventAt,
	}, metadata)
	return s.createSystemShipmentComment(
		ctx,
		req.Recipient.SourceShipmentID,
		tenantInfo,
		comment,
		metadata,
	)
}

type externalShipmentStatusCommentParams struct {
	StatusCode  string
	ReasonCode  string
	ReferenceID string
	EventAt     int64
}

// externalShipmentStatusComment builds the shared 214 comment wording and
// stamps the status details into metadata so recipient-matched and
// offer-matched statuses read identically.
func externalShipmentStatusComment(
	params externalShipmentStatusCommentParams,
	metadata map[string]any,
) string {
	comment := "EDI 214 shipment status received from trading partner: " + params.StatusCode
	if description := externalShipmentStatusDescription(params.StatusCode); description != "" {
		comment += " (" + description + ")"
	}
	if params.ReasonCode != "" {
		comment += ", reason " + params.ReasonCode
	}
	metadata["statusCode"] = params.StatusCode
	if params.ReasonCode != "" {
		metadata["reasonCode"] = params.ReasonCode
	}
	if params.ReferenceID != "" {
		metadata["referenceId"] = params.ReferenceID
	}
	if params.EventAt > 0 {
		metadata["eventAt"] = params.EventAt
	}
	return comment
}

type RecordTenderedShipmentStatusRequest struct {
	TenantInfo   pagination.TenantInfo
	Offer        *tender.TenderOffer
	StatusCode   string
	ReasonCode   string
	ReferenceID  string
	EventAt      int64
	Event        *edi.ShipmentStatusEvent
	RawReference string
}

// RecordTenderedShipmentStatus applies a carrier 214 that matched an accepted
// tender offer instead of a tender recipient row. The shipment comment and
// shipment event are always written — no recipient row is created and the
// shipment's tender status is never touched. Status codes that carry an
// arrival/departure meaning additionally drive the move's stop actuals;
// business rejections from that pipeline surface as warnings rather than
// failing the inbound routing.
func (s *Service) RecordTenderedShipmentStatus(
	ctx context.Context,
	req *RecordTenderedShipmentStatusRequest,
) ([]string, error) {
	if req == nil || req.Offer == nil || req.Offer.Tender == nil ||
		req.Offer.Tender.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"offer",
			errortypes.ErrRequired,
			"An accepted tender offer with its tender is required for status processing",
		)
	}
	target, warnings, err := s.resolveTendered214Target(ctx, req)
	if err != nil {
		return nil, err
	}
	if target != nil {
		resolved := *req
		resolved.EventAt = target.eventAt
		req = &resolved
	}

	offer := req.Offer
	shipmentID := offer.Tender.ShipmentID
	metadata := map[string]any{
		"tenderOfferId": offer.ID,
		"tenderId":      offer.TenderID,
		"carrierId":     offer.CarrierID,
	}
	comment := externalShipmentStatusComment(externalShipmentStatusCommentParams{
		StatusCode:  req.StatusCode,
		ReasonCode:  req.ReasonCode,
		ReferenceID: req.ReferenceID,
		EventAt:     req.EventAt,
	}, metadata)
	if err = s.createSystemShipmentComment(
		ctx,
		shipmentID,
		req.TenantInfo,
		comment,
		metadata,
	); err != nil {
		return nil, err
	}
	if err = s.recordTenderedShipmentStatusEvent(ctx, req, shipmentID); err != nil {
		return nil, err
	}
	applied, err := s.applyTendered214StopActuals(ctx, req, target)
	if err != nil {
		return nil, err
	}
	return append(warnings, applied...), nil
}

func (s *Service) recordTenderedShipmentStatusEvent(
	ctx context.Context,
	req *RecordTenderedShipmentStatusRequest,
	shipmentID pulid.ID,
) error {
	offer := req.Offer
	if s.shipmentEventRepo == nil {
		return nil
	}
	severity := shipmentevent.SeverityInfo
	if req.StatusCode == "A7" {
		severity = shipmentevent.SeverityDanger
	}
	actorLabel := "Carrier EDI"
	if offer.Carrier != nil && offer.Carrier.Name != "" {
		actorLabel = offer.Carrier.Name
	}
	occurredAt := req.EventAt
	if occurredAt <= 0 {
		occurredAt = timeutils.NowUnix()
	}
	eventMetadata := map[string]any{
		"tenderOfferId": offer.ID.String(),
		"tenderId":      offer.TenderID.String(),
		"carrierId":     offer.CarrierID.String(),
		"statusCode":    req.StatusCode,
	}
	if req.ReasonCode != "" {
		eventMetadata["reasonCode"] = req.ReasonCode
	}
	if err := s.shipmentEventRepo.Insert(ctx, &shipmentevent.Event{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     shipmentID,
		Type:           shipmentevent.TypeStatusChanged,
		Severity:       severity,
		ActorType:      shipmentevent.ActorEDI,
		ActorLabel:     actorLabel,
		Summary:        "Carrier reported EDI 214 status " + req.StatusCode,
		Metadata:       eventMetadata,
		CorrelationID:  offer.TenderID.String(),
		OccurredAt:     occurredAt,
	}); err != nil {
		s.l.Warn(
			"failed to record tendered carrier shipment status event",
			zap.String("tenderOfferId", offer.ID.String()),
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err),
		)
	}
	return nil
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
	case "CD":
		return "departed delivery location"
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
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	taken, _, err := s.DecideTender(ctx, &DecideTenderRequest{
		TenantInfo: tenantInfo,
		TransferID: transfer.ID,
		Accept:     transfer.Status == edi.TransferStatusApproved,
		Reason:     transfer.RejectionReason,
		ActorID:    actorID,
	})
	if err != nil {
		s.l.Warn(
			"the partner's flow failed to answer the EDI tender",
			zap.String("transferId", transfer.ID.String()),
			zap.Error(err),
		)
	}
	if taken {
		return
	}
	if _, err = s.GenerateDocument(
		ctx,
		tenderResponseRequest(tenantInfo, transfer, actorID),
	); err != nil {
		s.l.Warn(
			"failed to generate outbound 990 EDI tender response",
			zap.String("transferId", transfer.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) ListPartnersCursor(
	ctx context.Context,
	req *edipartnerports.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	return s.partnerRepo.ListCursor(ctx, req)
}

func (s *Service) ListCommunicationProfilesCursor(
	ctx context.Context,
	req *edipartnerports.ListEDICommunicationProfilesRequest,
) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
	return s.profileRepo.ListProfilesCursor(ctx, req)
}

func (s *Service) ListInboundTransfersCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListInboundCursor(ctx, req)
}

func (s *Service) ListOutboundTransfersCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return s.transferRepo.ListOutboundCursor(ctx, req)
}

func (s *Service) ListMessagesCursor(
	ctx context.Context,
	req *edimessageports.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	return s.messageRepo.ListMessagesCursor(ctx, req)
}

func (s *Service) ListMappingProfilesCursor(
	ctx context.Context,
	req *edipartnerports.ListEDIMappingProfilesRequest,
) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
	return s.mappingProfileRepo.ListMappingProfilesCursor(ctx, req)
}
