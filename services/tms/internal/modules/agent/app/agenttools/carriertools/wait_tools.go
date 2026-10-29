package carriertools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const paramWaitID = "waitId"

const waitUntilName = "agent_create_wait"

const cancelWaitName = "agent_cancel_wait"

const maxWaitHours = agentwait.MaxLifetimeSeconds / 3600

var waitKinds = agenttoolschema.Source("agent_wait_kinds", agentwait.AgentKinds())

var errNoWaitService = errors.New("waits are not available here")

// waitPolicy is the policy both wait tools share. A wait changes nothing in
// the world: it parks the agent's own work and hands it back later, so it
// runs on its own and leaves nothing for anyone outside the organization.
func waitPolicy(name, rationale string) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAgentRun,
		Operation:     permission.OpRead,
		Scope:         agenttooltrust.ToolScopeRun,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		CarriesTaint:  true,
		// A wait is part of the agent's own runs: what it reports is read
		// beside them in AI Control.
		Artifact:  "agent_run",
		Rationale: rationale,
	}
}

type waitUntilTool struct {
	waits agentwaitports.AgentWaitService
}

func NewWaitUntilTool(waits agentwaitports.AgentWaitService) agenttoolports.AgentTool {
	return &waitUntilTool{waits: waits}
}

func (t *waitUntilTool) Name() string { return waitUntilName }

func (t *waitUntilTool) Description() string {
	return "Park your work until something happens (\"let me know when it delivers\", \"tell " +
		"me when the driver gets there\") instead of checking back. It waits for a stop's " +
		"arrival or departure (StopArrival, StopDeparture: met when the truck is seen there " +
		"or the time is recorded), a reply about a shipment or from a carrier or customer " +
		"(Reply), an appointment coming round (AppointmentNear), detention free time running " +
		"out (FreeTimeEnding), a driver's drive time falling low (HOSDriveBelow), or a time " +
		"(Time). When it ends, or gives up after giveUpAfterHours, you are started again " +
		"here, told what happened and what you said you would do. A wait on something " +
		"already true is refused with what is true now. After setting one, end your turn: " +
		"say in one line what you wait for and what you will do then."
}

func (t *waitUntilTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"until": agenttoolschema.Enum("What to wait for.", waitKinds),
			changepreview.FieldDescription: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "What you are waiting for, in a few words a person reads " +
					"beside the conversation: \"Truck 2214 to reach Kroger DC\".",
			},
			"then": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "What you will do when it happens, so you pick up where you " +
					"meant to.",
			},
			"at": agenttoolschema.LocalDateTime("For Time: when to pick the work up."),
			toolparams.ParamShipmentMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"For StopArrival, StopDeparture and AppointmentNear: the move the stop is on, from "+
					"shipment_get.",
			),
			toolparams.ParamStopID: agenttoolschema.RecordIDText(permission.ResourceShipmentStop,
				"The stop, from shipment_get: required for AppointmentNear; for StopArrival "+
					"and StopDeparture, leave it out to wait for the move's next one."),
			changepreview.FieldShipmentID: agenttoolschema.RecordIDText(
				permission.ResourceShipment,
				"For Reply: a reply about this shipment, from shipment_list or the run subject.",
			),
			toolparams.ParamCarrierID: agenttoolschema.RecordIDText(permission.ResourceCarrier,
				"For Reply: a reply from this carrier, from carrier_list."),
			toolparams.ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"For Reply: a reply from this customer, from customer_list."),
			toolparams.ParamWorkerID: agenttoolschema.RecordIDText(permission.ResourceWorker,
				"For HOSDriveBelow: the driver, from worker_list."),
			"detentionOccurrenceId": agenttoolschema.KindID(
				"For FreeTimeEnding: the detention record, from detention_list_occurrences or the "+
					"run subject (detention.occurrence_opened names it).",
				recordkind.DetentionOccurrence,
			),
			"minutesBefore": map[string]any{
				toolschema.KeyType: toolschema.TypeInteger,
				toolschema.KeyDescription: "For AppointmentNear and FreeTimeEnding: how many minutes " +
					"before the appointment or the end of free time to pick the work up.",
			},
			"driveHoursBelow": map[string]any{
				toolschema.KeyType:        "number",
				toolschema.KeyDescription: "For HOSDriveBelow: pick the work up once the driver has less than this many hours of drive time left.",
			},
			"giveUpAfterHours": map[string]any{
				toolschema.KeyType: toolschema.TypeInteger,
				toolschema.KeyDescription: "How long to wait before giving up and picking the work up " +
					"anyway. Defaults to 24; at most 168.",
			},
		},
		toolschema.KeyRequired:             []string{"until", changepreview.FieldDescription},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *waitUntilTool) Policy() agenttoolports.ToolPolicy {
	return waitPolicy(t.Name(), "Parks the agent's own work until something happens and "+
		"starts it again then; it changes no record and reaches nobody.")
}

func (t *waitUntilTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	_, err := t.request(&params)

	return err
}

func (t *waitUntilTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}
	request, err := t.request(&params)
	if err != nil {
		return changepreview.WarnWouldFail(
			toolpreview.Build("Would park this work until something happens."),
			err,
		), nil
	}
	hours := request.LifetimeSeconds / 3600
	if hours == 0 {
		hours = agentwait.DefaultLifetimeSecs / 3600
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would park this work until: %s. Nothing changes meanwhile; the agent picks the "+
			"work up when it happens, or after %d hours if it does not.",
		request.Description, hours,
	)), nil
}

func (t *waitUntilTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *waitUntilTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if t.waits == nil {
		return nil, errNoWaitService
	}
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	wait, err := t.waits.Register(ctx, request)
	if err != nil {
		return nil, err
	}

	return &agentrecord.ToolExecutionResult{
		Action: "set",
		Kind:   "wait",
		Name:   wait.Description,
		IDs:    map[string]string{paramWaitID: wait.ID.String()},
	}, nil
}

func (t *waitUntilTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*agentwaitports.RegisterWaitRequest, error) {
	until, err := toolparams.RequireString(params.Params, "until")
	if err != nil {
		return nil, err
	}
	kind := agentwait.Kind(strings.TrimSpace(until))
	if !slices.Contains(agentwait.AgentKinds(), kind) {
		return nil, errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"until must be one of "+strings.Join(waitKinds.Names(), ", "))
	}
	description, err := toolparams.RequireString(params.Params, changepreview.FieldDescription)
	if err != nil {
		return nil, err
	}

	condition, err := waitCondition(kind, params)
	if err != nil {
		return nil, err
	}

	return &agentwaitports.RegisterWaitRequest{
		Actor:           params.Actor,
		DefinitionID:    params.DefinitionID,
		ThreadID:        params.ThreadID,
		RunID:           params.RunID,
		Delegated:       params.Delegated,
		Kind:            kind,
		Condition:       condition,
		Description:     description,
		Then:            toolparams.OptionalString(params.Params, "then"),
		LifetimeSeconds: giveUpAfter(params.Params),
	}, nil
}

func giveUpAfter(params map[string]any) int64 {
	hours := toolparams.OptionalInt64(params, "giveUpAfterHours")
	if hours <= 0 {
		return 0
	}

	return min(hours, maxWaitHours) * 3600
}

// waitCondition reads the arguments a kind of wait takes. The domain says
// which are required; this only reads what was sent.
func waitCondition(
	kind agentwait.Kind,
	params *agenttoolports.ToolExecuteParams,
) (agentwait.Condition, error) {
	var condition agentwait.Condition
	ids := []struct {
		key    string
		target *pulid.ID
	}{
		{toolparams.ParamShipmentMoveID, &condition.ShipmentMoveID},
		{toolparams.ParamStopID, &condition.StopID},
		{changepreview.FieldShipmentID, &condition.ShipmentID},
		{toolparams.ParamCarrierID, &condition.CarrierID},
		{toolparams.ParamCustomerID, &condition.CustomerID},
		{toolparams.ParamWorkerID, &condition.WorkerID},
		{"detentionOccurrenceId", &condition.DetentionOccurrenceID},
	}
	for _, field := range ids {
		id, given, err := toolparams.OptionalPulid(params.Params, field.key)
		if err != nil {
			return condition, errortypes.NewValidationError(field.key, errortypes.ErrInvalid,
				field.key+" is not a valid id")
		}
		if given {
			*field.target = id
		}
	}
	condition.MinutesBefore = int(toolparams.OptionalInt64(params.Params, "minutesBefore"))
	if hours, ok := params.Params["driveHoursBelow"].(float64); ok {
		condition.DriveMinutesBelow = int(math.Round(hours * 60))
	}

	if kind == agentwait.KindTime {
		at, err := toolparams.RequireLocalTime(
			params.Params,
			"at",
			timeutils.LoadLocation(params.Timezone),
		)
		if err != nil {
			return condition, err
		}
		condition.At = at
	}

	return condition, nil
}

type cancelWaitTool struct {
	waits agentwaitports.AgentWaitService
}

func NewCancelWaitTool(waits agentwaitports.AgentWaitService) agenttoolports.AgentTool {
	return &cancelWaitTool{waits: waits}
}

func (t *cancelWaitTool) Name() string { return cancelWaitName }

func (t *cancelWaitTool) Description() string {
	return "Cancel a wait you set with agent_create_wait, when what you were waiting for no longer " +
		"matters. The work is not picked up again. waitId is the id agent_create_wait returned."
}

func (t *cancelWaitTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramWaitID: agenttoolschema.KindID(
				"The wait to cancel, as agent_create_wait returned it (awt_…).",
				recordkind.Wait,
			),
		},
		toolschema.KeyRequired:             []string{paramWaitID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *cancelWaitTool) Policy() agenttoolports.ToolPolicy {
	return waitPolicy(t.Name(), "Cancels a wait the agent set on its own work; it changes "+
		"no record and reaches nobody.")
}

func (t *cancelWaitTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	_, err := toolparams.RequirePulid(params.Params, paramWaitID)

	return err
}

func (t *cancelWaitTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}
	preview := toolpreview.Build("Would cancel the wait; the parked work would not be picked up.")
	if _, err := toolparams.RequirePulid(params.Params, paramWaitID); err != nil {
		return changepreview.WarnWouldFail(preview, err), nil
	}

	return preview, nil
}

func (t *cancelWaitTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if t.waits == nil {
		return errNoWaitService
	}
	id, err := toolparams.RequirePulid(params.Params, paramWaitID)
	if err != nil {
		return err
	}

	_, err = t.waits.Cancel(ctx, &agentwaitports.CancelWaitRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		ThreadID:     params.ThreadID,
		DefinitionID: params.DefinitionID,
		By:           "the agent",
	})

	return err
}
