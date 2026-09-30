package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerPartnerSettingsRoutes(partnerSettings *gin.RouterGroup) {
	partnerSettings.GET(
		"/schemas/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerSettingSchemas,
	)
	partnerSettings.GET(
		"/schemas/:schemaID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerSettingSchema,
	)
	partnerSettings.GET(
		"/schemas/:schemaID/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerSettingSchemaFields,
	)
	partnerSettings.GET(
		"/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.searchPartnerSettingFields,
	)
	partnerSettings.GET(
		"/fields/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerSettingFieldOptions,
	)
	partnerSettings.POST(
		"/validate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.validatePartnerSettings,
	)
}

func (h *Handler) listPartnerSettingSchemas(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
			return h.service.ListPartnerSettingSchemas(
				c.Request.Context(),
				&repositories.ListEDIPartnerSettingSchemasRequest{
					Filter:   req,
					Standard: edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
					X12Version:     helpers.QueryString(c, "x12Version", ""),
					DocumentTypeID: helpers.QueryCatalogID(c, "documentTypeId"),
					SchemaVersion:  helpers.QueryInt64(c, "schemaVersion", 0),
					Status:         edi.PartnerSettingStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getPartnerSettingSchema(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	schema, err := h.service.GetPartnerSettingSchema(
		c.Request.Context(),
		repositories.GetEDIPartnerSettingSchemaRequest{
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

func (h *Handler) listPartnerSettingSchemaFields(c *gin.Context) {
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
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			fieldReq := h.partnerSettingFieldRequest(c, req)
			fieldReq.SchemaID = schemaID
			return h.service.ListPartnerSettingFields(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) searchPartnerSettingFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			return h.service.SearchPartnerSettingFields(
				c.Request.Context(),
				h.partnerSettingFieldRequest(c, req),
			)
		},
	)
}

func (h *Handler) selectPartnerSettingFieldOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	fieldReq := h.partnerSettingFieldRequest(c, queryOptionsFromSelect(req))

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			return h.service.SelectPartnerSettingFieldOptions(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) partnerSettingFieldRequest(
	c *gin.Context,
	req *pagination.QueryOptions,
) *repositories.ListEDIPartnerSettingFieldsRequest {
	fieldReq := &repositories.ListEDIPartnerSettingFieldsRequest{
		Filter:         req,
		Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
		TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
		Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
		Status:         edi.PartnerSettingStatus(helpers.QueryString(c, "status", "")),
		PathPrefix:     helpers.QueryString(c, "pathPrefix", ""),
		GroupKey:       helpers.QueryString(c, "groupKey", ""),
	}
	if _, ok := c.GetQuery("required"); ok {
		required := helpers.QueryBool(c, "required", false)
		fieldReq.Required = &required
	}
	if _, ok := c.GetQuery("secret"); ok {
		secret := helpers.QueryBool(c, "secret", false)
		fieldReq.Secret = &secret
	}
	if schemaID := helpers.QueryPulid(c, "schemaId"); schemaID.IsNotNil() {
		fieldReq.SchemaID = schemaID
	}
	return fieldReq
}

func (h *Handler) validatePartnerSettings(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.ValidatePartnerSettingsRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	diagnostics, err := h.service.ValidatePartnerSettings(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"diagnostics": diagnostics})
}
