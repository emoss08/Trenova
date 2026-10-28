package portaltools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"go.uber.org/zap"
)

const (
	paramSiteID      = "siteId"
	paramInstruction = "instruction"
	paramDetails     = "details"
	resultKind       = "portal task"
	resultRunID      = "portalRunId"
)

var ErrPortalActionNeedsAPerson = errors.New(
	"acting on a portal runs only from a person's approval of the proposal",
)

type ActOnPortalTool struct {
	portals agentextensionports.PortalAutomator
	waits   agentextensionports.PortalRunWaits
	l       *zap.Logger
}

func NewActOnPortalTool(
	portals agentextensionports.PortalAutomator,
	waits agentextensionports.PortalRunWaits,
	logger *zap.Logger,
) agenttoolports.AgentTool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ActOnPortalTool{portals: portals, waits: waits, l: logger.Named("tool.act-on-portal")}
}

func (t *ActOnPortalTool) Name() string { return agentextension.ToolActOnPortal }

func (t *ActOnPortalTool) Prerequisites() []string {
	return []string{agentextension.ToolListPortalSites, agentextension.ToolReadPortal}
}

func (t *ActOnPortalTool) Description() string {
	return "Propose doing one thing on an organization portal site through a cloud browser, " +
		"such as booking a dock appointment or accepting a load. It can also move an " +
		"appointment, upload a proof of delivery or update a status. A person sees the exact " +
		"instruction and details the browser will get and decides; once approved the portal " +
		"does it in the background and you pick the work up with what it said. Only sites " +
		"that allow actions can take one. Give one action per proposal, and the references " +
		"the portal knows in details, never Trenova record ids."
}

func (t *ActOnPortalTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSiteID: agenttoolschema.KindID(
			"The portal site to act on, from web_list_portal_sites. It must allow actions.",
			permission.KindPortalSite,
		),
		paramInstruction: map[string]any{
			toolschema.KeyType: "string",
			toolschema.KeyDescription: "Exactly what to do, in one plain sentence, such as " +
				"\"Book the 8:00 AM dock appointment on October 20 for this PO\".",
			toolschema.KeyMaxLength: agentextension.MaxPortalInstructionRunes,
		},
		paramDetails: agenttoolschema.StringMap(
			"The values the portal needs, by what the portal calls them, such as "+
				"{\"PO number\": \"4512\", \"Carrier\": \"Ridgeline Freight\"}.",
			agentextension.MaxPortalDetailValueRunes,
		),
	}, paramSiteID, paramInstruction)
}

func (t *ActOnPortalTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourcePortalAutomation,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Artifact:      string(permission.KindPortalRun),
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Acts on a third-party portal in the organization's name, which another " +
			"company then relies on; what it books or submits is undone, if at all, on that " +
			"portal. A person always decides.",
	}
}

func (t *ActOnPortalTool) SearchTerms() []string {
	return []string{
		"book appointment", "dock appointment", "schedule appointment", "upload pod",
		"accept load", "shipper portal", "portal", "submit on portal", "reschedule",
	}
}

func (t *ActOnPortalTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*agentextensionports.PortalActionRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	siteID, err := common.RequirePulid(params.Params, paramSiteID)
	if err != nil {
		return nil, err
	}
	instruction, err := common.RequireString(params.Params, paramInstruction)
	if err != nil {
		return nil, err
	}
	details, err := agentextension.PortalDetailsFrom(params.Params[paramDetails])
	if err != nil {
		return nil, errortypes.NewValidationError(paramDetails, errortypes.ErrInvalid, err.Error())
	}

	return &agentextensionports.PortalActionRequest{
		TenantInfo:     common.TenantFrom(*params),
		SiteID:         siteID,
		Instruction:    instruction,
		Details:        details,
		IdempotencyKey: params.IdempotencyKey,
		Owner: agentextensionports.PortalTaskOwner{
			Actor:        params.Actor,
			DefinitionID: params.DefinitionID,
			RunID:        params.RunID,
			ThreadID:     params.ThreadID,
			ProposalID:   params.ProposalID,
		},
	}, nil
}

func (t *ActOnPortalTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.portals.PlanPortalAction(ctx, req)

	return err
}

// Preview shows the instruction exactly as the browser will receive it, and
// the details it will type. It never reaches the vendor.
func (t *ActOnPortalTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	summary := "Would ask a browser to act on a portal."
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.portals.PlanPortalAction(ctx, req)
	if err != nil {
		if agentextensionservice.IsToolError(err) || common.IsRefusal(err) ||
			errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	sent := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourcePortalAutomation,
		ID:       plan.Site.ID,
		Label:    plan.Site.Name,
	}, &agent.MessagePreview{
		Channel: agent.MessageChannelPortal,
		To:      plan.Site.Domains,
		Subject: plan.Instruction,
		Body:    plan.SentTask,
	})

	summary = fmt.Sprintf("Would ask %s's browser to do this on %s: %s.",
		plan.Vendor, plan.Site.Name, strings.TrimRight(plan.Instruction, ". "))
	switch {
	case plan.UsesSignIn && plan.UsesAuthenticator:
		summary += " It signs in with the site's stored sign-in and answers its authenticator code."
	case plan.UsesSignIn:
		summary += " It signs in with the site's stored sign-in."
	}

	return toolpreview.Build(summary, sent), nil
}

func (t *ActOnPortalTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult hands the approved action to the vendor, parks the work
// that asked until the portal answers, and starts following the task. The
// approval is the idempotency key, so a retry finds the task it started.
func (t *ActOnPortalTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrPortalActionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	started, err := t.portals.StartPortalAction(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &agent.ToolExecutionResult{
		Action: "started",
		Kind:   resultKind,
		Name:   started.Site.Name,
		IDs:    map[string]string{resultRunID: started.RunID.String()},
		Record: &agent.RecordRef{
			EntityType: string(permission.KindPortalRun),
			ID:         started.RunID.String(),
		},
	}
	if started.Status.Finished() {
		result.Action = "ran"
		result.State = finishedState(started.Status)
		return result, nil
	}

	result.State = "Running on " + started.Site.Name + "."
	if t.registerWait(ctx, req, started) {
		result.State += " The work picks up here with what the portal said when it finishes."
	} else {
		result.State += " Its outcome appears under AI Control, Extensions when it finishes."
	}
	if err = t.portals.TrackPortalRun(
		ctx,
		req.TenantInfo,
		started.RunID,
		started.Site.Name,
	); err != nil {
		return nil, fmt.Errorf("follow the portal task: %w", err)
	}

	return result, nil
}

// registerWait parks the work until the portal answers. A wait that cannot be
// set does not undo an action the portal already has: the task is followed
// to its end either way, and its outcome is recorded.
func (t *ActOnPortalTool) registerWait(
	ctx context.Context,
	req *agentextensionports.PortalActionRequest,
	started *agentextensionports.PortalActionStarted,
) bool {
	if t.waits == nil {
		return false
	}
	if err := t.waits.RegisterPortalRunWait(ctx, &agentextensionports.RegisterPortalRunWaitRequest{
		TenantInfo: req.TenantInfo,
		RunID:      started.RunID,
		SiteName:   started.Site.Name,
		Owner:      req.Owner,
	}); err != nil {
		t.l.Warn("could not park the work on a portal task",
			zap.String("portalRun", started.RunID.String()), zap.Error(err))
		return false
	}

	return true
}

func finishedState(status agentextension.PortalRunStatus) string {
	switch status {
	case agentextension.PortalRunStatusSucceeded:
		return "Finished. Read what the portal said under AI Control, Extensions."
	case agentextension.PortalRunStatusStrayed:
		return "Stopped: the browser left the site's addresses, so what it found was withheld."
	default:
		return "Ended without succeeding (" + strings.ToLower(status.String()) + ")."
	}
}
