package formulatemplatehandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type previewRequest struct {
	Source     string             `json:"source"`
	Inputs     tarifftypes.Inputs `json:"inputs"`
	ShipmentID string             `json:"shipmentId"`
}

// @Summary Preview a formula's source
// @Description Checks the source, runs its tests, and rates it on the sample record or a real shipment. A source with errors returns its diagnostics and tests without a bill.
// @ID previewFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body previewRequest true "The source, the preview's input values and an optional shipment"
// @Success 200 {object} formulatemplateports.PreviewResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/test [post]
func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	shipmentID, ok := h.optionalShipment(c, authCtx, req.ShipmentID)
	if !ok {
		return
	}

	result, err := h.studioService.Preview(
		c.Request.Context(),
		&formulatemplatestudioservice.PreviewRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Source:     req.Source,
			Inputs:     req.Inputs,
			ShipmentID: shipmentID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

type scaffoldTestRequest struct {
	Source     string             `json:"source"`
	Inputs     tarifftypes.Inputs `json:"inputs"`
	ShipmentID string             `json:"shipmentId"`
	Name       string             `json:"name"`
}

// @Summary Write a test block from a real shipment
// @Description Rates the source on the shipment and returns a test block whose givens are the shipment's facts and whose expectations are the bill it produced, for the Studio's "Save as test".
// @ID scaffoldFormulaTemplateTest
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body scaffoldTestRequest true "The source, the shipment and the test's name"
// @Success 200 {object} formulatemplatestudioservice.ScaffoldTestResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/scaffold-test [post]
func (h *Handler) scaffoldTest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req scaffoldTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	shipmentID, ok := h.optionalShipment(c, authCtx, req.ShipmentID)
	if !ok {
		return
	}
	if shipmentID == nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Choose the shipment the test is written from",
		))
		return
	}

	result, err := h.studioService.ScaffoldTest(
		c.Request.Context(),
		&formulatemplatestudioservice.ScaffoldTestRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Source:     req.Source,
			Inputs:     req.Inputs,
			ShipmentID: *shipmentID,
			Name:       req.Name,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

// @Summary The context the browser checks formulas with
// @Description Returns what tariff.check needs besides the source: the organization's settings, the shapes of its rate tables, its accessorial charge codes, and the source of the newest approved version of every library a formula may import.
// @ID getFormulaTemplateContext
// @Tags Formula Templates
// @Produce json
// @Success 200 {object} formulatemplatestudioservice.StudioContext
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/context [get]
func (h *Handler) getContext(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.studioService.StudioContext(
		c.Request.Context(),
		pagination.FromAuthAsUser(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

type contextCellsRequest struct {
	Needs []rules.TariffNeed `json:"needs"`
	Whole bool               `json:"whole"`
}

// @Summary Rate table cells for the browser's tests
// @Description Returns the cells of the rate table key combinations a formula's tests read in the browser (each need from tariff.test), or with whole the named tables entire, with the stamp of the organization's rate tables they were read at.
// @ID getFormulaTemplateContextCells
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body contextCellsRequest true "The needs tariff.test answered, and whether to load their tables whole"
// @Success 200 {object} tariff.TableCells
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/context/cells [post]
func (h *Handler) getContextCells(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req contextCellsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.studioService.StudioCells(
		c.Request.Context(),
		&formulatemplatestudioservice.StudioCellsRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Needs:      req.Needs,
			Whole:      req.Whole,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) optionalShipment(
	c *gin.Context,
	authCtx *authctx.AuthContext,
	raw string,
) (*pulid.ID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}

	shipmentID, err := pulid.MustParse(raw)
	if err != nil {
		h.eh.HandleError(c, err)
		return nil, false
	}
	if !h.allowShipmentRead(c, authCtx) {
		return nil, false
	}
	return &shipmentID, true
}

func (h *Handler) allowShipmentRead(c *gin.Context, authCtx *authctx.AuthContext) bool {
	result, err := h.permEngine.Check(
		c.Request.Context(),
		middleware.BuildPermissionCheckRequest(
			authCtx,
			permission.ResourceShipment.String(),
			permission.OpRead,
		),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return false
	}

	if !result.Allowed {
		h.eh.HandleError(c, errortypes.NewAuthorizationError(
			"You don't have permission to read shipments",
		))
		return false
	}

	return true
}
