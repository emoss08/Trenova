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
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTransactionSetRoutes(transactionSets *gin.RouterGroup) {
	selectOptions := transactionSets.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectTransactionSetOptions,
	)
	selectOptions.GET(
		"/:transactionSetID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTransactionSetOption,
	)
}

func (h *Handler) selectTransactionSetOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDITransactionSet], error) {
			return h.service.SelectTransactionSetOptions(
				c.Request.Context(),
				&repositories.EDITransactionSetSelectOptionsRequest{
					SelectQueryRequest: req,
					Standard:           edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					Status:             edi.DocumentStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getTransactionSetOption(c *gin.Context) {
	transactionSetID := helpers.ParamCatalogID(c, "transactionSetID")
	if transactionSetID.IsNil() {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"transactionSetID",
			errortypes.ErrRequired,
			"Transaction set id is required",
		))
		return
	}

	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	req.Query = ""
	req.Pagination = pagination.Info{Limit: 1}

	result, err := h.service.SelectTransactionSetOptions(
		c.Request.Context(),
		&repositories.EDITransactionSetSelectOptionsRequest{
			SelectQueryRequest: req,
			IDs:                []pulid.ID{transactionSetID},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(result.Items) == 0 {
		h.eh.HandleError(c, errortypes.NewNotFoundError("EDI transaction set not found"))
		return
	}

	c.JSON(http.StatusOK, result.Items[0])
}
