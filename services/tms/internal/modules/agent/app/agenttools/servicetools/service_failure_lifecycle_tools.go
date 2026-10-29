package servicetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicefailuretools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramClearReasonCode = "clearReasonCode"

const paramInternalNotes = "internalNotes"

const paramX12Status = "x12StatusCodeOverride"

const paramX12Reason = "x12ReasonCodeOverride"

const paramX12Exception = "x12ExceptionCode"

const kindServiceFailure = "service failure"

const maxX12CodeChars = 3

var serviceFailureUserRefs = map[string]permission.Resource{
	changepreview.FieldReasonCodeID: permission.ResourceServiceFailureReasonCode,
	changepreview.FieldReviewedByID: permission.ResourceUser,
	changepreview.FieldVoidedByID:   permission.ResourceUser,
}

type serviceFailureLifecycle interface {
	GetByID(
		ctx context.Context,
		req *servicefailureports.GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	Update(
		ctx context.Context,
		req *servicefailureports.UpdateServiceFailureRequest,
		actor *permissionports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Review(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	Void(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	PreviewUpdate(
		ctx context.Context,
		req *servicefailureports.UpdateServiceFailureRequest,
	) (*servicefailureports.ServiceFailureLifecyclePreview, error)
	PreviewReview(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureLifecyclePreview, error)
	PreviewVoid(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureLifecyclePreview, error)
}

func serviceFailureIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceServiceFailure, "The failure",
		"exception_list_service_failures or the page you are on")
}

func serviceFailureRecord(failure *servicefailure.ServiceFailure) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceServiceFailure,
		ID:       failure.ID,
		Label:    "Service failure " + failure.Number,
		Version:  changepreview.PinnedVersion(failure.Version),
	}
}

func serviceFailureResult(
	action string,
	failure *servicefailure.ServiceFailure,
) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{Action: action, Kind: kindServiceFailure}
	if failure != nil {
		result.Name = failure.Number
		result.IDs = map[string]string{
			editools.ParamServiceFailureID: failure.ID.String(),
			toolparams.ParamShipmentID:     failure.ShipmentID.String(),
		}
		result.Record = changepreview.RecordOf(changepreview.ServiceFailureRecordEntity, failure.ID)
	}

	return result
}

func targetServiceFailure(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		editools.ParamServiceFailureID,
		permission.ResourceServiceFailure,
	)
}

func getServiceFailure(
	ctx context.Context,
	failures serviceFailureLifecycle,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*servicefailure.ServiceFailure, error) {
	return failures.GetByID(ctx, &servicefailureports.GetServiceFailureByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
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

func readServiceFailureEdit(params *agenttoolports.ToolExecuteParams) (*serviceFailureEdit, error) {
	id, err := toolparams.RequirePulid(params.Params, editools.ParamServiceFailureID)
	if err != nil {
		return nil, err
	}
	edit := &serviceFailureEdit{
		id:          id,
		tenant:      toolparams.TenantFrom(*params),
		clearReason: toolparams.OptionalBool(params.Params, paramClearReasonCode),
	}
	if edit.reasonCodeID, _, err = toolparams.OptionalPulid(
		params.Params,
		changepreview.FieldReasonCodeID,
	); err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a valid id: %w",
			changepreview.FieldReasonCodeID,
			err,
		)
	}
	if edit.clearReason && edit.reasonCodeID.IsNotNil() {
		return nil, fmt.Errorf(
			"send %s or %s, not both",
			changepreview.FieldReasonCodeID,
			paramClearReasonCode,
		)
	}
	if edit.notes, err = toolparams.OptionalBoundedText(
		params.Params,
		toolparams.ParamNote,
		toolparams.MaxOperationNoteChars,
	); err != nil {
		return nil, err
	}
	if edit.internalNotes, err = toolparams.OptionalBoundedText(
		params.Params, paramInternalNotes, toolparams.MaxOperationNoteChars); err != nil {
		return nil, err
	}
	if edit.x12Status, err = toolparams.OptionalBoundedText(
		params.Params,
		paramX12Status,
		maxX12CodeChars,
	); err != nil {
		return nil, err
	}
	if edit.x12Reason, err = toolparams.OptionalBoundedText(
		params.Params,
		paramX12Reason,
		maxX12CodeChars,
	); err != nil {
		return nil, err
	}
	if edit.x12Exception, err = toolparams.OptionalBoundedText(
		params.Params, paramX12Exception, maxX12CodeChars); err != nil {
		return nil, err
	}
	if edit.reasonCodeID.IsNil() && !edit.clearReason && edit.notes == nil &&
		edit.internalNotes == nil && edit.x12Status == nil && edit.x12Reason == nil &&
		edit.x12Exception == nil {
		return nil, toolparams.ErrNothingToChange
	}

	return edit, nil
}

func (e *serviceFailureEdit) request(
	current *servicefailure.ServiceFailure,
) *servicefailureports.UpdateServiceFailureRequest {
	req := &servicefailureports.UpdateServiceFailureRequest{
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

func NewUpdateServiceFailureTool(failures serviceFailureLifecycle) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "exception_update_service_failure",
		SearchTerms: []string{"change reason code", "edit service failure"},
		Artifact:    changepreview.ServiceFailureRecordEntity,
		Description: "Change the reason code, notes or EDI 214 code overrides on an open " +
			"or reviewed service failure. Send only what changes; an empty string clears " +
			"a note or override. It does not move the failure through its lifecycle: use " +
			"exception_review_service_failure, exception_resolve_service_failure or exception_void_service_failure for " +
			"that.",
		Resource:    permission.ResourceServiceFailure,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Edits a service failure's own notes and codes inside Trenova; the " +
			"overrides shape a later EDI 214, so a person approves each edit.",
		Properties: map[string]any{
			editools.ParamServiceFailureID: serviceFailureIDProperty(),
			changepreview.FieldReasonCodeID: agenttoolschema.RecordIDText(
				permission.ResourceServiceFailureReasonCode,
				"The reason code to set, from exception_list_service_failure_reason_codes.",
			),
			paramClearReasonCode: toolparams.BooleanProperty("True to remove the reason code. A " +
				"reviewed failure keeps its reason."),
			toolparams.ParamNote: toolschema.KeepEmpty(
				toolparams.StringProperty("What the customer may read "+
					"about the failure.", toolparams.MaxOperationNoteChars),
			),
			paramInternalNotes: toolschema.KeepEmpty(
				toolparams.StringProperty("What staff should know.",
					toolparams.MaxOperationNoteChars),
			),
			paramX12Status: toolschema.KeepEmpty(toolparams.StringProperty("EDI 214 status code "+
				"override, three characters.", maxX12CodeChars)),
			paramX12Reason: toolschema.KeepEmpty(toolparams.StringProperty("EDI 214 reason code "+
				"override, three characters.", maxX12CodeChars)),
			paramX12Exception: toolschema.KeepEmpty(
				toolparams.StringProperty("EDI 214 exception code, "+
					"three characters.", maxX12CodeChars),
			),
		},
		Required: []string{editools.ParamServiceFailureID},
		Target:   targetServiceFailure,
	}, toolspec.ReceivablePlan[*serviceFailureEdit, *servicefailureports.ServiceFailureLifecyclePreview]{
		Request: readServiceFailureEdit,
		Plan: func(
			ctx context.Context,
			edit *serviceFailureEdit,
			_ *agenttoolports.ToolExecuteParams,
		) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
			current, err := getServiceFailure(ctx, failures, edit.id, edit.tenant)
			if err != nil {
				return nil, err
			}

			return failures.PreviewUpdate(ctx, edit.request(current))
		},
		Refused: func(*serviceFailureEdit) string {
			return "Would update the service failure."
		},
		Render: func(
			_ *serviceFailureEdit,
			plan *servicefailureports.ServiceFailureLifecyclePreview,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				serviceFailureRecord(plan.Before),
				plan.Before,
				plan.After,
				toolpreview.Only(
					changepreview.FieldReasonCodeID,
					changepreview.FieldNotes,
					paramInternalNotes,
					paramX12Status,
					paramX12Reason,
					paramX12Exception,
				),
				toolpreview.WithRefs(serviceFailureUserRefs),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(
				"Would update service failure "+plan.Before.Number+".", change), nil
		},
		Run: func(
			ctx context.Context,
			edit *serviceFailureEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			current, err := getServiceFailure(ctx, failures, edit.id, edit.tenant)
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
	params *agenttoolports.ToolExecuteParams,
	notesRequired bool,
) (*serviceFailureTransition, error) {
	id, err := toolparams.RequirePulid(params.Params, editools.ParamServiceFailureID)
	if err != nil {
		return nil, err
	}
	transition := &serviceFailureTransition{id: id, tenant: toolparams.TenantFrom(*params)}
	if transition.reasonCodeID, _, err = toolparams.OptionalPulid(
		params.Params,
		changepreview.FieldReasonCodeID,
	); err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a valid id: %w",
			changepreview.FieldReasonCodeID,
			err,
		)
	}
	if notesRequired {
		transition.notes, err = toolparams.RequireBoundedText(
			params.Params,
			toolparams.ParamNote,
			toolparams.MaxOperationNoteChars,
		)
	} else {
		transition.notes, err = toolparams.BoundedText(
			params.Params,
			toolparams.ParamNote,
			toolparams.MaxOperationNoteChars,
		)
	}
	if err != nil {
		return nil, err
	}

	return transition, nil
}

func (t *serviceFailureTransition) request(
	current *servicefailure.ServiceFailure,
) *servicefailureports.ServiceFailureLifecycleRequest {
	return &servicefailureports.ServiceFailureLifecycleRequest{
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
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureLifecyclePreview, error)
	run func(
		failures serviceFailureLifecycle,
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
}

func newServiceFailureTransitionTool(
	failures serviceFailureLifecycle,
	spec *serviceFailureTransitionSpec,
) agenttoolports.AgentTool {
	properties := map[string]any{
		editools.ParamServiceFailureID: serviceFailureIDProperty(),
		toolparams.ParamNote: toolparams.StringProperty(
			"What was found, in a sentence a person reads on the "+
				"failure later.",
			toolparams.MaxOperationNoteChars,
		),
	}
	required := []string{editools.ParamServiceFailureID}
	if spec.notesRequired {
		required = append(required, toolparams.ParamNote)
	} else {
		properties[changepreview.FieldReasonCodeID] = agenttoolschema.RecordIDText(
			permission.ResourceServiceFailureReasonCode,
			"The reason code, from exception_list_service_failure_reason_codes. Required when the "+
				"failure has none; otherwise replaces it.",
		)
	}

	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        spec.name,
		Description: spec.description,
		SearchTerms: spec.searchTerms,
		Artifact:    changepreview.ServiceFailureRecordEntity,
		Resource:    permission.ResourceServiceFailure,
		Operation:   spec.operation,
		Egress:      agent.EgressExternalRecipient,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale:   spec.rationale,
		Properties:  properties,
		Required:    required,
		Target:      targetServiceFailure,
	}, toolspec.ReceivablePlan[*serviceFailureTransition, *servicefailureports.ServiceFailureLifecyclePreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*serviceFailureTransition, error) {
			return readServiceFailureTransition(params, spec.notesRequired)
		},
		Plan: func(
			ctx context.Context,
			transition *serviceFailureTransition,
			params *agenttoolports.ToolExecuteParams,
		) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
			current, err := getServiceFailure(ctx, failures, transition.id, transition.tenant)
			if err != nil {
				return nil, err
			}

			return spec.preview(failures, ctx, transition.request(current), params.Actor)
		},
		Refused: func(*serviceFailureTransition) string {
			return "Would " + spec.verb + " the service failure."
		},
		Render: func(
			_ *serviceFailureTransition,
			plan *servicefailureports.ServiceFailureLifecyclePreview,
		) (*agentpreview.ToolPreview, error) {
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

			return servicefailuretools.ServiceFailureLifecyclePreview(
				summary,
				change,
				plan,
				spec.action,
			), nil
		},
		Run: func(
			ctx context.Context,
			transition *serviceFailureTransition,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			current, err := getServiceFailure(ctx, failures, transition.id, transition.tenant)
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

func NewReviewServiceFailureTool(failures serviceFailureLifecycle) agenttoolports.AgentTool {
	return newServiceFailureTransitionTool(failures, &serviceFailureTransitionSpec{
		name:        "exception_review_service_failure",
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
			changepreview.FieldStatus,
			changepreview.FieldReasonCodeID,
			paramInternalNotes,
			changepreview.FieldReviewedAt,
			changepreview.FieldReviewedByID,
		},
		volatile: changepreview.FieldReviewedAt,
		preview:  serviceFailureLifecycle.PreviewReview,
		run:      serviceFailureLifecycle.Review,
	})
}

func NewVoidServiceFailureTool(failures serviceFailureLifecycle) agenttoolports.AgentTool {
	return newServiceFailureTransitionTool(failures, &serviceFailureTransitionSpec{
		name: "exception_void_service_failure",
		description: "Void a service failure that should never have been opened, with " +
			"the reason why: a wrong appointment window, a corrected arrival, a stop " +
			"that was not late. Voiding is final. For a failure that was real and is " +
			"now handled, use exception_resolve_service_failure instead.",
		operation: permission.OpArchive,
		rationale: "Voiding a failure is final and may send an EDI 214 to the customer's " +
			"trading partner; only a person approves it.",
		verb:          "void",
		action:        "voided",
		notesRequired: true,
		fields: []string{
			changepreview.FieldStatus,
			"voidReason",
			"voidedAt",
			changepreview.FieldVoidedByID,
		},
		volatile: "voidedAt",
		preview:  serviceFailureLifecycle.PreviewVoid,
		run:      serviceFailureLifecycle.Void,
	})
}
