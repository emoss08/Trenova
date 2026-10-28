package ediservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	serviceFailure214TriggerReference = "serviceFailure214Trigger"
	serviceFailureEvent               = "ServiceFailure"
)

type serviceFailure214Candidate struct {
	partner *edi.EDIPartner
}

func (s *Service) PreviewServiceFailure214ForLifecycle(
	ctx context.Context,
	req *ediports.ServiceFailure214LifecycleRequest,
) (*servicefailureports.ServiceFailure214LifecycleResult, error) {
	return s.serviceFailure214Lifecycle(ctx, req, false)
}

func (s *Service) GenerateServiceFailure214ForLifecycle(
	ctx context.Context,
	req *ediports.ServiceFailure214LifecycleRequest,
) (*servicefailureports.ServiceFailure214LifecycleResult, error) {
	return s.serviceFailure214Lifecycle(ctx, req, true)
}

func (s *Service) serviceFailure214Lifecycle(
	ctx context.Context,
	req *ediports.ServiceFailure214LifecycleRequest,
	generate bool,
) (*servicefailureports.ServiceFailure214LifecycleResult, error) {
	if err := validateServiceFailure214LifecycleRequest(req); err != nil {
		return nil, err
	}

	existing, err := s.messageRepo.GetServiceFailure214LifecycleMessage(
		ctx,
		edimessageports.GetServiceFailure214LifecycleMessageRequest{
			TenantInfo:       req.TenantInfo,
			ServiceFailureID: req.ServiceFailureID,
			Trigger:          string(req.Trigger),
		},
	)
	if err == nil {
		return duplicateServiceFailure214Result(req.Trigger, existing), nil
	}
	if !dberror.IsNotFoundError(err) {
		return nil, err
	}

	failure, source, err := s.serviceFailure214Source(ctx, req)
	if err != nil {
		return nil, err
	}
	candidate, result, err := s.resolveServiceFailure214Candidate(ctx, req, source)
	if err != nil || result != nil {
		return result, err
	}

	payload := buildServiceFailure214LifecyclePayload(failure, source, req.Trigger)
	if err = s.applyServiceFailure214ReasonMapping(
		ctx,
		req.TenantInfo,
		candidate.partner.ID,
		failure,
		payload.ShipmentStatus,
	); err != nil {
		return nil, err
	}
	diagnostics := serviceFailurePayloadDiagnostics(payload.ShipmentStatus)
	result = &servicefailureports.ServiceFailure214LifecycleResult{
		Trigger:      req.Trigger,
		Action:       ediports.ServiceFailureEDIActionSkipped,
		EDIPartnerID: candidate.partner.ID,
		Mandatory:    true,
		Diagnostics:  diagnostics,
	}
	if hasErrorDiagnostics(diagnostics) {
		result.Action = ediports.ServiceFailureEDIActionBlocked
		result.SkippedReason = "service failure 214 payload has blocking diagnostics"
		return result, nil
	}
	request := &ediports.GenerateEDIDocumentRequest{
		TenantInfo:       req.TenantInfo,
		EDIPartnerID:     candidate.partner.ID,
		ShipmentID:       failure.ShipmentID,
		ServiceFailureID: failure.ID,
		TransactionSet:   edi.TransactionSet214,
		Direction:        edi.DocumentDirectionOutbound,
		Payload:          &payload,
		GeneratedByID:    req.GeneratedByID,
	}
	preview, err := s.PreviewOutboundDocument(ctx, request)
	if err != nil {
		return nil, err
	}
	result.Diagnostics = append(result.Diagnostics, preview.Document.Diagnostics...)
	if preview.Blocking {
		result.Action = ediports.ServiceFailureEDIActionBlocked
		result.SkippedReason = "the partner's 214 map or rules refuse this service failure"
		return result, nil
	}
	if !generate {
		result.Action = ediports.ServiceFailureEDIActionSkipped
		result.SkippedReason = "ready_for_generation"
		return result, nil
	}

	message, err := s.GenerateDocument(ctx, request)
	if errors.Is(err, ediports.ErrSendDeferred) {
		result.Action = ediports.ServiceFailureEDIActionQueued
		result.SkippedReason = "the partner's flow sends it with its next batch"
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	result.Action = ediports.ServiceFailureEDIActionGenerated
	result.SkippedReason = ""
	result.MessageID = message.ID
	return result, nil
}

func (s *Service) GetServiceFailure214Status(
	ctx context.Context,
	req edimessageports.GetServiceFailure214StatusRequest,
) (*ediports.ServiceFailure214Status, error) {
	return s.messageRepo.GetServiceFailure214Status(ctx, req)
}

func (s *Service) applyServiceFailure214ReasonMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	failure *servicefailure.ServiceFailure,
	payload *edi.ShipmentStatusPayload,
) error {
	if payload == nil ||
		failure == nil ||
		strings.TrimSpace(failure.X12ReasonCodeOverride) != "" ||
		failure.ReasonCodeID == nil ||
		failure.ReasonCodeID.IsNil() {
		return nil
	}
	items, err := s.mappingProfileRepo.GetMappingItems(ctx, edipartnerports.GetMappingItemsRequest{
		PartnerID:   partnerID,
		TenantInfo:  tenantInfo,
		EntityTypes: []edi.MappingEntityType{edi.MappingEntityTypeServiceFailureReasonCode},
		SourceIDs:   []pulid.ID{*failure.ReasonCodeID},
	})
	if err != nil {
		return err
	}
	for _, item := range items {
		if item == nil || item.SourceID != *failure.ReasonCodeID {
			continue
		}
		reasonCode := normalizedX12Code(item.TargetID.String())
		if reasonCode == "" {
			continue
		}
		payload.StatusReasonCode = reasonCode
		payload.ReasonCode = reasonCode
		if strings.TrimSpace(item.TargetLabel) != "" {
			payload.ReasonDescription = strings.TrimSpace(item.TargetLabel)
		}
		return nil
	}
	return nil
}

func validateServiceFailure214LifecycleRequest(
	req *ediports.ServiceFailure214LifecycleRequest,
) error {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add(
			"request",
			errortypes.ErrRequired,
			"Service failure EDI lifecycle request is required",
		)
		return multiErr
	}
	if req.TenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if req.TenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if req.ServiceFailureID.IsNil() {
		multiErr.Add("serviceFailureId", errortypes.ErrRequired, "Service failure ID is required")
	}
	if req.ShipmentID.IsNil() {
		multiErr.Add("shipmentId", errortypes.ErrRequired, "Shipment ID is required")
	}
	if !req.Trigger.IsValid() {
		multiErr.Add("trigger", errortypes.ErrInvalid, "Service failure EDI trigger is invalid")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) serviceFailure214Source(
	ctx context.Context,
	req *ediports.ServiceFailure214LifecycleRequest,
) (*servicefailure.ServiceFailure, *shipment.Shipment, error) {
	failure := req.ServiceFailure
	if failure == nil {
		var err error
		failure, err = s.serviceFailureRepo.GetByID(
			ctx,
			&servicefailureports.GetServiceFailureByIDRequest{
				ID:         req.ServiceFailureID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			return nil, nil, err
		}
	}
	if failure.ShipmentID != req.ShipmentID {
		return nil, nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalidReference,
			"Shipment ID must match the service failure",
		)
	}
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         failure.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return failure, source, nil
}

func (s *Service) resolveServiceFailure214Candidate(
	ctx context.Context,
	req *ediports.ServiceFailure214LifecycleRequest,
	source *shipment.Shipment,
) (*serviceFailure214Candidate, *servicefailureports.ServiceFailure214LifecycleResult, error) {
	if source == nil || source.CustomerID.IsNil() {
		return nil, skippedServiceFailure214Result(
			req.Trigger,
			"shipment customer is not linked to an EDI partner",
		), nil
	}
	partners, err := s.partnerRepo.List(ctx, &edipartnerports.ListEDIPartnersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: pagination.MaxLimit},
		},
		CustomerID: source.CustomerID,
	})
	if err != nil {
		return nil, nil, err
	}
	if partners.Total == 0 {
		return nil, skippedServiceFailure214Result(
			req.Trigger,
			"no outbound EDI partner for shipment customer",
		), nil
	}

	candidates := make([]serviceFailure214Candidate, 0, len(partners.Items))
	activeOutboundPartners := 0
	capabilityDisabled := false
	noVersion := false
	flowSilent := false
	for _, partner := range partners.Items {
		if partner == nil {
			continue
		}
		if partner.Status != domaintypes.StatusActive || !partner.EnabledForOutbound {
			continue
		}
		activeOutboundPartners++
		enabled, err := s.shipmentStatusCapabilityEnabled(ctx, req.TenantInfo, partner)
		if err != nil {
			return nil, nil, err
		}
		if !enabled {
			capabilityDisabled = true
			continue
		}
		program, programErr := s.lading.Active(ctx, req.TenantInfo, partner.ID)
		if errors.Is(programErr, edilading.ErrNoProject) ||
			errors.Is(programErr, edilading.ErrNoActiveVersion) {
			noVersion = true
			continue
		}
		if programErr != nil {
			return nil, nil, programErr
		}
		if _, clause := program.Manifest.Sending(
			string(edi.TransactionSet214),
			serviceFailureEvent,
		); clause == nil {
			flowSilent = true
			continue
		}
		candidates = append(candidates, serviceFailure214Candidate{partner: partner})
	}

	switch len(candidates) {
	case 0:
		reason := "the partner's flow sends no 214 on a service failure"
		switch {
		case activeOutboundPartners == 0:
			reason = "EDI partner is inactive or outbound disabled"
		case capabilityDisabled:
			reason = "shipment status capability disabled"
		case noVersion && !flowSilent:
			reason = "the partner has no active Lading version"
		}
		return nil, skippedServiceFailure214Result(req.Trigger, reason), nil
	case 1:
		return &candidates[0], nil, nil
	default:
		result := skippedServiceFailure214Result(
			req.Trigger,
			"more than one of the customer's EDI partners sends a 214 on a service failure",
		)
		result.Action = ediports.ServiceFailureEDIActionBlocked
		result.Mandatory = true
		return nil, result, nil
	}
}

func (s *Service) shipmentStatusCapabilityEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
) (bool, error) {
	connection := partner.Connection
	if connection == nil && partner.EDIConnectionID.IsNotNil() {
		var err error
		connection, err = s.connectionRepo.GetConnectionByID(
			ctx,
			edipartnerports.GetEDIConnectionByIDRequest{
				ID:         partner.EDIConnectionID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return true, nil
			}
			return false, err
		}
	}
	if connection == nil {
		return true, nil
	}
	return connection.Capabilities.ShipmentStatus, nil
}

func buildServiceFailure214LifecyclePayload(
	failure *servicefailure.ServiceFailure,
	source *shipment.Shipment,
	trigger servicefailureports.ServiceFailureEDITrigger,
) edi.DocumentPayload {
	payload := buildServiceFailureShipmentStatusPayload(failure, source)
	if payload.ShipmentStatus == nil {
		return payload
	}
	if payload.ShipmentStatus.References == nil {
		payload.ShipmentStatus.References = map[string]string{}
	}
	payload.ShipmentStatus.References[serviceFailure214TriggerReference] = string(trigger)
	payload.ShipmentStatus.References["serviceFailureStatus"] = string(failure.Status)
	return payload
}

func normalizedX12Code(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func serviceFailure214Diagnostics(payload *edi.ShipmentStatusPayload) []edi.Diagnostic {
	if payload == nil {
		return []edi.Diagnostic{requiredServiceFailure214Diagnostic(
			"shipmentStatus",
			"EDI 214 shipment status payload is required",
			"Build the shipment status payload before generating the 214.",
		)}
	}
	diagnostics := make([]edi.Diagnostic, 0, 6)
	statusCode := normalizedX12Code(payload.StatusCode)
	reasonCode := normalizedX12Code(payload.StatusReasonCode)
	if payload.EventDate == 0 {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.eventDate",
			"Service failure 214 event date is required",
			"Set the service failure detection, creation, or actual arrival timestamp before generating the 214.",
		))
	}
	if payload.EventTime == 0 {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.eventTime",
			"Service failure 214 event time is required",
			"Set the service failure detection, creation, or actual arrival timestamp before generating the 214.",
		))
	}
	if statusCode == "SD" && reasonCode == "" {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.statusReasonCode",
			"X12 214 service failure status code SD requires a status reason code",
			"Set an override reason code or configure a default reason code on the service failure reason code.",
		))
	}
	return diagnostics
}

func requiredServiceFailure214Diagnostic(path, message, suggestedFix string) edi.Diagnostic {
	diagnostic := edi.Diagnostic{
		Severity:     edi.ValidationSeverityError,
		Code:         "required",
		SegmentID:    "AT7",
		Path:         path,
		Message:      message,
		SuggestedFix: suggestedFix,
	}
	if path == "shipmentStatus.statusReasonCode" {
		diagnostic.ElementPosition = 2
	}
	return diagnostic
}

func hasErrorDiagnostics(diagnostics []edi.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}
	return false
}

func skippedServiceFailure214Result(
	trigger servicefailureports.ServiceFailureEDITrigger,
	reason string,
) *servicefailureports.ServiceFailure214LifecycleResult {
	return &servicefailureports.ServiceFailure214LifecycleResult{
		Trigger:       trigger,
		Action:        ediports.ServiceFailureEDIActionSkipped,
		SkippedReason: reason,
		Diagnostics:   []edi.Diagnostic{},
	}
}

func duplicateServiceFailure214Result(
	trigger servicefailureports.ServiceFailureEDITrigger,
	message *edi.EDIMessage,
) *servicefailureports.ServiceFailure214LifecycleResult {
	return &servicefailureports.ServiceFailure214LifecycleResult{
		Trigger:      trigger,
		Action:       ediports.ServiceFailureEDIActionDuplicate,
		MessageID:    message.ID,
		EDIPartnerID: message.EDIPartnerID,
		Diagnostics:  []edi.Diagnostic{},
	}
}
