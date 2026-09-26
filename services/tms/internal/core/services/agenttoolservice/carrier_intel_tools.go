package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramCarrierIDs          = "carrierIds"
	paramDepth               = "depth"
	paramEnabled             = "enabled"
	paramSuggestionFields    = "fields"
	paramPolicyIDs           = "policyIds"
	paramDOTNumber           = "dotNumber"
	paramCarrierCode         = "code"
	paramEnrollMonitoring    = "enrollMonitoring"
	paramCarrierAssignmentID = "carrierAssignmentId"
	paramUnitType            = "unitType"
	paramVIN                 = "vin"
	paramPlateNumber         = "plateNumber"
	paramPlateState          = "plateState"
	paramUnitNumber          = "unitNumber"
	kindCarrierIntel         = "carrier intelligence"
	maxMonitoredCarriers     = 50
	maxSuggestionPolicies    = 20
	maxDOTNumberChars        = 12
	maxCarrierCodeChars      = 10
	maxVINChars              = 17
	maxPlateChars            = 15
	maxPlateStateChars       = 2
	maxUnitNumberChars       = 50
	maxReviewNoteChars       = 2000
)

var (
	lookupDepths = []carrierintel.LookupDepth{
		carrierintel.LookupDepthFMCSA,
		carrierintel.LookupDepthLite,
		carrierintel.LookupDepthFull,
	}
	unitTypes = []carrierintel.UnitType{
		carrierintel.UnitTypeTractor,
		carrierintel.UnitTypeTrailer,
		carrierintel.UnitTypeStraight,
	}
)

type carrierIntelOperator interface {
	VetCarrier(
		ctx context.Context,
		req *carrierintelservice.VetCarrierRequest,
	) (*carrierintelservice.FetchResult, error)
	PlanVetCarrier(
		ctx context.Context,
		req *carrierintelservice.VetCarrierRequest,
	) (*carrierintelservice.VetPlan, error)
	VetCustomerBroker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		force bool,
	) (*carrierintelservice.FetchResult, error)
	PlanVetCustomerBroker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*carrierintelservice.VetPlan, error)
	SetManualEnrollment(
		ctx context.Context,
		req *carrierintelservice.EnrollSubjectsRequest,
	) (int, error)
	PlanManualEnrollment(
		ctx context.Context,
		req *carrierintelservice.EnrollSubjectsRequest,
	) ([]carrierintelservice.EnrollmentChange, error)
	MarkReviewed(ctx context.Context, req *carrierintelservice.MarkReviewedRequest) error
	PlanMarkReviewed(
		ctx context.Context,
		req *carrierintelservice.MarkReviewedRequest,
	) (*carrierintelservice.SnapshotChange, error)
	SyncPlan(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierID pulid.ID,
	) (*carrierintel.SyncPlan, error)
	ApplySuggestions(
		ctx context.Context,
		req *carrierintelservice.ApplySuggestionsRequest,
	) (int, error)
	PlanApplySuggestions(
		ctx context.Context,
		req *carrierintelservice.ApplySuggestionsRequest,
	) (*carrierintelservice.SuggestionPlan, error)
	ImportProspect(
		ctx context.Context,
		req *carrierintelservice.ImportProspectRequest,
	) (*carrier.Carrier, error)
	PlanImportProspect(
		ctx context.Context,
		req *carrierintelservice.ImportProspectRequest,
	) (*carrierintelservice.ImportPlan, error)
	VerifyEquipment(
		ctx context.Context,
		req *carrierintelservice.VerifyEquipmentRequest,
	) (*carrierintel.CarrierEquipmentVerification, error)
	PlanVerifyEquipment(
		ctx context.Context,
		req *carrierintelservice.VerifyEquipmentRequest,
	) (*carrierintel.CarrierEquipmentVerification, *carrier.Carrier, error)
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *serviceports.PermissionCheckRequest,
	) (*serviceports.PermissionCheckResult, error)
}

// requireCarrierGrant is the second grant a resolver checks after the tool's own: a
// write that changes a carrier as well as its intelligence needs both.
func requireCarrierGrant(
	ctx context.Context,
	permissions permissionChecker,
	actor *serviceports.RequestActor,
	operation permission.Operation,
) error {
	if permissions == nil {
		return errors.New("the person's access to carriers could not be checked")
	}
	result, err := permissions.Check(ctx,
		actor.PermissionCheck(permission.ResourceCarrier, operation))
	if err != nil {
		return fmt.Errorf("authorize %s on carriers: %w", operation, err)
	}
	if !result.Allowed {
		return fmt.Errorf("the person you act for may not %s carrier records", operation)
	}

	return nil
}

func carrierIDProperty(what string) map[string]any {
	return idProperty(what + ", from list_carriers or get_carrier. Never guess one.")
}

func targetCarrier(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramCarrierID, permission.ResourceCarrier)
}

func carrierIntelResult(action string, ids map[string]string) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindCarrierIntel, IDs: ids}
	if id, err := pulid.Parse(ids[paramCarrierID]); err == nil {
		result.Record = recordOf(carrierRecordEntity, id)
	} else if id, err = pulid.Parse(ids[fieldCustomerID]); err == nil {
		result.Record = recordOf(customerRecordEntity, id)
	}

	return result
}

func vetPreview(
	subjectResource permission.Resource,
	subjectID pulid.ID,
	plan *carrierintelservice.VetPlan,
	force bool,
) *agent.ToolPreview {
	name := strings.TrimSpace(plan.Subject.Name)
	if name == "" {
		name = "DOT " + plan.Subject.DOTNumber
	}
	summary := fmt.Sprintf("Would ask %s for %s (DOT %s) at %s depth and record what it says.",
		plan.Provider.String(), name, plan.Subject.DOTNumber, plan.Depth)
	switch {
	case plan.Current == nil:
		summary += " There is no earlier result for it."
	case force:
		summary += " A fresh lookup is made even if the last one is still current, which " +
			"is charged."
	default:
		summary += " A result still current is reused instead of charged again."
	}
	preview := toolpreview.Build(summary, &agent.RecordChange{
		Resource:  subjectResource,
		EntityID:  subjectID,
		Label:     name,
		Operation: agent.PreviewOperationRun,
	})
	preview.Partial = true

	return preview
}

func newVetCarrierTool(intel carrierIntelOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "vet_carrier",
		artifact: carrierRecordEntity,
		description: "Look a carrier up with the organization's carrier intelligence provider " +
			"now: authority, insurance, safety and the organization's rules, recorded on the " +
			"carrier. Use it before tendering to a carrier nobody has checked lately. A " +
			"result still current is reused unless force is true.",
		resource:    permission.ResourceCarrierIntelligence,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Spends the organization's carrier intelligence budget on a lookup and " +
			"records the result; nothing is sent to the carrier.",
		properties: map[string]any{
			paramCarrierID: carrierIDProperty("The carrier"),
			paramDepth: enumProperty("How deep a lookup: FMCSA is the public record, Lite and "+
				"Full add the provider's profile. Defaults to Full.", lookupDepths),
			paramForceResend: booleanProperty("Look up again even when the last result is " +
				"still current. It is charged; only when the person asks."),
		},
		required: []string{paramCarrierID},
		target:   targetCarrier,
	}, receivablePlan[*carrierintelservice.VetCarrierRequest, *carrierintelservice.VetPlan]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*carrierintelservice.VetCarrierRequest, error) {
			carrierID, err := requirePulid(params.Params, paramCarrierID)
			if err != nil {
				return nil, err
			}
			depth, _, err := optionalEnum(params.Params, paramDepth, lookupDepths)
			if err != nil {
				return nil, err
			}
			force, err := optionalBoolPointer(params.Params, paramForceResend)
			if err != nil {
				return nil, err
			}

			return &carrierintelservice.VetCarrierRequest{
				TenantInfo: tenantFrom(*params),
				CarrierID:  carrierID,
				Depth:      depth,
				Force:      force != nil && *force,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *carrierintelservice.VetCarrierRequest,
			_ *serviceports.ToolExecuteParams,
		) (*carrierintelservice.VetPlan, error) {
			return intel.PlanVetCarrier(ctx, req)
		},
		refused: func(*carrierintelservice.VetCarrierRequest) string {
			return "Would vet the carrier."
		},
		render: func(
			req *carrierintelservice.VetCarrierRequest,
			plan *carrierintelservice.VetPlan,
		) (*agent.ToolPreview, error) {
			return vetPreview(permission.ResourceCarrier, req.CarrierID, plan, req.Force), nil
		},
		run: func(
			ctx context.Context,
			req *carrierintelservice.VetCarrierRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			result, err := intel.VetCarrier(ctx, req)
			if err != nil {
				return nil, err
			}

			return fetchResult(
				result,
				map[string]string{paramCarrierID: req.CarrierID.String()},
			), nil
		},
	})
}

func fetchResult(
	result *carrierintelservice.FetchResult,
	ids map[string]string,
) *agent.ToolExecutionResult {
	out := carrierIntelResult("vetted", ids)
	if result == nil || result.Snapshot == nil {
		return out
	}
	out.Name = string(result.Snapshot.RiskLevel)
	if result.FromCache {
		out.Action = "vetted from the current result"
	}

	return out
}

type brokerVet struct {
	tenant     pagination.TenantInfo
	customerID pulid.ID
	force      bool
}

func newVetCustomerBrokerTool(intel carrierIntelOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "vet_customer_broker",
		artifact: customerRecordEntity,
		description: "Check a customer that is a freight broker against the public FMCSA " +
			"record by its DOT number: broker authority and bond. Use it before taking " +
			"freight from a broker nobody has checked lately.",
		resource:    permission.ResourceCarrierIntelligence,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Spends the organization's carrier intelligence budget on a lookup and " +
			"records the result; nothing is sent to the customer.",
		properties: map[string]any{
			fieldCustomerID: idProperty("The broker customer, from list_customers. Never " +
				"guess one."),
			paramForceResend: booleanProperty("Look up again even when the last result is " +
				"still current. It is charged; only when the person asks."),
		},
		required: []string{fieldCustomerID},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, fieldCustomerID, permission.ResourceCustomer)
		},
	}, receivablePlan[*brokerVet, *carrierintelservice.VetPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*brokerVet, error) {
			customerID, err := requirePulid(params.Params, fieldCustomerID)
			if err != nil {
				return nil, err
			}
			force, err := optionalBoolPointer(params.Params, paramForceResend)
			if err != nil {
				return nil, err
			}

			return &brokerVet{
				tenant:     tenantFrom(*params),
				customerID: customerID,
				force:      force != nil && *force,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *brokerVet,
			_ *serviceports.ToolExecuteParams,
		) (*carrierintelservice.VetPlan, error) {
			return intel.PlanVetCustomerBroker(ctx, req.tenant, req.customerID)
		},
		refused: func(*brokerVet) string { return "Would vet the broker." },
		render: func(req *brokerVet, plan *carrierintelservice.VetPlan) (*agent.ToolPreview, error) {
			return vetPreview(permission.ResourceCustomer, req.customerID, plan, req.force), nil
		},
		run: func(
			ctx context.Context,
			req *brokerVet,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			result, err := intel.VetCustomerBroker(ctx, req.tenant, req.customerID, req.force)
			if err != nil {
				return nil, err
			}

			return fetchResult(
				result,
				map[string]string{fieldCustomerID: req.customerID.String()},
			), nil
		},
	})
}

type monitoringView struct {
	Monitoring carrierintel.DesiredState `json:"monitoring"`
}

func newSetCarrierMonitoringTool(intel carrierIntelOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "set_carrier_monitoring",
		artifact: carrierRecordEntity,
		description: "Turn continuous monitoring on or off for carriers, so the provider " +
			"reports authority, insurance and safety changes as they happen. Carriers " +
			"without a DOT number are skipped.",
		resource:    permission.ResourceCarrierIntelligence,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Monitoring is billed per carrier by the provider; turning it off again " +
			"undoes it, and nothing is sent to the carriers.",
		properties: map[string]any{
			paramCarrierIDs: toolschema.RecordSubset(permission.ResourceCarrier.String(),
				idListProperty("The carriers, from list_carriers.", maxMonitoredCarriers)),
			paramEnabled: booleanProperty("True to monitor them, false to stop."),
		},
		required: []string{paramCarrierIDs, paramEnabled},
	}, receivablePlan[*carrierintelservice.EnrollSubjectsRequest, []carrierintelservice.EnrollmentChange]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*carrierintelservice.EnrollSubjectsRequest, error) {
			ids, err := requirePulidSlice(params.Params, paramCarrierIDs, maxMonitoredCarriers)
			if err != nil {
				return nil, err
			}
			enabled, err := optionalBoolPointer(params.Params, paramEnabled)
			if err != nil {
				return nil, err
			}
			if enabled == nil {
				return nil, fmt.Errorf("missing required parameter %q", paramEnabled)
			}

			return &carrierintelservice.EnrollSubjectsRequest{
				TenantInfo: tenantFrom(*params),
				CarrierIDs: ids,
				Enroll:     *enabled,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *carrierintelservice.EnrollSubjectsRequest,
			_ *serviceports.ToolExecuteParams,
		) ([]carrierintelservice.EnrollmentChange, error) {
			return intel.PlanManualEnrollment(ctx, req)
		},
		refused: func(*carrierintelservice.EnrollSubjectsRequest) string {
			return "Would change carrier monitoring."
		},
		render: renderMonitoring,
		run: func(
			ctx context.Context,
			req *carrierintelservice.EnrollSubjectsRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			changed, err := intel.SetManualEnrollment(ctx, req)
			if err != nil {
				return nil, err
			}
			action := "monitoring stopped"
			if req.Enroll {
				action = "monitoring started"
			}
			out := carrierIntelResult(action, nil)
			out.Name = countOf(changed, "carrier")

			return out, nil
		},
	})
}

func renderMonitoring(
	req *carrierintelservice.EnrollSubjectsRequest,
	changes []carrierintelservice.EnrollmentChange,
) (*agent.ToolPreview, error) {
	records := make([]*agent.RecordChange, 0, len(changes))
	for idx := range changes {
		change := &changes[idx]
		if change.Before == change.After {
			continue
		}
		record, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceCarrier,
				ID:       change.CarrierID,
				Label:    change.Name,
			},
			&monitoringView{Monitoring: change.Before},
			&monitoringView{Monitoring: change.After},
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	verb := "stop monitoring"
	if req.Enroll {
		verb = "start monitoring"
	}
	summary := fmt.Sprintf("Would %s %s.", verb, countOf(len(records), "carrier"))
	if skipped := len(req.CarrierIDs) - len(changes); skipped > 0 {
		summary += fmt.Sprintf(" %d have no DOT number and are skipped.", skipped)
	}
	if unchanged := len(changes) - len(records); unchanged > 0 {
		summary += fmt.Sprintf(" %d are already that way.", unchanged)
	}

	return toolpreview.Build(summary, records...), nil
}

func newMarkCarrierIntelReviewedTool(intel carrierIntelOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "mark_carrier_intel_reviewed",
		artifact: carrierRecordEntity,
		description: "Record that a person reviewed a carrier's current intelligence " +
			"result, with what they checked. It is their sign-off, so it only runs once " +
			"they approve it.",
		resource:    permission.ResourceCarrierIntelligence,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Puts a person's name on a carrier risk review; only that person can " +
			"say they reviewed it.",
		properties: map[string]any{
			paramCarrierID: carrierIDProperty("The carrier"),
			paramNote: stringProperty("What was reviewed and concluded.",
				maxReviewNoteChars),
		},
		required: []string{paramCarrierID, paramNote},
		target:   targetCarrier,
	}, receivablePlan[*carrierintelservice.MarkReviewedRequest, *carrierintelservice.SnapshotChange]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*carrierintelservice.MarkReviewedRequest, error) {
			carrierID, err := requirePulid(params.Params, paramCarrierID)
			if err != nil {
				return nil, err
			}
			note, err := requireBoundedText(params.Params, paramNote, maxReviewNoteChars)
			if err != nil {
				return nil, err
			}

			return &carrierintelservice.MarkReviewedRequest{
				TenantInfo: tenantFrom(*params),
				CarrierID:  carrierID,
				Note:       note,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *carrierintelservice.MarkReviewedRequest,
			_ *serviceports.ToolExecuteParams,
		) (*carrierintelservice.SnapshotChange, error) {
			return intel.PlanMarkReviewed(ctx, req)
		},
		refused: func(*carrierintelservice.MarkReviewedRequest) string {
			return "Would mark the carrier's intelligence reviewed."
		},
		render: func(
			req *carrierintelservice.MarkReviewedRequest,
			plan *carrierintelservice.SnapshotChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{Resource: permission.ResourceCarrier, ID: req.CarrierID},
				plan.Before,
				plan.After,
				toolpreview.Only(fieldReviewedByID, fieldReviewedAt, "reviewNote"),
				toolpreview.Volatile(fieldReviewedAt),
				toolpreview.WithRefs(map[string]permission.Resource{
					fieldReviewedByID: permission.ResourceUser,
				}),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would record the person's review of the carrier's %s risk result.",
				strings.ToLower(string(plan.Before.RiskLevel)),
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *carrierintelservice.MarkReviewedRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := intel.MarkReviewed(ctx, req); err != nil {
				return nil, err
			}

			return carrierIntelResult("reviewed",
				map[string]string{paramCarrierID: req.CarrierID.String()}), nil
		},
	})
}

type suggestionChoice struct {
	req *carrierintelservice.ApplySuggestionsRequest
}

// resolve fills a call that named no suggestion with every one carrier
// intelligence makes now, which the preview then lists one by one.
func (c *suggestionChoice) resolve(
	ctx context.Context,
	intel carrierIntelOperator,
) (*carrierintelservice.ApplySuggestionsRequest, error) {
	if len(c.req.Fields) > 0 || len(c.req.PolicyIDs) > 0 {
		return c.req, nil
	}
	plan, err := intel.SyncPlan(ctx, c.req.TenantInfo, c.req.CarrierID)
	if err != nil {
		return nil, err
	}
	resolved := *c.req
	resolved.Fields = make([]carrierintel.SyncField, 0, len(plan.Suggestions))
	for idx := range plan.Suggestions {
		resolved.Fields = append(resolved.Fields, plan.Suggestions[idx].Field)
	}
	resolved.PolicyIDs = make([]pulid.ID, 0, len(plan.InsuranceSuggestions))
	for idx := range plan.InsuranceSuggestions {
		if policyID := plan.InsuranceSuggestions[idx].PolicyID; policyID.IsNotNil() {
			resolved.PolicyIDs = append(resolved.PolicyIDs, policyID)
		}
	}

	return &resolved, nil
}

func newApplyCarrierIntelSuggestionsTool(
	intel carrierIntelOperator,
	permissions permissionChecker,
) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "apply_carrier_intel_suggestions",
		artifact: carrierRecordEntity,
		description: "Correct a carrier's profile where its intelligence result disagrees: " +
			"name, DBA, MC number, safety rating, address, phone, email or insurance. Name the fields and policies to apply, or leave " +
			"both out to apply every suggestion there is now.",
		resource:    permission.ResourceCarrierIntelligence,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Edits the carrier's own profile inside Trenova from what the provider " +
			"reports; nothing is sent, and the old values are set back by hand.",
		properties: map[string]any{
			paramCarrierID: carrierIDProperty("The carrier"),
			paramSuggestionFields: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The profile fields to correct.",
				toolschema.KeyMaxItems:    len(carrierintel.AllSyncFields()),
				toolschema.KeyItems: enumProperty("A profile field.",
					carrierintel.AllSyncFields()),
			},
			paramPolicyIDs: idListProperty("The insurance policies to correct, by the policy "+
				"id on the page you are on.", maxSuggestionPolicies),
		},
		required: []string{paramCarrierID},
		target:   targetCarrier,
	}, receivablePlan[*suggestionChoice, *carrierintelservice.SuggestionPlan]{
		request: readSuggestionChoice,
		plan: func(
			ctx context.Context,
			choice *suggestionChoice,
			params *serviceports.ToolExecuteParams,
		) (*carrierintelservice.SuggestionPlan, error) {
			if err := requireCarrierGrant(
				ctx,
				permissions,
				params.Actor,
				permission.OpUpdate,
			); err != nil {
				return nil, err
			}
			req, err := choice.resolve(ctx, intel)
			if err != nil {
				return nil, err
			}

			return intel.PlanApplySuggestions(ctx, req)
		},
		refused: func(*suggestionChoice) string {
			return "Would correct the carrier's profile from its intelligence."
		},
		render: renderSuggestions,
		run: func(
			ctx context.Context,
			choice *suggestionChoice,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := requireCarrierGrant(
				ctx,
				permissions,
				params.Actor,
				permission.OpUpdate,
			); err != nil {
				return nil, err
			}
			req, err := choice.resolve(ctx, intel)
			if err != nil {
				return nil, err
			}
			applied, err := intel.ApplySuggestions(ctx, req)
			if err != nil {
				return nil, err
			}
			out := carrierIntelResult("suggestions applied",
				map[string]string{paramCarrierID: req.CarrierID.String()})
			out.Name = countOf(applied, "correction")

			return out, nil
		},
	})
}

func readSuggestionChoice(params *serviceports.ToolExecuteParams) (*suggestionChoice, error) {
	carrierID, err := requirePulid(params.Params, paramCarrierID)
	if err != nil {
		return nil, err
	}
	req := &carrierintelservice.ApplySuggestionsRequest{
		TenantInfo: tenantFrom(*params),
		CarrierID:  carrierID,
	}
	if _, given := params.Params[paramSuggestionFields]; given {
		var names []string
		if err = decodeParam(params.Params, paramSuggestionFields, &names); err != nil {
			return nil, err
		}
		for _, name := range names {
			field, fieldErr := requireEnum(map[string]any{paramSuggestionFields: name},
				paramSuggestionFields, carrierintel.AllSyncFields())
			if fieldErr != nil {
				return nil, fieldErr
			}
			req.Fields = append(req.Fields, field)
		}
	}
	if _, given := params.Params[paramPolicyIDs]; given {
		if req.PolicyIDs, err = requirePulidSlice(params.Params, paramPolicyIDs,
			maxSuggestionPolicies); err != nil {
			return nil, err
		}
	}

	return &suggestionChoice{req: req}, nil
}

func renderSuggestions(
	_ *suggestionChoice,
	plan *carrierintelservice.SuggestionPlan,
) (*agent.ToolPreview, error) {
	fields := make([]string, 0, len(plan.Fields))
	for _, update := range plan.Fields {
		fields = append(fields, update.Field.String())
	}
	changes := make([]*agent.RecordChange, 0, 1)
	if len(fields) > 0 {
		change, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceCarrier,
				ID:       plan.Before.ID,
				Label:    plan.Before.Name,
				Version:  pinnedVersion(plan.Before.Version),
			},
			plan.Before,
			plan.After,
			toolpreview.Only(fields...),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would correct %s on carrier %s from its intelligence result",
		countOf(len(plan.Fields), "profile field"), plan.Before.Name)
	if len(plan.Insurance) > 0 {
		summary += fmt.Sprintf(" and update %s", countOf(len(plan.Insurance), "insurance policy"))
	}

	return toolpreview.Build(summary+".", changes...), nil
}

func newImportSourcedCarrierTool(
	intel carrierIntelOperator,
	permissions permissionChecker,
) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "import_sourced_carrier",
		artifact: carrierRecordEntity,
		description: "Add a carrier found through carrier sourcing to the organization's " +
			"carriers by its DOT number, filled from the provider's profile. A DOT number " +
			"already on a carrier is refused.",
		resource:    permission.ResourceCarrierSourcing,
		operation:   permission.OpImport,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Creates a carrier inside Trenova from the provider's profile; nothing is " +
			"sent to it, and one added in error is made inactive.",
		properties: map[string]any{
			paramDOTNumber: stringProperty("The carrier's USDOT number, from the sourcing "+
				"search.", maxDOTNumberChars),
			paramCarrierCode: stringProperty("The short code to file it under. Left out, one "+
				"is made from its name.", maxCarrierCodeChars),
			paramEnrollMonitoring: booleanProperty("Start monitoring it at once. Monitoring " +
				"is billed per carrier."),
		},
		required: []string{paramDOTNumber},
	}, receivablePlan[*carrierintelservice.ImportProspectRequest, *carrierintelservice.ImportPlan]{
		request: importProspectRequest,
		plan: func(
			ctx context.Context,
			req *carrierintelservice.ImportProspectRequest,
			params *serviceports.ToolExecuteParams,
		) (*carrierintelservice.ImportPlan, error) {
			if err := requireCarrierGrant(
				ctx,
				permissions,
				params.Actor,
				permission.OpCreate,
			); err != nil {
				return nil, err
			}

			return intel.PlanImportProspect(ctx, req)
		},
		refused: func(*carrierintelservice.ImportProspectRequest) string {
			return "Would add the sourced carrier."
		},
		render: func(
			_ *carrierintelservice.ImportProspectRequest,
			plan *carrierintelservice.ImportPlan,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				toolpreview.Record{
					Resource: permission.ResourceCarrier,
					Label:    "DOT " + plan.DOTNumber,
				},
				&importView{
					DOTNumber:        plan.DOTNumber,
					Code:             plan.Code,
					Source:           plan.Provider.String(),
					EnrollMonitoring: plan.EnrollMonitoring,
				},
			)
			if err != nil {
				return nil, err
			}
			preview := toolpreview.Build(fmt.Sprintf(
				"Would add the carrier with DOT %s, filled from %s's profile.",
				plan.DOTNumber, plan.Provider.String(),
			), change)
			preview.Partial = true

			return preview, nil
		},
		run: func(
			ctx context.Context,
			req *carrierintelservice.ImportProspectRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := requireCarrierGrant(
				ctx,
				permissions,
				params.Actor,
				permission.OpCreate,
			); err != nil {
				return nil, err
			}
			created, err := intel.ImportProspect(ctx, req)
			if err != nil {
				return nil, err
			}
			out := carrierIntelResult("carrier added",
				map[string]string{paramCarrierID: created.ID.String()})
			out.Name = created.Name

			return out, nil
		},
	})
}

type importView struct {
	DOTNumber        string `json:"dotNumber"`
	Code             string `json:"code"`
	Source           string `json:"source"`
	EnrollMonitoring bool   `json:"enrollMonitoring"`
}

func newVerifyCarrierEquipmentTool(intel carrierIntelOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "verify_carrier_equipment",
		artifact: carrierRecordEntity,
		description: "Check that the truck or trailer a carrier sent for a load is registered " +
			"to that carrier, by VIN or plate, and record the answer on the assignment. Take " +
			"the numbers from the driver or the carrier's dispatch, never from a guess.",
		resource:    permission.ResourceEquipmentVerification,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Records an equipment check inside Trenova after a provider lookup; " +
			"nothing is sent to the carrier.",
		properties: map[string]any{
			paramCarrierAssignmentID: idProperty("The carrier assignment, from " +
				"list_rate_confirmations or get_shipment. Never guess one."),
			paramUnitType: enumProperty("What was checked.", unitTypes),
			paramVIN:      stringProperty("The vehicle identification number.", maxVINChars),
			paramPlateNumber: stringProperty("The license plate number.",
				maxPlateChars),
			paramPlateState: stringProperty("The plate's two-letter state.",
				maxPlateStateChars),
			paramUnitNumber: stringProperty("The carrier's own unit number.",
				maxUnitNumberChars),
		},
		required: []string{paramCarrierAssignmentID, paramUnitType},
	}, receivablePlan[*carrierintelservice.VerifyEquipmentRequest, *verificationPlan]{
		request: verifyEquipmentRequest,
		plan: func(
			ctx context.Context,
			req *carrierintelservice.VerifyEquipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*verificationPlan, error) {
			entity, carrierEntity, err := intel.PlanVerifyEquipment(ctx, req)
			if err != nil {
				return nil, err
			}

			return &verificationPlan{verification: entity, carrier: carrierEntity}, nil
		},
		refused: func(*carrierintelservice.VerifyEquipmentRequest) string {
			return "Would verify the carrier's equipment."
		},
		render: func(
			_ *carrierintelservice.VerifyEquipmentRequest,
			plan *verificationPlan,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				toolpreview.Record{
					Resource: permission.ResourceEquipmentVerification,
					Label:    plan.carrier.Name + " equipment check",
				},
				plan.verification,
				toolpreview.Only(paramCarrierID, "expectedDotNumber", paramUnitType, paramVIN,
					paramPlateNumber, paramPlateState, paramUnitNumber),
				toolpreview.WithRefs(map[string]permission.Resource{
					paramCarrierID: permission.ResourceCarrier,
				}),
			)
			if err != nil {
				return nil, err
			}
			preview := toolpreview.Build(fmt.Sprintf(
				"Would look the %s up and record whether it is registered to %s.",
				strings.ToLower(string(plan.verification.UnitType)), plan.carrier.Name,
			), change)
			preview.Partial = true

			return preview, nil
		},
		run: func(
			ctx context.Context,
			req *carrierintelservice.VerifyEquipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := intel.VerifyEquipment(ctx, req)
			if err != nil {
				return nil, err
			}
			out := carrierIntelResult("equipment checked",
				map[string]string{paramCarrierAssignmentID: req.CarrierAssignmentID.String()})
			out.Name = string(created.Result)

			return out, nil
		},
	})
}

type verificationPlan struct {
	verification *carrierintel.CarrierEquipmentVerification
	carrier      *carrier.Carrier
}

func verifyEquipmentRequest(
	params *serviceports.ToolExecuteParams,
) (*carrierintelservice.VerifyEquipmentRequest, error) {
	assignmentID, err := requirePulid(params.Params, paramCarrierAssignmentID)
	if err != nil {
		return nil, err
	}
	unitType, err := requireEnum(params.Params, paramUnitType, unitTypes)
	if err != nil {
		return nil, err
	}
	req := &carrierintelservice.VerifyEquipmentRequest{
		TenantInfo:          tenantFrom(*params),
		CarrierAssignmentID: assignmentID,
		UnitType:            unitType,
	}
	for key, spec := range map[string]struct {
		target *string
		limit  int
		upper  bool
	}{
		paramVIN:         {&req.VIN, maxVINChars, true},
		paramPlateNumber: {&req.PlateNumber, maxPlateChars, true},
		paramPlateState:  {&req.PlateState, maxPlateStateChars, true},
		paramUnitNumber:  {&req.UnitNumber, maxUnitNumberChars, false},
	} {
		value, textErr := boundedText(params.Params, key, spec.limit)
		if textErr != nil {
			return nil, textErr
		}
		if spec.upper {
			value = strings.ToUpper(value)
		}
		*spec.target = value
	}

	return req, nil
}

func importProspectRequest(
	params *serviceports.ToolExecuteParams,
) (*carrierintelservice.ImportProspectRequest, error) {
	dot, err := requireBoundedText(params.Params, paramDOTNumber, maxDOTNumberChars)
	if err != nil {
		return nil, err
	}
	code, err := boundedText(params.Params, paramCarrierCode, maxCarrierCodeChars)
	if err != nil {
		return nil, err
	}
	enroll, err := optionalBoolPointer(params.Params, paramEnrollMonitoring)
	if err != nil {
		return nil, err
	}

	return &carrierintelservice.ImportProspectRequest{
		TenantInfo:       tenantFrom(*params),
		DOTNumber:        dot,
		Code:             code,
		EnrollMonitoring: enroll != nil && *enroll,
	}, nil
}
