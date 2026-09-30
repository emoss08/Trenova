package edihandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerConnectionRoutes(connections *gin.RouterGroup) {
	connections.GET(
		"/",
		// h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listConnections,
	)
	connections.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createConnection,
	)
	connections.GET(
		"/:connectionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getConnection,
	)
	connections.POST(
		"/:connectionID/accept/",
		// h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.acceptConnection,
	)
	connections.POST(
		"/:connectionID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectConnection,
	)
	connections.POST(
		"/:connectionID/suspend/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.suspendConnection,
	)
	connections.POST(
		"/:connectionID/revoke/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.revokeConnection,
	)
}

func (h *Handler) listConnections(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIConnection], error) {
		return h.service.ListConnections(
			c.Request.Context(),
			&repositories.ListEDIConnectionsRequest{Filter: req},
		)
	})
}

func (h *Handler) createConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateEDIConnectionRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	connection, err := h.service.CreateConnection(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, connection)
}

func (h *Handler) getConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	connectionID, err := pulid.MustParse(c.Param("connectionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	connection, err := h.service.GetConnection(
		c.Request.Context(),
		repositories.GetEDIConnectionByIDRequest{
			ID:         connectionID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, connection)
}

func (h *Handler) acceptConnection(c *gin.Context) {
	h.connectionAction(c, h.service.AcceptConnection)
}

func (h *Handler) rejectConnection(c *gin.Context) {
	h.connectionAction(c, h.service.RejectConnection)
}

func (h *Handler) suspendConnection(c *gin.Context) {
	h.connectionAction(c, h.service.SuspendConnection)
}

func (h *Handler) revokeConnection(c *gin.Context) {
	h.connectionAction(c, h.service.RevokeConnection)
}

func (h *Handler) connectionAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.EDIConnectionActionRequest,
		*services.RequestActor,
	) (*edi.EDIConnection, error),
) {
	authCtx := authctx.GetAuthContext(c)
	connectionID, err := pulid.MustParse(c.Param("connectionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.EDIConnectionActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ConnectionID = connectionID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	connection, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, connection)
}
