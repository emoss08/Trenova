package agentextensionresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (r *AgentExtensionUsageResolver) CostThisMonthUsd(ctx context.Context, obj *agentextension.UsageSummary) (string, error) {
	return obj.CostThisMonthUSD.String(), nil
}

func (r *AgentPortalRunResolver) Details(ctx context.Context, obj *agentextensionports.PortalRunView) (map[string]any, error) {
	details := make(map[string]any, len(obj.Details))
	for key, value := range obj.Details {
		details[key] = value
	}

	return details, nil
}

func (r *AgentPortalRunResolver) CostUsd(ctx context.Context, obj *agentextensionports.PortalRunView) (string, error) {
	return obj.CostUSD.String(), nil
}

func (r *AgentWebWatchResolver) CostUsd(ctx context.Context, obj *agentextensionports.WebWatchView) (string, error) {
	return obj.CostUSD.String(), nil
}

func (r *QueryResolver) AgentExtensionCatalog(ctx context.Context) (*agentextensionports.AgentExtensionCatalogResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.AgentExtensionService.ListCatalog(ctx, resolvercore.TenantInfo(authCtx))
}

func (r *QueryResolver) AgentExtensionConfig(ctx context.Context, typeArg string) (*agentextensionports.AgentExtensionConfigResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.AgentExtensionService.GetConfig(
		ctx,
		resolvercore.TenantInfo(authCtx),
		agentextension.Type(typeArg),
	)
}

func (r *QueryResolver) AgentPortalSites(ctx context.Context) ([]*agentextensionports.PortalSiteView, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	sites, err := r.AgentPortalService.ListSites(ctx, resolvercore.TenantInfo(authCtx))
	if err != nil {
		return nil, err
	}

	return sliceutils.Pointers(sites), nil
}

func (r *QueryResolver) AgentPortalRuns(ctx context.Context, siteID *string, before *int, limit *int) (*agentextensionports.PortalRunListResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	req, err := portalRunsRequest(resolvercore.TenantInfo(authCtx), siteID, before, limit)
	if err != nil {
		return nil, err
	}

	return r.AgentPortalService.ListRuns(ctx, req)
}

func (r *QueryResolver) AgentPortalRun(ctx context.Context, id string) (*agentextensionports.PortalRunView, error) {
	req, err := r.portalRunAccess(ctx, id, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.AgentPortalService.PortalRunFor(ctx, req)
}

func (r *QueryResolver) AgentPortalRunLive(ctx context.Context, id string) (*agentextensionports.PortalLiveView, error) {
	req, err := r.portalRunAccess(ctx, id, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	return r.AgentPortalService.PortalRunLive(ctx, req)
}

func (r *QueryResolver) AgentPortalRunRecording(ctx context.Context, id string) (*agentextensionports.PortalRecordingResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	runID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.AgentPortalService.RunRecording(ctx, resolvercore.TenantInfo(authCtx), runID)
}

func (r *QueryResolver) AgentWebWatches(ctx context.Context, status *string, before *int, limit *int) (*agentextensionports.WebWatchListResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	req, err := webWatchesRequest(resolvercore.TenantInfo(authCtx), status, before, limit)
	if err != nil {
		return nil, err
	}

	return r.AgentWebWatchService.ListWebWatches(ctx, req)
}

func (r *QueryResolver) AgentWebWatchFindings(ctx context.Context, id string) ([]*agentextensionports.WebWatchFindingView, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpRead)
	if err != nil {
		return nil, err
	}

	watchID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	findings, err := r.AgentWebWatchService.WebWatchFindings(
		ctx,
		resolvercore.TenantInfo(authCtx),
		watchID,
	)
	if err != nil {
		return nil, err
	}

	return sliceutils.Pointers(findings.Items), nil
}

func (r *MutationResolver) UpdateAgentExtensionConfig(ctx context.Context, typeArg string, input gqlmodel.UpdateAgentExtensionConfigInput) (*agentextensionports.AgentExtensionConfigResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	req, err := updateConfigRequest(resolvercore.TenantInfo(authCtx), &input)
	if err != nil {
		return nil, err
	}

	return r.AgentExtensionService.UpdateConfig(ctx, agentextension.Type(typeArg), req)
}

func (r *MutationResolver) TestAgentExtension(ctx context.Context, typeArg string) (*agentextensionports.AgentExtensionTestResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	return r.AgentExtensionService.TestConnection(
		ctx,
		resolvercore.TenantInfo(authCtx),
		agentextension.Type(typeArg),
	)
}

func (r *MutationResolver) CreateAgentPortalSite(ctx context.Context, input gqlmodel.SaveAgentPortalSiteInput) (*agentextensionports.PortalSiteView, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	req := portalSiteRequest(&input)
	req.TenantInfo = resolvercore.TenantInfo(authCtx)

	return r.AgentPortalService.CreateSite(ctx, req)
}

func (r *MutationResolver) UpdateAgentPortalSite(ctx context.Context, id string, input gqlmodel.SaveAgentPortalSiteInput) (*agentextensionports.PortalSiteView, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	siteID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	req := portalSiteRequest(&input)
	req.TenantInfo = resolvercore.TenantInfo(authCtx)
	req.ID = siteID

	return r.AgentPortalService.UpdateSite(ctx, req)
}

func (r *MutationResolver) DeleteAgentPortalSite(ctx context.Context, id string) (bool, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentExtension, permission.OpUpdate)
	if err != nil {
		return false, err
	}

	siteID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return false, err
	}

	if err = r.AgentPortalService.DeleteSite(ctx, resolvercore.TenantInfo(authCtx), siteID); err != nil {
		return false, err
	}

	return true, nil
}

func (r *MutationResolver) ProvideAgentPortalSignInCode(ctx context.Context, id string, input gqlmodel.AgentPortalSignInCodeInput) (*agentextensionports.PortalRunView, error) {
	access, err := r.portalRunAccess(ctx, id, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	return r.AgentPortalService.ProvidePortalCode(ctx, &agentextensionports.ProvidePortalCodeRequest{
		TenantInfo:  access.TenantInfo,
		RunID:       access.RunID,
		Code:        input.Code,
		SignedIn:    input.SignedIn,
		Administers: access.Administers,
	})
}

func (r *MutationResolver) StopAgentWebWatch(ctx context.Context, id string) (*agentextensionports.WebWatchView, error) {
	authCtx, administers, err := r.requireAdministersOr(
		ctx,
		permission.OpUpdate,
		permission.ResourceWebResearch,
	)
	if err != nil {
		return nil, err
	}

	watchID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	tenant := resolvercore.TenantInfo(authCtx)
	watch, err := r.AgentWebWatchService.StopWebWatch(ctx, &agentextensionports.StopWebWatchRequest{
		TenantInfo:  tenant,
		WatchID:     watchID,
		StoppedBy:   tenant.UserID,
		Administers: administers,
	})
	if err != nil {
		return nil, err
	}
	if r.WebWatchWaits != nil {
		r.WebWatchWaits.EndWebWatchWaits(ctx, tenant, &extensionwebwatch.WebWatch{
			ID:     watch.ID,
			Status: watch.Status,
		})
	}

	return watch, nil
}

func (r *AgentExtensionCatalogItemResolver) Type(ctx context.Context, obj *agentextensionports.AgentExtensionCatalogItem) (string, error) {
	return string(obj.Type), nil
}

func (r *AgentExtensionCatalogItemResolver) Category(ctx context.Context, obj *agentextensionports.AgentExtensionCatalogItem) (string, error) {
	return string(obj.Category), nil
}

func (r *AgentExtensionCatalogItemResolver) Capability(ctx context.Context, obj *agentextensionports.AgentExtensionCatalogItem) (string, error) {
	return string(obj.Capability), nil
}

func (r *AgentExtensionCatalogItemResolver) Availability(ctx context.Context, obj *agentextensionports.AgentExtensionCatalogItem) (string, error) {
	return string(obj.Availability), nil
}

func (r *AgentExtensionCategoryOptionResolver) Value(ctx context.Context, obj *agentextensionports.AgentExtensionCategoryOption) (string, error) {
	return string(obj.Value), nil
}

func (r *AgentExtensionConfigFieldResolver) Type(ctx context.Context, obj *configspec.Field) (string, error) {
	return string(obj.Type), nil
}

func (r *AgentExtensionConfigResolver) Type(ctx context.Context, obj *agentextensionports.AgentExtensionConfigResponse) (string, error) {
	return string(obj.Type), nil
}

func (r *AgentExtensionConfigResolver) Availability(ctx context.Context, obj *agentextensionports.AgentExtensionConfigResponse) (string, error) {
	return string(obj.Availability), nil
}

func (r *AgentExtensionTestResultResolver) Type(ctx context.Context, obj *agentextensionports.AgentExtensionTestResponse) (string, error) {
	return string(obj.Type), nil
}

func (r *AgentPortalRunResolver) ExtensionType(ctx context.Context, obj *agentextensionports.PortalRunView) (string, error) {
	return string(obj.ExtensionType), nil
}

func (r *AgentPortalRunResolver) Mode(ctx context.Context, obj *agentextensionports.PortalRunView) (string, error) {
	return string(obj.Mode), nil
}

func (r *AgentPortalRunResolver) Status(ctx context.Context, obj *agentextensionports.PortalRunView) (string, error) {
	return string(obj.Status), nil
}

func (r *AgentWebWatchResolver) ExtensionType(ctx context.Context, obj *agentextensionports.WebWatchView) (string, error) {
	return string(obj.ExtensionType), nil
}

func (r *AgentWebWatchResolver) Cadence(ctx context.Context, obj *agentextensionports.WebWatchView) (string, error) {
	return string(obj.Cadence), nil
}

func (r *AgentWebWatchResolver) Status(ctx context.Context, obj *agentextensionports.WebWatchView) (string, error) {
	return string(obj.Status), nil
}
