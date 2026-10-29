package servicefailuretools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

type ServiceFailureDecider interface {
	EvaluateShipment(
		ctx context.Context,
		req *servicefailureports.EvaluateShipmentServiceFailuresRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureEvaluationResult, error)
	EvaluateStop(
		ctx context.Context,
		req *servicefailureports.EvaluateStopServiceFailuresRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureEvaluationResult, error)
	BulkEvaluate(
		ctx context.Context,
		req *servicefailureports.BulkEvaluateServiceFailuresRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureEvaluationResult, error)
	GetByID(
		ctx context.Context,
		req *servicefailureports.GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	Resolve(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
	PreviewEvaluateShipment(
		ctx context.Context,
		req *servicefailureports.EvaluateShipmentServiceFailuresRequest,
	) (*servicefailureports.ServiceFailureDetectionPlan, error)
}

// errBothEvaluationScopes answers a call whose shipmentId and shipmentIds name
// different shipments. One shipment checks its stops, or one of them; a list
// checks each shipment in it. Which the person meant cannot be read from the
// call, so the refusal spells out both calls.
var ErrBothEvaluationScopes = errors.New(
	"shipmentId and shipmentIds name different shipments, so it is not clear which to " +
		"check. To check one shipment, send shipmentId without shipmentIds, for example " +
		`{"shipmentId": "shp_…"}, adding stopId to check one of its stops. To check several ` +
		`shipments in one pass, send shipmentIds without shipmentId, for example ` +
		`{"shipmentIds": ["shp_…", "shp_…"]}`,
)

type EvaluateServiceFailuresTool struct {
	failures ServiceFailureDecider
}

func NewEvaluateServiceFailuresTool(failures ServiceFailureDecider) agenttoolports.AgentTool {
	return &EvaluateServiceFailuresTool{failures: failures}
}

func (t *EvaluateServiceFailuresTool) Name() string { return "exception_evaluate_service_failures" }

func (t *EvaluateServiceFailuresTool) Recipe() []string {
	return []string{
		toolparams.ToolShipmentGetTracking,
		"exception_list_service_failures",
		"exception_evaluate_service_failures",
	}
}

func (t *EvaluateServiceFailuresTool) Description() string {
	return "Run the service failure check on a shipment now, opening a failure for a " +
		"late or missed stop that has none. Every stop is compared against its window " +
		"and the grace period; send stopId to check one stop, or shipmentIds to check " +
		"several shipments in one pass. Use it when shipment_get_tracking shows a stop " +
		"late or overdue and exception_list_service_failures shows nothing for it, so the " +
		"failure is on record before anyone is told. It records what the stop " +
		"actuals prove and nothing else; it cannot create a failure from a guess."
}

func (t *EvaluateServiceFailuresTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(
				permission.ResourceShipment,
				"The shipment to check, from shipment_get_tracking, "+
					"shipment_list or this run's subject. For several shipments send "+
					"shipmentIds instead.",
			),
			toolparams.ParamStopID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentStop,
				"One stop of that shipment to check on its own, from "+
					"shipment_get_tracking or shipment_get. Needs shipmentId.",
			),
			"shipmentIds": agenttoolschema.RecordIDs(
				permission.ResourceShipment,
				"Several shipments to check in one pass, from "+
					"shipment_list. Leave it out when checking one "+
					"shipment by shipmentId.",
				MaxBulkEvaluationShipments,
			),
			"force": map[string]any{
				toolschema.KeyType: toolschema.TypeBoolean,
				toolschema.KeyDescription: "Re-check stops that were already evaluated. Default false; " +
					"set it only when an actual arrival was corrected.",
			},
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *EvaluateServiceFailuresTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceServiceFailure,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Opens service failures from stop actuals inside Trenova; an open failure " +
			"sends nothing.",
	}
}

func (t *EvaluateServiceFailuresTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

func (t *EvaluateServiceFailuresTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return err
	}

	scope, err := t.request(&params)
	if err != nil {
		return err
	}

	switch {
	case len(scope.shipmentIDs) > 0:
		_, err = t.failures.BulkEvaluate(
			ctx,
			&servicefailureports.BulkEvaluateServiceFailuresRequest{
				TenantInfo:  scope.tenant,
				ShipmentIDs: scope.shipmentIDs,
				Force:       scope.force,
			},
			params.Actor,
		)
	case scope.stopID.IsNotNil():
		_, err = t.failures.EvaluateStop(
			ctx,
			&servicefailureports.EvaluateStopServiceFailuresRequest{
				TenantInfo: scope.tenant,
				ShipmentID: scope.shipmentID,
				StopID:     scope.stopID,
				Force:      scope.force,
			},
			params.Actor,
		)
	default:
		_, err = t.failures.EvaluateShipment(ctx, scope.shipmentRequest(scope.shipmentID),
			params.Actor)
	}

	return err
}

const MaxBulkEvaluationShipments = 50

type evaluationScope struct {
	tenant      pagination.TenantInfo
	shipmentID  pulid.ID
	stopID      pulid.ID
	shipmentIDs []pulid.ID
	force       bool
}

func (s *evaluationScope) shipmentRequest(
	shipmentID pulid.ID,
) *servicefailureports.EvaluateShipmentServiceFailuresRequest {
	return &servicefailureports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: s.tenant,
		ShipmentID: shipmentID,
		Force:      s.force,
	}
}

func (t *EvaluateServiceFailuresTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*evaluationScope, error) {
	scope := &evaluationScope{
		tenant: toolparams.TenantFrom(params),
		force:  toolparams.OptionalBool(params.Params, "force"),
	}

	var err error
	if scope.shipmentID, _, err = toolparams.OptionalPulid(
		params.Params,
		toolparams.ParamShipmentID,
	); err != nil {
		return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", err)
	}
	if scope.stopID, _, err = toolparams.OptionalPulid(
		params.Params,
		toolparams.ParamStopID,
	); err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", toolparams.ParamStopID, err)
	}
	if _, given := params.Params["shipmentIds"]; given {
		ids, idsErr := toolparams.RequirePulidSlice(
			params.Params, "shipmentIds", MaxBulkEvaluationShipments)
		if idsErr != nil {
			return nil, idsErr
		}
		scope.shipmentIDs = sliceutils.Dedupe(ids)
	}
	scope.oneListed()

	switch {
	case len(scope.shipmentIDs) > 0 && scope.shipmentID.IsNotNil():
		return nil, ErrBothEvaluationScopes
	case len(scope.shipmentIDs) > 0 && scope.stopID.IsNotNil():
		return nil, errors.New("stopId checks one stop of one shipment; send it with shipmentId")
	case len(scope.shipmentIDs) == 0 && scope.shipmentID.IsNil():
		return nil, errors.New("parameter \"shipmentId\" is required unless shipmentIds is sent")
	}

	return scope, nil
}

// oneListed reads a list of one shipment as that shipment when the call also
// names it, or names a stop of it. Models fill every parameter they are
// offered, so a check of one shipment arrived as shipmentId and as shipmentIds
// of that same shipment, and was refused for naming both, though both said
// the same thing.
func (s *evaluationScope) oneListed() {
	if len(s.shipmentIDs) != 1 {
		return
	}
	only := s.shipmentIDs[0]
	switch {
	case s.shipmentID == only:
	case s.shipmentID.IsNil() && s.stopID.IsNotNil():
		s.shipmentID = only
	default:
		return
	}
	s.shipmentIDs = nil
}

// ResolveSelection files a call that named its one shipment twice as the
// single shipmentId it means, so the approval card shows one shipment rather
// than a list beside it. A call the scope cannot read is left as sent for
// Validate to refuse by name.
func (t *EvaluateServiceFailuresTool) ResolveSelection(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	if _, listed := params.Params["shipmentIds"]; !listed {
		return params.Params, nil
	}
	scope, err := t.request(&params)
	if err != nil || len(scope.shipmentIDs) > 0 {
		return params.Params, nil //nolint:nilerr // Validate refuses the call with its own reason
	}

	resolved := maputils.WithoutKeys(params.Params, "shipmentIds")
	resolved[toolparams.ParamShipmentID] = scope.shipmentID.String()

	return resolved, nil
}

type ResolveServiceFailureTool struct {
	failures serviceFailureResolver
}

func NewResolveServiceFailureTool(failures serviceFailureResolver) agenttoolports.AgentTool {
	return &ResolveServiceFailureTool{failures: failures}
}

func (t *ResolveServiceFailureTool) Name() string { return "exception_resolve_service_failure" }

func (t *ResolveServiceFailureTool) Recipe() []string {
	return []string{
		"exception_list_service_failures",
		"exception_get_service_failure",
		toolparams.ToolShipmentGetTracking,
		"exception_list_service_failure_reason_codes",
		"exception_resolve_service_failure",
	}
}

func (t *ResolveServiceFailureTool) SearchTerms() []string {
	return []string{"close service failure", "close out failure"}
}

func (t *ResolveServiceFailureTool) Description() string {
	return "Close an open service failure with the reason it happened and a note on " +
		"what was done. The reason code comes from exception_list_service_failure_reason_codes " +
		"and must match the cause; a failure that already carries one keeps it " +
		"unless you send another. Do not resolve a failure nobody has looked into: " +
		"the note should say what was learned, not that the stop was late."
}

func (t *ResolveServiceFailureTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"serviceFailureId": agenttoolschema.RecordIDText(
				permission.ResourceServiceFailure,
				"The failure's id, from exception_list_service_failures.",
			),
			"reasonCodeId": agenttoolschema.RecordIDText(
				permission.ResourceServiceFailureReasonCode,
				"The reason code, from exception_list_service_failure_reason_codes. "+
					"Required when the failure has none; otherwise replaces it.",
			),
			toolparams.ParamNote: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "What was found and done. Required.",
			},
		},
		toolschema.KeyRequired:             []string{"serviceFailureId", toolparams.ParamNote},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *ResolveServiceFailureTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceServiceFailure,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Resolving a failure generates an EDI 214 to the customer's trading " +
			"partner carrying the reason chosen.",
	}
}

func (t *ResolveServiceFailureTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "serviceFailureId", permission.ResourceServiceFailure)
}

func (t *ResolveServiceFailureTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	request, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.failures.Resolve(ctx, request, params.Actor)

	return err
}

// request is the resolution the preview shows and the write makes: an open
// failure, closed at the version read, with the reason code the call sends
// or the one already on file.
func (t *ResolveServiceFailureTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*servicefailureports.ServiceFailureLifecycleRequest, *servicefailure.ServiceFailure, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, nil, err
	}

	failureID, err := toolparams.RequirePulid(params.Params, "serviceFailureId")
	if err != nil {
		return nil, nil, err
	}
	notes, err := toolparams.RequireString(params.Params, toolparams.ParamNote)
	if err != nil {
		return nil, nil, err
	}

	tenant := toolparams.TenantFrom(params)
	existing, err := t.failures.GetByID(ctx, &servicefailureports.GetServiceFailureByIDRequest{
		ID:         failureID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, nil, err
	}
	if existing.IsTerminal() {
		return nil, nil, fmt.Errorf(
			"service failure %s is already %s and cannot be resolved again",
			existing.Number, existing.Status,
		)
	}

	request := &servicefailureports.ServiceFailureLifecycleRequest{
		TenantInfo: tenant,
		ID:         existing.ID,
		ShipmentID: existing.ShipmentID,
		Notes:      strings.TrimSpace(notes),
		Version:    existing.Version,
	}
	if raw := toolparams.OptionalString(params.Params, "reasonCodeId"); raw != "" {
		reasonID, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			return nil, nil, fmt.Errorf(
				"parameter \"reasonCodeId\" is not a valid id: %w", parseErr,
			)
		}
		request.ReasonCodeID = reasonID
	} else if existing.ReasonCodeID == nil || existing.ReasonCodeID.IsNil() {
		return nil, nil, errors.New(
			"this failure has no reason code yet; pick one from " +
				"exception_list_service_failure_reason_codes and send it as reasonCodeId",
		)
	}

	return request, existing, nil
}

func (t *EvaluateServiceFailuresTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plans, err := t.plans(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would check the shipment's stops for service failures."),
				err,
			), nil
		}

		return nil, err
	}

	summaries := make([]string, 0, len(plans))
	changes := make([]*agentpreview.RecordChange, 0, len(plans))
	for _, plan := range plans {
		planChanges, changeErr := detectedFailureChanges(plan)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, planChanges...)
		summaries = append(summaries, detectionSummary(plan))
	}

	return toolpreview.Build(strings.Join(summaries, " "), changes...), nil
}

func (t *EvaluateServiceFailuresTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.plans(ctx, &params)

	return err
}

func (t *EvaluateServiceFailuresTool) plans(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) ([]*servicefailureports.ServiceFailureDetectionPlan, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	scope, err := t.request(params)
	if err != nil {
		return nil, err
	}

	shipmentIDs := scope.shipmentIDs
	if len(shipmentIDs) == 0 {
		shipmentIDs = []pulid.ID{scope.shipmentID}
	}

	plans := make([]*servicefailureports.ServiceFailureDetectionPlan, 0, len(shipmentIDs))
	for _, shipmentID := range shipmentIDs {
		plan, planErr := t.failures.PreviewEvaluateShipment(ctx, scope.shipmentRequest(shipmentID))
		if planErr != nil {
			return nil, planErr
		}
		if scope.stopID.IsNotNil() {
			plan = restrictDetectionToStop(plan, scope.stopID)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

func (t *ResolveServiceFailureTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, existing, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	resolved, err := t.failures.PreviewResolve(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceServiceFailure,
			ID:       existing.ID,
			Label:    "Service failure " + existing.Number,
			Version:  changepreview.PinnedVersion(existing.Version),
		},
		resolved.Before,
		resolved.After,
		toolpreview.Only("status", "reasonCodeId", "internalNotes", "resolvedAt", "resolvedById"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"reasonCodeId": permission.ResourceServiceFailureReasonCode,
			"resolvedById": permission.ResourceUser,
		}),
		toolpreview.Volatile("resolvedAt"),
	)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would resolve service failure %s as %s.",
		existing.Number, reasonCodeWords(resolved.After))

	return ServiceFailureLifecyclePreview(summary, change, resolved, "resolved"), nil
}

func (t *ResolveServiceFailureTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}
