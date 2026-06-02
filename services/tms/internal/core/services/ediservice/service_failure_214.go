package ediservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const serviceFailure214TriggerReference = "serviceFailure214Trigger"

type serviceFailure214Settings struct {
	Enabled                 bool
	SendOnReviewed          bool
	SendOnResolved          bool
	MandatoryOnReviewed     bool
	MandatoryOnResolved     bool
	StatusCode              string
	RequireStatusReasonCode bool
	RequireLocation         bool
	RequireStop             bool
	RequireProNumber        bool
	RequireBOL              bool
	AcceptedReasonCodes     map[string]struct{}
}

type serviceFailure214Candidate struct {
	partner  *edi.EDIPartner
	profile  *edi.EDIPartnerDocumentProfile
	settings serviceFailure214Settings
}

func (s *Service) PreviewServiceFailure214ForLifecycle(
	ctx context.Context,
	req *services.ServiceFailure214LifecycleRequest,
) (*services.ServiceFailure214LifecycleResult, error) {
	return s.serviceFailure214Lifecycle(ctx, req, false)
}

func (s *Service) GenerateServiceFailure214ForLifecycle(
	ctx context.Context,
	req *services.ServiceFailure214LifecycleRequest,
) (*services.ServiceFailure214LifecycleResult, error) {
	return s.serviceFailure214Lifecycle(ctx, req, true)
}

func (s *Service) serviceFailure214Lifecycle(
	ctx context.Context,
	req *services.ServiceFailure214LifecycleRequest,
	generate bool,
) (*services.ServiceFailure214LifecycleResult, error) {
	if err := validateServiceFailure214LifecycleRequest(req); err != nil {
		return nil, err
	}

	existing, err := s.messageRepo.GetServiceFailure214LifecycleMessage(
		ctx,
		repositories.GetServiceFailure214LifecycleMessageRequest{
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

	payload := buildServiceFailure214LifecyclePayload(
		failure,
		source,
		candidate.settings,
		req.Trigger,
	)
	diagnostics := serviceFailurePayloadDiagnostics(payload.ShipmentStatus, candidate.settings)
	result = &services.ServiceFailure214LifecycleResult{
		Trigger:                  req.Trigger,
		Action:                   services.ServiceFailureEDIActionSkipped,
		EDIPartnerID:             candidate.partner.ID,
		PartnerDocumentProfileID: candidate.profile.ID,
		Mandatory:                candidate.settings.mandatory(req.Trigger),
		Diagnostics:              diagnostics,
	}
	if hasErrorDiagnostics(diagnostics) {
		result.Action = services.ServiceFailureEDIActionBlocked
		result.SkippedReason = "service failure 214 payload has blocking diagnostics"
		return result, nil
	}
	if !generate {
		result.Action = services.ServiceFailureEDIActionSkipped
		result.SkippedReason = "ready"
		return result, nil
	}

	message, err := s.GenerateDocument(ctx, &services.GenerateEDIDocumentRequest{
		TenantInfo:               req.TenantInfo,
		PartnerDocumentProfileID: candidate.profile.ID,
		EDIPartnerID:             candidate.partner.ID,
		ShipmentID:               failure.ShipmentID,
		ServiceFailureID:         failure.ID,
		TransactionSet:           edi.TransactionSet214,
		Direction:                edi.DocumentDirectionOutbound,
		Payload:                  &payload,
		GeneratedByID:            req.GeneratedByID,
	})
	if err != nil {
		return nil, err
	}
	result.Action = services.ServiceFailureEDIActionGenerated
	result.SkippedReason = ""
	result.MessageID = message.ID
	return result, nil
}

func validateServiceFailure214LifecycleRequest(req *services.ServiceFailure214LifecycleRequest) error {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Service failure EDI lifecycle request is required")
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
	req *services.ServiceFailure214LifecycleRequest,
) (*servicefailure.ServiceFailure, *shipment.Shipment, error) {
	failure := req.ServiceFailure
	if failure == nil {
		var err error
		failure, err = s.serviceFailureRepo.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
			ID:         req.ServiceFailureID,
			TenantInfo: req.TenantInfo,
		})
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
	source, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         failure.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
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
	req *services.ServiceFailure214LifecycleRequest,
	source *shipment.Shipment,
) (*serviceFailure214Candidate, *services.ServiceFailure214LifecycleResult, error) {
	if source == nil || source.CustomerID.IsNil() {
		return nil, skippedServiceFailure214Result(req.Trigger, "shipment customer is not linked to an EDI partner"), nil
	}
	partners, err := s.partnerRepo.List(ctx, &repositories.ListEDIPartnersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: pagination.MaxLimit},
		},
		CustomerID:         source.CustomerID,
		EnabledForOutbound: true,
		Status:             domaintypes.StatusActive,
	})
	if err != nil {
		return nil, nil, err
	}
	if partners.Total == 0 {
		return nil, skippedServiceFailure214Result(req.Trigger, "no outbound EDI partner for shipment customer"), nil
	}

	candidates := make([]serviceFailure214Candidate, 0, partners.Total)
	capabilityDisabled := false
	for _, partner := range partners.Items {
		if partner == nil || !partner.EnabledForOutbound {
			continue
		}
		enabled, err := s.shipmentStatusCapabilityEnabled(ctx, req.TenantInfo, partner)
		if err != nil {
			return nil, nil, err
		}
		if !enabled {
			capabilityDisabled = true
			continue
		}
		profiles, profileErr := s.documentProfileRepo.ListPartnerDocumentProfiles(
			ctx,
			&repositories.ListEDIPartnerDocumentProfilesRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: req.TenantInfo,
					Pagination: pagination.Info{Limit: pagination.MaxLimit},
				},
				PartnerID:      partner.ID,
				TransactionSet: edi.TransactionSet214,
				Direction:      edi.DocumentDirectionOutbound,
				Standard:       edi.EDIStandardX12,
				Status:         edi.DocumentStatusActive,
			},
		)
		if profileErr != nil {
			return nil, nil, profileErr
		}
		for _, profile := range profiles.Items {
			settings := parseServiceFailure214Settings(profile.PartnerSettings)
			if !settings.enabledForTrigger(req.Trigger) {
				continue
			}
			candidates = append(candidates, serviceFailure214Candidate{
				partner:  partner,
				profile:  profile,
				settings: settings,
			})
		}
	}

	switch len(candidates) {
	case 0:
		reason := "service failure 214 trigger disabled"
		if capabilityDisabled {
			reason = "shipment status capability disabled"
		}
		return nil, skippedServiceFailure214Result(req.Trigger, reason), nil
	case 1:
		return &candidates[0], nil, nil
	default:
		result := skippedServiceFailure214Result(req.Trigger, "ambiguous service failure 214 partner document profile")
		if hasMandatoryServiceFailure214Candidate(candidates, req.Trigger) {
			result.Action = services.ServiceFailureEDIActionBlocked
			result.Mandatory = true
		}
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
		connection, err = s.connectionRepo.GetConnectionByID(ctx, repositories.GetEDIConnectionByIDRequest{
			ID:         partner.EDIConnectionID,
			TenantInfo: tenantInfo,
		})
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
	settings serviceFailure214Settings,
	trigger services.ServiceFailureEDITrigger,
) edi.DocumentPayload {
	payload := buildServiceFailureShipmentStatusPayload(failure, source)
	if payload.ShipmentStatus == nil {
		return payload
	}
	if strings.TrimSpace(failure.X12StatusCodeOverride) == "" && settings.StatusCode != "" {
		payload.ShipmentStatus.StatusCode = settings.StatusCode
	}
	if payload.ShipmentStatus.References == nil {
		payload.ShipmentStatus.References = map[string]string{}
	}
	payload.ShipmentStatus.References[serviceFailure214TriggerReference] = string(trigger)
	payload.ShipmentStatus.References["serviceFailureStatus"] = string(failure.Status)
	return payload
}

func parseServiceFailure214Settings(settings map[string]any) serviceFailure214Settings {
	raw, _ := settings["serviceFailure214"].(map[string]any)
	parsed := serviceFailure214Settings{
		StatusCode:          normalizedX12Code(rawString(raw, "statusCode")),
		AcceptedReasonCodes: normalizedCodeSet(rawSlice(raw, "acceptedReasonCodes")),
	}
	parsed.Enabled = rawBool(raw, "enabled")
	parsed.SendOnReviewed = rawBool(raw, "sendOnReviewed")
	parsed.SendOnResolved = rawBool(raw, "sendOnResolved")
	parsed.MandatoryOnReviewed = rawBool(raw, "mandatoryOnReviewed")
	parsed.MandatoryOnResolved = rawBool(raw, "mandatoryOnResolved")
	parsed.RequireStatusReasonCode = rawBool(raw, "requireStatusReasonCode")
	parsed.RequireLocation = rawBool(raw, "requireLocation")
	parsed.RequireStop = rawBool(raw, "requireStop")
	parsed.RequireProNumber = rawBool(raw, "requireProNumber")
	parsed.RequireBOL = rawBool(raw, "requireBol")
	return parsed
}

func rawBool(settings map[string]any, key string) bool {
	value, _ := settings[key].(bool)
	return value
}

func rawString(settings map[string]any, key string) string {
	value, _ := settings[key].(string)
	return value
}

func rawSlice(settings map[string]any, key string) []string {
	value, ok := settings[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func normalizedCodeSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizedX12Code(value)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func normalizedX12Code(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func (s serviceFailure214Settings) enabledForTrigger(trigger services.ServiceFailureEDITrigger) bool {
	if !s.Enabled {
		return false
	}
	switch trigger {
	case services.ServiceFailureEDITriggerReviewed:
		return s.SendOnReviewed || s.MandatoryOnReviewed
	case services.ServiceFailureEDITriggerResolved:
		return s.SendOnResolved || s.MandatoryOnResolved
	default:
		return false
	}
}

func (s serviceFailure214Settings) mandatory(trigger services.ServiceFailureEDITrigger) bool {
	switch trigger {
	case services.ServiceFailureEDITriggerReviewed:
		return s.MandatoryOnReviewed
	case services.ServiceFailureEDITriggerResolved:
		return s.MandatoryOnResolved
	default:
		return false
	}
}

func serviceFailure214Diagnostics(
	payload *edi.ShipmentStatusPayload,
	settings serviceFailure214Settings,
) []edix12.Diagnostic {
	if payload == nil {
		return []edix12.Diagnostic{serviceFailure214Diagnostic(
			"required",
			"shipmentStatus",
			"EDI 214 shipment status payload is required",
			"Build the shipment status payload before generating the 214.",
		)}
	}
	diagnostics := make([]edix12.Diagnostic, 0, 6)
	statusCode := normalizedX12Code(payload.StatusCode)
	reasonCode := normalizedX12Code(payload.StatusReasonCode)
	if statusCode == "SD" && reasonCode == "" {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.statusReasonCode",
			"X12 214 service failure status code SD requires a status reason code",
			"Set an override reason code or configure a default reason code on the service failure reason code.",
		))
	}
	if statusCode != "SD" && settings.RequireStatusReasonCode && reasonCode == "" {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.statusReasonCode",
			"Partner profile requires a shipment status reason code",
			"Set an override reason code or configure a default reason code on the service failure reason code.",
		))
	}
	if settings.RequireBOL && strings.TrimSpace(payload.BOL) == "" {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.bol",
			"Partner profile requires a BOL for service failure 214 generation",
			"Set the shipment BOL before generating the 214.",
		))
	}
	if settings.RequireProNumber && strings.TrimSpace(payload.ProNumber) == "" {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.proNumber",
			"Partner profile requires a PRO number for service failure 214 generation",
			"Set the shipment PRO number before generating the 214.",
		))
	}
	if settings.RequireStop && payload.StopID.IsNil() {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.stopId",
			"Partner profile requires a stop for service failure 214 generation",
			"Link the service failure to a shipment stop before generating the 214.",
		))
	}
	if settings.RequireLocation && payload.LocationID.IsNil() && strings.TrimSpace(payload.LocationName) == "" {
		diagnostics = append(diagnostics, serviceFailure214Diagnostic(
			"required",
			"shipmentStatus.locationId",
			"Partner profile requires a location for service failure 214 generation",
			"Link the service failure stop to a location before generating the 214.",
		))
	}
	if len(settings.AcceptedReasonCodes) > 0 && reasonCode != "" {
		if _, ok := settings.AcceptedReasonCodes[reasonCode]; !ok {
			diagnostics = append(diagnostics, serviceFailure214Diagnostic(
				"unsupported_reason_code",
				"shipmentStatus.statusReasonCode",
				fmt.Sprintf("Status reason code %s is not accepted by the partner profile", reasonCode),
				"Use one of the accepted partner reason codes for this service failure 214.",
			))
		}
	}
	return diagnostics
}

func serviceFailure214Diagnostic(code, path, message, suggestedFix string) edix12.Diagnostic {
	diagnostic := edix12.Diagnostic{
		Severity:     edi.ValidationSeverityError,
		Code:         code,
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

func hasErrorDiagnostics(diagnostics []edix12.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == edi.ValidationSeverityError {
			return true
		}
	}
	return false
}

func hasMandatoryServiceFailure214Candidate(
	candidates []serviceFailure214Candidate,
	trigger services.ServiceFailureEDITrigger,
) bool {
	for _, candidate := range candidates {
		if candidate.settings.mandatory(trigger) {
			return true
		}
	}
	return false
}

func skippedServiceFailure214Result(
	trigger services.ServiceFailureEDITrigger,
	reason string,
) *services.ServiceFailure214LifecycleResult {
	return &services.ServiceFailure214LifecycleResult{
		Trigger:       trigger,
		Action:        services.ServiceFailureEDIActionSkipped,
		SkippedReason: reason,
		Diagnostics:   []edix12.Diagnostic{},
	}
}

func duplicateServiceFailure214Result(
	trigger services.ServiceFailureEDITrigger,
	message *edi.EDIMessage,
) *services.ServiceFailure214LifecycleResult {
	return &services.ServiceFailure214LifecycleResult{
		Trigger:                  trigger,
		Action:                   services.ServiceFailureEDIActionDuplicate,
		MessageID:                message.ID,
		EDIPartnerID:             message.EDIPartnerID,
		PartnerDocumentProfileID: message.PartnerDocumentProfileID,
		Diagnostics:              []edix12.Diagnostic{},
	}
}
