package edihandler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const ladingListLimit = 100

func (h *Handler) registerLadingRoutes(lading *gin.RouterGroup) {
	read := h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead)
	create := h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate)
	update := h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate)

	lading.GET("/catalog/", read, h.ladingCatalog)

	projects := lading.Group("/projects")
	projects.POST("/", create, h.createLadingProject)
	projects.GET("/by-partner/:partnerID/", read, h.getPartnerLadingProject)
	projects.GET("/:projectID/", read, h.getLadingProject)
	projects.GET("/:projectID/versions/", read, h.listLadingVersions)
	projects.POST("/:projectID/drafts/", create, h.newLadingDraft)
	projects.PUT("/:projectID/settings/", update, h.updateLadingSettings)

	versions := lading.Group("/versions")
	versions.GET("/:versionID/", read, h.getLadingVersion)
	versions.PUT("/:versionID/", update, h.saveLadingDraft)
	versions.POST("/:versionID/run/", read, h.runLadingVersion)
	versions.POST("/:versionID/test/", update, h.testLadingVersion)
	versions.POST("/:versionID/certify/", update, h.certifyLadingVersion)
	versions.POST("/:versionID/activate/", update, h.activateLadingVersion)
	versions.POST("/:versionID/archive/", update, h.archiveLadingVersion)
	versions.POST("/:versionID/shadow/", update, h.startLadingShadow)
	versions.DELETE("/:versionID/shadow/", update, h.stopLadingShadow)
	versions.GET("/:versionID/shadow-results/", read, h.ladingShadowResults)
	versions.POST("/:versionID/replays/", update, h.startLadingReplay)
	versions.GET("/:versionID/replays/", read, h.listLadingReplays)

	lading.GET("/replays/:replayID/", read, h.getLadingReplay)
}

func (h *Handler) registerFlowRoutes(flows *gin.RouterGroup) {
	read := h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead)
	update := h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate)

	flows.GET("/instances/", read, h.listFlowInstances)
	flows.GET("/instances/:instanceID/", read, h.getFlowInstance)
	flows.POST("/instances/:instanceID/release/", update, h.releaseFlowInstance)
}

func (h *Handler) ladingCatalog(c *gin.Context) {
	catalog, err := h.ladingRuntime.Client().Catalog(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", catalog)
}

func (h *Handler) createLadingProject(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(edilading.CreateProjectRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	project, err := h.projects.CreateProject(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusCreated, project)
}

func (h *Handler) getPartnerLadingProject(c *gin.Context) {
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	project, err := h.projects.GetProject(c.Request.Context(), ediports.GetLadingProjectRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		PartnerID:  partnerID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, project)
}

func (h *Handler) getLadingProject(c *gin.Context) {
	projectID, err := pulid.MustParse(c.Param("projectID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	project, err := h.projects.GetProject(c.Request.Context(), ediports.GetLadingProjectRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		ID:         projectID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, project)
}

func (h *Handler) listLadingVersions(c *gin.Context) {
	projectID, err := pulid.MustParse(c.Param("projectID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	versions, err := h.projects.ListVersions(
		c.Request.Context(),
		ediports.ListLadingVersionsRequest{
			TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
			ProjectID:  projectID,
			Limit:      queryLimit(c),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, gin.H{"items": versions})
}

func (h *Handler) newLadingDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	projectID, err := pulid.MustParse(c.Param("projectID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.NewDraftRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ProjectID = projectID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.projects.NewDraft(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusCreated, version)
}

func (h *Handler) updateLadingSettings(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	projectID, err := pulid.MustParse(c.Param("projectID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.UpdateSettingsRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ProjectID = projectID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	project, err := h.projects.UpdateSettings(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, project)
}

func (h *Handler) getLadingVersion(c *gin.Context) {
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	detail, err := h.projects.GetVersion(c.Request.Context(), ediports.GetLadingVersionRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		ID:         versionID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, detail)
}

func (h *Handler) saveLadingDraft(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.SaveDraftRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.projects.SaveDraft(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, version)
}

func (h *Handler) runLadingVersion(c *gin.Context) {
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.RunVersionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authctx.GetAuthContext(c))
	result, err := h.projects.Run(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) testLadingVersion(c *gin.Context) {
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	report, err := h.projects.Test(c.Request.Context(), &edilading.VersionRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		VersionID:  versionID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, report)
}

type versionAction func(
	context.Context,
	*edilading.VersionRequest,
	*permissionports.RequestActor,
) (*edi.LadingVersion, error)

func (h *Handler) ladingVersionAction(c *gin.Context, action versionAction) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := action(c.Request.Context(), &edilading.VersionRequest{
		TenantInfo: tenantInfoFromAuth(authCtx),
		VersionID:  versionID,
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, version)
}

func (h *Handler) certifyLadingVersion(c *gin.Context) {
	h.ladingVersionAction(c, h.projects.Certify)
}

func (h *Handler) activateLadingVersion(c *gin.Context) {
	h.ladingVersionAction(c, h.projects.Activate)
}

func (h *Handler) archiveLadingVersion(c *gin.Context) {
	h.ladingVersionAction(c, h.projects.Archive)
}

func (h *Handler) stopLadingShadow(c *gin.Context) {
	h.ladingVersionAction(c, h.projects.StopShadow)
}

func (h *Handler) startLadingShadow(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.ShadowRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.projects.StartShadow(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, version)
}

func (h *Handler) ladingShadowResults(c *gin.Context) {
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	report, err := h.projects.ShadowResults(
		c.Request.Context(),
		ediports.ListLadingShadowResultsRequest{
			TenantInfo:    tenantInfoFromAuth(authctx.GetAuthContext(c)),
			VersionID:     versionID,
			OnlyDiffering: helpers.QueryBool(c, "onlyDiffering"),
			Limit:         queryLimit(c),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, report)
}

func (h *Handler) startLadingReplay(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(edilading.StartReplayRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	replay, err := h.projects.StartReplay(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusAccepted, replay)
}

func (h *Handler) listLadingReplays(c *gin.Context) {
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	replays, err := h.projects.ListReplays(c.Request.Context(), ediports.ListLadingReplaysRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		VersionID:  versionID,
		Limit:      queryLimit(c),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, gin.H{"items": replays})
}

func (h *Handler) getLadingReplay(c *gin.Context) {
	replayID, err := pulid.MustParse(c.Param("replayID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	replay, err := h.projects.GetReplay(c.Request.Context(), ediports.GetLadingReplayRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		ID:         replayID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, replay)
}

func (h *Handler) listFlowInstances(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	statuses := make([]edi.FlowInstanceStatus, 0, 2)
	for _, status := range c.QueryArray("status") {
		statuses = append(statuses, edi.FlowInstanceStatus(status))
	}
	partnerID := pulid.ID(helpers.QueryString(c, "partnerId", ""))
	shipmentID := pulid.ID(helpers.QueryString(c, "shipmentId", ""))

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.FlowInstance], error) {
		return h.service.ListFlowInstances(c.Request.Context(), &ediports.ListFlowInstancesRequest{
			Filter:     req,
			PartnerID:  partnerID,
			ShipmentID: shipmentID,
			Statuses:   statuses,
		})
	})
}

func (h *Handler) getFlowInstance(c *gin.Context) {
	instanceID, err := pulid.MustParse(c.Param("instanceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	detail, err := h.service.GetFlowInstance(c.Request.Context(), ediports.GetFlowInstanceRequest{
		TenantInfo: tenantInfoFromAuth(authctx.GetAuthContext(c)),
		ID:         instanceID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, detail)
}

func (h *Handler) releaseFlowInstance(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	instanceID, err := pulid.MustParse(c.Param("instanceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	released, err := h.service.ReleaseFlowInstance(
		c.Request.Context(),
		ediports.GetFlowInstanceRequest{TenantInfo: tenantInfoFromAuth(authCtx), ID: instanceID},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, released)
}

func queryLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 || limit > ladingListLimit {
		return ladingListLimit
	}
	return limit
}
