package servicefailurehandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              servicefailureports.ServiceFailureService
	EDIPayloads          ediports.ServiceFailurePayloadBuilder
	EDIServiceFailure214 ediports.ServiceFailure214Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service     servicefailureports.ServiceFailureService
	ediPayloads ediports.ServiceFailurePayloadBuilder
	edi214      ediports.ServiceFailure214Service
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		ediPayloads: p.EDIPayloads,
		edi214:      p.EDIServiceFailure214,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/service-failures")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpCreate),
		h.createManual,
	)
	api.POST(
		"/evaluate-shipment/:shipmentID/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpCreate),
		h.evaluateShipment,
	)
	api.POST(
		"/evaluate-stop/:shipmentID/:stopID/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpCreate),
		h.evaluateStop,
	)
	api.POST(
		"/bulk-evaluate/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpCreate),
		h.bulkEvaluate,
	)
	api.GET(
		"/:serviceFailureID/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/:serviceFailureID/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:serviceFailureID/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpUpdate),
		h.update,
	)
	api.POST(
		"/:serviceFailureID/review/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpApprove),
		h.review,
	)
	api.POST(
		"/:serviceFailureID/resolve/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpUpdate),
		h.resolve,
	)
	api.POST(
		"/:serviceFailureID/void/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpArchive),
		h.void,
	)
	api.POST(
		"/:serviceFailureID/edi-214-payload/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpExport),
		h.buildEDI214Payload,
	)
	api.GET(
		"/:serviceFailureID/edi-214-readiness/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpRead),
		h.edi214Readiness,
	)
	api.GET(
		"/:serviceFailureID/edi-214-status/",
		h.pm.RequirePermission(permission.ResourceServiceFailure.String(), permission.OpRead),
		h.edi214Status,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	shipmentID, _ := pulid.MustParse(c.Query("shipmentId"))

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
			return h.service.List(
				c.Request.Context(),
				&servicefailureports.ListServiceFailuresRequest{
					Filter:     req,
					ShipmentID: shipmentID,
				},
			)
		},
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&servicefailureports.GetServiceFailureByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, entity)
}

func (h *Handler) createManual(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	_, err := h.service.CreateManual(
		c.Request.Context(),
		&servicefailureports.CreateManualServiceFailureRequest{
			TenantInfo: pagination.FromAuth(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
}

func (h *Handler) evaluateShipment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.EvaluateShipment(
		c.Request.Context(),
		&servicefailureports.EvaluateShipmentServiceFailuresRequest{
			TenantInfo: pagination.FromAuth(authCtx),
			ShipmentID: shipmentID,
			Force:      c.Query("force") == "true",
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) evaluateStop(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	stopID, err := pulid.MustParse(c.Param("stopID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	moveID, _ := pulid.MustParse(c.Query("shipmentMoveId"))
	result, err := h.service.EvaluateStop(
		c.Request.Context(),
		&servicefailureports.EvaluateStopServiceFailuresRequest{
			TenantInfo:     pagination.FromAuth(authCtx),
			ShipmentID:     shipmentID,
			ShipmentMoveID: moveID,
			StopID:         stopID,
			Force:          c.Query("force") == "true",
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) bulkEvaluate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(servicefailureports.BulkEvaluateServiceFailuresRequest)
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.BulkEvaluate(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(servicefailureports.UpdateServiceFailureRequest)
	req.ID = id
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if req.ShipmentID.IsNil() {
		current, getErr := h.service.GetByID(
			c.Request.Context(),
			&servicefailureports.GetServiceFailureByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			},
		)
		if getErr != nil {
			h.eh.HandleError(c, getErr)
			return
		}
		req.ShipmentID = current.ShipmentID
	}
	updated, err := h.service.Update(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, updated)
}

func (h *Handler) review(c *gin.Context) {
	h.lifecycle(
		c,
		func(ctx context.Context, req *servicefailureports.ServiceFailureLifecycleRequest, actor *permissionports.RequestActor) (*servicefailure.ServiceFailure, error) {
			return h.service.Review(ctx, req, actor)
		},
	)
}

func (h *Handler) resolve(c *gin.Context) {
	h.lifecycle(
		c,
		func(ctx context.Context, req *servicefailureports.ServiceFailureLifecycleRequest, actor *permissionports.RequestActor) (*servicefailure.ServiceFailure, error) {
			return h.service.Resolve(ctx, req, actor)
		},
	)
}

func (h *Handler) void(c *gin.Context) {
	h.lifecycle(
		c,
		func(ctx context.Context, req *servicefailureports.ServiceFailureLifecycleRequest, actor *permissionports.RequestActor) (*servicefailure.ServiceFailure, error) {
			return h.service.Void(ctx, req, actor)
		},
	)
}

type lifecycleFn func(context.Context, *servicefailureports.ServiceFailureLifecycleRequest, *permissionports.RequestActor) (*servicefailure.ServiceFailure, error)

func (h *Handler) lifecycle(c *gin.Context, fn lifecycleFn) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(servicefailureports.ServiceFailureLifecycleRequest)
	req.ID = id
	req.TenantInfo = pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if req.ShipmentID.IsNil() {
		current, getErr := h.service.GetByID(
			c.Request.Context(),
			&servicefailureports.GetServiceFailureByIDRequest{
				ID:         id,
				TenantInfo: req.TenantInfo,
			},
		)
		if getErr != nil {
			h.eh.HandleError(c, getErr)
			return
		}
		req.ShipmentID = current.ShipmentID
	}
	entity, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, entity)
}

func (h *Handler) buildEDI214Payload(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.ediPayloads.BuildShipmentStatusPayloadForServiceFailure(
		c.Request.Context(),
		&servicefailureports.BuildServiceFailureEDIPayloadRequest{
			TenantInfo:       pagination.FromAuth(authCtx),
			ServiceFailureID: id,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) edi214Readiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
	current, err := h.service.GetByID(
		c.Request.Context(),
		&servicefailureports.GetServiceFailureByIDRequest{
			ID:         id,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	trigger, nextStatus := readinessTrigger(c.Query("trigger"), current.Status)
	preview := *current
	preview.Status = nextStatus
	result, err := h.edi214.PreviewServiceFailure214ForLifecycle(
		c.Request.Context(),
		&ediports.ServiceFailure214LifecycleRequest{
			TenantInfo:       tenantInfo,
			ServiceFailureID: current.ID,
			ShipmentID:       current.ShipmentID,
			Trigger:          trigger,
			PreviousStatus:   current.Status,
			NewStatus:        nextStatus,
			ServiceFailure:   &preview,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) edi214Status(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("serviceFailureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	status, err := h.edi214.GetServiceFailure214Status(
		c.Request.Context(),
		edimessageports.GetServiceFailure214StatusRequest{
			TenantInfo:       pagination.FromAuth(authCtx),
			ServiceFailureID: id,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, status)
}

func readinessTrigger(
	value string,
	status servicefailure.Status,
) (servicefailureports.ServiceFailureEDITrigger, servicefailure.Status) {
	switch value {
	case string(servicefailureports.ServiceFailureEDITriggerResolved):
		return servicefailureports.ServiceFailureEDITriggerResolved, servicefailure.StatusResolved
	case string(servicefailureports.ServiceFailureEDITriggerReviewed):
		return servicefailureports.ServiceFailureEDITriggerReviewed, servicefailure.StatusReviewed
	default:
		if status == servicefailure.StatusOpen {
			return servicefailureports.ServiceFailureEDITriggerReviewed, servicefailure.StatusReviewed
		}
		return servicefailureports.ServiceFailureEDITriggerResolved, servicefailure.StatusResolved
	}
}
