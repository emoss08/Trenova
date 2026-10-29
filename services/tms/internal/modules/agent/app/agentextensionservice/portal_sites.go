package agentextensionservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const portalSiteSecretKind = "agent_portal_site"

var _ agentextensionports.PortalSites = (*Service)(nil)

func portalSiteScope(tenantInfo pagination.TenantInfo, siteID pulid.ID) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeAgentExtensionSecret,
		Tenant:       tenantInfo,
		ResourceKind: portalSiteSecretKind,
		Subject:      siteID.String(),
	}
}

func portalSiteView(site *extensionportal.PortalSite) agentextensionports.PortalSiteView {
	return agentextensionports.PortalSiteView{
		ID:               site.ID,
		Name:             site.Name,
		Purpose:          site.Purpose,
		Domains:          site.Domains,
		StartURL:         site.StartURL,
		BrowserProfileID: site.BrowserProfileID,
		AllowActions:     site.AllowActions,
		Enabled:          site.Enabled,
		HasSignIn:        site.HasSignIn(),
		HasAuthenticator: site.HasSignIn() && site.HasAuthenticator(),
		Version:          site.Version,
		CreatedAt:        site.CreatedAt,
		UpdatedAt:        site.UpdatedAt,
	}
}

func (s *Service) ListSites(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]agentextensionports.PortalSiteView, error) {
	sites, err := s.sites.List(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list portal sites").WithInternal(err)
	}

	views := make([]agentextensionports.PortalSiteView, 0, len(sites))
	for _, site := range sites {
		views = append(views, portalSiteView(site))
	}

	return views, nil
}

func (s *Service) CreateSite(
	ctx context.Context,
	req *agentextensionports.SavePortalSiteRequest,
) (*agentextensionports.PortalSiteView, error) {
	existing, err := s.sites.List(ctx, req.TenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list portal sites").WithInternal(err)
	}
	if len(existing) >= extensionportal.MaxPortalSitesPerTenant {
		return nil, errortypes.NewBusinessError(
			"An organization can set up at most 100 portal sites. Remove one you no longer use first.",
		)
	}

	site := &extensionportal.PortalSite{
		ID:             pulid.MustNew(extensionportal.PortalSiteIDPrefix),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CreatedByID:    req.TenantInfo.UserID,
		Version:        1,
	}
	if err = s.applySiteRequest(site, req); err != nil {
		return nil, err
	}

	created, err := s.sites.Create(ctx, site)
	if err != nil {
		return nil, err
	}
	s.logSiteChange(&siteChange{
		tenant:    req.TenantInfo,
		after:     created,
		operation: permission.OpCreate,
		comment:   "Portal site added for agents",
	})
	view := portalSiteView(created)

	return &view, nil
}

func (s *Service) UpdateSite(
	ctx context.Context,
	req *agentextensionports.SavePortalSiteRequest,
) (*agentextensionports.PortalSiteView, error) {
	site, err := s.sites.Get(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	if site.Version != req.Version {
		return nil, errortypes.NewConflictError(
			"This site was changed by someone else while you were editing. Reload and try again.",
		)
	}
	before := *site
	if err = s.applySiteRequest(site, req); err != nil {
		return nil, err
	}

	updated, err := s.sites.Update(ctx, site)
	if err != nil {
		return nil, err
	}
	s.logSiteChange(&siteChange{
		tenant:    req.TenantInfo,
		before:    &before,
		after:     updated,
		operation: permission.OpUpdate,
		comment:   "Portal site changed",
	})
	view := portalSiteView(updated)

	return &view, nil
}

// applySiteRequest sets what the form sent on the site. A sign-in left blank
// keeps the stored one; clearing it removes both parts.
func (s *Service) applySiteRequest(
	site *extensionportal.PortalSite,
	req *agentextensionports.SavePortalSiteRequest,
) error {
	site.Name = req.Name
	site.Purpose = req.Purpose
	site.Domains = req.Domains
	site.StartURL = req.StartURL
	site.BrowserProfileID = req.BrowserProfileID
	site.AllowActions = req.AllowActions
	site.Enabled = req.Enabled
	site.Normalize()

	multiErr := errortypes.NewMultiError()
	site.Validate(multiErr)
	s.checkNotTrenova(site, multiErr)

	username := strings.TrimSpace(req.Username)
	password := req.Password
	if !req.ClearSignIn && (username == "") != (password == "") && !site.HasSignIn() {
		multiErr.Add("password", errortypes.ErrRequired,
			"A sign-in needs both a username and a password")
	}
	authenticator, keyErr := extensionportal.NormalizeAuthenticatorKey(req.AuthenticatorKey)
	if keyErr != nil {
		multiErr.Add(
			"authenticatorKey",
			errortypes.ErrInvalid,
			"The authenticator key must be the setup key the portal showed for an authenticator app",
		)
	}
	signsIn := !req.ClearSignIn && (site.HasSignIn() || (username != "" && password != ""))
	if authenticator != "" && !signsIn {
		multiErr.Add(
			"authenticatorKey",
			errortypes.ErrInvalid,
			"An authenticator key is used with the site's sign-in; add the username and password too",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	if req.ClearSignIn {
		site.Credentials = map[string]any{}
		return nil
	}

	merged, err := s.secrets.Merge(
		extensionportal.PortalCredentialFields(),
		map[string]string{
			extensionportal.PortalCredentialUsername:      username,
			extensionportal.PortalCredentialPassword:      password,
			extensionportal.PortalCredentialAuthenticator: authenticator,
		},
		site.Credentials,
		portalSiteScope(pagination.TenantInfo{
			OrgID: site.OrganizationID,
			BuID:  site.BusinessUnitID,
		}, site.ID),
	)
	if err != nil {
		return err
	}
	if req.ClearAuthenticator && authenticator == "" {
		delete(merged, extensionportal.PortalCredentialAuthenticator)
	}
	site.Credentials = merged

	return nil
}

// checkNotTrenova refuses a site on Trenova's own address. An agent acts in
// Trenova through its own tools, as the person it works for and within their
// permissions; a browser signed in with a stored sign-in would act as
// whoever that sign-in belongs to.
func (s *Service) checkNotTrenova(
	site *extensionportal.PortalSite,
	multiErr *errortypes.MultiError,
) {
	if s.appHost != "" && extensionportal.HostAllowed(s.appHost, site.Domains) {
		multiErr.Add(
			"domains",
			errortypes.ErrInvalid,
			"Trenova itself cannot be a portal site: agents act in Trenova with their own tools, as the person they work for.",
		)
	}
}

func (s *Service) DeleteSite(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	site, err := s.sites.Get(ctx, tenantInfo, id)
	if err != nil {
		return err
	}
	if err = s.sites.Delete(ctx, tenantInfo, id); err != nil {
		return err
	}
	s.logSiteChange(&siteChange{
		tenant:    tenantInfo,
		before:    site,
		after:     site,
		operation: permission.OpDelete,
		comment:   "Portal site removed",
	})

	return nil
}

func (s *Service) ListRuns(
	ctx context.Context,
	req *agentextensionports.ListPortalRunsRequest,
) (*agentextensionports.PortalRunListResponse, error) {
	runs, err := s.runs.List(ctx, req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list portal runs").WithInternal(err)
	}
	sites, err := s.sites.List(ctx, req.TenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list portal sites").WithInternal(err)
	}
	names := make(map[pulid.ID]string, len(sites))
	for _, site := range sites {
		names[site.ID] = site.Name
	}

	views := make([]agentextensionports.PortalRunView, 0, len(runs))
	for _, run := range runs {
		views = append(views, portalRunView(run, names[run.SiteID]))
	}
	response := &agentextensionports.PortalRunListResponse{Items: views}
	limit := req.Limit
	if limit <= 0 {
		limit = len(runs)
	}
	if len(runs) > 0 && len(runs) >= limit {
		response.NextBefore = runs[len(runs)-1].StartedAt
	}

	return response, nil
}

func portalRunView(
	run *extensionportal.PortalRun,
	siteName string,
) agentextensionports.PortalRunView {
	return agentextensionports.PortalRunView{
		ID:              run.ID,
		SiteID:          run.SiteID,
		SiteName:        siteName,
		ExtensionType:   run.ExtensionType,
		Mode:            run.Mode,
		Status:          run.Status,
		Instruction:     run.Instruction,
		Details:         run.Details,
		UsedSignIn:      run.UsedSignIn,
		Answer:          run.Answer,
		VisitedURLs:     run.VisitedURLs,
		OffSiteURLs:     run.OffSiteURLs,
		Steps:           run.Steps,
		CostUSD:         run.CostUSD,
		Failure:         run.Failure,
		AgentRunID:      run.AgentRunID,
		ThreadID:        run.ThreadID,
		ProposalID:      run.ProposalID,
		RequestedByID:   run.RequestedByID,
		HasRecording:    run.VendorSessionID != "",
		CodePrompt:      run.CodePrompt,
		CodeExpiresAt:   run.CodeExpiresAt(),
		CodeAttempts:    run.CodeAttempts,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		VendorSessionID: run.VendorSessionID,
	}
}

// RunRecording asks the vendor for the task's recording. Its address is
// signed and short-lived, so it is fetched when someone opens it rather than
// stored.
func (s *Service) RunRecording(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentextensionports.PortalRecordingResponse, error) {
	run, err := s.runs.Get(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	if run.VendorSessionID == "" {
		return nil, errortypes.NewBusinessError("This portal task has no recording")
	}

	extension, err := s.settingsFor(ctx, tenantInfo, run.ExtensionType)
	if err != nil {
		return nil, asBusinessError(err)
	}
	provider, ok := s.browsers[run.ExtensionType]
	if !ok {
		return nil, errortypes.NewBusinessError("the {0} extension cannot browse",
			run.ExtensionType.String())
	}

	recording, err := provider.Recording(ctx, extension.values, run.VendorSessionID)
	if err != nil {
		return nil, asBusinessError(s.providerError(extension.failing(err, "find the recording")))
	}
	if recording.RecordingURL == "" && recording.ShareURL == "" {
		return nil, errortypes.NewBusinessError(
			"The vendor has no recording of this task. Recording may have been off, or the " +
				"video is still being prepared.",
		)
	}

	return &agentextensionports.PortalRecordingResponse{
		RecordingURL: recording.RecordingURL,
		ShareURL:     recording.ShareURL,
	}, nil
}

type siteChange struct {
	tenant    pagination.TenantInfo
	before    *extensionportal.PortalSite
	after     *extensionportal.PortalSite
	operation permission.Operation
	comment   string
}

func (s *Service) logSiteChange(change *siteChange) {
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentExtension,
		ResourceID:     change.after.ID.String(),
		Operation:      change.operation,
		UserID:         change.tenant.UserID,
		CurrentState:   siteSnapshot(change.after),
		PreviousState:  siteSnapshot(change.before),
		BusinessUnitID: change.after.BusinessUnitID,
		OrganizationID: change.after.OrganizationID,
	},
		auditservice.WithComment(change.comment),
		auditservice.WithDiff(siteSnapshot(change.before), siteSnapshot(change.after)),
		auditservice.WithCritical(),
	); err != nil {
		s.l.Error("failed to log portal site change", zap.Error(err))
	}
}

func siteSnapshot(site *extensionportal.PortalSite) map[string]any {
	if site == nil {
		return nil
	}

	return map[string]any{
		"id":               site.ID.String(),
		"name":             site.Name,
		"purpose":          site.Purpose,
		"domains":          site.Domains,
		"startUrl":         site.StartURL,
		"browserProfileId": site.BrowserProfileID,
		"allowActions":     site.AllowActions,
		"enabled":          site.Enabled,
		"hasSignIn":        site.HasSignIn(),
		"version":          site.Version,
	}
}
