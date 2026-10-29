package ediinboundapply

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
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
	payload *editransfer.LoadTenderPayload,
) (*editendermapping.MappingPreview, error) {
	return s.tenderMapping.BuildMappingPreview(ctx, partner, *payload)
}

type ApplyExternalTenderResponseRequest struct {
	Recipient       *editenderchange.TenderRecipient
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
	tenderStatus := shipmentvocab.TenderStatusAccepted
	comment := "EDI load tender accepted by trading partner."
	if !req.Accepted {
		tenderStatus = shipmentvocab.TenderStatusRejected
		reason := stringutils.FirstNonEmpty(req.RejectionReason, req.ReasonCode)
		comment = "EDI load tender rejected by trading partner."
		if reason != "" {
			comment = "EDI load tender rejected by trading partner: " + reason
		}
	}
	if recipient.SourceShipmentID.IsNotNil() {
		if err := s.shipments.SetTenderStatus(
			ctx,
			recipient.SourceShipmentID,
			tenantInfo,
			tenderStatus,
		); err != nil {
			return err
		}
		if err := s.shipments.CreateSystemComment(
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
		recipient.BaselineStatus = editenderchange.TenderRecipientBaselineStatusAccepted
	} else {
		recipient.Status = editenderchange.TenderRecipientStatusClosed
	}
	recipient.BaselineRecordedAt = timeutils.NowUnix()
	_, err := s.tenderRecipientRepo.UpdateTenderRecipient(ctx, recipient)
	return err
}

type RecordExternalShipmentStatusRequest struct {
	Recipient   *editenderchange.TenderRecipient
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
	return s.shipments.CreateSystemComment(
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
	Event        *edipayload.ShipmentStatusEvent
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
	if err = s.shipments.CreateSystemComment(
		ctx,
		shipmentID,
		req.TenantInfo,
		comment,
		metadata,
	); err != nil {
		return nil, err
	}
	s.recordTenderedShipmentStatusEvent(ctx, req, shipmentID)
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
) {
	offer := req.Offer
	if s.shipmentEventRepo == nil {
		return
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
