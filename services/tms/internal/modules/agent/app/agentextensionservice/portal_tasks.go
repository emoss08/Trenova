package agentextensionservice

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/portaljobs/portalcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	ReadBudget            = 4 * time.Minute
	readPollInterval      = 3 * time.Second
	maxPollFailures       = 3
	portalMetadataRun     = "trenovaPortalRun"
	portalMetadataMode    = "mode"
	portalSiteListLimit   = 10
	portalStrayedMessage  = "The browser reached pages outside this site's addresses, so what it found was withheld. A person can review the task under AI Control, Extensions."
	portalTimedOutMessage = "The portal did not answer within four minutes, so the task was stopped. Ask for one thing at a time, or name the page or reference to look at."

	portalOnTrenovaMessage = "Trenova itself cannot be a portal site: agents act in Trenova with their own tools, as the person they work for."
)

var (
	_ agentextensionports.PortalAutomator   = (*Service)(nil)
	_ agentextensionports.PortalRunFinisher = (*Service)(nil)
	_ agentextensionports.PortalSignInCodes = (*Service)(nil)
)

type portalCall struct {
	*extensionSettings
	settings agentextension.PortalAutomationSettings
	provider agentextensionports.BrowserProvider
}

func (s *Service) portalAutomation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*portalCall, error) {
	extension, err := s.serving(ctx, tenantInfo, agentextension.CapabilityPortalAutomation)
	if err != nil {
		return nil, err
	}

	return s.portalCallFor(extension)
}

func (s *Service) portalCallFor(extension *extensionSettings) (*portalCall, error) {
	provider, ok := s.browsers[extension.typ]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"the {0} extension cannot browse", extension.typ.String(),
		)
	}

	multiErr := errortypes.NewMultiError()
	agentextension.ValidateSettings(extension.typ, extension.values, multiErr)
	settings, err := agentextension.ParsePortalAutomationSettings(extension.values)
	if multiErr.HasErrors() || err != nil {
		s.l.Error("stored portal automation settings are invalid",
			zap.String("type", extension.typ.String()), zap.Error(errors.Join(multiErr, err)))
		return nil, &ToolError{Message: "The " + extension.def.title() + invalidSettingsSuffix}
	}

	return &portalCall{extensionSettings: extension, settings: settings, provider: provider}, nil
}

func (c *portalCall) metered(tenantInfo pagination.TenantInfo) meteredCall {
	return meteredCall{tenant: tenantInfo, typ: c.typ, limit: c.settings.DailyRequestLimit}
}

func summaryOf(site *agentextension.PortalSite) agentextensionports.PortalSiteSummary {
	return agentextensionports.PortalSiteSummary{
		ID:           site.ID,
		Name:         site.Name,
		Purpose:      site.Purpose,
		Domains:      site.Domains,
		StartURL:     site.StartURL,
		AllowActions: site.AllowActions,
		HasSignIn:    site.HasSignIn(),
	}
}

func (s *Service) ListPortalSites(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]agentextensionports.PortalSiteSummary, error) {
	sites, err := s.sites.List(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list portal sites").WithInternal(err)
	}

	summaries := make([]agentextensionports.PortalSiteSummary, 0, len(sites))
	for _, site := range sites {
		if site.Enabled {
			summaries = append(summaries, summaryOf(site))
		}
	}

	return summaries, nil
}

// site is the portal a task names, refused with the sites that exist when it
// is not one agents may use for this.
func (s *Service) site(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	acting bool,
) (*agentextension.PortalSite, error) {
	site, err := s.sites.Get(ctx, tenantInfo, id)
	if errortypes.IsNotFoundError(err) {
		return nil, &ToolError{Message: "There is no portal site with that id. " +
			s.knownSites(ctx, tenantInfo)}
	}
	if err != nil {
		return nil, err
	}
	if !site.Enabled {
		return nil, &ToolError{Message: site.Name + " is turned off for agents. " +
			s.knownSites(ctx, tenantInfo)}
	}
	if s.appHost != "" && agentextension.HostAllowed(s.appHost, site.Domains) {
		return nil, &ToolError{Message: portalOnTrenovaMessage + " Use the tools for it instead, " +
			"or tell the person what to do there."}
	}
	if acting && !site.AllowActions {
		return nil, &ToolError{Message: "Agents may only read " + site.Name + ", not act on " +
			"it. Tell the person what to do there, or ask an administrator to let agents act " +
			"on it under AI Control, Extensions."}
	}

	return site, nil
}

func (s *Service) knownSites(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	sites, err := s.ListPortalSites(ctx, tenantInfo)
	if err != nil || len(sites) == 0 {
		return "No portal sites are set up for agents."
	}

	var b strings.Builder
	b.WriteString("The sites set up are: ")
	for idx, site := range sites {
		if idx == portalSiteListLimit {
			b.WriteString(", and more; call web_list_portal_sites")
			break
		}
		if idx > 0 {
			b.WriteString(", ")
		}
		b.WriteString(site.Name)
		b.WriteString(" (")
		b.WriteString(site.ID.String())
		b.WriteString(")")
	}
	b.WriteString(".")

	return b.String()
}

func (s *Service) signIn(site *agentextension.PortalSite) (map[string]string, error) {
	if !site.HasSignIn() {
		return nil, nil //nolint:nilnil // a site without a sign-in sends no secrets
	}

	values, err := s.secrets.ReadAll(
		site.Credentials,
		agentextension.PortalCredentialFields(),
		portalSiteScope(pagination.TenantInfo{
			OrgID: site.OrganizationID,
			BuID:  site.BusinessUnitID,
		}, site.ID),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to read the site's sign-in").
			WithInternal(err)
	}

	secrets := map[string]string{
		agentextension.PortalSecretUsername: values[agentextension.PortalCredentialUsername],
		agentextension.PortalSecretPassword: values[agentextension.PortalCredentialPassword],
	}
	if key := values[agentextension.PortalCredentialAuthenticator]; key != "" {
		secrets[agentextension.PortalSecretAuthenticator] = key
	}

	return secrets, nil
}

func taskText(
	site *agentextension.PortalSite,
	mode agentextension.PortalRunMode,
	instruction string,
	details map[string]string,
) *agentextension.PortalTaskText {
	return &agentextension.PortalTaskText{
		SiteName:    site.Name,
		SitePurpose: site.Purpose,
		Domains:     site.Domains,
		Mode:        mode,
		Instruction: instruction,
		Details:     details,
		UsesSignIn:  site.HasSignIn(),

		UsesAuthenticator: site.HasSignIn() && site.HasAuthenticator(),
	}
}

func checkTask(instruction string, raw map[string]string) (map[string]string, error) {
	if err := agentextension.CheckPortalInstruction(instruction); err != nil {
		return nil, &ToolError{Message: stringutils.CapitalizeFirst(err.Error()) + "."}
	}

	return agentextension.NormalizePortalDetails(raw)
}

type portalStart struct {
	tenant pagination.TenantInfo
	call   *portalCall
	site   *agentextension.PortalSite
	text   *agentextension.PortalTaskText
	run    *agentextension.PortalRun
}

// begin records the run and hands the task to the vendor. The run is written
// first so a task the vendor took is never one Trenova has no record of.
func (s *Service) begin(ctx context.Context, start *portalStart) error {
	secrets, err := s.signIn(start.site)
	if err != nil {
		return err
	}

	created, err := s.runs.Create(ctx, start.run)
	if err != nil {
		return err
	}
	start.run = created

	started, err := start.call.provider.StartTask(ctx, &agentextensionports.PortalTaskRequest{
		Settings:       start.call.values,
		Instruction:    start.run.Instruction,
		Guidance:       start.text.Guidance(),
		StartURL:       start.site.StartURL,
		AllowedDomains: start.site.Domains,
		Secrets:        secrets,
		ProfileID:      start.site.BrowserProfileID,
		Metadata: map[string]string{
			portalMetadataRun:  created.ID.String(),
			portalMetadataMode: strings.ToLower(created.Mode.String()),
		},
	})
	if err != nil {
		failure := s.providerError(start.call.failing(err, "start the portal task"))
		s.finish(ctx, start.tenant, start.run, &agentextension.PortalRunOutcome{
			Status:  agentextension.PortalRunStatusFailed,
			Failure: failure.Error(),
		})
		return failure
	}

	start.run.VendorTaskID = started.TaskID
	start.run.VendorSessionID = started.SessionID
	updated, err := s.runs.Update(ctx, start.run)
	if err != nil {
		s.l.Error("the vendor started a portal task Trenova could not record",
			zap.String("portalRun", start.run.ID.String()), zap.Error(err))
		return err
	}
	start.run = updated

	return nil
}

func (s *Service) newRun(
	start *portalStart,
	owner agentextensionports.PortalTaskOwner,
	details map[string]string,
) *agentextension.PortalRun {
	run := &agentextension.PortalRun{
		ID:                pulid.MustNew(agentextension.PortalRunIDPrefix),
		OrganizationID:    start.tenant.OrgID,
		BusinessUnitID:    start.tenant.BuID,
		SiteID:            start.site.ID,
		ExtensionType:     start.call.typ,
		Mode:              start.text.Mode,
		Status:            agentextension.PortalRunStatusRunning,
		Instruction:       start.text.Task(),
		Details:           details,
		UsedSignIn:        start.site.HasSignIn(),
		AgentDefinitionID: owner.DefinitionID,
		AgentRunID:        owner.RunID,
		ThreadID:          owner.ThreadID,
		ProposalID:        owner.ProposalID,
		StartedAt:         s.now(),
		Version:           1,
	}
	if run.Details == nil {
		run.Details = map[string]string{}
	}
	if owner.Actor != nil {
		run.RequestedByID = owner.Actor.PersonUserID()
	}

	return run
}

// ReadPortal looks something up on a portal and waits for the answer within
// the tool call. A task still running at the budget is stopped rather than
// left to run unwatched. One the portal stopped for a sign-in code the person
// can relay is kept open on the code prompt and followed from there, and the
// outcome says it waits on the code.
func (s *Service) ReadPortal(
	ctx context.Context,
	req *agentextensionports.PortalReadRequest,
) (*agentextensionports.PortalReadOutcome, error) {
	details, err := checkTask(req.Question, req.Details)
	if err != nil {
		return nil, err
	}
	call, err := s.portalAutomation(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	site, err := s.site(ctx, req.TenantInfo, req.SiteID, false)
	if err != nil {
		return nil, err
	}

	start := &portalStart{
		tenant: req.TenantInfo,
		call:   call,
		site:   site,
		text:   taskText(site, agentextension.PortalRunModeRead, req.Question, details),
	}
	if _, err = s.reserve(ctx, call.metered(req.TenantInfo)); err != nil {
		return nil, err
	}
	start.run = s.newRun(start, req.Owner, details)
	if err = s.begin(ctx, start); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(ReadBudget)
	state, timedOut, err := s.await(ctx, call, start.run.VendorTaskID, deadline)
	if err != nil {
		finished := s.finish(ctx, req.TenantInfo, start.run, &agentextension.PortalRunOutcome{
			Status:  agentextension.PortalRunStatusFailed,
			Failure: "Trenova lost track of the task while it ran.",
		})
		s.release(ctx, call, finished)
		return nil, s.portalOutcomeError(finished, err, call)
	}

	outcome, asked := outcomeOf(start.run, state, site, timedOut)
	if asked && relays(start.run) {
		return s.readAwaitingCode(ctx, &awaitingRead{
			tenant:  req.TenantInfo,
			call:    call,
			site:    site,
			run:     start.run,
			outcome: outcome,
		})
	}

	finished := s.finish(ctx, req.TenantInfo, start.run, outcome)
	s.release(ctx, call, finished)
	if finished.Status != agentextension.PortalRunStatusSucceeded {
		return nil, s.portalOutcomeError(finished, nil, call)
	}

	return &agentextensionports.PortalReadOutcome{
		RunID:       finished.ID,
		Site:        summaryOf(site),
		Status:      finished.Status,
		Answer:      finished.Answer,
		VisitedURLs: finished.VisitedURLs,
		Steps:       finished.Steps,
		RetrievedAt: s.now(),
	}, nil
}

// await polls the task until it ends or the deadline passes, when it is
// stopped. A few polls in a row that fail end it with the last error.
func (s *Service) await(
	ctx context.Context,
	call *portalCall,
	taskID string,
	deadline time.Time,
) (*agentextensionports.PortalTaskState, bool, error) {
	failures := 0
	for {
		state, err := call.provider.TaskState(ctx, call.values, taskID)
		switch {
		case err != nil:
			failures++
			if failures >= maxPollFailures || ctx.Err() != nil {
				s.stopQuietly(ctx, call, taskID)
				return nil, false, err
			}
		case state.Phase != agentextensionports.PortalTaskRunning:
			return state, false, nil
		default:
			failures = 0
		}

		if !time.Now().Before(deadline) {
			s.stopQuietly(ctx, call, taskID)
			last, lastErr := call.provider.TaskState(ctx, call.values, taskID)
			if lastErr != nil || last == nil {
				last = &agentextensionports.PortalTaskState{}
			}
			return last, true, nil
		}

		timer := time.NewTimer(readPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.stopQuietly(context.WithoutCancel(ctx), call, taskID)
			return nil, false, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Service) stopQuietly(ctx context.Context, call *portalCall, taskID string) {
	if err := call.provider.StopTask(ctx, call.values, taskID); err != nil {
		s.l.Warn("could not stop a portal task", zap.String("task", taskID), zap.Error(err))
	}
}

// progressOf is what a run has done with its current task: the task's own
// pages, steps and cost, added to what the run held before a relayed code
// started that task.
func progressOf(
	run *agentextension.PortalRun,
	state *agentextensionports.PortalTaskState,
	site *agentextension.PortalSite,
) *agentextension.PortalRunOutcome {
	visited := state.VisitedURLs
	steps := state.Steps
	cost := state.CostUSD
	if run.Continued() {
		visited = append(slices.Clone(run.VisitedURLs), state.VisitedURLs...)
		steps += run.Steps
		cost = cost.Add(run.CostUSD)
	}

	return &agentextension.PortalRunOutcome{
		Answer:      state.Answer,
		VisitedURLs: visited,
		OffSiteURLs: agentextension.OffSite(visited, site.Domains),
		Steps:       steps,
		CostUSD:     cost,
	}
}

// outcomeOf decides how a task ended, and says whether the portal stopped it
// for a sign-in code. A task that reached a page outside the site's addresses
// is Strayed whatever it says it did: what it found may have come from
// anywhere.
func outcomeOf(
	run *agentextension.PortalRun,
	state *agentextensionports.PortalTaskState,
	site *agentextension.PortalSite,
	timedOut bool,
) (*agentextension.PortalRunOutcome, bool) {
	outcome := progressOf(run, state, site)
	succeeded := state.Phase == agentextensionports.PortalTaskFinished &&
		(state.Succeeded == nil || *state.Succeeded)
	asked := false

	switch {
	case len(outcome.OffSiteURLs) > 0:
		outcome.Status = agentextension.PortalRunStatusStrayed
		outcome.Failure = portalStrayedMessage
	case timedOut:
		outcome.Status = agentextension.PortalRunStatusTimedOut
		outcome.Failure = portalTimedOutMessage
	case agentextension.AsksForSignInCode(state.Answer, succeeded):
		asked = true
		outcome.Status = agentextension.PortalRunStatusFailed
		outcome.Failure = signInCodeMessage(site, run)
	case succeeded:
		outcome.Status = agentextension.PortalRunStatusSucceeded
	case state.Phase == agentextensionports.PortalTaskStopped:
		outcome.Status = agentextension.PortalRunStatusStopped
		outcome.Failure = "The vendor stopped the task before it finished."
	default:
		outcome.Status = agentextension.PortalRunStatusFailed
		outcome.Failure = state.Answer
		if strings.TrimSpace(outcome.Failure) == "" {
			outcome.Failure = "The browser could not finish the task and did not say why."
		}
	}

	return outcome, asked
}

// signInCodeMessage says what the person can change when a portal stopped the
// browser for a sign-in code: add the authenticator key, check the one stored,
// or use a browser profile that is already signed in. A run that already
// spent its relayed codes says so instead.
func signInCodeMessage(site *agentextension.PortalSite, run *agentextension.PortalRun) string {
	if run.CodeAttempts >= agentextension.MaxPortalCodeAttempts {
		return site.Name + " still asked for a sign-in code after " +
			strconv.Itoa(run.CodeAttempts) + " codes were entered, so the task was stopped. " +
			"Check that the codes come from this portal's sign-in, then ask again."
	}
	if site.HasSignIn() && site.HasAuthenticator() {
		return site.Name + " asked for a sign-in code the browser could not give from the " +
			"site's authenticator key: the key may be wrong, or the portal sends its codes " +
			"another way. An administrator can check the key under AI Control, Extensions, or " +
			"give the site a browser profile that is already signed in."
	}

	return site.Name + " asked for a sign-in code the browser was not given. An administrator " +
		"can add the portal's authenticator key to the site under AI Control, Extensions, or " +
		"give it a browser profile that is already signed in."
}

// finish records how the task ended and what it cost. A failure to record it
// is logged, not returned: the task has ended either way, and the caller
// reports what it knows.
func (s *Service) finish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	run *agentextension.PortalRun,
	outcome *agentextension.PortalRunOutcome,
) *agentextension.PortalRun {
	run.Finish(outcome, s.now())
	updated, err := s.runs.Update(ctx, run)
	if err != nil {
		s.l.Error("failed to record how a portal task ended",
			zap.String("portalRun", run.ID.String()), zap.Error(err))
	} else {
		run = updated
	}

	s.settle(ctx, meteredCall{tenant: tenantInfo, typ: run.ExtensionType}, settledCall{
		day:    timeutils.DayKeyUTC(run.StartedAt),
		failed: run.Status != agentextension.PortalRunStatusSucceeded,
		cost:   run.CostUSD,
	})

	return run
}

func (s *Service) portalOutcomeError(
	run *agentextension.PortalRun,
	cause error,
	call *portalCall,
) error {
	switch run.Status {
	case agentextension.PortalRunStatusStrayed:
		return &ToolError{Message: portalStrayedMessage}
	case agentextension.PortalRunStatusTimedOut:
		return &ToolError{Message: portalTimedOutMessage}
	}
	if cause != nil {
		return s.providerError(call.failing(cause, "follow the portal task"))
	}

	return &ToolError{Message: "The portal task did not succeed: " +
		stringutils.Ellipsize(run.Failure, agentextension.MaxPortalFailureRunes) +
		" Tell the person, and do not try the same task again unchanged."}
}

// PlanPortalAction is what acting on a portal would send, checked without
// reaching the vendor: the preview, the check before filing and the action
// itself all read it.
func (s *Service) PlanPortalAction(
	ctx context.Context,
	req *agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionPlan, error) {
	details, err := checkTask(req.Instruction, req.Details)
	if err != nil {
		return nil, err
	}
	call, err := s.portalAutomation(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	site, err := s.site(ctx, req.TenantInfo, req.SiteID, true)
	if err != nil {
		return nil, err
	}
	text := taskText(site, agentextension.PortalRunModeAct, req.Instruction, details)

	return &agentextensionports.PortalActionPlan{
		Site:        summaryOf(site),
		Instruction: strings.TrimSpace(req.Instruction),
		Details:     details,
		SentTask:    text.Task(),
		UsesSignIn:  site.HasSignIn(),
		Vendor:      call.def.Vendor,

		UsesAuthenticator: text.UsesAuthenticator,
	}, nil
}

// StartPortalAction hands an approved action to the vendor and returns once
// the vendor has it. It is keyed by the approval: an approval retried finds
// the task it started and does not act twice.
func (s *Service) StartPortalAction(
	ctx context.Context,
	req *agentextensionports.PortalActionRequest,
) (*agentextensionports.PortalActionStarted, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		return nil, errortypes.NewBusinessError("Acting on a portal needs the approval it runs from")
	}
	if existing, err := s.runs.GetByIdempotencyKey(ctx, req.TenantInfo, key); err == nil {
		return s.resumed(ctx, req.TenantInfo, existing)
	} else if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	details, err := checkTask(req.Instruction, req.Details)
	if err != nil {
		return nil, err
	}
	call, err := s.portalAutomation(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	site, err := s.site(ctx, req.TenantInfo, req.SiteID, true)
	if err != nil {
		return nil, err
	}

	start := &portalStart{
		tenant: req.TenantInfo,
		call:   call,
		site:   site,
		text:   taskText(site, agentextension.PortalRunModeAct, req.Instruction, details),
	}
	if _, err = s.reserve(ctx, call.metered(req.TenantInfo)); err != nil {
		return nil, err
	}
	start.run = s.newRun(start, req.Owner, details)
	start.run.IdempotencyKey = key
	if err = s.begin(ctx, start); err != nil {
		if errors.Is(err, agentextensionports.ErrPortalRunExists) {
			existing, getErr := s.runs.GetByIdempotencyKey(ctx, req.TenantInfo, key)
			if getErr != nil {
				return nil, getErr
			}
			return s.resumed(ctx, req.TenantInfo, existing)
		}
		return nil, err
	}

	return &agentextensionports.PortalActionStarted{
		RunID:  start.run.ID,
		Site:   summaryOf(site),
		Status: start.run.Status,
	}, nil
}

func (s *Service) resumed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	run *agentextension.PortalRun,
) (*agentextensionports.PortalActionStarted, error) {
	started := &agentextensionports.PortalActionStarted{
		RunID:   run.ID,
		Status:  run.Status,
		Resumed: true,
	}
	if site, err := s.sites.Get(ctx, tenantInfo, run.SiteID); err == nil {
		started.Site = summaryOf(site)
	}
	if run.Status == agentextension.PortalRunStatusRunning && run.VendorTaskID == "" {
		run = s.finish(ctx, tenantInfo, run, &agentextension.PortalRunOutcome{
			Status:  agentextension.PortalRunStatusFailed,
			Failure: "The vendor never took the task.",
		})
		started.Status = run.Status
	}
	if run.Status == agentextension.PortalRunStatusSucceeded {
		started.Answer = run.Answer
	}

	return started, nil
}

// TrackPortalRun starts the workflow that follows an action to its end.
func (s *Service) TrackPortalRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
	siteName string,
) error {
	if s.workflows == nil {
		return errortypes.NewBusinessError("Portal tasks cannot be followed here")
	}

	return portalcontract.Start(ctx, &portalcontract.StartParams{
		Workflows: s.workflows,
		Tenant:    tenantInfo,
		RunID:     runID,
		SiteName:  siteName,
		Now:       s.now(),
	})
}

func (s *Service) PortalRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) (*agentextension.PortalRun, error) {
	ctx = dbscope.WithValidTenant(ctx, tenantInfo.DBTenant())

	return s.runs.Get(ctx, tenantInfo, runID)
}

// PollPortalRun asks the vendor once how an action is going and records its
// end when it has ended. The run's own extension answers, even one turned off
// since: a task the vendor is running is followed to its end. A run waiting
// on a sign-in code is left waiting until the code's time runs out, when its
// browser is closed and it fails.
func (s *Service) PollPortalRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) (*agentextensionports.PortalPoll, error) {
	ctx = dbscope.WithValidTenant(ctx, tenantInfo.DBTenant())
	run, err := s.runs.Get(ctx, tenantInfo, runID)
	if err != nil {
		return nil, err
	}
	if run.Status.Finished() {
		return &agentextensionports.PortalPoll{Run: run}, nil
	}
	if run.VendorTaskID == "" {
		return &agentextensionports.PortalPoll{
			Run: s.finish(ctx, tenantInfo, run, &agentextension.PortalRunOutcome{
				Status:  agentextension.PortalRunStatusFailed,
				Failure: "The vendor never took the task.",
			}),
		}, nil
	}

	call, err := s.portalCallOf(ctx, tenantInfo, run.ExtensionType)
	if err != nil {
		return nil, err
	}
	if run.Status == agentextension.PortalRunStatusAwaitingCode {
		if expires := run.CodeExpiresAt(); expires != nil && s.now() >= *expires {
			finished := s.finish(ctx, tenantInfo, run, heldProgress(run,
				agentextension.PortalRunStatusFailed, portalCodeExpiredMessage))
			s.release(ctx, call, finished)
			return &agentextensionports.PortalPoll{Run: finished}, nil
		}
		return &agentextensionports.PortalPoll{Run: run}, nil
	}

	state, err := call.provider.TaskState(ctx, call.values, run.VendorTaskID)
	if err != nil {
		return nil, err
	}
	if state.Phase == agentextensionports.PortalTaskRunning {
		return &agentextensionports.PortalPoll{Run: run}, nil
	}

	site, err := s.sites.Get(ctx, tenantInfo, run.SiteID)
	if err != nil {
		return nil, err
	}

	outcome, asked := outcomeOf(run, state, site, false)
	if asked && relays(run) {
		awaiting, awaitErr := s.awaitCode(ctx, run, outcome)
		if awaitErr != nil {
			return nil, awaitErr
		}
		return &agentextensionports.PortalPoll{Run: awaiting, CodeRequested: true}, nil
	}

	finished := s.finish(ctx, tenantInfo, run, outcome)
	s.release(ctx, call, finished)

	return &agentextensionports.PortalPoll{Run: finished}, nil
}

// StopPortalRun stops an action that ran too long or could no longer be
// followed, and records it as ended with the reason.
func (s *Service) StopPortalRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
	status agentextension.PortalRunStatus,
	reason string,
) (*agentextension.PortalRun, error) {
	ctx = dbscope.WithValidTenant(ctx, tenantInfo.DBTenant())
	run, err := s.runs.Get(ctx, tenantInfo, runID)
	if err != nil {
		return nil, err
	}
	if run.Status.Finished() {
		return run, nil
	}

	if run.Status == agentextension.PortalRunStatusAwaitingCode {
		finished := s.finish(ctx, tenantInfo, run, heldProgress(run, status, reason))
		if call, callErr := s.portalCallOf(ctx, tenantInfo, run.ExtensionType); callErr == nil {
			s.release(ctx, call, finished)
		}
		return finished, nil
	}

	outcome := &agentextension.PortalRunOutcome{Status: status, Failure: reason}
	if run.VendorTaskID == "" {
		return s.finish(ctx, tenantInfo, run, outcome), nil
	}
	call, callErr := s.portalCallOf(ctx, tenantInfo, run.ExtensionType)
	if callErr != nil {
		return s.finish(ctx, tenantInfo, run, outcome), nil
	}

	s.stopQuietly(ctx, call, run.VendorTaskID)
	state, stateErr := call.provider.TaskState(ctx, call.values, run.VendorTaskID)
	site, siteErr := s.sites.Get(ctx, tenantInfo, run.SiteID)
	if stateErr == nil && siteErr == nil {
		outcome = progressOf(run, state, site)
		outcome.Answer = ""
		outcome.Status = status
		outcome.Failure = reason
	}
	finished := s.finish(ctx, tenantInfo, run, outcome)
	s.release(ctx, call, finished)

	return finished, nil
}

func (s *Service) portalCallOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*portalCall, error) {
	extension, err := s.settingsFor(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}

	return s.portalCallFor(extension)
}
