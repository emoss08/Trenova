package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramClearReasonCode = "clearReasonCode"
	paramInternalNotes   = "internalNotes"
	paramX12Status       = "x12StatusCodeOverride"
	paramX12Reason       = "x12ReasonCodeOverride"
	paramX12Exception    = "x12ExceptionCode"
	kindServiceFailure   = "service failure"
	maxX12CodeChars      = 3
)

var serviceFailureUserRefs = map[string]permission.Resource{
	fieldReasonCodeID: permission.ResourceServiceFailureReasonCode,
	fieldReviewedByID: permission.ResourceUser,
	fieldVoidedByID:   permission.ResourceUser,
}

type serviceFailureLifecycle interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	Update(
		ctx context.Context,
		req *serviceports.UpdateServiceFailureRequest,
		actor *serviceports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Review(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Void(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	PreviewUpdate(
		ctx context.Context,
		req *serviceports.UpdateServiceFailureRequest,
	) (*serviceports.ServiceFailureLifecyclePreview, error)
	PreviewReview(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ServiceFailureLifecyclePreview, error)
	PreviewVoid(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ServiceFailureLifecyclePreview, error)
}

func serviceFailureIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceServiceFailure, "The failure",
		"list_service_failures or the page you are on")
}

func serviceFailureRecord(failure *servicefailure.ServiceFailure) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceServiceFailure,
		ID:       failure.ID,
		Label:    "Service failure " + failure.Number,
		Version:  pinnedVersion(failure.Version),
	}
}

func serviceFailureResult(
	action string,
	failure *servicefailure.ServiceFailure,
) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindServiceFailure}
	if failure != nil {
		result.Name = failure.Number
		result.IDs = map[string]string{
			paramServiceFailureID: failure.ID.String(),
			paramShipmentID:       failure.ShipmentID.String(),
		}
		result.Record = recordOf(serviceFailureRecordEntity, failure.ID)
	}

	return result
}

func targetServiceFailure(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramServiceFailureID, permission.ResourceServiceFailure)
}

type serviceFailureEdit struct {
	id            pulid.ID
	tenant        pagination.TenantInfo
	reasonCodeID  pulid.ID
	clearReason   bool
	notes         *string
	internalNotes *string
	x12Status     *string
	x12Reason     *string
	x12Exception  *string
}

func readServiceFailureEdit(params *serviceports.ToolExecuteParams) (*serviceFailureEdit, error) {
	id, err := requirePulid(params.Params, paramServiceFailureID)
	if err != nil {
		return nil, err
	}
	edit := &serviceFailureEdit{
		id:          id,
		tenant:      tenantFrom(*params),
		clearReason: optionalBool(params.Params, paramClearReasonCode),
	}
	if edit.reasonCodeID, _, err = optionalPulid(params.Params, fieldReasonCodeID); err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", fieldReasonCodeID, err)
	}
	if edit.clearReason && edit.reasonCodeID.IsNotNil() {
		return nil, fmt.Errorf("send %s or %s, not both", fieldReasonCodeID, paramClearReasonCode)
	}
	if edit.notes, err = optionalBoundedText(
		params.Params,
		fieldNotes,
		maxOperationNoteChars,
	); err != nil {
		return nil, err
	}
	if edit.internalNotes, err = optionalBoundedText(
		params.Params, paramInternalNotes, maxOperationNoteChars); err != nil {
		return nil, err
	}
	if edit.x12Status, err = optionalBoundedText(
		params.Params,
		paramX12Status,
		maxX12CodeChars,
	); err != nil {
		return nil, err
	}
	if edit.x12Reason, err = optionalBoundedText(
		params.Params,
		paramX12Reason,
		maxX12CodeChars,
	); err != nil {
		return nil, err
	}
	if edit.x12Exception, err = optionalBoundedText(
		params.Params, paramX12Exception, maxX12CodeChars); err != nil {
		return nil, err
	}
	if edit.reasonCodeID.IsNil() && !edit.clearReason && edit.notes == nil &&
		edit.internalNotes == nil && edit.x12Status == nil && edit.x12Reason == nil &&
		edit.x12Exception == nil {
		return nil, errNothingToChange
	}

	return edit, nil
}

func (e *serviceFailureEdit) request(
	current *servicefailure.ServiceFailure,
) *serviceports.UpdateServiceFailureRequest {
	req := &serviceports.UpdateServiceFailureRequest{
		TenantInfo:            e.tenant,
		ID:                    current.ID,
		ShipmentID:            current.ShipmentID,
		ReasonCodeID:          e.reasonCodeID,
		ClearReasonCode:       e.clearReason,
		Notes:                 current.Notes,
		InternalNotes:         current.InternalNotes,
		X12StatusCodeOverride: current.X12StatusCodeOverride,
		X12ReasonCodeOverride: current.X12ReasonCodeOverride,
		X12ExceptionCode:      current.X12ExceptionCode,
		Version:               current.Version,
	}
	if e.notes != nil {
		req.Notes = *e.notes
	}
	if e.internalNotes != nil {
		req.InternalNotes = *e.internalNotes
	}
	if e.x12Status != nil {
		req.X12StatusCodeOverride = strings.ToUpper(*e.x12Status)
	}
	if e.x12Reason != nil {
		req.X12ReasonCodeOverride = strings.ToUpper(*e.x12Reason)
	}
	if e.x12Exception != nil {
		req.X12ExceptionCode = strings.ToUpper(*e.x12Exception)
	}

	return req
}

func newUpdateServiceFailureTool(failures serviceFailureLifecycle) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:        "update_service_failure",
		searchTerms: []string{"change reason code", "edit service failure"},
		artifact:    serviceFailureRecordEntity,
		description: "Change the reason code, notes or EDI 214 code overrides on an open " +
			"or reviewed service failure. Send only what changes; an empty string clears " +
			"a note or override. It does not move the failure through its lifecycle: use " +
			"review_service_failure, resolve_service_failure or void_service_failure for " +
			"that.",
		resource:    permission.ResourceServiceFailure,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Edits a service failure's own notes and codes inside Trenova; the " +
			"overrides shape a later EDI 214, so a person approves each edit.",
		properties: map[string]any{
			paramServiceFailureID: serviceFailureIDProperty(),
			fieldReasonCodeID: agenttoolschema.RecordIDText(
				permission.ResourceServiceFailureReasonCode,
				"The reason code to set, from list_service_failure_reason_codes.",
			),
			paramClearReasonCode: booleanProperty("True to remove the reason code. A " +
				"reviewed failure keeps its reason."),
			fieldNotes: toolschema.KeepEmpty(stringProperty("What the customer may read "+
				"about the failure.", maxOperationNoteChars)),
			paramInternalNotes: toolschema.KeepEmpty(stringProperty("What staff should know.",
				maxOperationNoteChars)),
			paramX12Status: toolschema.KeepEmpty(stringProperty("EDI 214 status code "+
				"override, three characters.", maxX12CodeChars)),
			paramX12Reason: toolschema.KeepEmpty(stringProperty("EDI 214 reason code "+
				"override, three characters.", maxX12CodeChars)),
			paramX12Exception: toolschema.KeepEmpty(stringProperty("EDI 214 exception code, "+
				"three characters.", maxX12CodeChars)),
		},
		required: []string{paramServiceFailureID},
		target:   targetServiceFailure,
	}, receivablePlan[*serviceFailureEdit, *serviceports.ServiceFailureLifecyclePreview]{
		request: readServiceFailureEdit,
		plan: func(
			ctx context.Context,
			edit *serviceFailureEdit,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.ServiceFailureLifecyclePreview, error) {
			current, err := failures.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
				ID:         edit.id,
				TenantInfo: edit.tenant,
			})
			if err != nil {
				return nil, err
			}

			return failures.PreviewUpdate(ctx, edit.request(current))
		},
		refused: func(*serviceFailureEdit) string {
			return "Would update the service failure."
		},
		render: func(
			_ *serviceFailureEdit,
			plan *serviceports.ServiceFailureLifecyclePreview,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				serviceFailureRecord(plan.Before),
				plan.Before,
				plan.After,
				toolpreview.Only(fieldReasonCodeID, fieldNotes, paramInternalNotes,
					paramX12Status, paramX12Reason, paramX12Exception),
				toolpreview.WithRefs(serviceFailureUserRefs),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(
				"Would update service failure "+plan.Before.Number+".", change), nil
		},
		run: func(
			ctx context.Context,
			edit *serviceFailureEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			current, err := failures.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
				ID:         edit.id,
				TenantInfo: edit.tenant,
			})
			if err != nil {
				return nil, err
			}
			saved, err := failures.Update(ctx, edit.request(current), params.Actor)
			if err != nil {
				return nil, err
			}

			return serviceFailureResult("updated", saved), nil
		},
	})
}

type serviceFailureTransition struct {
	id           pulid.ID
	tenant       pagination.TenantInfo
	reasonCodeID pulid.ID
	notes        string
}

func readServiceFailureTransition(
	params *serviceports.ToolExecuteParams,
	notesRequired bool,
) (*serviceFailureTransition, error) {
	id, err := requirePulid(params.Params, paramServiceFailureID)
	if err != nil {
		return nil, err
	}
	transition := &serviceFailureTransition{id: id, tenant: tenantFrom(*params)}
	if transition.reasonCodeID, _, err = optionalPulid(
		params.Params,
		fieldReasonCodeID,
	); err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", fieldReasonCodeID, err)
	}
	if notesRequired {
		transition.notes, err = requireBoundedText(params.Params, fieldNotes, maxOperationNoteChars)
	} else {
		transition.notes, err = boundedText(params.Params, fieldNotes, maxOperationNoteChars)
	}
	if err != nil {
		return nil, err
	}

	return transition, nil
}

func (t *serviceFailureTransition) request(
	current *servicefailure.ServiceFailure,
) *serviceports.ServiceFailureLifecycleRequest {
	return &serviceports.ServiceFailureLifecycleRequest{
		TenantInfo:   t.tenant,
		ID:           current.ID,
		ShipmentID:   current.ShipmentID,
		ReasonCodeID: t.reasonCodeID,
		Notes:        t.notes,
		Version:      current.Version,
	}
}

type serviceFailureTransitionSpec struct {
	name          string
	description   string
	operation     permission.Operation
	rationale     string
	verb          string
	action        string
	notesRequired bool
	fields        []string
	volatile      string
	searchTerms   []string
	preview       func(
		failures serviceFailureLifecycle,
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ServiceFailureLifecyclePreview, error)
	run func(
		failures serviceFailureLifecycle,
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
}

func newServiceFailureTransitionTool(
	failures serviceFailureLifecycle,
	spec *serviceFailureTransitionSpec,
) serviceports.AgentTool {
	properties := map[string]any{
		paramServiceFailureID: serviceFailureIDProperty(),
		fieldNotes: stringProperty("What was found, in a sentence a person reads on the "+
			"failure later.", maxOperationNoteChars),
	}
	required := []string{paramServiceFailureID}
	if spec.notesRequired {
		required = append(required, fieldNotes)
	} else {
		properties[fieldReasonCodeID] = agenttoolschema.RecordIDText(
			permission.ResourceServiceFailureReasonCode,
			"The reason code, from list_service_failure_reason_codes. Required when the "+
				"failure has none; otherwise replaces it.",
		)
	}

	return newReportingReceivableTool(&receivableSpec{
		name:        spec.name,
		description: spec.description,
		searchTerms: spec.searchTerms,
		artifact:    serviceFailureRecordEntity,
		resource:    permission.ResourceServiceFailure,
		operation:   spec.operation,
		egress:      agent.EgressExternalRecipient,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale:   spec.rationale,
		properties:  properties,
		required:    required,
		target:      targetServiceFailure,
	}, receivablePlan[*serviceFailureTransition, *serviceports.ServiceFailureLifecyclePreview]{
		request: func(params *serviceports.ToolExecuteParams) (*serviceFailureTransition, error) {
			return readServiceFailureTransition(params, spec.notesRequired)
		},
		plan: func(
			ctx context.Context,
			transition *serviceFailureTransition,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ServiceFailureLifecyclePreview, error) {
			current, err := failures.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
				ID:         transition.id,
				TenantInfo: transition.tenant,
			})
			if err != nil {
				return nil, err
			}

			return spec.preview(failures, ctx, transition.request(current), params.Actor)
		},
		refused: func(*serviceFailureTransition) string {
			return "Would " + spec.verb + " the service failure."
		},
		render: func(
			_ *serviceFailureTransition,
			plan *serviceports.ServiceFailureLifecyclePreview,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				serviceFailureRecord(plan.Before),
				plan.Before,
				plan.After,
				toolpreview.Only(spec.fields...),
				toolpreview.WithRefs(serviceFailureUserRefs),
				toolpreview.Volatile(spec.volatile),
			)
			if err != nil {
				return nil, err
			}
			summary := fmt.Sprintf("Would %s service failure %s.", spec.verb, plan.Before.Number)

			return serviceFailureLifecyclePreview(summary, change, plan, spec.action), nil
		},
		run: func(
			ctx context.Context,
			transition *serviceFailureTransition,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			current, err := failures.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
				ID:         transition.id,
				TenantInfo: transition.tenant,
			})
			if err != nil {
				return nil, err
			}
			saved, err := spec.run(failures, ctx, transition.request(current), params.Actor)
			if err != nil {
				return nil, err
			}

			return serviceFailureResult(spec.action, saved), nil
		},
	})
}

func newReviewServiceFailureTool(failures serviceFailureLifecycle) serviceports.AgentTool {
	return newServiceFailureTransitionTool(failures, &serviceFailureTransitionSpec{
		name:        "review_service_failure",
		searchTerms: []string{"mark reviewed"},
		description: "Mark an open service failure reviewed, with the reason code that " +
			"explains it, once a person has looked into what happened. A reviewed " +
			"failure keeps its reason and waits to be resolved. The customer's trading " +
			"partner may be sent an EDI 214, so a person approves it.",
		operation: permission.OpApprove,
		rationale: "Reviewing a failure is a person's attestation and may send an EDI " +
			"214 to the customer's trading partner; only a person approves it.",
		verb:   "review",
		action: "reviewed",
		fields: []string{
			fieldStatus,
			fieldReasonCodeID,
			paramInternalNotes,
			fieldReviewedAt,
			fieldReviewedByID,
		},
		volatile: fieldReviewedAt,
		preview:  serviceFailureLifecycle.PreviewReview,
		run:      serviceFailureLifecycle.Review,
	})
}

func newVoidServiceFailureTool(failures serviceFailureLifecycle) serviceports.AgentTool {
	return newServiceFailureTransitionTool(failures, &serviceFailureTransitionSpec{
		name: "void_service_failure",
		description: "Void a service failure that should never have been opened, with " +
			"the reason why: a wrong appointment window, a corrected arrival, a stop " +
			"that was not late. Voiding is final. For a failure that was real and is " +
			"now handled, use resolve_service_failure instead.",
		operation: permission.OpArchive,
		rationale: "Voiding a failure is final and may send an EDI 214 to the customer's " +
			"trading partner; only a person approves it.",
		verb:          "void",
		action:        "voided",
		notesRequired: true,
		fields:        []string{fieldStatus, "voidReason", "voidedAt", fieldVoidedByID},
		volatile:      "voidedAt",
		preview:       serviceFailureLifecycle.PreviewVoid,
		run:           serviceFailureLifecycle.Void,
	})
}
