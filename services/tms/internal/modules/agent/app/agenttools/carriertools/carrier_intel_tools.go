package carriertools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
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
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramCarrierIDs = "carrierIds"

const paramDepth = "depth"

const ParamEnabled = "enabled"

const paramSuggestionFields = "fields"

const paramPolicyIDs = "policyIds"

const paramDOTNumber = "dotNumber"

const paramCarrierCode = "code"

const paramEnrollMonitoring = "enrollMonitoring"

const paramCarrierAssignmentID = "carrierAssignmentId"

const paramUnitType = "unitType"

const paramVIN = "vin"

const paramPlateNumber = "plateNumber"

const paramPlateState = "plateState"

const paramUnitNumber = "unitNumber"

const kindCarrierIntel = "carrier intelligence"

const maxMonitoredCarriers = 50

const maxSuggestionPolicies = 20

const maxDOTNumberChars = 12

const maxCarrierCodeChars = 10

const maxVINChars = 17

const maxPlateChars = 15

const maxPlateStateChars = 2

const maxUnitNumberChars = 50

const maxReviewNoteChars = 2000

var lookupDepths = agenttoolschema.Source(
	"carrierIntel.lookupDepth",
	carrierintel.LookupDepthValues(),
)

var unitTypes = agenttoolschema.Source("carrierIntel.unitType", carrierintel.UnitTypeValues())

var syncFieldValues = agenttoolschema.Source("carrierIntel.syncField", carrierintel.AllSyncFields())

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
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

// requireCarrierGrant is the second grant a resolver checks after the tool's own: a
// write that changes a carrier as well as its intelligence needs both.
func requireCarrierGrant(
	ctx context.Context,
	permissions permissionChecker,
	actor *permissionports.RequestActor,
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
	return agenttoolschema.RecordID(permission.ResourceCarrier, what,
		"carrier_list or carrier_get")
}

func targetCarrier(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamCarrierID, permission.ResourceCarrier)
}

func carrierIntelResult(action string, ids map[string]string) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{Action: action, Kind: kindCarrierIntel, IDs: ids}
	if id, err := pulid.Parse(ids[toolparams.ParamCarrierID]); err == nil {
		result.Record = changepreview.RecordOf(changepreview.CarrierRecordEntity, id)
	} else if id, err = pulid.Parse(ids[changepreview.FieldCustomerID]); err == nil {
		result.Record = changepreview.RecordOf(changepreview.CustomerRecordEntity, id)
	}

	return result
}

func vetPreview(
	subjectResource permission.Resource,
	subjectID pulid.ID,
	plan *carrierintelservice.VetPlan,
	force bool,
) *agentpreview.ToolPreview {
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
	preview := toolpreview.Build(summary, &agentpreview.RecordChange{
		Resource:  subjectResource,
		EntityID:  subjectID,
		Label:     name,
		Operation: agentpreview.PreviewOperationRun,
	})
	preview.Partial = true

	return preview
}

func NewVetCarrierTool(intel carrierIntelOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "carrier_vet",
		Artifact: changepreview.CarrierRecordEntity,
		Description: "Look a carrier up with the organization's carrier intelligence provider " +
			"now: authority, insurance, safety and the organization's rules, recorded on the " +
			"carrier. Use it before tendering to a carrier nobody has checked lately. A " +
			"result still current is reused unless force is true.",
		Resource:    permission.ResourceCarrierIntelligence,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Spends the organization's carrier intelligence budget on a lookup and " +
			"records the result; nothing is sent to the carrier.",
		Properties: map[string]any{
			toolparams.ParamCarrierID: carrierIDProperty("The carrier"),
			paramDepth: agenttoolschema.Enum("How deep a lookup: FMCSA is the public record, "+
				"Lite and Full add the provider's profile. Defaults to Full.", lookupDepths),
			toolparams.ParamForceResend: toolparams.BooleanProperty(
				"Look up again even when the last result is " +
					"still current. It is charged; only when the person asks.",
			),
		},
		Required: []string{toolparams.ParamCarrierID},
		Target:   targetCarrier,
	}, toolspec.ReceivablePlan[*carrierintelservice.VetCarrierRequest, *carrierintelservice.VetPlan]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.VetCarrierRequest, error) {
			carrierID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCarrierID)
			if err != nil {
				return nil, err
			}
			depth, _, err := toolparams.OptionalEnum(params.Params, paramDepth, lookupDepths.Values)
			if err != nil {
				return nil, err
			}
			force, err := toolparams.OptionalBoolPointer(params.Params, toolparams.ParamForceResend)
			if err != nil {
				return nil, err
			}

			return &carrierintelservice.VetCarrierRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				CarrierID:  carrierID,
				Depth:      depth,
				Force:      force != nil && *force,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *carrierintelservice.VetCarrierRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.VetPlan, error) {
			return intel.PlanVetCarrier(ctx, req)
		},
		Refused: func(*carrierintelservice.VetCarrierRequest) string {
			return "Would vet the carrier."
		},
		Render: func(
			req *carrierintelservice.VetCarrierRequest,
			plan *carrierintelservice.VetPlan,
		) (*agentpreview.ToolPreview, error) {
			return vetPreview(permission.ResourceCarrier, req.CarrierID, plan, req.Force), nil
		},
		Run: func(
			ctx context.Context,
			req *carrierintelservice.VetCarrierRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			result, err := intel.VetCarrier(ctx, req)
			if err != nil {
				return nil, err
			}

			return fetchResult(
				result,
				map[string]string{toolparams.ParamCarrierID: req.CarrierID.String()},
			), nil
		},
	})
}

func fetchResult(
	result *carrierintelservice.FetchResult,
	ids map[string]string,
) *agentrecord.ToolExecutionResult {
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

func NewVetCustomerBrokerTool(intel carrierIntelOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "customer_vet_broker",
		Artifact: changepreview.CustomerRecordEntity,
		Description: "Check a customer that is a freight broker against the public FMCSA " +
			"record by its DOT number: broker authority and bond. Use it before taking " +
			"freight from a broker nobody has checked lately.",
		Resource:    permission.ResourceCarrierIntelligence,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Spends the organization's carrier intelligence budget on a lookup and " +
			"records the result; nothing is sent to the customer.",
		Properties: map[string]any{
			changepreview.FieldCustomerID: agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"The broker customer, from customer_list. Never "+
					"guess one.",
			),
			toolparams.ParamForceResend: toolparams.BooleanProperty(
				"Look up again even when the last result is " +
					"still current. It is charged; only when the person asks.",
			),
		},
		Required: []string{changepreview.FieldCustomerID},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(
				params,
				changepreview.FieldCustomerID,
				permission.ResourceCustomer,
			)
		},
	}, toolspec.ReceivablePlan[*brokerVet, *carrierintelservice.VetPlan]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*brokerVet, error) {
			customerID, err := toolparams.RequirePulid(params.Params, changepreview.FieldCustomerID)
			if err != nil {
				return nil, err
			}
			force, err := toolparams.OptionalBoolPointer(params.Params, toolparams.ParamForceResend)
			if err != nil {
				return nil, err
			}

			return &brokerVet{
				tenant:     toolparams.TenantFrom(*params),
				customerID: customerID,
				force:      force != nil && *force,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *brokerVet,
			_ *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.VetPlan, error) {
			return intel.PlanVetCustomerBroker(ctx, req.tenant, req.customerID)
		},
		Refused: func(*brokerVet) string { return "Would vet the broker." },
		Render: func(req *brokerVet, plan *carrierintelservice.VetPlan) (*agentpreview.ToolPreview, error) {
			return vetPreview(permission.ResourceCustomer, req.customerID, plan, req.force), nil
		},
		Run: func(
			ctx context.Context,
			req *brokerVet,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			result, err := intel.VetCustomerBroker(ctx, req.tenant, req.customerID, req.force)
			if err != nil {
				return nil, err
			}

			return fetchResult(
				result,
				map[string]string{changepreview.FieldCustomerID: req.customerID.String()},
			), nil
		},
	})
}

type monitoringView struct {
	Monitoring carrierintel.DesiredState `json:"monitoring"`
}

func NewSetCarrierMonitoringTool(intel carrierIntelOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "carrier_set_monitoring",
		Artifact: changepreview.CarrierRecordEntity,
		Description: "Turn continuous monitoring on or off for carriers, so the provider " +
			"reports authority, insurance and safety changes as they happen. Carriers " +
			"without a DOT number are skipped.",
		Resource:    permission.ResourceCarrierIntelligence,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Monitoring is billed per carrier by the provider; turning it off again " +
			"undoes it, and nothing is sent to the carriers.",
		Properties: map[string]any{
			paramCarrierIDs: toolschema.RecordSubset(permission.ResourceCarrier.String(),
				agenttoolschema.RecordIDs(
					permission.ResourceCarrier,
					"The carriers, from carrier_list.",
					maxMonitoredCarriers,
				)),
			ParamEnabled: toolparams.BooleanProperty("True to monitor them, false to stop."),
		},
		Required: []string{paramCarrierIDs, ParamEnabled},
	}, toolspec.ReceivablePlan[*carrierintelservice.EnrollSubjectsRequest, []carrierintelservice.EnrollmentChange]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.EnrollSubjectsRequest, error) {
			ids, err := toolparams.RequirePulidSlice(
				params.Params,
				paramCarrierIDs,
				maxMonitoredCarriers,
			)
			if err != nil {
				return nil, err
			}
			enabled, err := toolparams.OptionalBoolPointer(params.Params, ParamEnabled)
			if err != nil {
				return nil, err
			}
			if enabled == nil {
				return nil, fmt.Errorf("missing required parameter %q", ParamEnabled)
			}

			return &carrierintelservice.EnrollSubjectsRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				CarrierIDs: ids,
				Enroll:     *enabled,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *carrierintelservice.EnrollSubjectsRequest,
			_ *agenttoolports.ToolExecuteParams,
		) ([]carrierintelservice.EnrollmentChange, error) {
			return intel.PlanManualEnrollment(ctx, req)
		},
		Refused: func(*carrierintelservice.EnrollSubjectsRequest) string {
			return "Would change carrier monitoring."
		},
		Render: renderMonitoring,
		Run: func(
			ctx context.Context,
			req *carrierintelservice.EnrollSubjectsRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			changed, err := intel.SetManualEnrollment(ctx, req)
			if err != nil {
				return nil, err
			}
			action := "monitoring stopped"
			if req.Enroll {
				action = "monitoring started"
			}
			out := carrierIntelResult(action, nil)
			out.Name = changepreview.CountOf(changed, "carrier")

			return out, nil
		},
	})
}

func renderMonitoring(
	req *carrierintelservice.EnrollSubjectsRequest,
	changes []carrierintelservice.EnrollmentChange,
) (*agentpreview.ToolPreview, error) {
	records := make([]*agentpreview.RecordChange, 0, len(changes))
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
	summary := fmt.Sprintf("Would %s %s.", verb, changepreview.CountOf(len(records), "carrier"))
	if skipped := len(req.CarrierIDs) - len(changes); skipped > 0 {
		summary += fmt.Sprintf(" %d have no DOT number and are skipped.", skipped)
	}
	if unchanged := len(changes) - len(records); unchanged > 0 {
		summary += fmt.Sprintf(" %d are already that way.", unchanged)
	}

	return toolpreview.Build(summary, records...), nil
}

func NewMarkCarrierIntelReviewedTool(intel carrierIntelOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "carrier_record_intelligence_review",
		Artifact: changepreview.CarrierRecordEntity,
		Description: "Record that a person reviewed a carrier's current intelligence " +
			"result, with what they checked. It is their sign-off, so it only runs once " +
			"they approve it.",
		Resource:    permission.ResourceCarrierIntelligence,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Puts a person's name on a carrier risk review; only that person can " +
			"say they reviewed it.",
		Properties: map[string]any{
			toolparams.ParamCarrierID: carrierIDProperty("The carrier"),
			toolparams.ParamNote: toolparams.StringProperty("What was reviewed and concluded.",
				maxReviewNoteChars),
		},
		Required: []string{toolparams.ParamCarrierID, toolparams.ParamNote},
		Target:   targetCarrier,
	}, toolspec.ReceivablePlan[*carrierintelservice.MarkReviewedRequest, *carrierintelservice.SnapshotChange]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.MarkReviewedRequest, error) {
			carrierID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCarrierID)
			if err != nil {
				return nil, err
			}
			note, err := toolparams.RequireBoundedText(
				params.Params,
				toolparams.ParamNote,
				maxReviewNoteChars,
			)
			if err != nil {
				return nil, err
			}

			return &carrierintelservice.MarkReviewedRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				CarrierID:  carrierID,
				Note:       note,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *carrierintelservice.MarkReviewedRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*carrierintelservice.SnapshotChange, error) {
			return intel.PlanMarkReviewed(ctx, req)
		},
		Refused: func(*carrierintelservice.MarkReviewedRequest) string {
			return "Would mark the carrier's intelligence reviewed."
		},
		Render: func(
			req *carrierintelservice.MarkReviewedRequest,
			plan *carrierintelservice.SnapshotChange,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{Resource: permission.ResourceCarrier, ID: req.CarrierID},
				plan.Before,
				plan.After,
				toolpreview.Only(
					changepreview.FieldReviewedByID,
					changepreview.FieldReviewedAt,
					"reviewNote",
				),
				toolpreview.Volatile(changepreview.FieldReviewedAt),
				toolpreview.WithRefs(map[string]permission.Resource{
					changepreview.FieldReviewedByID: permission.ResourceUser,
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
		Run: func(
			ctx context.Context,
			req *carrierintelservice.MarkReviewedRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := intel.MarkReviewed(ctx, req); err != nil {
				return nil, err
			}

			return carrierIntelResult("reviewed",
				map[string]string{toolparams.ParamCarrierID: req.CarrierID.String()}), nil
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

func NewApplyCarrierIntelSuggestionsTool(
	intel carrierIntelOperator,
	permissions permissionChecker,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "carrier_apply_intelligence_suggestions",
		SearchTerms: []string{"carrier intelligence fix", "intelligence corrections"},
		Artifact:    changepreview.CarrierRecordEntity,
		Description: "Correct a carrier's profile where its intelligence result disagrees: " +
			"name, DBA, MC number, safety rating, address, phone, email or insurance. Name the fields and policies to apply, or leave " +
			"both out to apply every suggestion there is now.",
		Resource:    permission.ResourceCarrierIntelligence,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Edits the carrier's own profile inside Trenova from what the provider " +
			"reports; nothing is sent, and the old values are set back by hand.",
		Properties: map[string]any{
			toolparams.ParamCarrierID: carrierIDProperty("The carrier"),
			paramSuggestionFields: toolschema.KeepEmpty(map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The profile fields to correct.",
				toolschema.KeyMaxItems:    len(syncFieldValues.Values),
				toolschema.KeyItems: agenttoolschema.Enum(
					"A profile field.",
					syncFieldValues,
				),
			}),
			paramPolicyIDs: toolschema.KeepEmpty(agenttoolschema.KindIDs(
				"The insurance policies to correct, by the policy "+
					"id on the page you are on.",
				maxSuggestionPolicies,
				permission.KindInsurancePolicy,
			)),
		},
		Required: []string{toolparams.ParamCarrierID},
		Target:   targetCarrier,
	}, toolspec.ReceivablePlan[*suggestionChoice, *carrierintelservice.SuggestionPlan]{
		Request: readSuggestionChoice,
		Plan: func(
			ctx context.Context,
			choice *suggestionChoice,
			params *agenttoolports.ToolExecuteParams,
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
		Refused: func(*suggestionChoice) string {
			return "Would correct the carrier's profile from its intelligence."
		},
		Render: renderSuggestions,
		Run: func(
			ctx context.Context,
			choice *suggestionChoice,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
				map[string]string{toolparams.ParamCarrierID: req.CarrierID.String()})
			out.Name = changepreview.CountOf(applied, "correction")

			return out, nil
		},
	})
}

func readSuggestionChoice(params *agenttoolports.ToolExecuteParams) (*suggestionChoice, error) {
	carrierID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCarrierID)
	if err != nil {
		return nil, err
	}
	req := &carrierintelservice.ApplySuggestionsRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		CarrierID:  carrierID,
	}
	if _, given := params.Params[paramSuggestionFields]; given {
		var names []string
		if err = toolparams.DecodeParam(params.Params, paramSuggestionFields, &names); err != nil {
			return nil, err
		}
		for _, name := range names {
			field, fieldErr := toolparams.RequireEnum(map[string]any{paramSuggestionFields: name},
				paramSuggestionFields, syncFieldValues.Values)
			if fieldErr != nil {
				return nil, fieldErr
			}
			req.Fields = append(req.Fields, field)
		}
	}
	if _, given := params.Params[paramPolicyIDs]; given {
		if req.PolicyIDs, err = toolparams.RequirePulidSlice(params.Params, paramPolicyIDs,
			maxSuggestionPolicies); err != nil {
			return nil, err
		}
	}

	return &suggestionChoice{req: req}, nil
}

func renderSuggestions(
	_ *suggestionChoice,
	plan *carrierintelservice.SuggestionPlan,
) (*agentpreview.ToolPreview, error) {
	fields := make([]string, 0, len(plan.Fields))
	for _, update := range plan.Fields {
		fields = append(fields, update.Field.String())
	}
	changes := make([]*agentpreview.RecordChange, 0, 1)
	if len(fields) > 0 {
		change, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceCarrier,
				ID:       plan.Before.ID,
				Label:    plan.Before.Name,
				Version:  changepreview.PinnedVersion(plan.Before.Version),
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
		changepreview.CountOf(len(plan.Fields), "profile field"), plan.Before.Name)
	if len(plan.Insurance) > 0 {
		summary += fmt.Sprintf(
			" and update %s",
			changepreview.CountOf(len(plan.Insurance), "insurance policy"),
		)
	}

	return toolpreview.Build(summary+".", changes...), nil
}

func NewImportSourcedCarrierTool(
	intel carrierIntelOperator,
	permissions permissionChecker,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "carrier_import",
		Artifact: changepreview.CarrierRecordEntity,
		Description: "Add a carrier found through carrier sourcing to the organization's " +
			"carriers by its DOT number, filled from the provider's profile. A DOT number " +
			"already on a carrier is refused.",
		Resource:    permission.ResourceCarrierSourcing,
		Operation:   permission.OpImport,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Creates a carrier inside Trenova from the provider's profile; nothing is " +
			"sent to it, and one added in error is made inactive.",
		Properties: map[string]any{
			paramDOTNumber: toolparams.StringProperty(
				"The carrier's USDOT number, from the sourcing "+
					"search.",
				maxDOTNumberChars,
			),
			paramCarrierCode: toolparams.StringProperty(
				"The short code to file it under. Left out, one "+
					"is made from its name.",
				maxCarrierCodeChars,
			),
			paramEnrollMonitoring: toolparams.BooleanProperty(
				"Start monitoring it at once. Monitoring " +
					"is billed per carrier.",
			),
		},
		Required: []string{paramDOTNumber},
	}, toolspec.ReceivablePlan[*carrierintelservice.ImportProspectRequest, *carrierintelservice.ImportPlan]{
		Request: importProspectRequest,
		Plan: func(
			ctx context.Context,
			req *carrierintelservice.ImportProspectRequest,
			params *agenttoolports.ToolExecuteParams,
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
		Refused: func(*carrierintelservice.ImportProspectRequest) string {
			return "Would add the sourced carrier."
		},
		Render: renderSourcedCarrierImport,
		Run: func(
			ctx context.Context,
			req *carrierintelservice.ImportProspectRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
				map[string]string{toolparams.ParamCarrierID: created.ID.String()})
			out.Name = created.Name

			return out, nil
		},
	})
}

func renderSourcedCarrierImport(
	_ *carrierintelservice.ImportProspectRequest,
	plan *carrierintelservice.ImportPlan,
) (*agentpreview.ToolPreview, error) {
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
}

type importView struct {
	DOTNumber        string `json:"dotNumber"`
	Code             string `json:"code"`
	Source           string `json:"source"`
	EnrollMonitoring bool   `json:"enrollMonitoring"`
}

func NewVerifyCarrierEquipmentTool(intel carrierIntelOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "carrier_verify_equipment",
		Artifact: changepreview.CarrierRecordEntity,
		Description: "Check that the truck or trailer a carrier sent for a load is registered " +
			"to that carrier, by VIN or plate, and record the answer on the assignment. Take " +
			"the numbers from the driver or the carrier's dispatch, never from a guess.",
		Resource:    permission.ResourceEquipmentVerification,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Records an equipment check inside Trenova after a provider lookup; " +
			"nothing is sent to the carrier.",
		Properties: map[string]any{
			paramCarrierAssignmentID: agenttoolschema.KindID("The carrier assignment, from "+
				"rate_list_confirmations or shipment_get. Never guess one.",
				permission.KindCarrierAssignment),
			paramUnitType: agenttoolschema.Enum("What was checked.", unitTypes),
			paramVIN: toolparams.StringProperty(
				"The vehicle identification number.",
				maxVINChars,
			),
			paramPlateNumber: toolparams.StringProperty("The license plate number.",
				maxPlateChars),
			paramPlateState: toolparams.StringProperty("The plate's two-letter state.",
				maxPlateStateChars),
			paramUnitNumber: toolparams.StringProperty("The carrier's own unit number.",
				maxUnitNumberChars),
		},
		Required: []string{paramCarrierAssignmentID, paramUnitType},
	}, toolspec.ReceivablePlan[*carrierintelservice.VerifyEquipmentRequest, *verificationPlan]{
		Request: verifyEquipmentRequest,
		Plan: func(
			ctx context.Context,
			req *carrierintelservice.VerifyEquipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*verificationPlan, error) {
			entity, carrierEntity, err := intel.PlanVerifyEquipment(ctx, req)
			if err != nil {
				return nil, err
			}

			return &verificationPlan{verification: entity, carrier: carrierEntity}, nil
		},
		Refused: func(*carrierintelservice.VerifyEquipmentRequest) string {
			return "Would verify the carrier's equipment."
		},
		Render: func(
			_ *carrierintelservice.VerifyEquipmentRequest,
			plan *verificationPlan,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(
				toolpreview.Record{
					Resource: permission.ResourceEquipmentVerification,
					Label:    plan.carrier.Name + " equipment check",
				},
				plan.verification,
				toolpreview.Only(
					toolparams.ParamCarrierID,
					"expectedDotNumber",
					paramUnitType,
					paramVIN,
					paramPlateNumber,
					paramPlateState,
					paramUnitNumber,
				),
				toolpreview.WithRefs(map[string]permission.Resource{
					toolparams.ParamCarrierID: permission.ResourceCarrier,
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
		Run: func(
			ctx context.Context,
			req *carrierintelservice.VerifyEquipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	params *agenttoolports.ToolExecuteParams,
) (*carrierintelservice.VerifyEquipmentRequest, error) {
	assignmentID, err := toolparams.RequirePulid(params.Params, paramCarrierAssignmentID)
	if err != nil {
		return nil, err
	}
	unitType, err := toolparams.RequireEnum(params.Params, paramUnitType, unitTypes.Values)
	if err != nil {
		return nil, err
	}
	req := &carrierintelservice.VerifyEquipmentRequest{
		TenantInfo:          toolparams.TenantFrom(*params),
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
		value, textErr := toolparams.BoundedText(params.Params, key, spec.limit)
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
	params *agenttoolports.ToolExecuteParams,
) (*carrierintelservice.ImportProspectRequest, error) {
	dot, err := toolparams.RequireBoundedText(params.Params, paramDOTNumber, maxDOTNumberChars)
	if err != nil {
		return nil, err
	}
	code, err := toolparams.BoundedText(params.Params, paramCarrierCode, maxCarrierCodeChars)
	if err != nil {
		return nil, err
	}
	enroll, err := toolparams.OptionalBoolPointer(params.Params, paramEnrollMonitoring)
	if err != nil {
		return nil, err
	}

	return &carrierintelservice.ImportProspectRequest{
		TenantInfo:       toolparams.TenantFrom(*params),
		DOTNumber:        dot,
		Code:             code,
		EnrollMonitoring: enroll != nil && *enroll,
	}, nil
}
