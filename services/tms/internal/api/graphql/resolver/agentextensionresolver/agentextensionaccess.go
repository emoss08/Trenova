package agentextensionresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	maxPortalRunsPerPage = 200
	maxWebWatchesPerPage = 100
)

func (d *Deps) requireAdministersOr(
	ctx context.Context,
	administer permission.Operation,
	own permission.Resource,
) (*authctx.AuthContext, bool, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, false, errortypes.NewAuthenticationError("Authentication required")
	}

	administers := d.HasPermission(ctx, authCtx, permission.ResourceAgentExtension, administer)
	if !administers && !d.HasPermission(ctx, authCtx, own, permission.OpRead) {
		return nil, false, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}",
			permission.ResourceAgentExtension, administer,
		)
	}

	return authCtx, administers, nil
}

func (d *Deps) portalRunAccess(
	ctx context.Context,
	id string,
	administer permission.Operation,
) (*agentextensionports.PortalRunAccessRequest, error) {
	authCtx, administers, err := d.requireAdministersOr(
		ctx,
		administer,
		permission.ResourcePortalAutomation,
	)
	if err != nil {
		return nil, err
	}

	runID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.PortalRunAccessRequest{
		TenantInfo:  resolvercore.TenantInfo(authCtx),
		RunID:       runID,
		Administers: administers,
	}, nil
}

func portalRunsRequest(
	tenantInfo pagination.TenantInfo,
	siteID *string,
	before *int,
	limit *int,
) (*agentextensionports.ListPortalRunsRequest, error) {
	site, err := gqlid.ParseOptionalIDField("siteId", siteID)
	if err != nil {
		return nil, err
	}
	since, err := beforeOf(before)
	if err != nil {
		return nil, err
	}
	size, err := limitOf(limit, maxPortalRunsPerPage)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.ListPortalRunsRequest{
		TenantInfo: tenantInfo,
		SiteID:     site,
		Before:     since,
		Limit:      size,
	}, nil
}

func webWatchesRequest(
	tenantInfo pagination.TenantInfo,
	status *string,
	before *int,
	limit *int,
) (*agentextensionports.ListWebWatchesRequest, error) {
	since, err := beforeOf(before)
	if err != nil {
		return nil, err
	}
	size, err := limitOf(limit, maxWebWatchesPerPage)
	if err != nil {
		return nil, err
	}

	req := &agentextensionports.ListWebWatchesRequest{
		TenantInfo: tenantInfo,
		Before:     since,
		Limit:      size,
	}
	if status != nil && *status != "" {
		watchStatus := extensionwebwatch.WebWatchStatus(*status)
		if !watchStatus.IsValid() {
			return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid,
				"status must be Active, Ended, Stopped or Failed")
		}
		req.Status = watchStatus
	}

	return req, nil
}

func beforeOf(before *int) (int64, error) {
	if before == nil {
		return 0, nil
	}
	if *before < 0 {
		return 0, errortypes.NewValidationError("before", errortypes.ErrInvalid,
			"before must be a Unix time in seconds")
	}

	return int64(*before), nil
}

func limitOf(limit *int, most int) (int, error) {
	if limit == nil {
		return 0, nil
	}
	if *limit < 1 || *limit > most {
		return 0, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
			"limit must be a whole number from 1 to {0}", most)
	}

	return *limit, nil
}

func updateConfigRequest(
	tenantInfo pagination.TenantInfo,
	input *gqlmodel.UpdateAgentExtensionConfigInput,
) (*agentextensionports.UpdateAgentExtensionRequest, error) {
	configuration := make(map[string]string, len(input.Configuration))
	for _, setting := range input.Configuration {
		if _, seen := configuration[setting.Key]; seen {
			return nil, errortypes.NewValidationError(
				"configuration", errortypes.ErrDuplicate, "Each setting may be given once",
			)
		}
		configuration[setting.Key] = setting.Value
	}

	return &agentextensionports.UpdateAgentExtensionRequest{
		TenantInfo:    tenantInfo,
		Enabled:       input.Enabled,
		Availability:  agentextension.Availability(input.Availability),
		Configuration: configuration,
		Version:       int64(input.Version),
	}, nil
}

func portalSiteRequest(
	input *gqlmodel.SaveAgentPortalSiteInput,
) *agentextensionports.SavePortalSiteRequest {
	return &agentextensionports.SavePortalSiteRequest{
		Name:               input.Name,
		Purpose:            input.Purpose,
		Domains:            input.Domains,
		StartURL:           input.StartURL,
		Username:           input.Username,
		Password:           input.Password,
		AuthenticatorKey:   input.AuthenticatorKey,
		ClearSignIn:        input.ClearSignIn,
		ClearAuthenticator: input.ClearAuthenticator,
		BrowserProfileID:   input.BrowserProfileID,
		AllowActions:       input.AllowActions,
		Enabled:            input.Enabled,
		Version:            int64(input.Version),
	}
}
