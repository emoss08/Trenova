package extensionwebwatchservice

import (
	"cmp"
	"context"
	"errors"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/extensionsecrets"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/webwatchjobs/webwatchcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	webWatchAPIBase       = "/api/v1"
	webWatchMetadataKey   = "trenovaWebWatch"
	webWatchNamePrefix    = "Trenova: "
	maxVendorMonitorName  = 250
	secondsPerDay         = 24 * 60 * 60
	defaultWebWatchesPage = 25

	webWatchNeedsPublicAddress = "Watching the web needs Trenova at a public https address, " +
		"so the vendor can report what it finds. An administrator sets it as app.webBaseUrl."
)

var (
	_ agentextensionports.WebWatcher         = (*Service)(nil)
	_ agentextensionports.WebWatchRunner     = (*Service)(nil)
	_ agentextensionports.WebWatchDeliveries = (*Service)(nil)
)

type webWatchCall struct {
	*agentextensionservice.ExtensionSettings
	excluded []string
	allowed  int
	provider agentextensionports.WebWatchProvider
}

// webWatching is the extension serving web research, when its vendor can
// watch the web on a schedule.
func (s *Service) webWatching(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*webWatchCall, error) {
	extension, err := s.core.Serving(ctx, tenantInfo, agentextension.CapabilityWebResearch)
	if err != nil {
		return nil, err
	}

	return s.webWatchCallFor(extension)
}

// webWatchCallOf is the extension a watch was set with, even one turned off
// since: a watch the vendor is running is followed to its end.
func (s *Service) webWatchCallOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*webWatchCall, error) {
	extension, err := s.core.SettingsFor(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}

	return s.webWatchCallFor(extension)
}

func (s *Service) webWatchCallFor(
	extension *agentextensionservice.ExtensionSettings,
) (*webWatchCall, error) {
	spec, _ := agentextension.SpecFor(extension.Type)
	provider, ok := s.watchers[extension.Type]
	if !ok || spec.Omits(agentextension.ToolWatchWeb) {
		return nil, &agentextensionservice.ToolError{
			Message: extension.Def.Vendor + " cannot watch the web on a " +
				"schedule. Search again later instead, or an administrator can use Exa for web " +
				"research under AI Control, Extensions.",
		}
	}

	multiErr := errortypes.NewMultiError()
	agentextension.ValidateSettings(extension.Type, extension.Values, multiErr)
	settings, err := agentextension.ParseWebResearchSettings(extension.Values)
	if multiErr.HasErrors() || err != nil {
		s.l.Error("stored web research settings are invalid",
			zap.String("type", extension.Type.String()), zap.Error(errors.Join(multiErr, err)))
		return nil, &agentextensionservice.ToolError{
			Message: "The " + extension.Def.Title() + agentextensionservice.InvalidSettingsSuffix,
		}
	}

	return &webWatchCall{
		ExtensionSettings: extension,
		excluded:          settings.ExcludedDomains,
		allowed:           extensionwebwatch.MaxActiveWebWatches(extension.Type, extension.Values),
		provider:          provider,
	}, nil
}

// webhookBase is the public https address the vendor reports to, refused
// when Trenova is not reachable at one: the vendor will not deliver to a
// private or local address.
func (s *Service) webhookBase() (string, error) {
	parsed, err := url.Parse(s.webBase)
	if s.webBase == "" || err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return "", &agentextensionservice.ToolError{Message: webWatchNeedsPublicAddress}
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return "", &agentextensionservice.ToolError{Message: webWatchNeedsPublicAddress}
	}
	if ip := net.ParseIP(host); ip != nil &&
		(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return "", &agentextensionservice.ToolError{Message: webWatchNeedsPublicAddress}
	}

	return strings.TrimSuffix(s.webBase, "/") + webWatchAPIBase, nil
}

func (s *Service) draftWebWatch(
	req *agentextensionports.WebWatchRequest,
	call *webWatchCall,
) (*extensionwebwatch.WebWatch, int, error) {
	days, err := extensionwebwatch.WatchDays(req.Days)
	if err != nil {
		return nil, 0, err
	}
	cadence := req.Cadence
	if cadence == "" {
		cadence = extensionwebwatch.WebWatchCadenceDaily
	}

	watch := &extensionwebwatch.WebWatch{
		ID:                pulid.MustNew(extensionwebwatch.WebWatchIDPrefix),
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		ExtensionType:     call.Type,
		Query:             req.Query,
		Purpose:           req.Purpose,
		IncludeDomains:    req.Sites,
		Cadence:           cadence,
		Status:            extensionwebwatch.WebWatchStatusActive,
		Secrets:           map[string]any{},
		EndsAt:            s.now() + int64(days)*secondsPerDay,
		AgentDefinitionID: req.Owner.DefinitionID,
		AgentRunID:        req.Owner.RunID,
		ThreadID:          req.Owner.ThreadID,
		Version:           1,
	}
	if req.Owner.Actor != nil {
		watch.RequestedByID = req.Owner.Actor.PersonUserID()
	}
	watch.Normalize()

	multiErr := errortypes.NewMultiError()
	watch.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, 0, multiErr
	}
	for _, text := range []string{watch.Query, watch.Purpose} {
		if err = agentextension.CheckSearchQuery(text); err != nil {
			return nil, 0, &agentextensionservice.ToolError{
				Message: textcase.CapitalizeFirst(err.Error()) + ".",
			}
		}
	}

	return watch, days, nil
}

// PlanWebWatch is what setting a watch would do, checked without reaching
// the vendor: the preview, the check before running and the watch itself all
// read it.
func (s *Service) PlanWebWatch(
	ctx context.Context,
	req *agentextensionports.WebWatchRequest,
) (*agentextensionports.WebWatchPlan, error) {
	call, err := s.webWatching(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return s.planWith(ctx, req, call)
}

func (s *Service) planWith(
	ctx context.Context,
	req *agentextensionports.WebWatchRequest,
	call *webWatchCall,
) (*agentextensionports.WebWatchPlan, error) {
	watch, days, err := s.draftWebWatch(req, call)
	if err != nil {
		return nil, err
	}
	if _, err = s.webhookBase(); err != nil {
		return nil, err
	}

	active, err := s.watches.CountActive(ctx, req.TenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to count web watches").WithInternal(err)
	}
	if active >= call.allowed {
		return nil, &agentextensionservice.ToolError{Message: "This organization already has " +
			strconv.Itoa(active) + " web watches running, the most " + call.Def.Vendor +
			" is set to allow. Stop one the person no longer needs with web_stop_watch, or an " +
			"administrator can raise the limit under AI Control, Extensions."}
	}

	checks := extensionwebwatch.WebWatchChecks(watch.Cadence, days)

	return &agentextensionports.WebWatchPlan{
		Query:          watch.Query,
		Purpose:        watch.Purpose,
		Sites:          watch.IncludeDomains,
		Cadence:        watch.Cadence,
		Days:           days,
		EndsAt:         watch.EndsAt,
		Vendor:         call.Def.Vendor,
		CostPerCheck:   extensionwebwatch.ExaMonitorRunCostUSD,
		EstimatedCost:  extensionwebwatch.ExaMonitorRunCostUSD.Mul(decimal.NewFromInt(checks)),
		ActiveWatches:  active,
		WatchesAllowed: call.allowed,
	}, nil
}

// WatchWeb sets a watch. The watch is recorded before the vendor is asked,
// so a monitor the vendor made is never one Trenova has no record of, and a
// watch the vendor could not take, or that could not be followed, ends at
// once with the monitor removed.
func (s *Service) WatchWeb(
	ctx context.Context,
	req *agentextensionports.WebWatchRequest,
) (*agentextensionports.WebWatchView, error) {
	call, err := s.webWatching(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.planWith(ctx, req, call); err != nil {
		return nil, err
	}
	if s.workflows == nil {
		return nil, errortypes.NewBusinessError("Web watches cannot be followed here")
	}
	base, err := s.webhookBase()
	if err != nil {
		return nil, err
	}

	draft, _, err := s.draftWebWatch(req, call)
	if err != nil {
		return nil, err
	}
	watch, err := s.watches.Create(ctx, draft)
	if err != nil {
		return nil, err
	}

	created, err := call.provider.CreateWatch(ctx, &agentextensionports.WebWatchCreateRequest{
		Settings:       call.Values,
		Name:           vendorMonitorName(watch.Purpose),
		Query:          watch.Query,
		IncludeDomains: watch.IncludeDomains,
		ExcludeDomains: excludedFor(call.excluded, watch.IncludeDomains),
		Cadence:        watch.Cadence,
		WebhookURL: base + extensionwebwatch.WebWatchWebhookPath(
			watch.OrganizationID, watch.BusinessUnitID, watch.ID,
		),
		Metadata: map[string]string{webWatchMetadataKey: watch.ID.String()},
	})
	if err != nil {
		failure := s.core.ProviderError(call.Failing(err, "set the web watch"))
		s.endQuietly(ctx, watch, extensionwebwatch.WebWatchStatusFailed, failure.Error())
		return nil, failure
	}

	watch.VendorMonitorID = created.VendorID
	watch.Secrets, err = s.secrets.Merge(
		extensionwebwatch.WebWatchSecretFields(),
		map[string]string{extensionwebwatch.WebWatchSecretField: created.Secret},
		nil,
		extensionsecrets.WebWatchScope(req.TenantInfo, watch.ID),
	)
	if err == nil {
		watch, err = s.watches.Update(ctx, watch)
	}
	if err == nil {
		err = webwatchcontract.Start(ctx, &webwatchcontract.StartParams{
			Workflows: s.workflows,
			Tenant:    req.TenantInfo,
			WatchID:   watch.ID,
			Summary:   stringutils.TruncateRunes(watch.Purpose, 80),
			Now:       s.now(),
		})
	}
	if err != nil {
		s.l.Error("a web watch the vendor took could not be kept",
			zap.String("webWatch", watch.ID.String()), zap.Error(err))
		s.removeMonitor(ctx, call, created.VendorID)
		s.endQuietly(ctx, watch, extensionwebwatch.WebWatchStatusFailed,
			"Trenova could not keep track of the watch, so it was removed.")
		return nil, errortypes.NewBusinessError("The web watch could not be set").WithInternal(err)
	}

	view := webWatchView(watch)

	return &view, nil
}

// excludedFor is the organization's excluded sites for a watch over the
// whole web. A watch that names its sites searches only those, so it sends
// none: a site it asks for is one it means.
func excludedFor(excluded, included []string) []string {
	if len(included) > 0 {
		return nil
	}

	return excluded
}

func vendorMonitorName(purpose string) string {
	return stringutils.TruncateRunes(webWatchNamePrefix+purpose, maxVendorMonitorName)
}

func (s *Service) endQuietly(
	ctx context.Context,
	watch *extensionwebwatch.WebWatch,
	status extensionwebwatch.WebWatchStatus,
	failure string,
) {
	watch.End(status, failure, s.now())
	if _, err := s.watches.Update(ctx, watch); err != nil {
		s.l.Error("failed to record how a web watch ended",
			zap.String("webWatch", watch.ID.String()), zap.Error(err))
	}
}

func (s *Service) removeMonitor(ctx context.Context, call *webWatchCall, vendorID string) {
	if vendorID == "" {
		return
	}
	if err := call.provider.DeleteWatch(context.WithoutCancel(ctx), call.Values, vendorID); err != nil {
		s.l.Warn("could not remove a web watch from its vendor",
			zap.String("monitor", vendorID), zap.Error(err))
	}
}

func (s *Service) ListWebWatches(
	ctx context.Context,
	req *agentextensionports.ListWebWatchesRequest,
) (*agentextensionports.WebWatchListResponse, error) {
	if req.Limit <= 0 {
		req.Limit = defaultWebWatchesPage
	}
	watches, err := s.watches.List(ctx, req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list web watches").WithInternal(err)
	}

	response := &agentextensionports.WebWatchListResponse{
		Items: make([]agentextensionports.WebWatchView, 0, len(watches)),
	}
	for _, watch := range watches {
		response.Items = append(response.Items, webWatchView(watch))
	}
	if len(watches) == req.Limit {
		response.NextBefore = watches[len(watches)-1].CreatedAt
	}

	return response, nil
}

func (s *Service) WebWatchFindings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watchID pulid.ID,
) (*agentextensionports.WebWatchFindingsResponse, error) {
	if _, err := s.watches.Get(ctx, tenantInfo, watchID); err != nil {
		return nil, err
	}
	findings, err := s.findings.ListByWatch(ctx, tenantInfo, watchID, 0)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list what the watch found").
			WithInternal(err)
	}

	response := &agentextensionports.WebWatchFindingsResponse{
		Items: make([]agentextensionports.WebWatchFindingView, 0, len(findings)),
	}
	for _, finding := range findings {
		response.Items = append(response.Items, agentextensionports.WebWatchFindingView{
			ID:          finding.ID,
			Summary:     finding.Summary,
			Results:     finding.Results,
			FoundAt:     finding.FoundAt,
			DeliveredAt: finding.DeliveredAt,
		})
	}

	return response, nil
}

// StopWebWatch removes the vendor's monitor and records the watch stopped.
// Only the person it was set for, or someone who administers agents, may
// stop one; a watch already ended is returned as it ended.
func (s *Service) StopWebWatch(
	ctx context.Context,
	req *agentextensionports.StopWebWatchRequest,
) (*agentextensionports.WebWatchView, error) {
	watch, err := s.watches.Get(ctx, req.TenantInfo, req.WatchID)
	if err != nil {
		return nil, err
	}
	if !maySteer(req, watch) {
		return nil, errortypes.NewNotFoundError("Web watch not found")
	}
	if !watch.Active() {
		view := webWatchView(watch)
		return &view, nil
	}

	if watch.VendorMonitorID != "" {
		call, callErr := s.webWatchCallOf(ctx, req.TenantInfo, watch.ExtensionType)
		if callErr != nil {
			return nil, agentextensionservice.AsBusinessError(callErr)
		}
		if err = call.provider.DeleteWatch(ctx, call.Values, watch.VendorMonitorID); err != nil {
			return nil, agentextensionservice.AsBusinessError(
				s.core.ProviderError(call.Failing(err, "stop the web watch")),
			)
		}
	}

	watch.End(extensionwebwatch.WebWatchStatusStopped, "", s.now())
	watch.StoppedByID = req.StoppedBy
	updated, err := s.watches.Update(ctx, watch)
	if err != nil {
		return nil, err
	}
	if s.workflows != nil {
		if signalErr := webwatchcontract.Check(ctx, s.workflows, updated.ID); signalErr != nil {
			s.l.Warn("could not tell a stopped web watch's workflow to end",
				zap.String("webWatch", updated.ID.String()), zap.Error(signalErr))
		}
	}

	view := webWatchView(updated)

	return &view, nil
}

// CheckWebWatch reads the vendor's latest runs for a watch, records each one
// it bills and each that found something new, and returns what is not yet
// handed to the work waiting on it. A watch whose time ran out, or that its
// vendor turned off, ends here, its monitor removed.
func (s *Service) CheckWebWatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watchID pulid.ID,
) (*agentextensionports.WebWatchCheck, error) {
	ctx = dbscope.WithValidTenant(ctx, tenantInfo.DBTenant())
	watch, err := s.watches.Get(ctx, tenantInfo, watchID)
	if errortypes.IsNotFoundError(err) {
		return &agentextensionports.WebWatchCheck{Ended: true}, nil
	}
	if err != nil {
		return nil, err
	}
	if !watch.Active() {
		return &agentextensionports.WebWatchCheck{Watch: watch, Ended: true}, nil
	}

	call, err := s.webWatchCallOf(ctx, tenantInfo, watch.ExtensionType)
	if err != nil {
		if watch.Due(s.now()) {
			return s.endUnreachable(ctx, tenantInfo, watch)
		}
		return nil, err
	}

	if watch.Due(s.now()) {
		if err = call.provider.DeleteWatch(ctx, call.Values, watch.VendorMonitorID); err != nil {
			return nil, err
		}
		return s.closeWatch(ctx, tenantInfo, watch, extensionwebwatch.WebWatchStatusEnded, "")
	}

	state, err := call.provider.WatchState(ctx, call.Values, watch.VendorMonitorID)
	if err != nil {
		return nil, err
	}
	if state.Disabled {
		s.removeMonitor(ctx, call, watch.VendorMonitorID)
		return s.closeWatch(ctx, tenantInfo, watch, extensionwebwatch.WebWatchStatusFailed,
			call.Def.Vendor+" turned the watch off after its requests kept failing. An "+
				"administrator can check the API key under AI Control, Extensions.")
	}

	after := int64(0)
	if watch.LastRunAt != nil {
		after = *watch.LastRunAt
	}
	runs, err := call.provider.WatchRuns(ctx, &agentextensionports.WebWatchRunsRequest{
		Settings: call.Values,
		VendorID: watch.VendorMonitorID,
		After:    after,
	})
	if err != nil {
		return nil, err
	}
	billed, err := s.recordRuns(ctx, watch, call, runs)
	if err != nil {
		return nil, err
	}

	checked := s.now()
	watch.LastCheckedAt = &checked
	updated, err := s.watches.Update(ctx, watch)
	if err != nil {
		return nil, err
	}
	for _, outcome := range billed {
		s.core.Settle(
			ctx,
			agentextensionservice.MeteredCall{Tenant: tenantInfo, Type: updated.ExtensionType},
			outcome,
		)
	}

	undelivered, err := s.findings.ListUndelivered(ctx, tenantInfo, updated.ID)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.WebWatchCheck{Watch: updated, Undelivered: undelivered}, nil
}

// recordRuns counts each settled run newer than the last one counted, oldest
// first, and stops at the first that has not settled, so it is counted once
// it has. A run already recorded as a finding is not recorded twice. What the
// runs cost is returned rather than metered here, so it is metered only once
// the watch that counted them is saved.
func (s *Service) recordRuns(
	ctx context.Context,
	watch *extensionwebwatch.WebWatch,
	call *webWatchCall,
	vendorRuns []agentextensionports.WebWatchRun,
) ([]agentextensionservice.SettledCall, error) {
	runs := slices.Clone(vendorRuns)
	slices.SortFunc(runs, func(a, b agentextensionports.WebWatchRun) int {
		return cmp.Compare(a.CreatedAt, b.CreatedAt)
	})

	billed := make([]agentextensionservice.SettledCall, 0, len(runs))
	for idx := range runs {
		run := &runs[idx]
		if watch.LastRunAt != nil && run.CreatedAt <= *watch.LastRunAt {
			continue
		}
		if !run.Phase.Settled() {
			break
		}

		watch.CountRun(run.CreatedAt, run.CostUSD)
		billed = append(billed, agentextensionservice.SettledCall{
			Day:    timeutils.DayKeyUTC(run.SettledAt),
			Failed: run.Phase == agentextensionports.WebWatchRunFailed,
			Cost:   run.CostUSD,
		})

		switch run.Phase {
		case agentextensionports.WebWatchRunFailed:
			watch.NoteFailure(runFailureMessage(call.Def.Vendor, run.Failure))
		case agentextensionports.WebWatchRunCompleted:
			if err := s.recordFinding(ctx, watch, run); err != nil {
				return nil, err
			}
		case agentextensionports.WebWatchRunPending, agentextensionports.WebWatchRunCancelled:
		}
	}

	return billed, nil
}

func (s *Service) recordFinding(
	ctx context.Context,
	watch *extensionwebwatch.WebWatch,
	run *agentextensionports.WebWatchRun,
) error {
	results := extensionwebwatch.BoundedResults(run.Results)
	if len(results) == 0 {
		return nil
	}

	_, err := s.findings.Create(ctx, &extensionwebwatch.WebWatchFinding{
		ID:             pulid.MustNew(extensionwebwatch.WebWatchFindingIDPrefix),
		OrganizationID: watch.OrganizationID,
		BusinessUnitID: watch.BusinessUnitID,
		WatchID:        watch.ID,
		VendorRunID:    run.ID,
		Summary:        extensionwebwatch.BoundedSummary(run.Summary),
		Results:        results,
		FoundAt:        run.SettledAt,
	})
	switch {
	case errors.Is(err, agentextensionports.ErrWebWatchFindingExists):
		return nil
	case err != nil:
		return err
	default:
		watch.FindingCount++
		return nil
	}
}

// runFailureMessage says why a run failed in words an administrator can act
// on, from the vendor's reason code.
func runFailureMessage(vendor, reason string) string {
	switch reason {
	case "api_key_invalid", "forbidden":
		return vendor + " rejected the organization's API key on the last check. An " +
			"administrator can update the key under AI Control, Extensions."
	case "insufficient_credits":
		return "The organization's " + vendor + " account ran out of credits on the last " +
			"check. An administrator can add credits in the " + vendor + " dashboard."
	case "rate_limited":
		return vendor + " was limiting requests on the last check; the next one tries again."
	case "source_not_available":
		return vendor + " cannot search one of the sites the watch names."
	default:
		return vendor + " could not run the last check; the next one tries again."
	}
}

func (s *Service) closeWatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watch *extensionwebwatch.WebWatch,
	status extensionwebwatch.WebWatchStatus,
	failure string,
) (*agentextensionports.WebWatchCheck, error) {
	watch.End(status, failure, s.now())
	updated, err := s.watches.Update(ctx, watch)
	if err != nil {
		return nil, err
	}
	undelivered, err := s.findings.ListUndelivered(ctx, tenantInfo, updated.ID)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.WebWatchCheck{
		Watch:       updated,
		Undelivered: undelivered,
		Ended:       true,
	}, nil
}

// endUnreachable ends a watch whose time ran out while its extension can no
// longer be read, saying where its monitor may still be running.
func (s *Service) endUnreachable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watch *extensionwebwatch.WebWatch,
) (*agentextensionports.WebWatchCheck, error) {
	def, _ := agentextensionservice.DefinitionFor(watch.ExtensionType)

	return s.closeWatch(ctx, tenantInfo, watch, extensionwebwatch.WebWatchStatusEnded,
		"The watch's time ran out, but its extension is no longer set up, so Trenova could "+
			"not remove it from "+def.Vendor+". Delete it in the "+def.Vendor+" dashboard.")
}

func (s *Service) MarkWebWatchDelivered(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	findings []*extensionwebwatch.WebWatchFinding,
) error {
	ids := make([]pulid.ID, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.ID)
	}

	return s.findings.MarkDelivered(
		dbscope.WithValidTenant(ctx, tenantInfo.DBTenant()),
		tenantInfo,
		ids,
		s.now(),
	)
}

// ReceiveWebWatchDelivery checks a vendor's webhook against the watch's
// secret and asks the workflow following the watch to look now. What the
// delivery says is never used: the workflow reads the runs from the vendor.
func (s *Service) ReceiveWebWatchDelivery(
	ctx context.Context,
	req *agentextensionports.ReceiveWebWatchDeliveryRequest,
) error {
	ctx = dbscope.WithValidTenant(ctx, req.TenantInfo.DBTenant())
	watch, err := s.watches.Get(ctx, req.TenantInfo, req.WatchID)
	if err != nil {
		return err
	}
	provider, ok := s.watchers[watch.ExtensionType]
	if !ok || !watch.HasSecret() {
		return errortypes.NewNotFoundError("Web watch not found")
	}

	values, err := s.secrets.ReadAll(
		watch.Secrets,
		extensionwebwatch.WebWatchSecretFields(),
		extensionsecrets.WebWatchScope(req.TenantInfo, watch.ID),
	)
	if err != nil {
		return errortypes.NewBusinessError("failed to read the web watch's secret").
			WithInternal(err)
	}
	if err = provider.VerifyDelivery(
		&req.Delivery,
		values[extensionwebwatch.WebWatchSecretField],
	); err != nil {
		return errortypes.NewAuthenticationError("The delivery's signature is not valid")
	}
	if !watch.Active() || s.workflows == nil {
		return nil
	}

	return webwatchcontract.Check(ctx, s.workflows, watch.ID)
}

// maySteer is the person a watch was set for, the agent whose background work
// set one for no person, or someone who administers agents; anyone else is
// told there is no such watch.
func maySteer(
	req *agentextensionports.StopWebWatchRequest,
	watch *extensionwebwatch.WebWatch,
) bool {
	switch {
	case req.Administers:
		return true
	case watch.RequestedByID.IsNotNil():
		return watch.RequestedByID == req.StoppedBy
	default:
		return watch.AgentDefinitionID.IsNotNil() && watch.AgentDefinitionID == req.DefinitionID
	}
}

func webWatchView(watch *extensionwebwatch.WebWatch) agentextensionports.WebWatchView {
	return agentextensionports.WebWatchView{
		ID:                watch.ID,
		ExtensionType:     watch.ExtensionType,
		Query:             watch.Query,
		Purpose:           watch.Purpose,
		Sites:             watch.IncludeDomains,
		Cadence:           watch.Cadence,
		Status:            watch.Status,
		EndsAt:            watch.EndsAt,
		LastCheckedAt:     watch.LastCheckedAt,
		LastRunAt:         watch.LastRunAt,
		RunCount:          watch.RunCount,
		FindingCount:      watch.FindingCount,
		CostUSD:           watch.CostUSD,
		Failure:           watch.Failure,
		AgentDefinitionID: watch.AgentDefinitionID,
		AgentRunID:        watch.AgentRunID,
		ThreadID:          watch.ThreadID,
		RequestedByID:     watch.RequestedByID,
		StoppedByID:       watch.StoppedByID,
		FinishedAt:        watch.FinishedAt,
		CreatedAt:         watch.CreatedAt,
	}
}
