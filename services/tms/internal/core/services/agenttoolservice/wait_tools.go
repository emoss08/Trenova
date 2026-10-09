package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramWaitID    = "waitId"
	waitUntilName  = "wait_until"
	cancelWaitName = "cancel_wait"
	maxWaitHours   = agentwait.MaxLifetimeSeconds / 3600
)

var waitKinds = agenttoolschema.Source("agent_wait_kinds", agentwait.AllKinds())

var errNoWaitService = errors.New("waits are not available here")

// waitPolicy is the policy both wait tools share. A wait changes nothing in
// the world: it parks the agent's own work and hands it back later, so it
// runs on its own and leaves nothing for anyone outside the organization.
func waitPolicy(name, rationale string) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          name,
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentRun,
		Operation:     permission.OpRead,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		CarriesTaint:  true,
		// A wait is part of the agent's own runs: what it reports is read
		// beside them in AI Control.
		Artifact:  "agent_run",
		Rationale: rationale,
	}
}

type waitUntilTool struct {
	waits serviceports.AgentWaitService
}

func newWaitUntilTool(waits serviceports.AgentWaitService) serviceports.AgentTool {
	return &waitUntilTool{waits: waits}
}

func (t *waitUntilTool) Name() string { return waitUntilName }

func (t *waitUntilTool) Description() string {
	return "Park this work until something happens, instead of checking back. Use it when the " +
		"next step depends on the world: a truck reaching or leaving a stop (StopArrival, " +
		"StopDeparture), a reply about a shipment or from a carrier or customer (Reply), an " +
		"appointment coming round (AppointmentNear), detention free time running out " +
		"(FreeTimeEnding), a driver's drive time falling low (HOSDriveBelow), or a time (Time). " +
		"Nothing runs while it waits. When it ends, or gives up after giveUpAfterHours, you are " +
		"started again in this same conversation or on this same record, told what happened and " +
		"what you said you would do. A wait on something already true is refused with what is " +
		"true now. After setting one, end your turn: tell the person in one line what you are " +
		"waiting for and what you will do then."
}

func (t *waitUntilTool) ParamSchema() map[string]any {
	id := func(description string) map[string]any {
		return map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: description,
		}
	}

	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"until": agenttoolschema.Enum("What to wait for.", waitKinds),
			fieldDescription: map[string]any{
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
			paramShipmentMoveID: id("For StopArrival, StopDeparture and AppointmentNear: the " +
				"move the stop is on."),
			paramStopID: id("The stop: required for AppointmentNear; for StopArrival and " +
				"StopDeparture, leave it out to wait for the move's next one."),
			fieldShipmentID: id("For Reply: a reply about this shipment."),
			paramCarrierID:  id("For Reply: a reply from this carrier."),
			paramCustomerID: id("For Reply: a reply from this customer."),
			paramWorkerID:   id("For HOSDriveBelow: the driver."),
			"detentionOccurrenceId": id("For FreeTimeEnding: the detention record " +
				"(detention.occurrence_opened names it)."),
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
		toolschema.KeyRequired:             []string{"until", fieldDescription},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *waitUntilTool) Policy() serviceports.ToolPolicy {
	return waitPolicy(t.Name(), "Parks the agent's own work until something happens and "+
		"starts it again then; it changes no record and reaches nobody.")
}

func (t *waitUntilTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	_, err := t.request(&params)

	return err
}

func (t *waitUntilTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}
	request, err := t.request(&params)
	if err != nil {
		return warnWouldFail(
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
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *waitUntilTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
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

	return &agent.ToolExecutionResult{
		Action: "set",
		Kind:   "wait",
		Name:   wait.Description,
		IDs:    map[string]string{paramWaitID: wait.ID.String()},
	}, nil
}

func (t *waitUntilTool) request(
	params *serviceports.ToolExecuteParams,
) (*serviceports.RegisterWaitRequest, error) {
	until, err := requireString(params.Params, "until")
	if err != nil {
		return nil, err
	}
	kind := agentwait.Kind(strings.TrimSpace(until))
	if !kind.IsValid() {
		return nil, errortypes.NewValidationError("until", errortypes.ErrInvalid,
			"until must be one of "+strings.Join(waitKinds.Names(), ", "))
	}
	description, err := requireString(params.Params, fieldDescription)
	if err != nil {
		return nil, err
	}

	condition, err := waitCondition(kind, params)
	if err != nil {
		return nil, err
	}

	return &serviceports.RegisterWaitRequest{
		Actor:           params.Actor,
		DefinitionID:    params.DefinitionID,
		ThreadID:        params.ThreadID,
		RunID:           params.RunID,
		Delegated:       params.Delegated,
		Kind:            kind,
		Condition:       condition,
		Description:     description,
		Then:            optionalString(params.Params, "then"),
		LifetimeSeconds: giveUpAfter(params.Params),
	}, nil
}

func giveUpAfter(params map[string]any) int64 {
	hours := optionalInt64(params, "giveUpAfterHours")
	if hours <= 0 {
		return 0
	}

	return min(hours, maxWaitHours) * 3600
}

// waitCondition reads the arguments a kind of wait takes. The domain says
// which are required; this only reads what was sent.
func waitCondition(
	kind agentwait.Kind,
	params *serviceports.ToolExecuteParams,
) (agentwait.Condition, error) {
	var condition agentwait.Condition
	ids := []struct {
		key    string
		target *pulid.ID
	}{
		{paramShipmentMoveID, &condition.ShipmentMoveID},
		{paramStopID, &condition.StopID},
		{fieldShipmentID, &condition.ShipmentID},
		{paramCarrierID, &condition.CarrierID},
		{paramCustomerID, &condition.CustomerID},
		{paramWorkerID, &condition.WorkerID},
		{"detentionOccurrenceId", &condition.DetentionOccurrenceID},
	}
	for _, field := range ids {
		id, given, err := optionalPulid(params.Params, field.key)
		if err != nil {
			return condition, errortypes.NewValidationError(field.key, errortypes.ErrInvalid,
				field.key+" is not a valid id")
		}
		if given {
			*field.target = id
		}
	}
	condition.MinutesBefore = int(optionalInt64(params.Params, "minutesBefore"))
	if hours, ok := params.Params["driveHoursBelow"].(float64); ok {
		condition.DriveMinutesBelow = int(math.Round(hours * 60))
	}

	if kind == agentwait.KindTime {
		at, err := requireLocalTime(params.Params, "at", timeutils.LoadLocation(params.Timezone))
		if err != nil {
			return condition, err
		}
		condition.At = at
	}

	return condition, nil
}

type cancelWaitTool struct {
	waits serviceports.AgentWaitService
}

func newCancelWaitTool(waits serviceports.AgentWaitService) serviceports.AgentTool {
	return &cancelWaitTool{waits: waits}
}

func (t *cancelWaitTool) Name() string { return cancelWaitName }

func (t *cancelWaitTool) Description() string {
	return "Cancel a wait you set with wait_until, when what you were waiting for no longer " +
		"matters. The work is not picked up again. waitId is the id wait_until returned."
}

func (t *cancelWaitTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramWaitID: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The wait to cancel, as wait_until returned it (awt_…).",
			},
		},
		toolschema.KeyRequired:             []string{paramWaitID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *cancelWaitTool) Policy() serviceports.ToolPolicy {
	return waitPolicy(t.Name(), "Cancels a wait the agent set on its own work; it changes "+
		"no record and reaches nobody.")
}

func (t *cancelWaitTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	_, err := requirePulid(params.Params, paramWaitID)

	return err
}

func (t *cancelWaitTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}
	preview := toolpreview.Build("Would cancel the wait; the parked work would not be picked up.")
	if _, err := requirePulid(params.Params, paramWaitID); err != nil {
		return warnWouldFail(preview, err), nil
	}

	return preview, nil
}

func (t *cancelWaitTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if t.waits == nil {
		return errNoWaitService
	}
	id, err := requirePulid(params.Params, paramWaitID)
	if err != nil {
		return err
	}

	_, err = t.waits.Cancel(ctx, &serviceports.CancelWaitRequest{
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
