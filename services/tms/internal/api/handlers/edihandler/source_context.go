package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerSourceContextRoutes(sourceContext *gin.RouterGroup) {
	sourceContext.GET(
		"/schemas/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listSourceContextSchemas,
	)
	sourceContext.GET(
		"/schemas/:schemaID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getSourceContextSchema,
	)
	sourceContext.GET(
		"/schemas/:schemaID/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listSourceContextSchemaFields,
	)
	sourceContext.GET(
		"/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.searchSourceContextFields,
	)
	sourceContext.GET(
		"/fields/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectSourceContextFieldOptions,
	)
}

func (h *Handler) listSourceContextSchemas(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
			return h.service.ListSourceContextSchemas(
				c.Request.Context(),
				&repositories.ListEDISourceContextSchemasRequest{
					Filter:   req,
					Standard: edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					X12Version:    helpers.QueryString(c, "x12Version", ""),
					ContextKey:    helpers.QueryString(c, "contextKey", ""),
					SchemaVersion: helpers.QueryInt64(c, "schemaVersion", 0),
					Status: edi.SourceContextFieldStatus(
						helpers.QueryString(c, "status", ""),
					),
				},
			)
		},
	)
}

func (h *Handler) getSourceContextSchema(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	schema, err := h.service.GetSourceContextSchema(
		c.Request.Context(),
		repositories.GetEDISourceContextSchemaRequest{
			ID:         schemaID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, schema)
}

func (h *Handler) listSourceContextSchemaFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			fieldReq := h.sourceContextFieldRequest(c, req)
			fieldReq.SchemaID = schemaID
			return h.service.ListSourceContextFields(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) searchSourceContextFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			return h.service.SearchSourceContextFields(
				c.Request.Context(),
				h.sourceContextFieldRequest(c, req),
			)
		},
	)
}

func (h *Handler) selectSourceContextFieldOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	fieldReq := h.sourceContextFieldRequest(c, queryOptionsFromSelect(req))

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			return h.service.SelectSourceContextFieldOptions(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) sourceContextFieldRequest(
	c *gin.Context,
	req *pagination.QueryOptions,
) *repositories.ListEDISourceContextFieldsRequest {
	fieldReq := &repositories.ListEDISourceContextFieldsRequest{
		Filter: req,
		Status: edi.SourceContextFieldStatus(
			helpers.QueryString(c, "status", ""),
		),
		Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
		TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
		Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
		SourceKind:     edi.SourceContextKind(helpers.QueryString(c, "sourceKind", "")),
		PathPrefix:     helpers.QueryString(c, "pathPrefix", ""),
	}
	if _, ok := c.GetQuery("repeated"); ok {
		repeated := helpers.QueryBool(c, "repeated", false)
		fieldReq.Repeated = &repeated
	}
	if schemaID := helpers.QueryPulid(c, "schemaId"); schemaID.IsNotNil() {
		fieldReq.SchemaID = schemaID
	}
	return fieldReq
}
