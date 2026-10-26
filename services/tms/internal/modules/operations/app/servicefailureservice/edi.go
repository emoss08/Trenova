package servicefailureservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

type serviceFailure214Params struct {
	previous *servicefailure.ServiceFailure
	current  *servicefailure.ServiceFailure
	actor    *permissionports.RequestActor
}

// preflightServiceFailure214 is the 214 a lifecycle change would generate,
// checked before the change is saved: a mandatory 214 that would be blocked
// refuses the change.
func (s *service) preflightServiceFailure214(
	ctx context.Context,
	params serviceFailure214Params,
) (*servicefailureports.ServiceFailure214LifecycleResult, error) {
	req := serviceFailure214Request(params)
	if req == nil || s.ediService == nil {
		return nil, nil //nolint:nilnil // a nil result means the change generates no 214
	}
	result, err := s.ediService.PreviewServiceFailure214ForLifecycle(ctx, req)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"edi",
			errortypes.ErrInvalidOperation,
			"Service failure EDI 214 preflight failed: {0}", err.Error(),
		)
	}
	if result.Action != ediports.ServiceFailureEDIActionBlocked || !result.Mandatory {
		return result, nil
	}
	if len(result.Diagnostics) == 0 {
		return nil, errortypes.NewValidationError(
			"edi",
			errortypes.ErrInvalidOperation,
			strings.TrimSpace(result.SkippedReason),
		)
	}
	return nil, serviceFailure214DiagnosticsError(result.Diagnostics)
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
	case ediports.ServiceFailureEDIActionGenerated,
		ediports.ServiceFailureEDIActionDuplicate,
		ediports.ServiceFailureEDIActionQueued:
		s.recordServiceFailure214Result(ctx, params, result)
	case ediports.ServiceFailureEDIActionBlocked:
		s.l.Warn(
			"service failure EDI 214 generation blocked",
			zap.String("serviceFailureId", params.current.ID.String()),
			zap.String("trigger", string(result.Trigger)),
			zap.String("reason", result.SkippedReason),
		)
	case ediports.ServiceFailureEDIActionSkipped:
	default:
	}
}

func serviceFailure214Request(
	params serviceFailure214Params,
) *ediports.ServiceFailure214LifecycleRequest {
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
	return &ediports.ServiceFailure214LifecycleRequest{
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

func serviceFailure214Trigger(
	status servicefailure.Status,
) (servicefailureports.ServiceFailureEDITrigger, bool) {
	switch status {
	case servicefailure.StatusReviewed:
		return servicefailureports.ServiceFailureEDITriggerReviewed, true
	case servicefailure.StatusResolved:
		return servicefailureports.ServiceFailureEDITriggerResolved, true
	default:
		return "", false
	}
}

func serviceFailure214DiagnosticsError(diagnostics []edi.Diagnostic) error {
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
	result *servicefailureports.ServiceFailure214LifecycleResult,
) {
	metadata := serviceFailureLifecycleMetadata(params.previous, params.current, params.actor)
	metadata["ediMessageId"] = result.MessageID.String()
	metadata["ediPartnerId"] = result.EDIPartnerID.String()
	metadata["serviceFailure214Trigger"] = string(result.Trigger)
	metadata["serviceFailure214Action"] = string(result.Action)

	comment := "Service failure EDI 214 generated"
	if result.Action == ediports.ServiceFailureEDIActionDuplicate {
		comment = "Service failure EDI 214 already generated"
	}
	if result.Action == ediports.ServiceFailureEDIActionQueued {
		comment = "Service failure EDI 214 queued to go with the partner's next batch"
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
	logParams := &auditports.LogActionParams{
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
	opts := []auditports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(metadata),
	}
	if err := s.auditService.LogAction(logParams, opts...); err != nil {
		s.l.Warn("failed to log service failure EDI audit", zap.Error(err))
	}
}
