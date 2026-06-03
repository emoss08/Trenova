package servicefailureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

type serviceFailure214Params struct {
	previous *servicefailure.ServiceFailure
	current  *servicefailure.ServiceFailure
	actor    *services.RequestActor
}

func (s *service) preflightServiceFailure214(
	ctx context.Context,
	params serviceFailure214Params,
) error {
	req := serviceFailure214Request(params)
	if req == nil || s.ediService == nil {
		return nil
	}
	result, err := s.ediService.PreviewServiceFailure214ForLifecycle(ctx, req)
	if err != nil {
		return errortypes.NewValidationError(
			"edi",
			errortypes.ErrInvalidOperation,
			"Service failure EDI 214 preflight failed: "+err.Error(),
		)
	}
	if result.Action != services.ServiceFailureEDIActionBlocked || !result.Mandatory {
		return nil
	}
	if len(result.Diagnostics) == 0 {
		return errortypes.NewValidationError(
			"edi",
			errortypes.ErrInvalidOperation,
			strings.TrimSpace(result.SkippedReason),
		)
	}
	return serviceFailure214DiagnosticsError(result.Diagnostics)
}

func (s *service) generateServiceFailure214(
	ctx context.Context,
	params serviceFailure214Params,
) {
	req := serviceFailure214Request(params)
	if req == nil || s.ediService == nil {
		return
	}
	req.ServiceFailure = nil
	result, err := s.ediService.GenerateServiceFailure214ForLifecycle(ctx, req)
	if err != nil {
		s.l.Warn(
			"failed to generate service failure EDI 214",
			zap.Error(err),
			zap.String("serviceFailureId", params.current.ID.String()),
			zap.String("trigger", string(req.Trigger)),
		)
		return
	}
	switch result.Action {
	case services.ServiceFailureEDIActionGenerated, services.ServiceFailureEDIActionDuplicate:
		s.recordServiceFailure214Result(ctx, params, result)
	case services.ServiceFailureEDIActionBlocked:
		s.l.Warn(
			"service failure EDI 214 generation blocked",
			zap.String("serviceFailureId", params.current.ID.String()),
			zap.String("trigger", string(result.Trigger)),
			zap.String("reason", result.SkippedReason),
		)
	case services.ServiceFailureEDIActionSkipped:
	default:
	}
}

func serviceFailure214Request(
	params serviceFailure214Params,
) *services.ServiceFailure214LifecycleRequest {
	if params.current == nil {
		return nil
	}
	trigger, ok := serviceFailure214Trigger(params.current.Status)
	if !ok {
		return nil
	}
	previousStatus := servicefailure.StatusOpen
	if params.previous != nil {
		previousStatus = params.previous.Status
	}
	return &services.ServiceFailure214LifecycleRequest{
		TenantInfo:       serviceFailureTenantInfo(params.current),
		ServiceFailureID: params.current.ID,
		ShipmentID:       params.current.ShipmentID,
		Trigger:          trigger,
		PreviousStatus:   previousStatus,
		NewStatus:        params.current.Status,
		GeneratedByID:    params.actor.UserIDOrNil(),
		ServiceFailure:   params.current,
	}
}

func serviceFailure214Trigger(status servicefailure.Status) (services.ServiceFailureEDITrigger, bool) {
	switch status {
	case servicefailure.StatusReviewed:
		return services.ServiceFailureEDITriggerReviewed, true
	case servicefailure.StatusResolved:
		return services.ServiceFailureEDITriggerResolved, true
	default:
		return "", false
	}
}

func serviceFailure214DiagnosticsError(diagnostics []edix12.Diagnostic) error {
	multiErr := errortypes.NewMultiError()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != edi.ValidationSeverityError {
			continue
		}
		field := strings.TrimSpace(diagnostic.Path)
		if field == "" {
			field = "edi"
		}
		multiErr.Add(field, errortypes.ErrInvalid, diagnostic.Message)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *service) recordServiceFailure214Result(
	ctx context.Context,
	params serviceFailure214Params,
	result *services.ServiceFailure214LifecycleResult,
) {
	metadata := serviceFailureLifecycleMetadata(params.previous, params.current, params.actor)
	metadata["ediMessageId"] = result.MessageID.String()
	metadata["ediPartnerId"] = result.EDIPartnerID.String()
	metadata["ediPartnerDocumentProfileId"] = result.PartnerDocumentProfileID.String()
	metadata["serviceFailure214Trigger"] = string(result.Trigger)
	metadata["serviceFailure214Action"] = string(result.Action)

	comment := "Service failure EDI 214 generated"
	if result.Action == services.ServiceFailureEDIActionDuplicate {
		comment = "Service failure EDI 214 already generated"
	}
	s.logServiceFailureEDIAction(params, comment, metadata)
	s.comment(ctx, commentParams{
		entity:   params.current,
		comment:  comment,
		metadata: metadata,
	})
}

func (s *service) logServiceFailureEDIAction(
	params serviceFailure214Params,
	comment string,
	metadata map[string]any,
) {
	auditActor := params.actor.AuditActorOrSystem()
	logParams := &services.LogActionParams{
		Resource:       permission.ResourceServiceFailure,
		ResourceID:     params.current.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		OrganizationID: params.current.OrganizationID,
		BusinessUnitID: params.current.BusinessUnitID,
	}
	opts := []services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(metadata),
	}
	if err := s.auditService.LogAction(logParams, opts...); err != nil {
		s.l.Warn("failed to log service failure EDI audit", zap.Error(err))
	}
}
