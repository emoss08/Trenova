// Package scheduletools holds the tool an agent proposes a scheduled request
// with: a request repeated in the conversation it was asked in, kept once the
// person approves it.
package scheduletools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramRequest    = "request"
	paramRepeat     = "repeat"
	paramTime       = "time"
	paramTimezone   = "timezone"
	paramScheduleID = "scheduleId"

	labelRunes = 80

	threadEntity = "assistant_thread"
)

var repeats = agenttoolschema.Source("conversation_schedule_repeats", conversationschedule.Repeats)

var (
	errNoSchedules = errors.New("scheduled requests are not available here")
	errNeedsPerson = errors.New(
		"a schedule is asked for by a person; there is nobody in this run to ask for one",
	)
	errNeedsConversation = errors.New(
		"a schedule posts into the conversation it was asked in, and this run has none",
	)
)

var scheduleFields = []string{"prompt", "cadence", "timezone"}

var scheduleLabels = map[string]string{
	"prompt":   "Request",
	"cadence":  "Runs",
	"timezone": "Clock",
}

// CreateScheduleTool proposes repeating a request in the conversation the
// agent is answering, the way a person schedules one by typing "every day at
// 9:30, …". It never keeps one on its own: a person approves every schedule,
// so an agent cannot set its own workload.
type CreateScheduleTool struct {
	planner conversationscheduleports.ConversationSchedulePlanner
}

func NewCreateScheduleTool(
	planner conversationscheduleports.ConversationSchedulePlanner,
) agenttoolports.AgentTool {
	return &CreateScheduleTool{planner: planner}
}

func (t *CreateScheduleTool) Name() string { return agentdefinition.CoreToolSchedule }

func (t *CreateScheduleTool) SearchTerms() []string {
	return []string{
		"schedule this", "every day", "everyday", "daily", "every morning", "every weekday",
		"run this again", "ask this again", "repeat this request",
	}
}

func (t *CreateScheduleTool) Description() string {
	return "Repeat a request in this conversation on a schedule, when the person asks for " +
		"something every day, every weekday or one day a week (\"schedule this to run " +
		"everyday at 9:30 AM EST\", \"give me this board every Monday at 8\"). Each run asks " +
		"request again here, as the person, and the answer posts into this conversation. " +
		"Write request in full so it stands alone: when they say \"this\", write out what " +
		"they asked for. time is 24-hour HH:MM. timezone is the clock they named " +
		"(America/New_York for EST or Eastern); leave it out for their own clock, which is " +
		"the organization's unless your context names theirs. The person " +
		"approves it before it is kept; then say in one line when it runs. It cannot run " +
		"hourly or monthly: say so. To email a saved report to people, use " +
		"report_create_schedule instead."
}

func (t *CreateScheduleTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramRequest: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: fmt.Sprintf("What each run asks, in the person's "+
					"words and in full, at most %d characters: \"Give me today's operations "+
					"board: active shipments, on-time %%, at-risk, unassigned, revenue today\".",
					conversationschedule.MaxPromptLength),
			},
			paramRepeat: agenttoolschema.Enum(
				"How often it runs: day for every day, weekday for Monday to Friday, or one "+
					"day of the week.",
				repeats,
			),
			paramTime: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "When it runs, 24-hour HH:MM: 09:30, 17:00.",
			},
			paramTimezone: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The clock the time is on, as an IANA name " +
					"(America/New_York, America/Chicago), when the person named one. Leave it " +
					"out for their own clock, the one your context names as theirs.",
			},
		},
		toolschema.KeyRequired:             []string{paramRequest, paramRepeat, paramTime},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *CreateScheduleTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAssistant,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		// The schedule is read beside the conversation its answers post into.
		Artifact: threadEntity,
		Rationale: "Repeats a request in the person's own conversation, asked as them with " +
			"the access they have when it runs. A person approves every schedule, so an " +
			"agent never sets its own workload.",
	}
}

func (t *CreateScheduleTool) request(
	params *agenttoolports.ToolExecuteParams,
) (conversationscheduleports.PlanConversationScheduleRequest, error) {
	person := params.Actor.PersonUserID()
	if person.IsNil() {
		return conversationscheduleports.PlanConversationScheduleRequest{}, errNeedsPerson
	}
	if params.ThreadID.IsNil() {
		return conversationscheduleports.PlanConversationScheduleRequest{}, errNeedsConversation
	}

	request, err := toolparams.RequireString(params.Params, paramRequest)
	if err != nil {
		return conversationscheduleports.PlanConversationScheduleRequest{}, err
	}
	repeat, err := toolparams.RequireString(params.Params, paramRepeat)
	if err != nil {
		return conversationscheduleports.PlanConversationScheduleRequest{}, err
	}
	at, err := toolparams.RequireString(params.Params, paramTime)
	if err != nil {
		return conversationscheduleports.PlanConversationScheduleRequest{}, err
	}

	actor := *params.Actor
	actor.UserID = person

	return conversationscheduleports.PlanConversationScheduleRequest{
		ThreadID: params.ThreadID,
		Edit: conversationschedule.Edit{
			Prompt: request,
			Repeat: repeat,
			Time:   at,
		},
		Timezone:   toolparams.OptionalString(params.Params, paramTimezone),
		ProposalID: params.ProposalID,
		Actor:      actor,
	}, nil
}

// plan is what the call would keep. A conversation the approver cannot read
// is not theirs, and only the person whose conversation it is can schedule a
// request in it.
func (t *CreateScheduleTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*conversationscheduleports.PlannedConversationSchedule, error) {
	if t.planner == nil {
		return nil, errNoSchedules
	}
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	planned, err := t.planner.Plan(ctx, req)
	if errortypes.IsNotFoundError(err) {
		return nil, notTheirs()
	}

	return planned, err
}

func (t *CreateScheduleTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	// Approving the proposal again finds the schedule it kept, which Execute
	// answers with; only a new proposal for it is refused.
	if planned.Existing != nil && !params.ApprovedFromProposal() {
		return alreadyKept(planned.Existing)
	}

	return nil
}

func (t *CreateScheduleTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}
	fallback := toolpreview.Build("Would repeat a request in this conversation on a schedule.")

	planned, err := t.plan(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) || errors.Is(err, errNeedsPerson) ||
			errors.Is(err, errNeedsConversation) {
			return changepreview.WarnWouldFail(fallback, err), nil
		}

		return nil, err
	}
	if planned.Existing != nil {
		return changepreview.WarnWouldFail(fallback, alreadyKept(planned.Existing)), nil
	}

	schedule := planned.Schedule
	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceAssistant,
			Label:    stringutils.Ellipsize(schedule.Prompt, labelRunes),
		},
		schedule,
		toolpreview.Only(scheduleFields...),
		toolpreview.Labels(scheduleLabels),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary(schedule), change), nil
}

func (t *CreateScheduleTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *CreateScheduleTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if t.planner == nil {
		return nil, errNoSchedules
	}
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	kept, err := t.planner.CreatePlanned(ctx, req)
	if errortypes.IsNotFoundError(err) {
		return nil, notTheirs()
	}
	if err != nil {
		return nil, err
	}

	action, schedule := "created", kept.Schedule
	if kept.Existing != nil {
		action, schedule = "found", kept.Existing
	}

	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "schedule",
		Name:   schedule.Cadence + " · " + stringutils.Ellipsize(schedule.Prompt, labelRunes),
		IDs:    map[string]string{paramScheduleID: schedule.ID.String()},
		Record: &agentrecord.RecordRef{EntityType: threadEntity, ID: schedule.ThreadID.String()},
	}, nil
}

// summary says what the schedule would do and when it would first run, on
// its own clock.
func summary(schedule *conversationschedule.Schedule) string {
	line := fmt.Sprintf(
		"Would ask %q here %s, on %s time, and post each answer into this conversation.",
		schedule.Prompt,
		when(schedule.Cadence),
		schedule.Timezone,
	)
	if schedule.NextRunAt == nil {
		return line
	}
	zone, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return line
	}

	return line + " First run: " +
		time.Unix(*schedule.NextRunAt, 0).In(zone).Format("Mon Jan 2, 3:04 PM MST") + "."
}

// when is a cadence label as words inside a sentence: "Every Monday · 8:00
// AM" is "every Monday at 8:00 AM".
func when(cadence string) string {
	cadence = strings.Replace(cadence, " · ", " at ", 1)
	if rest, ok := strings.CutPrefix(cadence, "Every "); ok {
		return "every " + rest
	}

	return cadence
}

func alreadyKept(existing *conversationschedule.Schedule) error {
	return errortypes.NewValidationError(
		paramRequest,
		errortypes.ErrDuplicate,
		fmt.Sprintf("This conversation already asks %q %s (%s); tell the person it is set "+
			"up rather than adding it again",
			existing.Prompt, when(existing.Cadence), existing.ID),
	)
}

func notTheirs() error {
	return errortypes.NewBusinessError(
		"Only the person whose conversation this is can schedule a request in it",
	)
}
