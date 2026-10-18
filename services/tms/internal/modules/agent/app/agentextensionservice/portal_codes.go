package agentextensionservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const portalCodeExpiredMessage = "No sign-in code was entered within 10 minutes, so the browser " +
	"was closed and the task stopped. Ask again when the person can enter the code as soon as " +
	"it arrives."

// relays says a run can wait for a person to relay the sign-in code the portal
// sent them: it belongs to a conversation someone answers in, its browser is
// still open, and it has codes left to try.
func relays(run *agentextension.PortalRun) bool {
	return run.ThreadID.IsNotNil() && run.VendorSessionID != "" &&
		run.CodeAttempts < agentextension.MaxPortalCodeAttempts
}

// awaitCode records a run as waiting on a sign-in code, its browser left open
// on the portal's code prompt.
func (s *Service) awaitCode(
	ctx context.Context,
	run *agentextension.PortalRun,
	progress *agentextension.PortalRunOutcome,
) (*agentextension.PortalRun, error) {
	version := run.Version
	run.AwaitCode(progress, s.now())
	updated, err := s.runs.Update(ctx, run)
	if err != nil {
		run.Version = version
		s.l.Error("failed to record a portal task waiting on a sign-in code",
			zap.String("portalRun", run.ID.String()), zap.Error(err))
		return nil, err
	}

	return updated, nil
}

// heldProgress ends a run with what it did before it waited on a code: no task
// is running in its browser to ask.
func heldProgress(
	run *agentextension.PortalRun,
	status agentextension.PortalRunStatus,
	failure string,
) *agentextension.PortalRunOutcome {
	return &agentextension.PortalRunOutcome{
		Status:      status,
		Failure:     failure,
		VisitedURLs: run.VisitedURLs,
		OffSiteURLs: run.OffSiteURLs,
		Steps:       run.Steps,
		CostUSD:     run.CostUSD,
	}
}

// release closes the browser a run ended in. A task's browser stays open after
// the task so a relayed code can continue in it, so it is closed here once the
// run has ended rather than when the vendor's task does.
func (s *Service) release(ctx context.Context, call *portalCall, run *agentextension.PortalRun) {
	if run.VendorSessionID == "" || !run.Status.Finished() {
		return
	}

	err := call.provider.StopSession(context.WithoutCancel(ctx), call.values, run.VendorSessionID)
	if err != nil {
		s.l.Warn("could not close a portal task's browser",
			zap.String("portalRun", run.ID.String()), zap.Error(err))
	}
}

// mayReach says a person may enter a run's code, open its browser or follow it: the
// person it was asked for, or someone who administers agents. Anyone else is told
// there is no such run.
func mayReach(administers bool, run *agentextension.PortalRun, userID pulid.ID) bool {
	if administers {
		return true
	}

	return run.RequestedByID.IsNotNil() && run.RequestedByID == userID
}

type awaitingRead struct {
	tenant  pagination.TenantInfo
	call    *portalCall
	site    *agentextension.PortalSite
	run     *agentextension.PortalRun
	outcome *agentextension.PortalRunOutcome
}

// readAwaitingCode keeps a read open on the portal's code prompt and follows
// it from there, so the answer reaches the conversation once the person has
// entered the code. A read that cannot be followed ends as the code request it
// is.
func (s *Service) readAwaitingCode(
	ctx context.Context,
	read *awaitingRead,
) (*agentextensionports.PortalReadOutcome, error) {
	current := read.run
	awaiting, err := s.awaitCode(ctx, read.run, read.outcome)
	if err == nil {
		current = awaiting
		err = s.TrackPortalRun(ctx, read.tenant, awaiting.ID, read.site.Name)
	}
	if err != nil {
		s.l.Error("could not keep a portal read open for a sign-in code",
			zap.String("portalRun", read.run.ID.String()), zap.Error(err))
		finished := s.finish(ctx, read.tenant, current, read.outcome)
		s.release(ctx, read.call, finished)
		return nil, s.portalOutcomeError(finished, nil, read.call)
	}

	return &agentextensionports.PortalReadOutcome{
		RunID:       awaiting.ID,
		Site:        summaryOf(read.site),
		Status:      awaiting.Status,
		VisitedURLs: awaiting.VisitedURLs,
		Steps:       awaiting.Steps,
		RetrievedAt: s.now(),
		CodePrompt:  awaiting.CodePrompt,
	}, nil
}

// ProvidePortalCode continues a run waiting on a sign-in code with the code a
// person relayed. The code goes to the vendor only as a secret the browser
// types, in a follow-up task in the same browser, so it never reaches a model
// or the run's record. Only the person the run was asked for, or someone who
// administers agents, may enter one.
func (s *Service) ProvidePortalCode(
	ctx context.Context,
	req *agentextensionports.ProvidePortalCodeRequest,
) (*agentextensionports.PortalRunView, error) {
	ctx = dbscope.WithValidTenant(ctx, req.TenantInfo.DBTenant())
	run, err := s.runs.Get(ctx, req.TenantInfo, req.RunID)
	if err != nil {
		return nil, err
	}
	if !mayReach(req.Administers, run, req.TenantInfo.UserID) {
		return nil, errortypes.NewNotFoundError("Portal run not found")
	}
	if run.Status != agentextension.PortalRunStatusAwaitingCode {
		return nil, errortypes.NewBusinessError(
			"This portal task is not waiting for a sign-in code",
		)
	}
	if expires := run.CodeExpiresAt(); expires != nil && s.now() >= *expires {
		return nil, errortypes.NewBusinessError(
			"The time to enter this code has run out. Ask the agent to try the portal again.",
		)
	}

	instruction := agentextension.ContinueAfterSignIn(run.Instruction)
	code := ""
	switch {
	case req.SignedIn && strings.TrimSpace(req.Code) != "":
		return nil, errortypes.NewValidationError("code", errortypes.ErrInvalid,
			"Enter the code, or say you signed in yourself, not both")
	case !req.SignedIn:
		code, err = agentextension.NormalizeSignInCode(req.Code)
		if err != nil {
			return nil, errortypes.NewValidationError("code", errortypes.ErrInvalid,
				"A sign-in code is 4 to 12 letters or digits, as the portal sent it")
		}
		instruction = agentextension.ContinueWithCode(run.Instruction)
	}

	call, err := s.portalCallOf(ctx, req.TenantInfo, run.ExtensionType)
	if err != nil {
		return nil, asBusinessError(err)
	}
	site, err := s.sites.Get(ctx, req.TenantInfo, run.SiteID)
	if err != nil {
		return nil, err
	}
	secrets, err := s.signIn(site)
	if err != nil {
		return nil, err
	}
	if secrets == nil {
		secrets = make(map[string]string, 1)
	}
	if code != "" {
		secrets[agentextension.PortalSecretSignInCode] = code
	}

	text := taskText(site, run.Mode, "", run.Details)
	started, err := call.provider.StartTask(ctx, &agentextensionports.PortalTaskRequest{
		Settings:       call.values,
		Instruction:    instruction,
		Guidance:       text.Guidance(),
		AllowedDomains: site.Domains,
		Secrets:        secrets,
		SessionID:      run.VendorSessionID,
		Metadata: map[string]string{
			portalMetadataRun:  run.ID.String(),
			portalMetadataMode: strings.ToLower(run.Mode.String()),
		},
	})
	if err != nil {
		failure := s.providerError(call.failing(err, "continue the portal task"))
		finished := s.finish(ctx, req.TenantInfo, run, heldProgress(run,
			agentextension.PortalRunStatusFailed, failure.Error()))
		s.release(ctx, call, finished)
		return nil, asBusinessError(failure)
	}

	run.ContinuedWith(started.TaskID)
	updated, err := s.runs.Update(ctx, run)
	if err != nil {
		s.l.Error("the vendor took a sign-in code Trenova could not record",
			zap.String("portalRun", run.ID.String()), zap.Error(err))
		s.stopQuietly(ctx, call, started.TaskID)
		return nil, err
	}

	view := portalRunView(updated, site.Name)

	return &view, nil
}

// PortalRunFor is one run, for the person it was asked for or someone who
// administers agents: the Desk's card follows it while it runs.
func (s *Service) PortalRunFor(
	ctx context.Context,
	req *agentextensionports.PortalRunAccessRequest,
) (*agentextensionports.PortalRunView, error) {
	run, err := s.accessibleRun(ctx, req)
	if err != nil {
		return nil, err
	}

	name := ""
	if site, siteErr := s.sites.Get(ctx, req.TenantInfo, run.SiteID); siteErr == nil {
		name = site.Name
	}
	view := portalRunView(run, name)

	return &view, nil
}

// PortalRunLive is the address of the browser a run is in, while it is open:
// a person watches the task there, or signs in themselves when the portal
// asked for a code. It is read when asked for, never stored: it opens a
// browser signed in to the portal.
func (s *Service) PortalRunLive(
	ctx context.Context,
	req *agentextensionports.PortalRunAccessRequest,
) (*agentextensionports.PortalLiveView, error) {
	run, err := s.accessibleRun(ctx, req)
	if err != nil {
		return nil, err
	}
	if run.Status.Finished() || run.VendorSessionID == "" {
		return nil, errortypes.NewBusinessError("This portal task's browser is closed")
	}

	call, err := s.portalCallOf(ctx, req.TenantInfo, run.ExtensionType)
	if err != nil {
		return nil, asBusinessError(err)
	}
	live, err := call.provider.LiveView(ctx, call.values, run.VendorSessionID)
	if err != nil {
		return nil, asBusinessError(s.providerError(call.failing(err, "open the live browser")))
	}
	if live == "" {
		return nil, errortypes.NewBusinessError("The vendor has no live view of this browser")
	}

	return &agentextensionports.PortalLiveView{LiveURL: live}, nil
}

func (s *Service) accessibleRun(
	ctx context.Context,
	req *agentextensionports.PortalRunAccessRequest,
) (*agentextension.PortalRun, error) {
	ctx = dbscope.WithValidTenant(ctx, req.TenantInfo.DBTenant())
	run, err := s.runs.Get(ctx, req.TenantInfo, req.RunID)
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewNotFoundError("Portal run not found")
	}
	if err != nil {
		return nil, err
	}
	if !mayReach(req.Administers, run, req.TenantInfo.UserID) {
		return nil, errortypes.NewNotFoundError("Portal run not found")
	}

	return run, nil
}
