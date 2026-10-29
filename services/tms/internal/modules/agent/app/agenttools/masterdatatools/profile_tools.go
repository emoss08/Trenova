package masterdatatools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	fieldProfileAbout    = "about"
	fieldProfileGuidance = "guidance"
	profileAboutMe       = "me"
	profileAboutOrg      = "organization"
)

var profileSubjects = agenttoolschema.Source(
	"agent.profileSubject",
	[]string{profileAboutOrg, profileAboutMe},
)

var (
	errGuidanceNeedsPerson = errors.New(
		"guidance is given by a person; there is nobody in this run to give it",
	)
	errGuidanceAbout = errors.New(
		"about must be organization or me",
	)
	errGuidanceRequired = errors.New(
		"guidance is required: one plain sentence from what the person said",
	)
)

type profileGuide interface {
	AddGuidance(
		ctx context.Context,
		req *agentprofileports.AddGuidanceRequest,
	) (*agentprofile.Guidance, error)
	PlanGuidance(
		ctx context.Context,
		req *agentprofileports.AddGuidanceRequest,
	) (*agentprofile.Guidance, error)
}

// GuideProfileTool writes down, in a line, what a person said the agents'
// picture of their organization or of their own work is missing or gets
// wrong. The profile itself is never edited: the next write-up honours every
// line of guidance, so a person steers it without having to word it.
type GuideProfileTool struct {
	profiles profileGuide
}

func NewGuideProfileTool(profiles agentprofileports.Service) agenttoolports.AgentTool {
	return &GuideProfileTool{profiles: profiles}
}

func (t *GuideProfileTool) Name() string { return "agent_guide_profile" }

func (t *GuideProfileTool) SearchTerms() []string {
	return []string{"profile", "about our business", "about my work", "what we do", "my role"}
}

func (t *GuideProfileTool) Description() string {
	return "Correct or add to the profile agents are given about the organization's business " +
		"or the person's own work, when they say it is wrong or missing something. " +
		"about organization is the business as a whole: what it runs, for whom, on which " +
		"lanes, how it is set up (\"the Laredo lane is seasonal\", \"we stopped running " +
		"reefers\"). about me is the person's own work (\"I only handle the Midwest " +
		"accounts\", \"I'm in billing now\"). Write one plain sentence in your own words " +
		"from what they said; the profile is written again with it. Guidance about the " +
		"organization is kept at once only when the person may manage agent memory; " +
		"otherwise it is offered for an administrator to approve, and you should say so. " +
		"Not for a rule about one customer, record or task, or how to do something: use " +
		"agent_create_memory for those."
}

func (t *GuideProfileTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			fieldProfileAbout: agenttoolschema.Enum(
				"organization for the business as a whole; me for the person in the "+
					"conversation's own work.",
				profileSubjects,
			),
			fieldProfileGuidance: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: fmt.Sprintf("One plain sentence, at most %d "+
					"characters, saying what the profile should take into account, written so "+
					"a reader with no other context understands it.",
					agentprofile.MaxGuidanceChars),
			},
		},
		toolschema.KeyRequired:             []string{fieldProfileAbout, fieldProfileGuidance},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *GuideProfileTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:                t.Name(),
		Kind:                agenttooltrust.ToolKindAction,
		Resource:            permission.ResourceAssistant,
		Operation:           permission.OpCreate,
		Scope:               agenttooltrust.ToolScopeTenant,
		DefaultTier:         agent.TierActWithApproval,
		MaxTier:             agent.TierAutoExecute,
		Egress:              []agent.EgressClass{agent.EgressPersonal, agent.EgressInternal},
		Classify:            classifyGuideProfile,
		PersonalRunsUnasked: true,
		Effect:              agenttooltrust.ToolEffectChange,
		Reversible:          true,
		ReadsExternal:       agenttooltrust.ExternalReadNever,
		CarriesTaint:        true,
		TaintHold: &agenttoolports.TaintHold{
			Description: "Guidance recorded after the run read text from outside the " +
				"organization waits for a person's approval, because every later prompt is " +
				"written from it.",
			Applies: func(agenttoolports.ToolExecuteParams) bool { return true },
		},
		Rationale: "Adds a line every later write-up of a profile honours. Guidance about the " +
			"person's own work reaches only their conversations; guidance about the " +
			"organization reaches everyone's.",
	}
}

// classifyGuideProfile calls guidance about the person's own work personal: it
// shapes only the profile their own conversations read. Guidance about the
// organization shapes every conversation, so it is internal.
func classifyGuideProfile(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	actor := params.Actor
	if toolparams.OptionalString(params.Params, fieldProfileAbout) != profileAboutMe ||
		actor == nil || actor.PrincipalType != permissionports.PrincipalTypeUser ||
		actor.UserID.IsNil() {
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}

	return agenttoolports.CallPolicy{Egress: agent.EgressPersonal}
}

func (t *GuideProfileTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*agentprofileports.AddGuidanceRequest, error) {
	person := params.Actor.PersonUserID()
	if person.IsNil() {
		return nil, errGuidanceNeedsPerson
	}

	content := strings.TrimSpace(toolparams.OptionalString(params.Params, fieldProfileGuidance))
	switch {
	case content == "":
		return nil, errGuidanceRequired
	case utf8.RuneCountInString(content) > agentprofile.MaxGuidanceChars:
		return nil, fmt.Errorf("guidance is at most %d characters; say it in one sentence",
			agentprofile.MaxGuidanceChars)
	}

	tenant := toolparams.TenantFrom(*params)
	var ref agentprofileports.SubjectRef
	switch toolparams.OptionalString(params.Params, fieldProfileAbout) {
	case profileAboutOrg:
		ref = agentprofileports.OrganizationSubject(tenant)
	case profileAboutMe:
		ref = agentprofileports.UserSubject(tenant, person)
	default:
		return nil, errGuidanceAbout
	}

	return &agentprofileports.AddGuidanceRequest{
		Ref:      ref,
		Content:  content,
		Source:   agentprofile.GuidanceSourceAgent,
		ThreadID: params.ThreadID,
		Actor:    guidanceActor(params.Actor, person),
	}, nil
}

// guidanceActor is the person the guidance is from. An agent writes it down,
// but it is their words, and their name is on it.
func guidanceActor(
	actor *permissionports.RequestActor,
	person pulid.ID,
) *permissionports.RequestActor {
	if actor == nil {
		return nil
	}
	recorded := *actor
	recorded.UserID = person

	return &recorded
}

func (t *GuideProfileTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the Tool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.profiles.AddGuidance(ctx, req)

	return err
}

var guidanceFields = []string{"subject", changepreview.FieldContent}

var guidanceLabels = map[string]string{"subject": "About", changepreview.FieldContent: "Guidance"}

func (t *GuideProfileTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned, err := t.profiles.PlanGuidance(ctx, req)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build(guidanceSummary(req, false)),
				err,
			), nil
		}

		return nil, err
	}

	change, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceAgentMemory,
			Label:    stringutils.Ellipsize(req.Content, memoryLabelRunes),
		},
		planned,
		toolpreview.Only(guidanceFields...),
		toolpreview.Labels(guidanceLabels),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(guidanceSummary(req, planned.IsSuggested()), change), nil
}

// guidanceSummary says what the line would do: shape the organization's or
// the person's own profile, at once or once an administrator keeps it.
func guidanceSummary(req *agentprofileports.AddGuidanceRequest, offered bool) string {
	if req.Ref.Subject == agentprofile.SubjectUser {
		return "Would add this to what agents are told about your own work; your profile " +
			"would be written again with it."
	}
	if offered {
		return "Would offer this for an administrator to add to what agents are told about " +
			"the organization's business; its profile would be written again with it once " +
			"they do."
	}

	return "Would add this to what agents are told about the organization's business; " +
		"its profile would be written again with it."
}

func (t *GuideProfileTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}
