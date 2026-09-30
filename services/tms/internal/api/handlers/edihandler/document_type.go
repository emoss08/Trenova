package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerDocumentTypeRoutes(documentTypes *gin.RouterGroup) {
	selectOptions := documentTypes.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectDocumentTypeOptions,
	)
	selectOptions.GET(
		"/:documentTypeID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getDocumentTypeOption,
	)
	documentTypes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listDocumentTypes,
	)
}

func (h *Handler) listDocumentTypes(c *gin.Context) {
	entities, err := h.service.ListDocumentTypes(
		c.Request.Context(),
		repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
			TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
			Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
			Status:         edi.DocumentStatus(helpers.QueryString(c, "status", "")),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entities)
}

func (h *Handler) selectDocumentTypeOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIDocumentType], error) {
			return h.service.SelectDocumentTypeOptions(
				c.Request.Context(),
				&repositories.EDIDocumentTypeSelectOptionsRequest{
					SelectQueryRequest: req,
					Standard:           edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getDocumentTypeOption(c *gin.Context) {
	documentTypeID := helpers.ParamCatalogID(c, "documentTypeID")
	if documentTypeID.IsNil() {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"documentTypeID",
			errortypes.ErrRequired,
			"Document type id is required",
		))
		return
	}

	entities, err := h.service.ListDocumentTypes(
		c.Request.Context(),
		repositories.ListEDIDocumentTypesRequest{},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	for _, entity := range entities {
		if entity.ID == documentTypeID {
			c.JSON(http.StatusOK, entity)
			return
		}
	}
	h.eh.HandleError(c, errortypes.NewNotFoundError("EDI document type not found"))
}
