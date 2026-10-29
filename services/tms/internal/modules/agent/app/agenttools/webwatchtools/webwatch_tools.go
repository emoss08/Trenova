// Package webwatchtools lets an agent keep watching the web for a person:
// set a scheduled search that hands back only what is new, and stop one.
package webwatchtools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"go.uber.org/zap"
)

const (
	paramPurpose = "purpose"
	paramQuery   = "query"
	paramSites   = "sites"
	paramCadence = "cadence"
	paramDays    = "days"
	paramWatchID = "watchId"
	resultKind   = "web watch"
	resultID     = "webWatchId"
	dayLayout    = "Jan 2, 2006"
)

var (
	cadences = agenttoolschema.Source("web_watch_cadences", extensionwebwatch.AllWebWatchCadences())

	ErrDelegatedWatch = errors.New(
		"a task another agent handed you cannot set a web watch; tell that agent what to watch",
	)
	errNoWatcher = errors.New("web watches are not available here")
)

// watchPolicy is the policy both watch tools share. A watch is web research
// that goes on: it sends the search to the organization's vendor, spends on
// its account and reaches nobody, so it runs on its own; a turn that read
// outside content still proposes it, as every later write.
func watchPolicy(name, rationale string) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceWebResearch,
		Operation:     permission.OpRead,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		Artifact:      string(recordkind.WebWatch),
		ReadsExternal: agenttooltrust.ExternalReadNever,
		CarriesTaint:  true,
		Rationale:     rationale,
	}
}

type WatchWebTool struct {
	watches agentextensionports.WebWatcher
	waits   agentextensionports.WebWatchWaits
	l       *zap.Logger
}

func NewWatchWebTool(
	watches agentextensionports.WebWatcher,
	waits agentextensionports.WebWatchWaits,
	logger *zap.Logger,
) agenttoolports.AgentTool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &WatchWebTool{watches: watches, waits: waits, l: logger.Named("tool.watch-web")}
}

func (t *WatchWebTool) Name() string { return agentextension.ToolWatchWeb }

func (t *WatchWebTool) Description() string {
	return "Keep watching the web for something new for the person, such as a rule being " +
		"finalized, a carrier or customer in the news, or a rate change announced. The " +
		"organization's research vendor runs the search on a schedule and " +
		"hands back only pages it has not found before, and this conversation picks the work " +
		"up with them. Use it when the person asks to be told when something happens; for a " +
		"question about now, use web_search. A watch lasts 30 days unless asked otherwise."
}

func (t *WatchWebTool) Prerequisites() []string {
	return []string{agentextension.ToolWebSearch}
}

func (t *WatchWebTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramPurpose: map[string]any{
			toolschema.KeyType: "string",
			toolschema.KeyDescription: "What the person wants to hear about, in their terms, " +
				"such as \"FMCSA finalizing the broker transparency rule\".",
			toolschema.KeyMaxLength: extensionwebwatch.MaxWebWatchPurposeRunes,
		},
		paramQuery: map[string]any{
			toolschema.KeyType: "string",
			toolschema.KeyDescription: "The search to run each time, written as a description " +
				"of the pages wanted, such as \"FMCSA final rule broker transparency 49 CFR " +
				"371.3\". Never a Trenova record id.",
			toolschema.KeyMaxLength: extensionwebwatch.MaxWebWatchQueryRunes,
		},
		paramSites: map[string]any{
			toolschema.KeyType: "array",
			toolschema.KeyDescription: "Optional: search only these sites, such as " +
				"[\"fmcsa.dot.gov\", \"federalregister.gov\"].",
			toolschema.KeyItems:    map[string]any{toolschema.KeyType: "string"},
			toolschema.KeyMaxItems: extensionwebwatch.MaxWebWatchDomains,
		},
		paramCadence: agenttoolschema.Enum(
			"How often to look. Daily unless the person needs to hear sooner or less often.",
			cadences,
		),
		paramDays: toolparams.IntegerProperty(
			"How many days to keep watching, from 1 to 90. Defaults to 30.",
			extensionwebwatch.MinWebWatchDays, extensionwebwatch.MaxWebWatchDays,
		),
	}, paramPurpose, paramQuery)
}

func (t *WatchWebTool) Policy() agenttoolports.ToolPolicy {
	return watchPolicy(t.Name(), "Sets a scheduled search on the organization's web research "+
		"vendor for the person; it changes no record, reaches nobody, and ends on its own or "+
		"when stopped.")
}

func (t *WatchWebTool) SearchTerms() []string {
	return []string{
		"watch", "monitor", "keep an eye on", "let me know when", "tell me when", "alert me",
		"track", "follow", "news", "regulation", "rule change", "announcement",
	}
}

func (t *WatchWebTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*agentextensionports.WebWatchRequest, error) {
	purpose, err := toolparams.RequireString(params.Params, paramPurpose)
	if err != nil {
		return nil, err
	}
	query, err := toolparams.RequireString(params.Params, paramQuery)
	if err != nil {
		return nil, err
	}
	cadence := extensionwebwatch.WebWatchCadence(
		toolparams.OptionalString(params.Params, paramCadence),
	)
	if cadence != "" && !cadence.IsValid() {
		return nil, toolparams.ErrUnknownValue(paramCadence, string(cadence), cadences.Names())
	}

	return &agentextensionports.WebWatchRequest{
		TenantInfo: toolparams.TenantFrom(params),
		Query:      query,
		Purpose:    purpose,
		Sites:      toolparams.StringSliceParam(params.Params, paramSites),
		Cadence:    cadence,
		Days:       int(toolparams.OptionalInt64(params.Params, paramDays)),
		Owner: agentextensionports.PortalTaskOwner{
			Actor:        params.Actor,
			DefinitionID: params.DefinitionID,
			RunID:        params.RunID,
			ThreadID:     params.ThreadID,
			ProposalID:   params.ProposalID,
		},
	}, nil
}

func (t *WatchWebTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return err
	}
	if params.Delegated {
		return ErrDelegatedWatch
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.watches.PlanWebWatch(ctx, req)

	return err
}

// Preview says what the vendor will search, how often and until when, and
// what that costs at the vendor's published price. It never reaches the
// vendor.
func (t *WatchWebTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}
	summary := "Would keep watching the web."
	req, err := t.request(&params)
	if err != nil {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
	}

	plan, err := t.watches.PlanWebWatch(ctx, req)
	if err != nil {
		if agentextensionservice.IsToolError(err) || changepreview.IsRefusal(err) ||
			errortypes.IsMultiError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}
		return nil, err
	}

	return toolpreview.Build(planSummary(plan)), nil
}

func planSummary(plan *agentextensionports.WebWatchPlan) string {
	where := "the web"
	if len(plan.Sites) > 0 {
		where = fmt.Sprintf("%d site(s)", len(plan.Sites))
	}

	return fmt.Sprintf("Would ask %s to search %s %s until %s for %s, with the search %q, and "+
		"bring back only what it has not found before. About $%s in all at %s's published "+
		"price ($%s a check).",
		plan.Vendor, where, plan.Cadence.Words(), endDay(plan.EndsAt), plan.Purpose,
		plan.Query, plan.EstimatedCost.StringFixed(2), plan.Vendor,
		plan.CostPerCheck.StringFixed(3))
}

func endDay(at int64) string {
	return time.Unix(at, 0).UTC().Format(dayLayout)
}

func (t *WatchWebTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult sets the watch and parks the work on it. A watch whose
// work could not be parked is stopped at once: what it found would reach no
// one, and it would bill for nothing.
func (t *WatchWebTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return nil, err
	}
	if params.Delegated {
		return nil, ErrDelegatedWatch
	}
	if t.waits == nil {
		return nil, errNoWatcher
	}

	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	view, err := t.watches.WatchWeb(ctx, req)
	if err != nil {
		return nil, err
	}

	watch := &extensionwebwatch.WebWatch{
		ID:             view.ID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Purpose:        view.Purpose,
		Status:         view.Status,
		EndsAt:         view.EndsAt,
	}
	if err = t.waits.RegisterWebWatchWait(ctx, &agentextensionports.RegisterWebWatchWaitRequest{
		TenantInfo: req.TenantInfo,
		Watch:      watch,
		Owner:      req.Owner,
	}); err != nil {
		t.l.Warn("could not park the work on a web watch; stopping the watch",
			zap.String("webWatch", view.ID.String()), zap.Error(err))
		if _, stopErr := t.watches.StopWebWatch(ctx, &agentextensionports.StopWebWatchRequest{
			TenantInfo:  req.TenantInfo,
			WatchID:     view.ID,
			Administers: true,
		}); stopErr != nil {
			t.l.Error("a web watch whose work could not be parked could not be stopped",
				zap.String("webWatch", view.ID.String()), zap.Error(stopErr))
		}
		return nil, fmt.Errorf("park the work on the web watch: %w", err)
	}

	return &agentrecord.ToolExecutionResult{
		Action: "set",
		Kind:   resultKind,
		Name:   view.Purpose,
		IDs:    map[string]string{resultID: view.ID.String()},
		Record: &agentrecord.RecordRef{
			EntityType: string(recordkind.WebWatch),
			ID:         view.ID.String(),
		},
		State: fmt.Sprintf("Watching %s until %s. What it finds comes back to this work as it "+
			"is found.", view.Cadence.Words(), endDay(view.EndsAt)),
	}, nil
}

type StopWebWatchTool struct {
	watches agentextensionports.WebWatcher
	waits   agentextensionports.WebWatchWaits
}

func NewStopWebWatchTool(
	watches agentextensionports.WebWatcher,
	waits agentextensionports.WebWatchWaits,
) agenttoolports.AgentTool {
	return &StopWebWatchTool{watches: watches, waits: waits}
}

func (t *StopWebWatchTool) Name() string { return agentextension.ToolStopWebWatch }

func (t *StopWebWatchTool) Description() string {
	return "Stop a web watch the person no longer needs, so the vendor stops searching and " +
		"billing for it. Find its id with web_list_watches."
}

func (t *StopWebWatchTool) Prerequisites() []string {
	return []string{agentextension.ToolListWebWatches}
}

func (t *StopWebWatchTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramWatchID: agenttoolschema.KindID(
			"The web watch to stop, from web_list_watches.",
			recordkind.WebWatch,
		),
	}, paramWatchID)
}

func (t *StopWebWatchTool) Policy() agenttoolports.ToolPolicy {
	return watchPolicy(t.Name(), "Stops a scheduled search the person set; it changes no "+
		"record and reaches nobody.")
}

func (t *StopWebWatchTool) SearchTerms() []string {
	return []string{"stop watching", "stop monitoring", "cancel watch", "no longer need"}
}

func (t *StopWebWatchTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*agentextensionports.StopWebWatchRequest, error) {
	watchID, err := toolparams.RequirePulid(params.Params, paramWatchID)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.StopWebWatchRequest{
		TenantInfo:   toolparams.TenantFrom(params),
		WatchID:      watchID,
		StoppedBy:    params.Actor.PersonUserID(),
		DefinitionID: params.DefinitionID,
	}, nil
}

func (t *StopWebWatchTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return err
	}
	_, err := t.request(&params)

	return err
}

func (t *StopWebWatchTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}
	summary := "Would stop a web watch, so the vendor searches and bills for it no more."
	if _, err := t.request(&params); err != nil {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
	}

	return toolpreview.Build(summary), nil
}

func (t *StopWebWatchTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *StopWebWatchTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, &params); err != nil {
		return nil, err
	}
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	view, err := t.watches.StopWebWatch(ctx, req)
	if err != nil {
		return nil, err
	}
	if t.waits != nil {
		t.waits.EndWebWatchWaits(ctx, req.TenantInfo, &extensionwebwatch.WebWatch{
			ID:     view.ID,
			Status: view.Status,
		})
	}

	return &agentrecord.ToolExecutionResult{
		Action: "stopped",
		Kind:   resultKind,
		Name:   view.Purpose,
		IDs:    map[string]string{resultID: view.ID.String()},
		Record: &agentrecord.RecordRef{
			EntityType: string(recordkind.WebWatch),
			ID:         view.ID.String(),
		},
		State: "The watch is " + view.Status.String() + ".",
	}, nil
}
