package edisetuphandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerDocumentTypeRoutes(documentTypes *gin.RouterGroup) {
	selectOptions := documentTypes.Group("/select-options", responseguard.References())
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
	entities, err := h.setup.ListDocumentTypes(
		c.Request.Context(),
		edipartnerports.ListEDIDocumentTypesRequest{
			Standard: edidocument.EDIStandard(helpers.QueryString(c, "standard", "")),
			TransactionSet: edidocument.TransactionSet(
				helpers.QueryString(c, "transactionSet", ""),
			),
			Direction: edidocument.DocumentDirection(helpers.QueryString(c, "direction", "")),
			Status:    edidocument.DocumentStatus(helpers.QueryString(c, "status", "")),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, entities)
}

func (h *Handler) selectDocumentTypeOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edidocument.EDIDocumentType], error) {
			return h.setup.SelectDocumentTypeOptions(
				c.Request.Context(),
				&edipartnerports.EDIDocumentTypeSelectOptionsRequest{
					SelectQueryRequest: req,
					Standard: edidocument.EDIStandard(
						helpers.QueryString(c, "standard", ""),
					),
					TransactionSet: edidocument.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edidocument.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status: edidocument.DocumentStatus(helpers.QueryString(c, "status", "")),
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

	entities, err := h.setup.ListDocumentTypes(
		c.Request.Context(),
		edipartnerports.ListEDIDocumentTypesRequest{},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	for _, entity := range entities {
		if entity.ID == documentTypeID {
			helpers.JSON(c, http.StatusOK, entity)
			return
		}
	}
	h.eh.HandleError(c, errortypes.NewNotFoundError("EDI document type not found"))
}
