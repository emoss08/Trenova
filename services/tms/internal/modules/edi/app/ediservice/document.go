package ediservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const provisionalInterchange = "000000000"

func (s *Service) ListDocumentTypes(
	ctx context.Context,
	req ediports.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return s.documentTypeRepo.ListDocumentTypes(ctx, req)
}

func (s *Service) SelectDocumentTypeOptions(
	ctx context.Context,
	req *ediports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	return s.documentTypeRepo.SelectDocumentTypeOptions(ctx, req)
}

func (s *Service) documentTypeFor(
	ctx context.Context,
	transactionSet edi.TransactionSet,
	direction edi.DocumentDirection,
) (*edi.EDIDocumentType, error) {
	documentTypes, err := s.documentTypeRepo.ListDocumentTypes(
		ctx,
		ediports.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: transactionSet,
			Direction:      direction,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(documentTypes) == 0 {
		return nil, fmt.Errorf("document type %s %s is not seeded", transactionSet, direction)
	}
	return documentTypes[0], nil
}

// outbound is one message Trenova writes: the partner's running version, the map that writes
// it and the flow clause it belongs to, and the document it writes.
type outbound struct {
	partner      *edi.EDIPartner
	program      *edilading.Program
	programs     *edilading.Programs
	set          edi.TransactionSet
	mapName      string
	flow         *lading.Flow
	clause       *lading.Clause
	event        string
	key          string
	payload      edi.DocumentPayload
	document     jsonutils.RawJSON
	documentType *edi.EDIDocumentType
	shipmentID   pulid.ID
}

func (s *Service) programs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
) (*edilading.Programs, error) {
	programs, err := s.lading.Load(ctx, tenantInfo, partner.ID)
	switch {
	case errors.Is(err, edilading.ErrNoProject):
		return nil, errortypes.NewBusinessError(
			"{0} has no Lading project; create one from the partner's EDI page",
			partner.Name,
		)
	case err != nil:
		return nil, err
	case programs.Active == nil:
		return nil, errortypes.NewBusinessError(
			"{0}'s Lading project has no active version; certify and activate one first",
			partner.Name,
		)
	}
	return programs, nil
}

func (s *Service) resolveOutbound(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*outbound, error) {
	if req == nil || req.EDIPartnerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ediPartnerId",
			errortypes.ErrRequired,
			"EDI partner is required",
		)
	}
	set := req.TransactionSet
	if set == "" && req.Payload != nil {
		set = req.Payload.TransactionSet
	}
	if set == "" {
		return nil, errortypes.NewValidationError(
			"transactionSet",
			errortypes.ErrRequired,
			"Transaction set is required",
		)
	}
	if set.IsAcknowledgment() {
		return nil, errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrRequired,
			"An acknowledgment is written from the message it answers. Choose that message.",
		)
	}
	partner, err := s.partnerRepo.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         req.EDIPartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	programs, err := s.programs(ctx, req.TenantInfo, partner)
	if err != nil {
		return nil, err
	}
	payload, err := s.resolvePayload(ctx, req, set)
	if err != nil {
		return nil, err
	}
	payload.TransactionSet = set
	o := &outbound{
		partner:    partner,
		program:    programs.Active,
		programs:   programs,
		set:        set,
		payload:    payload,
		event:      flowEventFor(req, &payload),
		shipmentID: documentShipmentID(payload),
	}
	if err = o.chooseMap(req.Map); err != nil {
		return nil, err
	}
	if o.key, o.document, err = s.documentFor(ctx, req.TenantInfo, o); err != nil {
		return nil, err
	}
	if o.documentType, err = s.documentTypeFor(ctx, set, edi.DocumentDirectionOutbound); err != nil {
		return nil, err
	}
	return o, nil
}

// chooseMap picks the map that writes the set: the one asked for, else the one the flow writes
// it with for this event, else the partner's only outbound map for the set.
func (o *outbound) chooseMap(asked string) error {
	manifest := o.program.Manifest
	set := string(o.set)
	if o.event != "" {
		o.flow, o.clause = manifest.Sending(set, o.event)
	}
	if o.clause == nil && o.set == edi.TransactionSet990 {
		o.flow, o.clause = manifest.Responding(set)
	}
	switch {
	case asked != "":
		o.mapName = asked
	case o.clause != nil && o.clause.Map != "":
		o.mapName = o.clause.Map
	default:
		for index := range manifest.Maps {
			candidate := &manifest.Maps[index]
			if candidate.Direction == lading.DirectionOutbound && candidate.Set == set {
				o.mapName = candidate.Name
				break
			}
		}
	}
	found := manifest.Map(o.mapName)
	if found == nil || found.Direction != lading.DirectionOutbound || found.Set != set {
		return errortypes.NewBusinessError(
			"{0}'s active Lading version writes no {1}",
			o.partner.Name,
			set,
		)
	}
	return nil
}

func flowEventFor(req *PreviewEDIDocumentRequest, payload *edi.DocumentPayload) string {
	switch {
	case req.InvoiceID.IsNotNil():
		return "InvoicePosted"
	case req.ServiceFailureID.IsNotNil():
		return "ServiceFailure"
	case req.ShipmentEventID.IsNotNil():
		if payload.ShipmentStatus != nil {
			return statusEvent(payload.ShipmentStatus.StatusCode)
		}
	case payload.LoadTender != nil && req.TransferID.IsNil():
		if payload.LoadTender.PurposeCode == edi.LoadTenderPurposeCancel {
			return "TenderCancelled"
		}
		return "TenderOffered"
	}
	return ""
}

func statusEvent(code string) string {
	switch ladingdoc.EventKindOf(code) {
	case "ArrivedPickup", "ArrivedDelivery":
		return "StopArrived"
	case "DepartedPickup", "DepartedDelivery":
		return "StopDeparted"
	case "Delivered":
		return "Delivered"
	case "EtaChanged":
		return "EtaChanged"
	case "Dispatched":
		return "Dispatched"
	default:
		return ""
	}
}

func (s *Service) zones(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	payload *edi.DocumentPayload,
) ladingdoc.Zones {
	zones := ladingdoc.Zones{Locations: map[pulid.ID]string{}}
	context, err := s.lading.Context(ctx, tenantInfo)
	if err == nil {
		zones.Default = context.Org.Zone
	}
	ids := make([]pulid.ID, 0)
	tender := payload.LoadTender
	if tender == nil {
		tender = payload.Shipment
	}
	if tender != nil {
		for moveIndex := range tender.Moves {
			for stopIndex := range tender.Moves[moveIndex].Stops {
				if id := tender.Moves[moveIndex].Stops[stopIndex].LocationID; id.IsNotNil() {
					ids = append(ids, id)
				}
			}
		}
	}
	if payload.ShipmentStatus != nil && payload.ShipmentStatus.LocationID.IsNotNil() {
		ids = append(ids, payload.ShipmentStatus.LocationID)
	}
	if len(ids) == 0 || s.locationRepo == nil {
		return zones
	}
	locations, err := s.locationRepo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
		TenantInfo:  tenantInfo,
		LocationIDs: ids,
	})
	if err != nil {
		s.l.Warn("failed to read location time zones for an EDI document", zap.Error(err))
		return zones
	}
	for _, location := range locations {
		zones.Locations[location.ID] = location.Timezone
	}
	return zones
}

// documentFor converts the payload to the document the map reads, and the key the partner
// knows the load by.
func (s *Service) documentFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	o *outbound,
) (string, jsonutils.RawJSON, error) {
	payload := &o.payload
	zones := s.zones(ctx, tenantInfo, payload)
	var (
		key      string
		document jsonutils.RawJSON
		err      error
	)
	switch {
	case payload.LoadTender != nil || payload.Shipment != nil:
		tender := payload.LoadTender
		if tender == nil {
			tender = payload.Shipment
		}
		key = tender.ShipmentID.String()
		if tender.TenderOfferID.IsNotNil() {
			key = tender.TenderOfferID.String()
		}
		document, err = ladingdoc.FromLoadTender(tender, key, zones)
	case payload.TenderResponse != nil:
		key = s.tenderKey(ctx, tenantInfo, payload.TenderResponse.TransferID)
		document, err = ladingdoc.FromTenderResponse(
			payload.TenderResponse,
			key,
			"",
			time.Now().Unix(),
			zones.Default,
		)
	case payload.ShipmentStatus != nil:
		status := payload.ShipmentStatus
		key = stringutils.FirstNonEmpty(status.ShipperReference, status.BOL, status.ShipmentID.String())
		document, err = ladingdoc.FromShipmentStatus(status, key, zones)
	case payload.FreightInvoice != nil:
		invoicePayload := payload.FreightInvoice
		key = stringutils.FirstNonEmpty(
			invoicePayload.ReferenceNumbers["shipmentId"],
			invoicePayload.BOL,
			invoicePayload.ShipmentID.String(),
		)
		document, err = ladingdoc.FromFreightInvoice(invoicePayload, key, zones.Default)
	default:
		return "", nil, errortypes.NewValidationError(
			"payload",
			errortypes.ErrRequired,
			"The {0} has nothing to write",
			o.set,
		)
	}
	if err != nil {
		return "", nil, errortypes.NewValidationError("payload", errortypes.ErrInvalid, err.Error())
	}
	return key, document, nil
}

// tenderKey is the partner's identifier of the load a transfer tendered.
func (s *Service) tenderKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	transferID pulid.ID,
) string {
	if transferID.IsNil() {
		return ""
	}
	transfer, err := s.transferRepo.GetTransferByID(ctx, ediports.GetEDITransferByIDRequest{
		ID:         transferID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return transferID.String()
	}
	if key, ok := transfer.TenderPayload.RatingDetail["externalShipmentId"].(string); ok && key != "" {
		return key
	}
	return transferID.String()
}

type writeOptions struct {
	control        lading.Control
	flowInstanceID pulid.ID
	flowSendID     string
}

// render writes the document with the map; with no control numbers it writes a provisional
// message for a preview.
func (s *Service) render(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	o *outbound,
	control lading.Control,
) (*lading.RunResult, error) {
	result, err := s.lading.Run(ctx, &edilading.RunRequest{
		TenantInfo: tenantInfo,
		PartnerID:  o.partner.ID,
		Program:    o.program,
		Map:        o.mapName,
		Input:      lading.DocumentInput(o.document),
		Control:    control,
		Now:        time.Now(),
		Rules:      true,
	})
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error.ValidationError("document", errortypes.ErrInvalid)
	}
	if result.Output == nil {
		return nil, errortypes.NewBusinessError("Map {0} wrote no message", o.mapName)
	}
	return result, nil
}

func outputDiagnostics(output *lading.WriteResult) []edi.Diagnostic {
	diagnostics := make(
		[]edi.Diagnostic,
		0,
		len(output.Findings)+len(output.Validation)+len(output.Outcomes),
	)
	for index := range output.Findings {
		finding := &output.Findings[index]
		diagnostics = append(diagnostics, edi.Diagnostic{
			Severity: severityOf(finding.Severity),
			Code:     finding.Code,
			Path:     finding.Path,
			Message:  finding.Message.String(),
		})
	}
	for index := range output.Validation {
		finding := &output.Validation[index]
		diagnostic := edix12inspect.FindingDiagnostic(finding, i18n.Default)
		diagnostics = append(diagnostics, edi.Diagnostic{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
		})
	}
	for index := range output.Outcomes {
		outcome := &output.Outcomes[index]
		severity := edi.ValidationSeverityError
		if outcome.Kind == lading.OutcomeWarn {
			severity = edi.ValidationSeverityWarning
		}
		diagnostics = append(diagnostics, edi.Diagnostic{
			Severity: severity,
			Code:     "lading-" + outcome.Kind,
			Path:     outcome.Path,
			Message:  stringutils.FirstNonEmpty(outcome.Rule+": "+outcome.Message, outcome.Message),
		})
	}
	return diagnostics
}

func severityOf(severity string) edi.ValidationSeverity {
	switch severity {
	case lading.SeverityError:
		return edi.ValidationSeverityError
	case lading.SeverityWarning:
		return edi.ValidationSeverityWarning
	default:
		return edi.ValidationSeverityInfo
	}
}

func (s *Service) preview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	o *outbound,
) (*EDIDocumentPreview, *lading.WriteResult, error) {
	result, err := s.render(ctx, tenantInfo, o, lading.Control{})
	if err != nil {
		return nil, nil, err
	}
	output := result.Output
	return &EDIDocumentPreview{
		PartnerName:              o.partner.Name,
		RawX12:                   output.Text,
		SegmentCount:             segmentCount(output.Transaction),
		X12Version:               interchangeVersion(o.program),
		InterchangeControlNumber: provisionalInterchange,
		GroupControlNumber:       "0",
		TransactionControlNumber: "0000",
		Diagnostics:              outputDiagnostics(output),
		LadingVersionID:          o.program.Version.ID,
		Map:                      o.mapName,
		Document:                 output.Document,
		Lineage:                  output.Lineage,
		Outcomes:                 output.Outcomes,
	}, output, nil
}

func segmentCount(transaction string) int64 {
	count := int64(0)
	for _, line := range strings.Split(transaction, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func interchangeVersion(program *edilading.Program) string {
	if program.Manifest.Partner != nil && program.Manifest.Partner.Interchange != nil &&
		program.Manifest.Partner.Interchange.Version != "" {
		return program.Manifest.Partner.Interchange.Version
	}
	return edi.DefaultX12204Version
}

func (s *Service) PreviewDocument(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
) (*EDIDocumentPreview, error) {
	if req != nil && req.SourceMessageID.IsNotNil() {
		return s.previewMessageAcknowledgment(ctx, req)
	}
	o, err := s.resolveOutbound(ctx, req)
	if err != nil {
		return nil, err
	}
	preview, _, err := s.preview(ctx, req.TenantInfo, o)
	return preview, err
}

type OutboundDocumentPreview struct {
	Document *EDIDocumentPreview
	Blocking bool
}

// PreviewOutboundDocument writes a message as it would be sent, without numbering or keeping it.
func (s *Service) PreviewOutboundDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*OutboundDocumentPreview, error) {
	o, err := s.resolveOutbound(ctx, previewRequest(req))
	if err != nil {
		return nil, err
	}
	preview, output, err := s.preview(ctx, req.TenantInfo, o)
	if err != nil {
		return nil, err
	}
	return &OutboundDocumentPreview{Document: preview, Blocking: output.Blocked()}, nil
}

// PlanGenerateDocument is the message GenerateDocument would write, refused as it would be.
func (s *Service) PlanGenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*EDIDocumentPreview, error) {
	preview, err := s.PreviewOutboundDocument(ctx, req)
	if err != nil {
		return nil, err
	}
	if preview.Blocking {
		return nil, diagnosticsToValidationError(preview.Document.Diagnostics)
	}
	return preview.Document, nil
}

func previewRequest(req *GenerateEDIDocumentRequest) *PreviewEDIDocumentRequest {
	return &PreviewEDIDocumentRequest{
		TenantInfo:       req.TenantInfo,
		Map:              req.Map,
		EDIPartnerID:     req.EDIPartnerID,
		ShipmentID:       req.ShipmentID,
		TransferID:       req.TransferID,
		InvoiceID:        req.InvoiceID,
		ShipmentEventID:  req.ShipmentEventID,
		ServiceFailureID: req.ServiceFailureID,
		SourceMessageID:  req.SourceMessageID,
		TransactionSet:   req.TransactionSet,
		Direction:        req.Direction,
		Payload:          req.Payload,
		CarrierSCAC:      req.CarrierSCAC,
	}
}

// GenerateDocument writes and keeps a message for Trenova's side of an exchange. A message a
// partner's flow sends on one of Trenova's events goes through the flow, which may keep it to
// send later with others (ediports.ErrSendDeferred); any other is written at once.
func (s *Service) GenerateDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, error) {
	if req != nil && req.SourceMessageID.IsNotNil() {
		return s.generateMessageAcknowledgment(ctx, req)
	}
	if existing, found, err := s.generatedDocument(ctx, req); err != nil || found {
		return existing, err
	}
	o, err := s.resolveOutbound(ctx, previewRequest(req))
	if err != nil {
		return nil, err
	}
	if o.flow != nil && o.clause != nil && o.clause.Kind == lading.ClauseSend && req.Map == "" {
		return s.sendThroughFlow(ctx, req, o)
	}
	return s.write(ctx, req, o, &writeOptions{})
}

func (s *Service) sendThroughFlow(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
	o *outbound,
) (*edi.EDIMessage, error) {
	dispatched, err := s.flows.Dispatch(ctx, &edilading.DispatchRequest{
		TenantInfo:  req.TenantInfo,
		PartnerID:   o.partner.ID,
		Program:     o.program,
		Flow:        o.flow.Name,
		FallbackKey: o.key,
		Event: lading.FlowEvent{
			Kind:     lading.EventTrenova,
			Event:    o.event,
			Document: o.document,
		},
		ShipmentID: o.shipmentID,
		TransferID: req.TransferID,
		ActorID:    req.GeneratedByID,
	})
	if err != nil {
		return nil, err
	}
	sent, err := s.ExecuteFlowActions(ctx, req.TenantInfo, dispatched, &FlowActionContext{
		GeneratedByID:        req.GeneratedByID,
		DisableDeliveryQueue: req.DisableDeliveryQueue,
		IdempotencyKey:       req.IdempotencyKey,
		Payload:              &o.payload,
	})
	if err != nil {
		return nil, err
	}
	for _, message := range sent {
		if message.TransactionSet == o.set {
			return message, nil
		}
	}
	return nil, ediports.ErrSendDeferred
}

// write numbers, writes and keeps one outbound message.
func (s *Service) write(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
	o *outbound,
	options *writeOptions,
) (*edi.EDIMessage, error) {
	allocated, err := s.controlNumberRepo.AllocateControlNumbers(
		ctx,
		ediports.AllocateEDIControlNumbersRequest{
			TenantInfo:     req.TenantInfo,
			PartnerID:      o.partner.ID,
			DocumentTypeID: o.documentType.ID,
			Kinds: []edi.ControlNumberKind{
				edi.ControlNumberKindInterchange,
				edi.ControlNumberKindGroup,
				edi.ControlNumberKindTransaction,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	control := lading.Control{
		Interchange: uint64(allocated[edi.ControlNumberKindInterchange]),
		Group:       uint64(allocated[edi.ControlNumberKindGroup]),
		Transaction: uint64(allocated[edi.ControlNumberKindTransaction]),
	}
	result, err := s.render(ctx, req.TenantInfo, o, control)
	if err != nil {
		return nil, err
	}
	output := result.Output
	diagnostics := outputDiagnostics(output)
	if output.Blocked() {
		return nil, diagnosticsToValidationError(diagnostics)
	}
	shipmentID := o.shipmentID
	if shipmentID.IsNil() {
		shipmentID = req.ShipmentID
	}
	message := &edi.EDIMessage{
		BusinessUnitID:           req.TenantInfo.BuID,
		OrganizationID:           req.TenantInfo.OrgID,
		EDIPartnerID:             o.partner.ID,
		DocumentTypeID:           o.documentType.ID,
		LadingVersionID:          o.program.Version.ID,
		FlowInstanceID:           options.flowInstanceID,
		FlowSendID:               options.flowSendID,
		ShipmentID:               shipmentID,
		TransferID:               req.TransferID,
		InvoiceID:                req.InvoiceID,
		Direction:                edi.DocumentDirectionOutbound,
		Standard:                 edi.EDIStandardX12,
		TransactionSet:           o.set,
		X12Version:               interchangeVersion(o.program),
		Status:                   edi.MessageStatusGenerated,
		InterchangeControlNumber: fmt.Sprintf("%09d", control.Interchange),
		GroupControlNumber:       strconv.FormatUint(control.Group, 10),
		TransactionControlNumber: fmt.Sprintf("%04d", control.Transaction),
		SegmentCount:             segmentCount(output.Transaction),
		RawX12:                   output.Text,
		PayloadSnapshot:          o.payload,
		Document:                 output.Document,
		Lineage:                  output.Lineage,
		Outcomes:                 output.Outcomes,
		AckStatus:                ackStatusFor(o),
		GeneratedByID:            req.GeneratedByID,
		IdempotencyKey:           req.IdempotencyKey,
	}
	created, err := s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		ediports.CreateEDIMessageWithDiagnosticsRequest{
			Message:     message,
			Diagnostics: messageDiagnostics(diagnostics),
		},
	)
	if err != nil {
		if req.IdempotencyKey != "" && dberror.IsUniqueConstraintViolation(err) {
			existing, found, lookupErr := s.generatedDocument(ctx, req)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if found {
				return existing, nil
			}
		}
		return nil, err
	}
	s.shadowOutbound(ctx, req.TenantInfo, o, created)
	if !req.SuppressTenderRecipientUpsert {
		if err = s.upsertExternalTenderRecipient(ctx, created, o.partner); err != nil {
			return nil, err
		}
	}
	if !req.DisableDeliveryQueue {
		if err = s.queueMessageForDelivery(ctx, created); err != nil {
			s.l.Warn(
				"failed to queue EDI message for delivery",
				zap.String("messageId", created.ID.String()),
				zap.Error(err),
			)
		}
	}
	return created, nil
}

func (s *Service) shadowOutbound(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	o *outbound,
	message *edi.EDIMessage,
) {
	if s.replayer == nil || o.programs == nil || o.programs.Shadow == nil {
		return
	}
	s.replayer.Shadow(ctx, &edilading.ShadowRunRequest{
		TenantInfo: tenantInfo,
		Programs:   o.programs,
		Map:        o.mapName,
		MessageID:  message.ID,
		Set:        o.set,
		Input:      lading.DocumentInput(o.document),
		Now:        time.Now(),
	})
}

// ackStatusFor is pending when the flow that sends the set expects the partner to acknowledge it.
func ackStatusFor(o *outbound) edi.MessageAcknowledgmentStatus {
	manifest := o.program.Manifest
	for index := range manifest.Flows {
		flow := &manifest.Flows[index]
		if manifest.Expecting(flow.Name) == nil {
			continue
		}
		for clause := range flow.Clauses {
			candidate := &flow.Clauses[clause]
			if (candidate.Kind == lading.ClauseSend || candidate.Kind == lading.ClauseRespond) &&
				candidate.Set == string(o.set) {
				return edi.MessageAcknowledgmentStatusPending
			}
		}
	}
	return edi.MessageAcknowledgmentStatusNotExpected
}

func messageDiagnostics(diagnostics []edi.Diagnostic) []*edi.EDIMessageValidationError {
	out := make([]*edi.EDIMessageValidationError, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, &edi.EDIMessageValidationError{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	return out
}

func (s *Service) generatedDocument(
	ctx context.Context,
	req *GenerateEDIDocumentRequest,
) (*edi.EDIMessage, bool, error) {
	if req.IdempotencyKey == "" {
		return nil, false, nil
	}

	existing, err := s.messageRepo.GetMessageByIdempotencyKey(
		ctx,
		ediports.GetEDIMessageByIdempotencyKeyRequest{
			TenantInfo:     req.TenantInfo,
			IdempotencyKey: req.IdempotencyKey,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if existing.TransactionSet != req.TransactionSet {
		return nil, false, errortypes.NewBusinessError(
			"This idempotency key already names a {0} document, not a {1}",
			existing.TransactionSet,
			req.TransactionSet,
		)
	}

	if !req.DisableDeliveryQueue {
		if err = s.queueMessageForDelivery(ctx, existing); err != nil {
			s.l.Warn(
				"failed to queue EDI message for delivery",
				zap.String("messageId", existing.ID.String()),
				zap.Error(err),
			)
		}
	}

	return existing, true, nil
}

func (s *Service) ListMessages(
	ctx context.Context,
	req *ediports.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	return s.messageRepo.ListMessages(ctx, req)
}

func (s *Service) GetMessage(
	ctx context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	return s.messageRepo.GetMessageByID(ctx, req)
}

func (s *Service) InspectX12(
	ctx context.Context,
	req *InspectX12Request,
) (*edix12inspect.InspectX12Result, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"inspection",
			errortypes.ErrRequired,
			"Inspection request is required",
		)
	}
	return edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         req.RawX12,
		TransactionSet: req.TransactionSet,
		X12Version:     req.X12Version,
		Envelope:       req.Envelope,
		Diagnostics:    req.Diagnostics,
	})
}

func (s *Service) InspectMessage(
	ctx context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*EDIMessageInspection, error) {
	message, err := s.messageRepo.GetMessageByID(ctx, req)
	if err != nil {
		return nil, err
	}
	diagnostics := make([]edi.Diagnostic, 0, len(message.ValidationErrors))
	for _, diagnostic := range message.ValidationErrors {
		if strings.HasPrefix(diagnostic.Code, edix12inspect.FindingCodePrefix) {
			continue
		}
		diagnostics = append(diagnostics, edi.Diagnostic{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	inspection, err := edix12inspect.InspectX12(ctx, &edix12inspect.InspectX12Request{
		RawX12:         message.RawX12,
		TransactionSet: message.TransactionSet,
		X12Version:     message.X12Version,
		Diagnostics:    diagnostics,
	})
	if err != nil {
		return nil, err
	}
	return &EDIMessageInspection{
		Message:    message,
		Inspection: *inspection,
		Provenance: EDIInspectionProvenance{
			MessageID:       message.ID,
			LadingVersionID: message.LadingVersionID,
			FlowInstanceID:  message.FlowInstanceID,
			GeneratedAt:     message.GeneratedAt,
			GeneratedByID:   message.GeneratedByID,
		},
	}, nil
}

//nolint:cyclop,funlen,gocognit // Payload resolution is explicit per transaction set to preserve validation behavior.
func (s *Service) resolvePayload(
	ctx context.Context,
	req *PreviewEDIDocumentRequest,
	transactionSet edi.TransactionSet,
) (edi.DocumentPayload, error) {
	if req.Payload != nil {
		payload := *req.Payload
		payload.Normalize()
		return payload, nil
	}
	if !req.TransferID.IsNil() {
		if transactionSet != edi.TransactionSet204 && transactionSet != edi.TransactionSet990 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"transferId",
				"transfer",
				transactionSet,
				edi.TransactionSet204,
				edi.TransactionSet990,
			)
		}
		transfer, err := s.transferRepo.GetTransferByID(ctx, ediports.GetEDITransferByIDRequest{
			ID:         req.TransferID,
			TenantInfo: req.TenantInfo,
			Direction:  "outbound",
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if transactionSet == edi.TransactionSet990 {
			return buildTenderResponsePayload(transfer), nil
		}
		return edi.NewLoadTenderDocumentPayload(transfer.TenderPayload), nil
	}
	if !req.InvoiceID.IsNil() {
		if transactionSet != edi.TransactionSet210 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"invoiceId",
				"invoice",
				transactionSet,
				edi.TransactionSet210,
			)
		}
		invoiceEntity, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
			ID:         req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if invoiceEntity.Status == invoice.StatusVoided {
			return edi.DocumentPayload{}, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"Invoice {0} has been voided and cannot be sent as a freight invoice",
				invoiceEntity.Number,
			)
		}
		return buildFreightInvoicePayload(invoiceEntity), nil
	}
	if !req.ShipmentEventID.IsNil() {
		if transactionSet != edi.TransactionSet214 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"shipmentEventId",
				"shipment event",
				transactionSet,
				edi.TransactionSet214,
			)
		}
		event, err := s.shipmentEventRepo.GetByID(
			ctx,
			shipmenteventports.GetShipmentEventByIDRequest{
				ID:         req.ShipmentEventID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		if req.ShipmentID.IsNotNil() && req.ShipmentID != event.ShipmentID {
			return edi.DocumentPayload{}, errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalidReference,
				"Shipment ID must match the shipment event",
			)
		}
		source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:         event.ShipmentID,
			TenantInfo: req.TenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return buildShipmentEventStatusPayload(event, source), nil
	}
	if !req.ServiceFailureID.IsNil() {
		if transactionSet != edi.TransactionSet214 {
			return edi.DocumentPayload{}, sourceTransactionSetError(
				"serviceFailureId",
				"service failure",
				transactionSet,
				edi.TransactionSet214,
			)
		}
		result, err := s.BuildShipmentStatusPayloadForServiceFailure(
			ctx,
			&servicefailureports.BuildServiceFailureEDIPayloadRequest{
				TenantInfo:       req.TenantInfo,
				ServiceFailureID: req.ServiceFailureID,
			},
		)
		if err != nil {
			return edi.DocumentPayload{}, err
		}
		return result.Payload, nil
	}
	if req.ShipmentID.IsNil() {
		return edi.DocumentPayload{}, missingSourceError(transactionSet)
	}
	if transactionSet != edi.TransactionSet204 && transactionSet != edi.TransactionSet214 {
		return edi.DocumentPayload{}, sourceTransactionSetError(
			"shipmentId",
			"shipment",
			transactionSet,
			edi.TransactionSet204,
			edi.TransactionSet214,
		)
	}
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return edi.DocumentPayload{}, err
	}
	if transactionSet == edi.TransactionSet214 {
		return buildShipmentStatusPayload(source), nil
	}
	return edi.NewLoadTenderDocumentPayload(buildTenderPayload(source)), nil

}

func sourceTransactionSetError(
	field string,
	source string,
	actual edi.TransactionSet,
	allowed ...edi.TransactionSet,
) error {
	allowedValues := make([]string, 0, len(allowed))
	for _, transactionSet := range allowed {
		allowedValues = append(allowedValues, string(transactionSet))
	}
	return errortypes.NewValidationError(
		field,
		errortypes.ErrInvalidReference,
		"{0} source cannot be used with transaction set {1}; expected {2}",
		source,
		actual,
		strings.Join(allowedValues, " or "),
	)
}

func missingSourceError(transactionSet edi.TransactionSet) error {
	//nolint:exhaustive // Only transaction sets with source-specific validation need custom errors.
	switch transactionSet {
	case edi.TransactionSet210:
		return errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrRequired,
			"Invoice ID or payload is required for 210 documents",
		)
	case edi.TransactionSet214:
		return errortypes.NewValidationError(
			"shipmentEventId",
			errortypes.ErrRequired,
			"Shipment event ID, shipment ID, or payload is required for 214 documents",
		)
	case edi.TransactionSet990:
		return errortypes.NewValidationError(
			"transferId",
			errortypes.ErrRequired,
			"Transfer ID or payload is required for 990 documents",
		)
	case edi.TransactionSet997, edi.TransactionSet999:
		return errortypes.NewValidationError(
			"sourceMessageId",
			errortypes.ErrRequired,
			"Source message ID or payload is required for acknowledgment documents",
		)
	default:
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment, transfer, invoice, shipment event, source message, or payload is required",
		)
	}
}

func documentShipmentID(payload edi.DocumentPayload) pulid.ID {
	payload.Normalize()
	if payload.Shipment != nil {
		return payload.Shipment.ShipmentID
	}
	if payload.ShipmentStatus != nil {
		return payload.ShipmentStatus.ShipmentID
	}
	if payload.TenderResponse != nil {
		return payload.TenderResponse.ShipmentID
	}
	if payload.FreightInvoice != nil {
		return payload.FreightInvoice.ShipmentID
	}
	return pulid.Nil
}

func diagnosticsToValidationError(diagnostics []edi.Diagnostic) error {
	multiErr := errortypes.NewMultiError()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		field := stringutils.FirstNonEmpty(diagnostic.Path, diagnostic.SegmentID, "edi")
		multiErr.Add(field, errortypes.ErrInvalid, diagnostic.Message)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
